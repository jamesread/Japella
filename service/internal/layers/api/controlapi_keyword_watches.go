package api

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/keywordwatch"
)

func (s *ControlApi) canManageKeywordWatches(ctx context.Context, socialAccountID uint32) error {
	au, err := s.requireAuthenticatedUser(ctx)
	if err != nil {
		return err
	}
	if canViewAllSocialAccounts(au) {
		return nil
	}
	if s.DB.CanUserAccessSocialAccount(au.User.ID, socialAccountID, "manage") {
		return nil
	}
	return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("manage access required for keyword watches"))
}

func keywordWatchToProto(w *db.KeywordWatch) *controlv1.KeywordWatch {
	if w == nil {
		return nil
	}
	return &controlv1.KeywordWatch{
		Id:              w.ID,
		SocialAccountId: w.SocialAccountID,
		Keyword:         w.Keyword,
		Enabled:         w.Enabled,
		Created:         formatNonZeroTime(w.CreatedAt),
		Updated:         formatNonZeroTime(w.UpdatedAt),
	}
}

func formatNonZeroTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func keywordWatchStateToProto(st *db.KeywordWatchState) *controlv1.KeywordWatchState {
	if st == nil {
		return nil
	}
	out := &controlv1.KeywordWatchState{
		SocialAccountId: st.SocialAccountID,
		AcquisitionMode: st.AcquisitionMode,
		NativeFidelity:  st.NativeFidelity,
		LastError:       st.LastError,
	}
	out.BackoffUntil = nullTimeRFC3339(st.BackoffUntil)
	out.LastSuccessAt = nullTimeRFC3339(st.LastSuccessAt)
	out.LastErrorAt = nullTimeRFC3339(st.LastErrorAt)
	out.LastPollAt = nullTimeRFC3339(st.LastPollAt)
	return out
}

func nullTimeRFC3339(nt sql.NullTime) string {
	if !nt.Valid {
		return ""
	}
	return nt.Time.Format(time.RFC3339)
}

func keywordMatchToProto(m *db.KeywordMatch, keyword string) *controlv1.KeywordMatch {
	return &controlv1.KeywordMatch{
		Id:                m.ID,
		KeywordWatchId:    m.KeywordWatchID,
		Keyword:           keyword,
		Connector:         m.Connector,
		RemoteId:          m.RemoteID,
		RemoteUrl:         m.RemoteURL,
		AuthorId:          m.AuthorID,
		AuthorHandle:      m.AuthorHandle,
		ContentSnippet:    m.ContentSnippet,
		MatchedAt:         m.MatchedAt.Format(time.RFC3339),
		AcquisitionMode:   m.AcquisitionMode,
		WebhookDispatched: m.WebhookDispatchedAt.Valid,
	}
}

func (s *ControlApi) ListKeywordWatches(ctx context.Context, req *connect.Request[controlv1.ListKeywordWatchesRequest]) (*connect.Response[controlv1.ListKeywordWatchesResponse], error) {
	sa, err := s.managedKeywordWatchAccount(ctx, req.Msg.SocialAccountId)
	if err != nil {
		return nil, err
	}
	return s.listKeywordWatchesForAccount(sa)
}

func (s *ControlApi) listKeywordWatchesForAccount(sa *db.SocialAccount) (*connect.Response[controlv1.ListKeywordWatchesResponse], error) {
	watches, err := s.DB.ListKeywordWatches(sa.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list keyword watches"))
	}
	st, _ := s.DB.GetKeywordWatchState(sa.ID)
	matches, _ := s.DB.ListRecentKeywordMatches(sa.ID, 20)

	keywordByWatch := map[uint32]string{}
	outWatches := make([]*controlv1.KeywordWatch, 0, len(watches))
	for _, w := range watches {
		keywordByWatch[w.ID] = w.Keyword
		outWatches = append(outWatches, keywordWatchToProto(w))
	}
	outMatches := make([]*controlv1.KeywordMatch, 0, len(matches))
	for _, m := range matches {
		outMatches = append(outMatches, keywordMatchToProto(m, keywordByWatch[m.KeywordWatchID]))
	}

	supportsSearch, supportsNative, fidelity := s.keywordWatchCapabilities(sa, watches)
	return connect.NewResponse(&controlv1.ListKeywordWatchesResponse{
		Watches:        outWatches,
		State:          keywordWatchStateToProto(st),
		RecentMatches:  outMatches,
		SupportsSearch: supportsSearch,
		SupportsNative: supportsNative,
		NativeFidelity: fidelity,
	}), nil
}

func (s *ControlApi) keywordWatchCapabilities(sa *db.SocialAccount, watches []*db.KeywordWatch) (supportsSearch, supportsNative bool, fidelity string) {
	fidelity = connector.NativeFidelitySearchOnly
	c := s.keywordWatchConnector(sa)
	if c == nil {
		return false, false, fidelity
	}
	csa := toConnectorSA(sa)
	if _, ok := c.(connector.ConnectorWithKeywordSearch); ok {
		supportsSearch = true
	}
	return supportsSearch, s.nativeKeywordWatchCaps(c, csa, watches, &fidelity), fidelity
}

func (s *ControlApi) keywordWatchConnector(sa *db.SocialAccount) connector.BaseConnector {
	if s.cc == nil || sa == nil {
		return nil
	}
	return s.cc.Get(sa.Connector)
}

func (s *ControlApi) nativeKeywordWatchCaps(c connector.BaseConnector, csa *connector.SocialAccount, watches []*db.KeywordWatch, fidelity *string) bool {
	watcher, ok := c.(connector.ConnectorWithKeywordWatch)
	if !ok {
		return false
	}
	keywords := enabledWatchKeywords(watches)
	*fidelity = watcher.NativeWatchFidelity(csa, keywords)
	supportsNative := watcher.PreferNativeWatch(csa, keywords) || *fidelity != connector.NativeFidelitySearchOnly
	if len(keywords) == 0 {
		supportsNative = true
		*fidelity = watcher.NativeWatchFidelity(csa, []string{"#example"})
	}
	return supportsNative
}

func enabledWatchKeywords(watches []*db.KeywordWatch) []string {
	keywords := make([]string, 0, len(watches))
	for _, w := range watches {
		if w.Enabled {
			keywords = append(keywords, w.Keyword)
		}
	}
	return keywords
}

func (s *ControlApi) ensureKeywordWatchConnector(sa *db.SocialAccount) error {
	if s.cc == nil {
		return nil
	}
	c := s.cc.Get(sa.Connector)
	if c == nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("connector not available"))
	}
	if _, ok := c.(connector.ConnectorWithKeywordSearch); ok {
		return nil
	}
	if _, ok := c.(connector.ConnectorWithKeywordWatch); ok {
		return nil
	}
	return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("connector does not support keyword watches"))
}

func (s *ControlApi) managedKeywordWatchAccount(ctx context.Context, socialAccountID uint32) (*db.SocialAccount, error) {
	if socialAccountID == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("social_account_id is required"))
	}
	if err := s.canManageKeywordWatches(ctx, socialAccountID); err != nil {
		return nil, err
	}
	sa, err := s.DB.GetSocialAccount(socialAccountID)
	if err != nil || sa == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("social account not found"))
	}
	return sa, nil
}

func (s *ControlApi) CreateKeywordWatch(ctx context.Context, req *connect.Request[controlv1.CreateKeywordWatchRequest]) (*connect.Response[controlv1.CreateKeywordWatchResponse], error) {
	sa, err := s.managedKeywordWatchAccount(ctx, req.Msg.SocialAccountId)
	if err != nil {
		return nil, err
	}
	if err := s.ensureKeywordWatchConnector(sa); err != nil {
		return nil, err
	}
	return s.createKeywordWatchRecord(sa.ID, req.Msg.Keyword)
}

func (s *ControlApi) createKeywordWatchRecord(socialAccountID uint32, keyword string) (*connect.Response[controlv1.CreateKeywordWatchResponse], error) {
	display, norm, err := keywordwatch.NormalizeKeyword(keyword)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	count, err := s.DB.CountKeywordWatches(socialAccountID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count watches"))
	}
	if count >= keywordwatch.MaxKeywordsPerAccount {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("maximum of %d keywords per account", keywordwatch.MaxKeywordsPerAccount))
	}
	id, err := s.DB.CreateKeywordWatch(socialAccountID, display, norm, true)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create keyword watch: %w", err))
	}
	s.notifyKeywordWatchesChanged(socialAccountID)
	return connect.NewResponse(&controlv1.CreateKeywordWatchResponse{Id: id}), nil
}

func (s *ControlApi) notifyKeywordWatchesChanged(socialAccountID uint32) {
	if s.keywordWatch != nil {
		s.keywordWatch.OnWatchesChanged(socialAccountID)
	}
}

func (s *ControlApi) loadManagedKeywordWatch(ctx context.Context, id uint32) (*db.KeywordWatch, error) {
	if id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	existing, err := s.DB.GetKeywordWatch(id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load keyword watch"))
	}
	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("keyword watch not found"))
	}
	if err := s.canManageKeywordWatches(ctx, existing.SocialAccountID); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *ControlApi) UpdateKeywordWatch(ctx context.Context, req *connect.Request[controlv1.UpdateKeywordWatchRequest]) (*connect.Response[controlv1.UpdateKeywordWatchResponse], error) {
	existing, err := s.loadManagedKeywordWatch(ctx, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	display, norm, err := keywordwatch.NormalizeKeyword(req.Msg.Keyword)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.DB.UpdateKeywordWatch(req.Msg.Id, display, norm, req.Msg.Enabled); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update keyword watch"))
	}
	s.notifyKeywordWatchesChanged(existing.SocialAccountID)
	return connect.NewResponse(&controlv1.UpdateKeywordWatchResponse{}), nil
}

func (s *ControlApi) DeleteKeywordWatch(ctx context.Context, req *connect.Request[controlv1.DeleteKeywordWatchRequest]) (*connect.Response[controlv1.DeleteKeywordWatchResponse], error) {
	existing, err := s.loadManagedKeywordWatch(ctx, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	if err := s.DB.DeleteKeywordWatch(req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete keyword watch"))
	}
	s.notifyKeywordWatchesChanged(existing.SocialAccountID)
	return connect.NewResponse(&controlv1.DeleteKeywordWatchResponse{}), nil
}
