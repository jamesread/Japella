package dummy

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"github.com/jamesread/japella/internal/connector"
)

// Simulated engagement for posts published through the dummy connector, so
// the post_engagement job and its UI can be exercised without real networks.
var (
	engagementMu    sync.Mutex
	engagementByURL = map[string]*connector.PostEngagement{}
)

// FetchPostEngagement returns counters that grow a little on every call.
func (c *Connector) FetchPostEngagement(sa *connector.SocialAccount, remoteID string, postURL string) (*connector.PostEngagement, error) {
	key := postURL
	if key == "" {
		key = remoteID
	}
	if !strings.HasPrefix(key, "https://dummy.local/posts/") && !strings.HasPrefix(key, "dummy-") {
		return nil, fmt.Errorf("not a dummy post: %q", key)
	}

	engagementMu.Lock()
	defer engagementMu.Unlock()

	e, ok := engagementByURL[key]
	if !ok {
		e = &connector.PostEngagement{}
		engagementByURL[key] = e
	}

	e.Likes += uint32(rand.Intn(4))
	if rand.Intn(3) == 0 {
		e.Reposts++
	}
	if rand.Intn(4) == 0 {
		e.Replies++
	}
	if rand.Intn(10) == 0 {
		e.Quotes++
	}
	e.Impressions += uint32(5 + rand.Intn(40))

	out := *e
	return &out, nil
}
