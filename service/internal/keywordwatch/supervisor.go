package keywordwatch

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	log "github.com/sirupsen/logrus"
)

// Supervisor manages long-lived native keyword watch sessions per account.
type Supervisor struct {
	service *Service

	mu      sync.Mutex
	handles map[uint32]connector.WatchHandle
	modes   map[uint32]string // account -> native|search
}

func NewSupervisor(service *Service) *Supervisor {
	return &Supervisor{
		service: service,
		handles: map[uint32]connector.WatchHandle{},
		modes:   map[uint32]string{},
	}
}

func (sup *Supervisor) Counts() (native, other int) {
	sup.mu.Lock()
	defer sup.mu.Unlock()
	native = len(sup.handles)
	for _, mode := range sup.modes {
		if mode != db.WatchModeNative {
			other++
		}
	}
	return native, other
}

func (sup *Supervisor) StopAccount(accountID uint32) {
	sup.mu.Lock()
	h := sup.handles[accountID]
	delete(sup.handles, accountID)
	sup.modes[accountID] = db.WatchModeSearch
	sup.mu.Unlock()
	if h != nil {
		h.Stop()
	}
}

func (sup *Supervisor) StopAll() {
	sup.mu.Lock()
	ids := make([]uint32, 0, len(sup.handles))
	for id := range sup.handles {
		ids = append(ids, id)
	}
	sup.mu.Unlock()
	for _, id := range ids {
		sup.StopAccount(id)
	}
}

func watchKeywords(watches []*db.KeywordWatch) []string {
	keywords := make([]string, 0, len(watches))
	for _, w := range watches {
		keywords = append(keywords, w.Keyword)
	}
	return keywords
}

func (sup *Supervisor) asKeywordWatcher(accountID uint32, c connector.BaseConnector) (connector.ConnectorWithKeywordWatch, bool) {
	if c == nil {
		sup.fallbackSearch(accountID, db.WatchFidelitySearchOnly, "connector not available")
		return nil, false
	}
	watcher, ok := c.(connector.ConnectorWithKeywordWatch)
	if !ok {
		sup.fallbackSearch(accountID, db.WatchFidelitySearchOnly, "native watch unsupported")
		return nil, false
	}
	return watcher, true
}

func (sup *Supervisor) nativeWatcher(accountID uint32) (*db.SocialAccount, connector.ConnectorWithKeywordWatch, bool) {
	sa, err := sup.service.DB.GetSocialAccount(accountID)
	if err != nil || sa == nil || !sa.Active {
		sup.StopAccount(accountID)
		return nil, nil, false
	}
	watcher, ok := sup.asKeywordWatcher(accountID, sup.service.Connectors.Get(sa.Connector))
	if !ok {
		return nil, nil, false
	}
	return sa, watcher, true
}

func (sup *Supervisor) handleNativeStartError(accountID uint32, fidelity string, err error) bool {
	if rl := connector.AsRateLimited(err); rl != nil {
		st := &db.KeywordWatchState{
			SocialAccountID: accountID,
			AcquisitionMode: db.WatchModeDegraded,
			NativeFidelity:  fidelity,
		}
		sup.service.enterBackoff(st, rl, "native rate limited")
		_ = sup.service.DB.UpsertKeywordWatchState(st)
		sup.mu.Lock()
		sup.modes[accountID] = db.WatchModeDegraded
		sup.mu.Unlock()
		return true
	}
	if err != nil {
		log.Warnf("keyword watch native start account=%d: %v", accountID, err)
		sup.fallbackSearch(accountID, fidelity, err.Error())
		return true
	}
	return false
}

func (sup *Supervisor) markNativeRunning(accountID uint32, fidelity string, handle connector.WatchHandle) {
	_ = sup.service.DB.EnsureKeywordWatchState(accountID)
	st, _ := sup.service.DB.GetKeywordWatchState(accountID)
	if st == nil {
		st = &db.KeywordWatchState{SocialAccountID: accountID}
	}
	st.AcquisitionMode = db.WatchModeNative
	st.NativeFidelity = fidelity
	st.LastSuccessAt = db.NullTime(time.Now())
	st.LastError = ""
	st.BackoffUntil = db.NullTime(time.Time{})
	_ = sup.service.DB.UpsertKeywordWatchState(st)

	sup.mu.Lock()
	sup.handles[accountID] = handle
	sup.modes[accountID] = db.WatchModeNative
	sup.mu.Unlock()
}

func (sup *Supervisor) startNativeWatch(ctx context.Context, sa *db.SocialAccount, watcher connector.ConnectorWithKeywordWatch, watches []*db.KeywordWatch, fidelity string, keywords []string) {
	csa := toConnectorSA(sa)
	sup.StopAccount(sa.ID)
	watchCopy := append([]*db.KeywordWatch(nil), watches...)
	sink := func(post *connector.RemotePost, matched []string) {
		sup.service.HandleNativeMatch(sa, watchCopy, post, matched)
	}
	handle, err := watcher.StartKeywordWatch(ctx, csa, keywords, sink)
	if sup.handleNativeStartError(sa.ID, fidelity, err) {
		return
	}
	sup.markNativeRunning(sa.ID, fidelity, handle)
}

func serviceReady(sup *Supervisor) bool {
	return sup.service != nil && sup.service.DB != nil && sup.service.Connectors != nil
}

// EnsureAccount starts or restarts native watch when PreferNativeWatch is true.
func (sup *Supervisor) EnsureAccount(ctx context.Context, accountID uint32, watches []*db.KeywordWatch) {
	if !serviceReady(sup) {
		return
	}
	sa, watcher, ok := sup.nativeWatcher(accountID)
	if !ok {
		return
	}
	keywords := watchKeywords(watches)
	csa := toConnectorSA(sa)
	fidelity := watcher.NativeWatchFidelity(csa, keywords)
	if !watcher.PreferNativeWatch(csa, keywords) {
		sup.fallbackSearch(accountID, fidelity, "")
		return
	}
	sup.startNativeWatch(ctx, sa, watcher, watches, fidelity, keywords)
}

func (sup *Supervisor) fallbackSearch(accountID uint32, fidelity, errMsg string) {
	sup.StopAccount(accountID)
	_ = sup.service.DB.EnsureKeywordWatchState(accountID)
	st, _ := sup.service.DB.GetKeywordWatchState(accountID)
	if st == nil {
		st = &db.KeywordWatchState{SocialAccountID: accountID}
	}
	st.AcquisitionMode = db.WatchModeSearch
	if fidelity == "" {
		fidelity = db.WatchFidelitySearchOnly
	}
	st.NativeFidelity = fidelity
	if errMsg != "" {
		st.LastError = truncateErr(errMsg)
		st.LastErrorAt = db.NullTime(time.Now())
	}
	_ = sup.service.DB.UpsertKeywordWatchState(st)
	sup.mu.Lock()
	sup.modes[accountID] = db.WatchModeSearch
	sup.mu.Unlock()
}

// NotifyStreamFailure moves an account from native to search with backoff.
func (sup *Supervisor) NotifyStreamFailure(accountID uint32, err error) {
	sup.StopAccount(accountID)
	st, _ := sup.service.DB.GetKeywordWatchState(accountID)
	if st == nil {
		st = &db.KeywordWatchState{SocialAccountID: accountID, NativeFidelity: db.WatchFidelityHashtagOnly}
	}
	if rl := connector.AsRateLimited(err); rl != nil {
		sup.service.enterBackoff(st, rl, "native stream rate limited")
	} else {
		st.AcquisitionMode = db.WatchModeDegraded
		wait := 10 * time.Minute
		st.BackoffUntil = db.NullTime(time.Now().Add(wait))
		st.LastError = truncateErr(err.Error())
		st.LastErrorAt = db.NullTime(time.Now())
	}
	_ = sup.service.DB.UpsertKeywordWatchState(st)
}

func hashtagAppearsIn(content, kw string) bool {
	if !IsHashtagKeyword(kw) {
		return false
	}
	plain := strings.ToLower(StripHTML(content))
	tag := strings.ToLower(HashtagTag(kw))
	return strings.Contains(plain, "#"+tag) || strings.Contains(plain, tag)
}

// MatchKeywordsAgainstPost returns which configured keywords appear in post content.
func MatchKeywordsAgainstPost(content string, keywords []string) []string {
	out := make([]string, 0)
	for _, kw := range keywords {
		if ContainsKeyword(content, kw) || hashtagAppearsIn(content, kw) {
			out = append(out, kw)
		}
	}
	return out
}
