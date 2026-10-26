package api

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/keywordwatch"
	"github.com/jamesread/japella/internal/rbac"
)

func (s *ControlApi) canManageKeywordWatches(ctx context.Context, socialAccountID uint32) error {
	au := s.getAuthenticatedUser(ctx)
	if au == nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	canViewAll := au.HasPermission(rbac.PermissionSocialAccountsViewAll) || (au.RBAC != nil && au.RBAC.IsSuperuser)
	if canViewAll {
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
	created, updated := "", ""
	if !w.CreatedAt.IsZero() {
		created = w.CreatedAt.Format(time.RFC3339)
	}
	if !w.UpdatedAt.IsZero() {
		updated = w.UpdatedAt.Format(time.RFC3339)
	}
	return &controlv1.KeywordWatch{
		Id:              w.ID,
		SocialAccountId: w.SocialAccountID,
		Keyword:         w.Keyword,
		Enabled:         w.Enabled,
		Created:         created,
		Updated:         updated,
	}
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
	if st.BackoffUntil.Valid {
		out.BackoffUntil = st.BackoffUntil.Time.Format(time.RFC3339)
	}
	if st.LastSuccessAt.Valid {
		out.LastSuccessAt = st.LastSuccessAt.Time.Format(time.RFC3339)
	}
	if st.LastErrorAt.Valid {
		out.LastErrorAt = st.LastErrorAt.Time.Format(time.RFC3339)
	}
	if st.LastPollAt.Valid {
		out.LastPollAt = st.LastPollAt.Time.Format(time.RFC3339)
	}
	return out
}

func (s *ControlApi) ListKeywordWatches(ctx context.Context, req *connect.Request[controlv1.ListKeywordWatchesRequest]) (*connect.Response[controlv1.ListKeywordWatchesResponse], error) {
	if req.Msg.SocialAccountId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("social_account_id is required"))
	}
	if err := s.canManageKeywordWatches(ctx, req.Msg.SocialAccountId); err != nil {
		return nil, err
	}

	sa, err := s.DB.GetSocialAccount(req.Msg.SocialAccountId)
	if err != nil || sa == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("social account not found"))
	}

	watches, err := s.DB.ListKeywordWatches(req.Msg.SocialAccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list keyword watches"))
	}
	st, _ := s.DB.GetKeywordWatchState(req.Msg.SocialAccountId)
	matches, _ := s.DB.ListRecentKeywordMatches(req.Msg.SocialAccountId, 20)

	keywordByWatch := map[uint32]string{}
	for _, w := range watches {
		keywordByWatch[w.ID] = w.Keyword
	}

	outWatches := make([]*controlv1.KeywordWatch, 0, len(watches))
	for _, w := range watches {
		outWatches = append(outWatches, keywordWatchToProto(w))
	}
	outMatches := make([]*controlv1.KeywordMatch, 0, len(matches))
	for _, m := range matches {
		outMatches = append(outMatches, &controlv1.KeywordMatch{
			Id:                m.ID,
			KeywordWatchId:    m.KeywordWatchID,
			Keyword:           keywordByWatch[m.KeywordWatchID],
			Connector:         m.Connector,
			RemoteId:          m.RemoteID,
			RemoteUrl:         m.RemoteURL,
			AuthorId:          m.AuthorID,
			AuthorHandle:      m.AuthorHandle,
			ContentSnippet:    m.ContentSnippet,
			MatchedAt:         m.MatchedAt.Format(time.RFC3339),
			AcquisitionMode:   m.AcquisitionMode,
			WebhookDispatched: m.WebhookDispatchedAt.Valid,
		})
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
	if s.cc == nil || sa == nil {
		return false, false, fidelity
	}
	c := s.cc.Get(sa.Connector)
	if c == nil {
		return false, false, fidelity
	}
	csa := toConnectorSA(sa)
	if _, ok := c.(connector.ConnectorWithKeywordSearch); ok {
		supportsSearch = true
	}
	if watcher, ok := c.(connector.ConnectorWithKeywordWatch); ok {
		keywords := make([]string, 0, len(watches))
		for _, w := range watches {
			if w.Enabled {
				keywords = append(keywords, w.Keyword)
			}
		}
		fidelity = watcher.NativeWatchFidelity(csa, keywords)
		supportsNative = watcher.PreferNativeWatch(csa, keywords) || fidelity != connector.NativeFidelitySearchOnly
		// PreferNativeWatch may be false while still advertising hashtag_only capability once keywords fit.
		if len(keywords) == 0 {
			supportsNative = true // connector implements the interface
			fidelity = watcher.NativeWatchFidelity(csa, []string{"#example"})
		}
	}
	return supportsSearch, supportsNative, fidelity
}

func (s *ControlApi) CreateKeywordWatch(ctx context.Context, req *connect.Request[controlv1.CreateKeywordWatchRequest]) (*connect.Response[controlv1.CreateKeywordWatchResponse], error) {
	if req.Msg.SocialAccountId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("social_account_id is required"))
	}
	if err := s.canManageKeywordWatches(ctx, req.Msg.SocialAccountId); err != nil {
		return nil, err
	}
	sa, err := s.DB.GetSocialAccount(req.Msg.SocialAccountId)
	if err != nil || sa == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("social account not found"))
	}
	if s.cc != nil {
		c := s.cc.Get(sa.Connector)
		if c == nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("connector not available"))
		}
		if _, ok := c.(connector.ConnectorWithKeywordSearch); !ok {
			if _, ok := c.(connector.ConnectorWithKeywordWatch); !ok {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("connector does not support keyword watches"))
			}
		}
	}

	display, norm, err := keywordwatch.NormalizeKeyword(req.Msg.Keyword)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	count, err := s.DB.CountKeywordWatches(req.Msg.SocialAccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count watches"))
	}
	if count >= keywordwatch.MaxKeywordsPerAccount {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("maximum of %d keywords per account", keywordwatch.MaxKeywordsPerAccount))
	}

	id, err := s.DB.CreateKeywordWatch(req.Msg.SocialAccountId, display, norm, true)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create keyword watch: %w", err))
	}
	if s.keywordWatch != nil {
		s.keywordWatch.OnWatchesChanged(req.Msg.SocialAccountId)
	}
	return connect.NewResponse(&controlv1.CreateKeywordWatchResponse{Id: id}), nil
}

func (s *ControlApi) UpdateKeywordWatch(ctx context.Context, req *connect.Request[controlv1.UpdateKeywordWatchRequest]) (*connect.Response[controlv1.UpdateKeywordWatchResponse], error) {
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	existing, err := s.DB.GetKeywordWatch(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load keyword watch"))
	}
	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("keyword watch not found"))
	}
	if err := s.canManageKeywordWatches(ctx, existing.SocialAccountID); err != nil {
		return nil, err
	}
	display, norm, err := keywordwatch.NormalizeKeyword(req.Msg.Keyword)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.DB.UpdateKeywordWatch(req.Msg.Id, display, norm, req.Msg.Enabled); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update keyword watch"))
	}
	if s.keywordWatch != nil {
		s.keywordWatch.OnWatchesChanged(existing.SocialAccountID)
	}
	return connect.NewResponse(&controlv1.UpdateKeywordWatchResponse{}), nil
}

func (s *ControlApi) DeleteKeywordWatch(ctx context.Context, req *connect.Request[controlv1.DeleteKeywordWatchRequest]) (*connect.Response[controlv1.DeleteKeywordWatchResponse], error) {
	if req.Msg.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}
	existing, err := s.DB.GetKeywordWatch(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load keyword watch"))
	}
	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("keyword watch not found"))
	}
	if err := s.canManageKeywordWatches(ctx, existing.SocialAccountID); err != nil {
		return nil, err
	}
	if err := s.DB.DeleteKeywordWatch(req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete keyword watch"))
	}
	if s.keywordWatch != nil {
		s.keywordWatch.OnWatchesChanged(existing.SocialAccountID)
	}
	return connect.NewResponse(&controlv1.DeleteKeywordWatchResponse{}), nil
}
