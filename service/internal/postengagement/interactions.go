package postengagement

import (
	"sort"
	"time"

	"github.com/jamesread/japella/internal/db"
)

// Interaction kinds shown in an account's recent interactions.
const (
	InteractionPost       = "post"
	InteractionReply      = "reply"
	InteractionEngagement = "engagement"
)

// Interaction is one entry in an account's recent activity: a post it
// published, a reply it received, or a detected increase in engagement.
type Interaction struct {
	Kind        string
	At          time.Time
	PostID      uint32
	PostContent string
	PostURL     string

	// Set for InteractionEngagement.
	Counts Counts

	// Set for InteractionReply.
	Reply *db.PostReply
}

// InteractionSources are the raw rows for one account and window.
type InteractionSources struct {
	Posts   []*db.Post
	Replies []*db.AccountPostReply
	Events  []*db.PostEngagementEvent

	// RepliesFromThreads is true when the account's connector fetches reply
	// threads: replies are then listed individually, and the replies part of
	// engagement events is left out of the timeline to avoid showing them twice.
	RepliesFromThreads bool
}

// InteractionSummary is the timeline plus totals for the window.
type InteractionSummary struct {
	Posts  uint32
	Counts Counts
	Daily  []DayBucket
	Items  []Interaction
}

// SummariseInteractions merges posts, replies and engagement events into one
// timeline (newest first, at most maxItems) with totals and one bucket per
// calendar day for the `days` days ending on `today`. Day buckets count
// posts published and interactions detected that day.
func SummariseInteractions(today time.Time, days int, src InteractionSources, maxItems int) InteractionSummary {
	sum := InteractionSummary{Daily: Daily(today, days, nil)}
	index := make(map[string]int, len(sum.Daily))
	for i, d := range sum.Daily {
		index[d.Date] = i
	}
	addToDay := func(at time.Time, posts uint32, c Counts) {
		if i, ok := index[DateKey(at)]; ok {
			sum.Daily[i].Posts += posts
			sum.Daily[i].Counts = sum.Daily[i].Counts.Add(c)
		}
	}

	items := make([]Interaction, 0, len(src.Posts)+len(src.Replies)+len(src.Events))

	for _, p := range src.Posts {
		at := p.CreatedAt
		if p.ScheduledAt.Valid {
			at = p.ScheduledAt.Time
		}
		sum.Posts++
		addToDay(at, 1, Counts{})
		items = append(items, Interaction{
			Kind:        InteractionPost,
			At:          at,
			PostID:      p.ID,
			PostContent: p.Content,
			PostURL:     p.PostURL.String,
		})
	}

	for _, e := range src.Events {
		c := Counts{Likes: e.Likes, Reposts: e.Reposts, Replies: e.Replies, Quotes: e.Quotes}
		if src.RepliesFromThreads {
			// Counted from the threads below instead.
			c.Replies = 0
		}
		sum.Counts = sum.Counts.Add(c)
		addToDay(e.DetectedAt, 0, c)
		if c.Interactions() == 0 {
			continue
		}
		items = append(items, Interaction{
			Kind:        InteractionEngagement,
			At:          e.DetectedAt,
			PostID:      e.PostID,
			PostContent: e.PostContent,
			PostURL:     e.PostURL.String,
			Counts:      c,
		})
	}

	if src.RepliesFromThreads {
		for _, r := range src.Replies {
			if !r.PostedAt.Valid {
				continue
			}
			one := Counts{Replies: 1}
			sum.Counts = sum.Counts.Add(one)
			addToDay(r.PostedAt.Time, 0, one)
			reply := r.PostReply
			items = append(items, Interaction{
				Kind:        InteractionReply,
				At:          r.PostedAt.Time,
				PostID:      r.PostID,
				PostContent: r.PostContent,
				PostURL:     r.PostURL.String,
				Reply:       &reply,
			})
		}
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].At.After(items[j].At) })
	if maxItems > 0 && len(items) > maxItems {
		items = items[:maxItems]
	}
	sum.Items = items
	return sum
}
