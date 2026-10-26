package dummy

import (
	"sync"

	"github.com/jamesread/japella/internal/connector"
)

// Simulated like state, so the feed's like button can be tried locally.
var (
	likesMu sync.Mutex
	likes   = map[string]bool{}
)

// SetPostLiked records the like locally; counts are 1 when liked.
func (c *Connector) SetPostLiked(sa *connector.SocialAccount, remoteID string, remoteURL string, liked bool) (*connector.LikeResult, error) {
	key := remoteURL
	if key == "" {
		key = remoteID
	}

	likesMu.Lock()
	defer likesMu.Unlock()

	likes[key] = liked
	res := &connector.LikeResult{Liked: liked}
	if liked {
		res.LikeCount = 1
	}
	return res, nil
}
