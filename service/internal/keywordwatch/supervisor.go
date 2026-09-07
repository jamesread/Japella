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

// EnsureAccount starts or restarts native watch when PreferNativeWatch is true.
func (sup *Supervisor) EnsureAccount(ctx context.Context, accountID uint32, watches []*db.KeywordWatch) {
	if sup.service == nil || sup.service.DB == nil || sup.service.Connectors == nil {
		return
	}
	sa, err := sup.service.DB.GetSocialAccount(accountID)
	if err != nil || sa == nil || !sa.Active {
		sup.StopAccount(accountID)
		return
	}
	c := sup.service.Connectors.Get(sa.Connector)
	if c == nil {
		sup.fallbackSearch(accountID, db.WatchFidelitySearchOnly, "connector not available")
		return
	}
	watcher, ok := c.(connector.ConnectorWithKeywordWatch)
	if !ok {
		sup.fallbackSearch(accountID, db.WatchFidelitySearchOnly, "native watch unsupported")
		return
	}

	keywords := make([]string, 0, len(watches))
	for _, w := range watches {
		keywords = append(keywords, w.Keyword)
	}
	csa := toConnectorSA(sa)
	fidelity := watcher.NativeWatchFidelity(csa, keywords)
	if !watcher.PreferNativeWatch(csa, keywords) {
		sup.fallbackSearch(accountID, fidelity, "")
		return
	}

	sup.StopAccount(accountID)

	watchCopy := append([]*db.KeywordWatch(nil), watches...)
	sink := func(post *connector.RemotePost, matched []string) {
		sup.service.HandleNativeMatch(sa, watchCopy, post, matched)
	}

	handle, err := watcher.StartKeywordWatch(ctx, csa, keywords, sink)
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
		return
	}
	if err != nil {
		log.Warnf("keyword watch native start account=%d: %v", accountID, err)
		sup.fallbackSearch(accountID, fidelity, err.Error())
		return
	}

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

// MatchKeywordsAgainstPost returns which configured keywords appear in post content.
func MatchKeywordsAgainstPost(content string, keywords []string) []string {
	out := make([]string, 0)
	for _, kw := range keywords {
		if ContainsKeyword(content, kw) {
			out = append(out, kw)
		} else if IsHashtagKeyword(kw) {
			tag := strings.ToLower(HashtagTag(kw))
			if strings.Contains(strings.ToLower(StripHTML(content)), "#"+tag) ||
				strings.Contains(strings.ToLower(StripHTML(content)), tag) {
				out = append(out, kw)
			}
		}
	}
	return out
}
