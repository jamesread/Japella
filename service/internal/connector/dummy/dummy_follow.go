package dummy

import (
	"sync"

	"github.com/jamesread/japella/internal/connector"
)

// Simulated follows, so the follow toggle can be tried locally.
var (
	followsMu sync.Mutex
	follows   = map[string]bool{}
)

func followKey(sa *connector.SocialAccount, authorID string) string {
	return sa.Identity + "\x00" + authorID
}

func (c *Connector) GetFollowState(sa *connector.SocialAccount, authorID string) (*connector.FollowState, error) {
	followsMu.Lock()
	defer followsMu.Unlock()
	return &connector.FollowState{Following: follows[followKey(sa, authorID)]}, nil
}

func (c *Connector) SetFollowing(sa *connector.SocialAccount, authorID string, follow bool) (*connector.FollowState, error) {
	followsMu.Lock()
	defer followsMu.Unlock()
	follows[followKey(sa, authorID)] = follow
	return &connector.FollowState{Following: follow}, nil
}
