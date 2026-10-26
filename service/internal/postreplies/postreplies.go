// Package postreplies decides when to refresh the reply thread under a
// published post and turns stored (flat) replies into a conversation tree.
package postreplies

import (
	"sort"
	"strings"

	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
)

// MaxReplies caps how many replies are stored per post.
const MaxReplies = 500

// ShouldFetch reports whether the reply thread needs refreshing, given the
// stored engagement row (may be nil) and the freshly fetched replies counter.
// Threads are fetched once there are replies, then again whenever the counter
// moves, so steady posts cost no extra requests.
func ShouldFetch(stored *db.PostEngagement, replyCount uint32) bool {
	if stored == nil || !stored.RepliesFetchedAt.Valid {
		return replyCount > 0
	}
	return stored.RepliesFetchedCount != replyCount
}

// ToDB converts a connector fetch into rows to store, dropping replies
// without an id and duplicates, and capping the total at MaxReplies.
func ToDB(postID uint32, fetched *connector.PostReplies) []*db.PostReply {
	if fetched == nil {
		return nil
	}

	seen := map[string]bool{}
	rows := make([]*db.PostReply, 0, len(fetched.Replies))
	for _, r := range fetched.Replies {
		if r == nil || r.RemoteID == "" || seen[r.RemoteID] || r.RemoteID == fetched.RootRemoteID {
			continue
		}
		if len(rows) >= MaxReplies {
			break
		}
		seen[r.RemoteID] = true

		parent := r.ParentRemoteID
		if parent == fetched.RootRemoteID {
			// Store top-level replies with an empty parent so the tree does
			// not depend on how the platform spells the root id.
			parent = ""
		}

		rows = append(rows, &db.PostReply{
			PostID:          postID,
			RemoteID:        r.RemoteID,
			ParentRemoteID:  parent,
			RemoteURL:       r.RemoteURL,
			AuthorName:      truncate(r.AuthorName, 255),
			AuthorHandle:    truncate(r.AuthorHandle, 255),
			AuthorAvatarURL: truncate(r.AuthorAvatarURL, 1024),
			Content:         strings.TrimSpace(r.Content),
			PostedAt:        db.NullTimeOf(r.PostedDate),
		})
	}
	return rows
}

// Node is a reply and the replies to it.
type Node struct {
	Reply   *db.PostReply
	Replies []*Node
}

// BuildThread arranges stored replies into a tree. Replies whose parent is
// missing (deleted, or beyond the fetch depth) are shown at the top level, as
// are replies that would otherwise form a cycle in malformed data. Siblings
// are ordered oldest first.
func BuildThread(rows []*db.PostReply) []*Node {
	nodes := make(map[string]*Node, len(rows))
	parentOf := make(map[string]string, len(rows))
	for _, r := range rows {
		nodes[r.RemoteID] = &Node{Reply: r}
		parentOf[r.RemoteID] = r.ParentRemoteID
	}

	roots := make([]*Node, 0)
	for _, r := range rows {
		n := nodes[r.RemoteID]
		parent, ok := nodes[r.ParentRemoteID]
		if !ok || isAncestorOrSelf(r.RemoteID, r.ParentRemoteID, parentOf) {
			roots = append(roots, n)
			continue
		}
		parent.Replies = append(parent.Replies, n)
	}

	sortNodes(roots)
	return roots
}

// isAncestorOrSelf reports whether id appears on the parent chain starting at from.
func isAncestorOrSelf(id, from string, parentOf map[string]string) bool {
	for steps := 0; from != "" && steps <= len(parentOf); steps++ {
		if from == id {
			return true
		}
		from = parentOf[from]
	}
	return false
}

func sortNodes(nodes []*Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		a, b := nodes[i].Reply.PostedAt, nodes[j].Reply.PostedAt
		if a.Valid != b.Valid {
			return a.Valid
		}
		return a.Time.Before(b.Time)
	})
	for _, n := range nodes {
		sortNodes(n.Replies)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.ToValidUTF8(s[:n], "")
}
