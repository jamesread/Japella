package dummy

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/intconv"
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

	e.Likes += intconv.ToUint32(simRandIntn(4))
	if simRandIntn(3) == 0 {
		e.Reposts++
	}
	if simRandIntn(4) == 0 {
		e.Replies++
	}
	if simRandIntn(10) == 0 {
		e.Quotes++
	}
	e.Impressions += intconv.ToUint32(5 + simRandIntn(40))

	out := *e
	return &out, nil
}
