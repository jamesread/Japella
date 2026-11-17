package keywordwatch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/webhook"
	log "github.com/sirupsen/logrus"
)

// ConnectorResolver returns a live connector by protocol name.
type ConnectorResolver interface {
	Get(protocol string) connector.BaseConnector
}

// Service processes keyword matches (search + native) with dedupe and webhooks.
type Service struct {
	DB         *db.DB
	Connectors ConnectorResolver
	Webhook    *webhook.Dispatcher

	supervisor *Supervisor
}

func NewService(dbc *db.DB, connectors ConnectorResolver, wh *webhook.Dispatcher) *Service {
	s := &Service{
		DB:         dbc,
		Connectors: connectors,
		Webhook:    wh,
	}
	s.supervisor = NewSupervisor(s)
	return s
}

func (s *Service) Supervisor() *Supervisor {
	return s.supervisor
}

func (s *Service) cvarInt(key string, fallback int) int {
	if s.DB == nil {
		return fallback
	}
	v := int(s.DB.GetCvarInt(key))
	if v <= 0 {
		return fallback
	}
	return v
}

func (s *Service) lookback() time.Duration {
	return time.Duration(s.cvarInt(db.CvarKeys.KeywordWatchSearchLookbackSeconds, 900)) * time.Second
}

func (s *Service) maxMatchesPerPoll() int {
	return s.cvarInt(db.CvarKeys.KeywordWatchMaxMatchesPerPoll, 50)
}

func (s *Service) maxEmitsPerHour() int {
	return s.cvarInt(db.CvarKeys.KeywordWatchMaxEmitsPerHour, 100)
}

func (s *Service) nativeRetry() time.Duration {
	return time.Duration(s.cvarInt(db.CvarKeys.KeywordWatchNativeRetrySeconds, 1800)) * time.Second
}

func groupWatchesByAccount(watches []*db.KeywordWatch) map[uint32][]*db.KeywordWatch {
	byAccount := map[uint32][]*db.KeywordWatch{}
	for _, w := range watches {
		byAccount[w.SocialAccountID] = append(byAccount[w.SocialAccountID], w)
	}
	return byAccount
}

func skipSearchPoll(st *db.KeywordWatchState) bool {
	if st == nil {
		return false
	}
	if st.AcquisitionMode == db.WatchModeNative {
		// Native path owns this account; skip search unless backoff forced degraded.
		return true
	}
	return false
}

func (s *Service) pollGroupedSearches(ctx context.Context, watches []*db.KeywordWatch) {
	for accountID, accountWatches := range groupWatchesByAccount(watches) {
		if err := ctx.Err(); err != nil {
			return
		}
		st, _ := s.DB.GetKeywordWatchState(accountID)
		if skipSearchPoll(st) {
			continue
		}
		s.pollAccountSearch(ctx, accountID, accountWatches)
	}
}

// ProcessSearchPoll runs search acquisition for all accounts currently in search/degraded/evaluating mode.
func (s *Service) ProcessSearchPoll(ctx context.Context) {
	if s.DB == nil || s.Connectors == nil {
		return
	}
	watches, err := s.DB.ListEnabledKeywordWatches()
	if err != nil || len(watches) == 0 {
		return
	}
	s.pollGroupedSearches(ctx, watches)
	// Re-evaluate native after backoff for accounts stuck in search.
	s.maybePromoteToNative(ctx)
}

func (s *Service) asKeywordSearcher(accountID uint32, c connector.BaseConnector) (connector.ConnectorWithKeywordSearch, bool) {
	if c == nil {
		s.recordError(accountID, "connector not available", db.WatchFidelitySearchOnly)
		return nil, false
	}
	searcher, ok := c.(connector.ConnectorWithKeywordSearch)
	if !ok {
		s.recordError(accountID, "connector does not support keyword search", db.WatchFidelitySearchOnly)
		return nil, false
	}
	return searcher, true
}

func (s *Service) searcherForAccount(accountID uint32) (*db.SocialAccount, connector.ConnectorWithKeywordSearch, bool) {
	sa, err := s.DB.GetSocialAccount(accountID)
	if err != nil || sa == nil || !sa.Active {
		return nil, nil, false
	}
	searcher, ok := s.asKeywordSearcher(accountID, s.Connectors.Get(sa.Connector))
	if !ok {
		return nil, nil, false
	}
	return sa, searcher, true
}

func (s *Service) searchWatchState(accountID uint32) *db.KeywordWatchState {
	_ = s.DB.EnsureKeywordWatchState(accountID)
	st, _ := s.DB.GetKeywordWatchState(accountID)
	if st == nil {
		st = &db.KeywordWatchState{
			SocialAccountID: accountID,
			AcquisitionMode: db.WatchModeSearch,
			NativeFidelity:  db.WatchFidelitySearchOnly,
		}
	}
	st.AcquisitionMode = db.WatchModeSearch
	st.LastPollAt = db.NullTime(time.Now())
	return st
}

func (s *Service) handleSearchPosts(sa *db.SocialAccount, w *db.KeywordWatch, posts []*connector.RemotePost, limit int, emitted *int) {
	for _, post := range posts {
		if !ContainsKeyword(post.Content, w.Keyword) {
			continue
		}
		if s.acceptMatch(sa, w, post, db.WatchModeSearch) {
			*emitted++
		}
		if *emitted >= limit {
			return
		}
	}
}

func (s *Service) handleSearchWatchErr(st *db.KeywordWatchState, accountID uint32, keyword string, err error) (stop bool) {
	if rl := connector.AsRateLimited(err); rl != nil {
		s.enterBackoff(st, rl, "search rate limited")
		_ = s.DB.UpsertKeywordWatchState(st)
		s.supervisor.StopAccount(accountID)
		return true
	}
	st.LastError = truncateErr(err.Error())
	st.LastErrorAt = db.NullTime(time.Now())
	_ = s.DB.UpsertKeywordWatchState(st)
	log.Warnf("keyword watch search account=%d keyword=%q: %v", accountID, keyword, err)
	return false
}

func (s *Service) runSearchWatches(ctx context.Context, sa *db.SocialAccount, searcher connector.ConnectorWithKeywordSearch, st *db.KeywordWatchState, watches []*db.KeywordWatch) {
	csa := toConnectorSA(sa)
	since := time.Now().Add(-s.lookback())
	limit := s.maxMatchesPerPoll()
	emitted := 0
	for _, w := range watches {
		posts, err := searcher.SearchRecentByKeyword(ctx, csa, w.Keyword, since, limit)
		if err != nil {
			if s.handleSearchWatchErr(st, sa.ID, w.Keyword, err) {
				return
			}
			continue
		}
		s.handleSearchPosts(sa, w, posts, limit, &emitted)
		if emitted >= limit {
			break
		}
	}
	st.LastSuccessAt = db.NullTime(time.Now())
	st.LastError = ""
	_ = s.DB.UpsertKeywordWatchState(st)
}

func (s *Service) pollAccountSearch(ctx context.Context, accountID uint32, watches []*db.KeywordWatch) {
	sa, searcher, ok := s.searcherForAccount(accountID)
	if !ok {
		return
	}
	s.runSearchWatches(ctx, sa, searcher, s.searchWatchState(accountID), watches)
}

func suppressMatchWebhook(w *db.KeywordWatch, post *connector.RemotePost) bool {
	return !w.CreatedAt.IsZero() && !post.PostedAt.IsZero() && post.PostedAt.Before(w.CreatedAt)
}

func newKeywordMatch(sa *db.SocialAccount, w *db.KeywordWatch, post *connector.RemotePost, mode string, now time.Time) *db.KeywordMatch {
	return &db.KeywordMatch{
		SocialAccountID: sa.ID,
		KeywordWatchID:  w.ID,
		Connector:       sa.Connector,
		RemoteID:        post.RemoteID,
		RemoteURL:       post.RemoteURL,
		AuthorID:        post.AuthorID,
		AuthorHandle:    post.AuthorHandle,
		ContentSnippet:  TruncateSnippet(post.Content),
		MatchedAt:       now,
		PostedAt:        db.NullTime(post.PostedAt),
		AcquisitionMode: mode,
	}
}

func (s *Service) emitOrSuppressMatch(sa *db.SocialAccount, w *db.KeywordWatch, m *db.KeywordMatch, id uint32, suppress bool) {
	if suppress {
		return
	}
	if !s.allowEmit(sa.ID) {
		_ = s.DB.MarkKeywordMatchWebhookDispatched(id)
		return
	}
	s.dispatchMatch(w, m, id)
}

func prepareKeywordMatch(sa *db.SocialAccount, w *db.KeywordWatch, post *connector.RemotePost, mode string) (m *db.KeywordMatch, suppress bool) {
	now := time.Now()
	m = newKeywordMatch(sa, w, post, mode, now)
	// Suppress webhook for posts older than watch creation (backfill seed).
	suppress = suppressMatchWebhook(w, post)
	if suppress {
		m.WebhookDispatchedAt = db.NullTime(now)
	}
	return m, suppress
}

func (s *Service) acceptMatch(sa *db.SocialAccount, w *db.KeywordWatch, post *connector.RemotePost, mode string) (inserted bool) {
	if post == nil || post.RemoteID == "" {
		return false
	}
	m, suppress := prepareKeywordMatch(sa, w, post, mode)
	ok, id, err := s.DB.TryInsertKeywordMatch(m)
	if err != nil || !ok {
		return false
	}
	s.emitOrSuppressMatch(sa, w, m, id, suppress)
	return true
}

func (s *Service) emitState(accountID uint32) *db.KeywordWatchState {
	st, _ := s.DB.GetKeywordWatchState(accountID)
	if st != nil {
		return st
	}
	_ = s.DB.EnsureKeywordWatchState(accountID)
	st, _ = s.DB.GetKeywordWatchState(accountID)
	return st
}

func resetEmitHour(st *db.KeywordWatchState) {
	hour := time.Now().UTC().Truncate(time.Hour)
	if st.EmitsHourBucket.Valid && st.EmitsHourBucket.Time.Equal(hour) {
		return
	}
	st.EmitsHourBucket = db.NullTime(hour)
	st.EmitsThisHour = 0
}

func (s *Service) allowEmit(accountID uint32) bool {
	st := s.emitState(accountID)
	if st == nil {
		return true
	}
	resetEmitHour(st)
	if st.EmitsThisHour >= s.maxEmitsPerHour() {
		return false
	}
	st.EmitsThisHour++
	_ = s.DB.UpsertKeywordWatchState(st)
	return true
}

func (s *Service) dispatchMatch(w *db.KeywordWatch, m *db.KeywordMatch, matchID uint32) {
	if s.Webhook == nil {
		_ = s.DB.MarkKeywordMatchWebhookDispatched(matchID)
		return
	}
	payload := map[string]any{
		"match": map[string]any{
			"keyword":           w.Keyword,
			"keyword_watch_id":  w.ID,
			"acquisition_mode":  m.AcquisitionMode,
			"matched_at":        m.MatchedAt.UTC().Format(time.RFC3339),
			"social_account_id": m.SocialAccountID,
			"connector":         m.Connector,
			"remote_id":         m.RemoteID,
			"remote_url":        m.RemoteURL,
			"author_id":         m.AuthorID,
			"author_handle":     m.AuthorHandle,
			"content":           m.ContentSnippet,
		},
	}
	s.Webhook.Dispatch(context.Background(), webhook.EventKeywordMatched, payload)
	_ = s.DB.MarkKeywordMatchWebhookDispatched(matchID)
}

func (s *Service) enterBackoff(st *db.KeywordWatchState, rl *connector.RateLimitedError, reason string) {
	wait := 15 * time.Minute
	if rl != nil && rl.RetryAfter > 0 {
		wait = rl.RetryAfter
	}
	if wait > 2*time.Hour {
		wait = 2 * time.Hour
	}
	st.AcquisitionMode = db.WatchModeDegraded
	st.BackoffUntil = db.NullTime(time.Now().Add(wait))
	st.LastError = reason
	st.LastErrorAt = db.NullTime(time.Now())
}

func (s *Service) recordError(accountID uint32, msg, fidelity string) {
	_ = s.DB.EnsureKeywordWatchState(accountID)
	st, _ := s.DB.GetKeywordWatchState(accountID)
	if st == nil {
		st = &db.KeywordWatchState{SocialAccountID: accountID}
	}
	st.AcquisitionMode = db.WatchModeSearch
	st.NativeFidelity = fidelity
	st.LastError = truncateErr(msg)
	st.LastErrorAt = db.NullTime(time.Now())
	_ = s.DB.UpsertKeywordWatchState(st)
}

func inNativeBackoff(st *db.KeywordWatchState, now time.Time, retryAfter time.Duration) bool {
	if st.BackoffUntil.Valid && now.Before(st.BackoffUntil.Time) {
		return true
	}
	// Only retry native if we've been in search long enough since last error, or never tried.
	return st.LastErrorAt.Valid && now.Sub(st.LastErrorAt.Time) < retryAfter && st.AcquisitionMode == db.WatchModeDegraded
}

func shouldRetryNative(st *db.KeywordWatchState, now time.Time, retryAfter time.Duration) bool {
	if st.AcquisitionMode == db.WatchModeNative {
		return false
	}
	return !inNativeBackoff(st, now, retryAfter)
}

func (s *Service) promoteAccountIfReady(ctx context.Context, st *db.KeywordWatchState, now time.Time, retryAfter time.Duration) {
	if !shouldRetryNative(st, now, retryAfter) {
		return
	}
	watches, err := s.DB.ListKeywordWatches(st.SocialAccountID)
	if err != nil {
		return
	}
	enabled := filterEnabled(watches)
	if len(enabled) == 0 {
		return
	}
	s.supervisor.EnsureAccount(ctx, st.SocialAccountID, enabled)
}

func (s *Service) maybePromoteToNative(ctx context.Context) {
	states, err := s.DB.ListKeywordWatchStates()
	if err != nil {
		return
	}
	retryAfter := s.nativeRetry()
	now := time.Now()
	for _, st := range states {
		s.promoteAccountIfReady(ctx, st, now, retryAfter)
	}
}

// OnWatchesChanged refreshes native supervision for an account.
func (s *Service) OnWatchesChanged(accountID uint32) {
	watches, err := s.DB.ListKeywordWatches(accountID)
	if err != nil {
		return
	}
	enabled := filterEnabled(watches)
	if len(enabled) == 0 {
		s.supervisor.StopAccount(accountID)
		return
	}
	s.supervisor.EnsureAccount(context.Background(), accountID, enabled)
}

// StartNativeAll tries native watches for all accounts with enabled keywords.
func (s *Service) StartNativeAll(ctx context.Context) {
	watches, err := s.DB.ListEnabledKeywordWatches()
	if err != nil {
		return
	}
	byAccount := map[uint32][]*db.KeywordWatch{}
	for _, w := range watches {
		byAccount[w.SocialAccountID] = append(byAccount[w.SocialAccountID], w)
	}
	for accountID, list := range byAccount {
		s.supervisor.EnsureAccount(ctx, accountID, list)
	}
}

func watchesByNormalized(watches []*db.KeywordWatch) map[string]*db.KeywordWatch {
	byNorm := map[string]*db.KeywordWatch{}
	for _, w := range watches {
		byNorm[w.KeywordNormalized] = w
		byNorm[strings.ToLower(w.Keyword)] = w
	}
	return byNorm
}

func lookupWatch(byNorm map[string]*db.KeywordWatch, kw string) *db.KeywordWatch {
	if w := byNorm[strings.ToLower(kw)]; w != nil {
		return w
	}
	_, n, err := NormalizeKeyword(kw)
	if err != nil {
		return nil
	}
	return byNorm[n]
}

func (s *Service) acceptNativeKeywords(sa *db.SocialAccount, byNorm map[string]*db.KeywordWatch, post *connector.RemotePost, matchedKeywords []string) {
	for _, kw := range matchedKeywords {
		w := lookupWatch(byNorm, kw)
		if w == nil || !ContainsKeyword(post.Content, w.Keyword) {
			continue
		}
		s.acceptMatch(sa, w, post, db.WatchModeNative)
	}
}

// HandleNativeMatch is called from native watch sinks.
func (s *Service) HandleNativeMatch(sa *db.SocialAccount, watches []*db.KeywordWatch, post *connector.RemotePost, matchedKeywords []string) {
	if sa == nil || post == nil {
		return
	}
	s.acceptNativeKeywords(sa, watchesByNormalized(watches), post, matchedKeywords)
}

func filterEnabled(watches []*db.KeywordWatch) []*db.KeywordWatch {
	out := make([]*db.KeywordWatch, 0, len(watches))
	for _, w := range watches {
		if w != nil && w.Enabled {
			out = append(out, w)
		}
	}
	return out
}

func truncateErr(s string) string {
	if len(s) > 1000 {
		return s[:1000]
	}
	return s
}

func toConnectorSA(sa *db.SocialAccount) *connector.SocialAccount {
	return &connector.SocialAccount{
		Id:         sa.ID,
		Connector:  sa.Connector,
		Identity:   sa.Identity,
		Did:        sa.Did,
		OAuthToken: sa.OAuth2Token,
		Homeserver: sa.Homeserver,
	}
}

// StatusSummary returns a short human summary for jobs status.
func (s *Service) StatusSummary() string {
	if s.supervisor == nil {
		return "supervisor inactive"
	}
	native, search := s.supervisor.Counts()
	return fmt.Sprintf("native=%d search_or_idle=%d", native, search)
}

// CleanupMatches retains recent matches per watch.
func (s *Service) CleanupMatches() {
	keep := s.cvarInt(db.CvarKeys.KeywordWatchMatchRetention, 200)
	if err := s.DB.CleanupOldKeywordMatches(keep); err != nil {
		log.Warnf("keyword match cleanup: %v", err)
	}
}
