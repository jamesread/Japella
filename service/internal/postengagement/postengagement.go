// Package postengagement compares engagement snapshots for published posts
// and decides when users should be notified about new activity.
package postengagement

import (
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
)

// Counts is a plain set of engagement counters.
type Counts struct {
	Likes       uint32 `json:"likes"`
	Reposts     uint32 `json:"reposts"`
	Replies     uint32 `json:"replies"`
	Quotes      uint32 `json:"quotes"`
	Impressions uint32 `json:"impressions"`
}

// Delta is the increase in each counter since the previous snapshot. Counters
// that went down (unlikes, deleted replies) are reported as 0.
type Delta struct {
	Likes       uint32 `json:"likes"`
	Reposts     uint32 `json:"reposts"`
	Replies     uint32 `json:"replies"`
	Quotes      uint32 `json:"quotes"`
	Impressions uint32 `json:"impressions"`
}

// Comparison is the result of comparing a fresh fetch with the stored snapshot.
type Comparison struct {
	Current  Counts
	Previous Counts
	Delta    Delta

	// Baseline is true when there was no earlier successful fetch to compare
	// against; nothing should be notified in that case.
	Baseline bool

	// Changed is true when any counter differs from the stored snapshot.
	Changed bool
}

// HasNewActivity reports whether there is new user-facing interaction worth
// notifying about. Impressions alone are not considered activity.
func (c Comparison) HasNewActivity() bool {
	if c.Baseline {
		return false
	}
	return c.Delta.Likes > 0 || c.Delta.Reposts > 0 || c.Delta.Replies > 0 || c.Delta.Quotes > 0
}

// FromConnector converts a connector snapshot to Counts.
func FromConnector(e *connector.PostEngagement) Counts {
	if e == nil {
		return Counts{}
	}
	return Counts{Likes: e.Likes, Reposts: e.Reposts, Replies: e.Replies, Quotes: e.Quotes, Impressions: e.Impressions}
}

// FromDB converts a stored row to Counts.
func FromDB(pe *db.PostEngagement) Counts {
	if pe == nil {
		return Counts{}
	}
	return Counts{Likes: pe.Likes, Reposts: pe.Reposts, Replies: pe.Replies, Quotes: pe.Quotes, Impressions: pe.Impressions}
}

// HasBaseline reports whether a stored row came from at least one successful fetch.
// Rows created only by failed fetches have no meaningful counters.
func HasBaseline(pe *db.PostEngagement) bool {
	return pe != nil && pe.LastChangedAt.Valid
}

// Compare compares the stored row (may be nil) with freshly fetched counts.
func Compare(stored *db.PostEngagement, current Counts) Comparison {
	cmp := Comparison{Current: current}

	if !HasBaseline(stored) {
		cmp.Baseline = true
		cmp.Changed = true
		return cmp
	}

	prev := FromDB(stored)
	cmp.Previous = prev
	cmp.Changed = prev != current
	cmp.Delta = Delta{
		Likes:       increase(prev.Likes, current.Likes),
		Reposts:     increase(prev.Reposts, current.Reposts),
		Replies:     increase(prev.Replies, current.Replies),
		Quotes:      increase(prev.Quotes, current.Quotes),
		Impressions: increase(prev.Impressions, current.Impressions),
	}
	return cmp
}

func increase(prev, cur uint32) uint32 {
	if cur > prev {
		return cur - prev
	}
	return 0
}

// ToDB builds the row to upsert for a post.
func ToDB(postID uint32, c Counts) *db.PostEngagement {
	return &db.PostEngagement{
		PostID:      postID,
		Likes:       c.Likes,
		Reposts:     c.Reposts,
		Replies:     c.Replies,
		Quotes:      c.Quotes,
		Impressions: c.Impressions,
	}
}
