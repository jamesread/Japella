package dummy

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/jamesread/japella/internal/connector"
)

// Simulated reply threads, kept in step with the simulated replies counter.
var repliesByURL = map[string][]*connector.PostReply{}

var dummyRepliers = []string{"Ada", "Grace", "Linus", "Margaret", "Ken"}

var dummyReplyLines = []string{
	"Great post!",
	"I'm not sure I agree with this.",
	"Thanks for sharing.",
	"Can you say more about that?",
	"Ha, so true.",
	"Bookmarked for later.",
}

// FetchPostReplies returns one reply per simulated reply counted by
// FetchPostEngagement; roughly half answer an earlier reply rather than the post.
func (c *Connector) FetchPostReplies(sa *connector.SocialAccount, remoteID string, postURL string) (*connector.PostReplies, error) {
	key := postURL
	if key == "" {
		key = remoteID
	}

	engagementMu.Lock()
	defer engagementMu.Unlock()

	e, ok := engagementByURL[key]
	if !ok {
		return nil, fmt.Errorf("not a dummy post: %q", key)
	}

	replies := repliesByURL[key]
	for uint32(len(replies)) < e.Replies {
		n := len(replies) + 1
		parent := key
		if n > 1 && rand.Intn(2) == 0 {
			parent = replies[rand.Intn(len(replies))].RemoteID
		}
		name := dummyRepliers[rand.Intn(len(dummyRepliers))]
		replies = append(replies, &connector.PostReply{
			RemoteID:       fmt.Sprintf("%s#reply-%d", key, n),
			ParentRemoteID: parent,
			RemoteURL:      fmt.Sprintf("%s#reply-%d", key, n),
			AuthorName:     name,
			AuthorHandle:   "@" + name + "@dummy.local",
			Content:        dummyReplyLines[rand.Intn(len(dummyReplyLines))],
			PostedDate:     time.Now(),
		})
	}
	repliesByURL[key] = replies

	out := make([]*connector.PostReply, len(replies))
	copy(out, replies)
	return &connector.PostReplies{RootRemoteID: key, Replies: out}, nil
}

// ReplyToPost adds a reply from the account to a simulated thread.
func (c *Connector) ReplyToPost(sa *connector.SocialAccount, remoteID string, remoteURL string, content string) (*connector.PostReply, error) {
	key := remoteURL
	if key == "" {
		key = remoteID
	}

	engagementMu.Lock()
	defer engagementMu.Unlock()

	replies := repliesByURL[key]
	reply := &connector.PostReply{
		RemoteID:       fmt.Sprintf("%s#reply-%d", key, len(replies)+1),
		ParentRemoteID: key,
		RemoteURL:      fmt.Sprintf("%s#reply-%d", key, len(replies)+1),
		AuthorName:     sa.Identity,
		AuthorHandle:   sa.Identity,
		Content:        content,
		PostedDate:     time.Now(),
	}
	repliesByURL[key] = append(replies, reply)
	return reply, nil
}
