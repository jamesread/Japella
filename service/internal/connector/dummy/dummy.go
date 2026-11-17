package dummy

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/keywordwatch"
	"github.com/jamesread/japella/internal/utils"
)

const Protocol = "dummy"

// KeywordPool is the set of terms the simulator sprinkles into posts.
var KeywordPool = []string{
	"japella", "rust", "golang", "coffee", "news", "weather", "cats", "devops", "mastodon", "bluesky",
}

var templates = []string{
	"Simulated update about %s — posted at %s",
	"Just saw %s trending in the dummy network (%s)",
	"Thinking out loud: %s is interesting today [%s]",
	"Dummy firehose sample mentioning %s @ %s",
	"#%s random chatter from the simulator (%s)",
	"Weekend notes on %s — %s",
}

type simPost struct {
	RemoteID     string
	RemoteURL    string
	AuthorID     string
	AuthorHandle string
	Content      string
	PostedAt     time.Time
}

// Connector is a local development driver that invents posts for keyword-watch testing.
type Connector struct {
	db *db.DB
	utils.LogComponent

	mu       sync.RWMutex
	posts    []simPost
	seq      uint64
	stopSim  context.CancelFunc
	interval time.Duration
}

func (c *Connector) SetStartupConfiguration(startup *connector.ControllerStartupConfiguration) {
	if startup != nil {
		c.db = startup.DB
	}
}

// simRandIntn picks simulated demo content; it is not security-sensitive.
func simRandIntn(n int) int {
	return rand.Intn(n) //nolint:gosec // simulated demo data, not security-sensitive
}

func (c *Connector) Start() {
	c.SetPrefix("Dummy")
	if c.interval <= 0 {
		c.interval = 15 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec // cancel is stored in c.stopSim and called by Stop
	c.stopSim = cancel
	c.seedInitialPosts()
	go c.runSimulator(ctx)
	c.Logger().Infof("Dummy connector started (sim interval %s)", c.interval)
}

func (c *Connector) GetIdentity() string { return "dummy-simulator" }
func (c *Connector) GetProtocol() string { return Protocol }
func (c *Connector) GetIcon() string     { return "mdi:flask-outline" }
func (c *Connector) GetDisplayName() string {
	return "Dummy Simulator"
}

func (c *Connector) OnRefresh(socialAccount *db.SocialAccount) error {
	c.Logger().Infof("Dummy refresh for account %d (no-op)", socialAccount.ID)
	return nil
}

func (c *Connector) seedInitialPosts() {
	now := time.Now()
	for i := range 30 {
		c.appendPost(now.Add(-time.Duration(30-i) * time.Minute))
	}
}

func (c *Connector) runSimulator(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.appendPost(time.Now())
		}
	}
}

func (c *Connector) appendPost(at time.Time) {
	kw := KeywordPool[simRandIntn(len(KeywordPool))]
	tpl := templates[simRandIntn(len(templates))]
	content := fmt.Sprintf(tpl, kw, at.UTC().Format(time.RFC3339))
	// Occasionally include a second keyword.
	if simRandIntn(3) == 0 {
		other := KeywordPool[simRandIntn(len(KeywordPool))]
		if other != kw {
			content += " also mentioning " + other
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	id := fmt.Sprintf("dummy-%d", c.seq)
	authorN := (c.seq % 5) + 1
	p := simPost{
		RemoteID:     id,
		RemoteURL:    fmt.Sprintf("https://dummy.local/posts/%s", id),
		AuthorID:     fmt.Sprintf("author-%d", authorN),
		AuthorHandle: fmt.Sprintf("simuser%d", authorN),
		Content:      content,
		PostedAt:     at,
	}
	c.posts = append(c.posts, p)
	const maxKeep = 200
	if len(c.posts) > maxKeep {
		c.posts = c.posts[len(c.posts)-maxKeep:]
	}
	c.Logger().Debugf("simulated post %s: %s", id, content)
}

func (c *Connector) snapshot() []simPost {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]simPost, len(c.posts))
	copy(out, c.posts)
	return out
}

func (c *Connector) PostToWall(sa *connector.SocialAccount, message string, mediaURLs []string) *connector.PostResult {
	at := time.Now()
	c.mu.Lock()
	c.seq++
	id := fmt.Sprintf("dummy-out-%d", c.seq)
	url := fmt.Sprintf("https://dummy.local/posts/%s", id)
	c.posts = append(c.posts, simPost{
		RemoteID:     id,
		RemoteURL:    url,
		AuthorID:     "self",
		AuthorHandle: sa.Identity,
		Content:      message,
		PostedAt:     at,
	})
	c.mu.Unlock()
	return &connector.PostResult{URL: url}
}

func (c *Connector) FetchRecentPosts(sa *connector.SocialAccount) ([]*connector.FeedPost, error) {
	posts := c.snapshot()
	out := make([]*connector.FeedPost, 0, len(posts))
	for i := len(posts) - 1; i >= 0 && len(out) < 40; i-- {
		p := posts[i]
		out = append(out, &connector.FeedPost{
			Content:    p.Content,
			PostedDate: p.PostedAt,
			AuthorID:   p.AuthorID,
			AuthorName: p.AuthorHandle,
			RemoteURL:  p.RemoteURL,
			RemoteID:   p.RemoteID,
		})
	}
	return out, nil
}

func (c *Connector) PreferNativeWatch(sa *connector.SocialAccount, keywords []string) bool {
	return len(keywords) > 0
}

func (c *Connector) NativeWatchFidelity(sa *connector.SocialAccount, keywords []string) string {
	return connector.NativeFidelityExactRule
}

func (c *Connector) SearchRecentByKeyword(ctx context.Context, sa *connector.SocialAccount, keyword string, since time.Time, limit int) ([]*connector.RemotePost, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 25
	}
	return filterPostsByKeyword(c.snapshot(), keyword, since, limit), nil
}

func filterPostsByKeyword(posts []simPost, keyword string, since time.Time, limit int) []*connector.RemotePost {
	out := make([]*connector.RemotePost, 0)
	for _, p := range slices.Backward(posts) {
		if !matchesKeywordSince(p, keyword, since) {
			continue
		}
		out = append(out, simPostToRemote(p))
		if len(out) >= limit {
			break
		}
	}
	return out
}

func matchesKeywordSince(p simPost, keyword string, since time.Time) bool {
	if !since.IsZero() && p.PostedAt.Before(since) {
		return false
	}
	return keywordwatch.ContainsKeyword(p.Content, keyword)
}

func simPostToRemote(p simPost) *connector.RemotePost {
	return &connector.RemotePost{
		RemoteID:     p.RemoteID,
		RemoteURL:    p.RemoteURL,
		AuthorID:     p.AuthorID,
		AuthorHandle: p.AuthorHandle,
		Content:      p.Content,
		PostedAt:     p.PostedAt,
	}
}

type watchHandle struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func (h *watchHandle) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
	h.wg.Wait()
}

func (c *Connector) StartKeywordWatch(ctx context.Context, sa *connector.SocialAccount, keywords []string, sink connector.MatchSink) (connector.WatchHandle, error) {
	if sink == nil {
		return nil, fmt.Errorf("match sink required")
	}
	runCtx, cancel := context.WithCancel(ctx)
	h := &watchHandle{cancel: cancel}
	h.wg.Go(func() {
		c.runNativeWatch(runCtx, keywords, sink)
	})
	return h, nil
}

func (c *Connector) runNativeWatch(ctx context.Context, keywords []string, sink connector.MatchSink) {
	seen := map[string]struct{}{}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	c.emitMatchingPosts(keywords, sink, seen)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.emitMatchingPosts(keywords, sink, seen)
		}
	}
}

func (c *Connector) emitMatchingPosts(keywords []string, sink connector.MatchSink, seen map[string]struct{}) {
	for _, p := range c.snapshot() {
		if _, ok := seen[p.RemoteID]; ok {
			continue
		}
		matched := keywordwatch.MatchKeywordsAgainstPost(p.Content, keywords)
		if len(matched) == 0 {
			continue
		}
		seen[p.RemoteID] = struct{}{}
		sink(simPostToRemote(p), matched)
	}
	// Bound seen set
	if len(seen) > 500 {
		clear(seen)
	}
}

// CreateAccount registers an active dummy social account owned by userID.
func (c *Connector) CreateAccount(identity string, ownerUserID uint32) (uint32, error) {
	if c.db == nil {
		return 0, fmt.Errorf("database not configured")
	}
	created, err := c.registerDummyAccount(normalizeDummyIdentity(identity))
	if err != nil {
		return 0, err
	}
	if err := c.assignAccountOwner(created.ID, ownerUserID); err != nil {
		return created.ID, err
	}
	return created.ID, nil
}

func (c *Connector) registerDummyAccount(identity string) (*db.SocialAccount, error) {
	sa := &db.SocialAccount{
		Connector: Protocol,
		Identity:  identity,
		Did:       "did:dummy:" + identity,
		Active:    true,
		State:     "active",
	}
	if err := c.db.RegisterAccount(sa); err != nil {
		return nil, err
	}
	created, err := c.db.GetSocialAccountByConnectorAndIdentity(Protocol, identity, sa.Did)
	if err != nil || created == nil {
		return nil, fmt.Errorf("created dummy account but could not reload: %w", err)
	}
	return created, nil
}

func normalizeDummyIdentity(identity string) string {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return fmt.Sprintf("dummy-%d", time.Now().Unix()%100000)
	}
	return identity
}

func (c *Connector) assignAccountOwner(accountID, ownerUserID uint32) error {
	if ownerUserID == 0 {
		return nil
	}
	return c.db.SetSocialAccountOwner(accountID, ownerUserID)
}
