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
	// Likes and comments the account itself made through Japella.
	InteractionLiked     = "liked"
	InteractionCommented = "commented"
)

// Interaction is one entry in an account's recent activity: a post it
// published, a reply it received, a detected increase in engagement, or a
// like or comment it made.
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

	// Set for InteractionLiked and InteractionCommented: the post is someone
	// else's, so PostID is 0.
	PostAuthor string
	Comment    string
	CommentURL string
}

// InteractionSources are the raw rows for one account and window.
type InteractionSources struct {
	Posts   []*db.Post
	Replies []*db.AccountPostReply
	Events  []*db.PostEngagementEvent
	// Likes and comments the account made.
	Activity []*db.AccountActivity

	// RepliesFromThreads is true when the account's connector fetches reply
	// threads: replies are then listed individually, and the replies part of
	// engagement events is left out of the timeline to avoid showing them twice.
	RepliesFromThreads bool
}

// InteractionSummary is the timeline plus totals for the window.
type InteractionSummary struct {
	Posts  uint32
	Counts Counts
	// Likes and comments the account made, kept apart from Counts (received).
	LikesGiven   uint32
	CommentsMade uint32
	Daily        []DayBucket
	Items        []Interaction
}

// SummariseInteractions merges posts, replies and engagement events into one
// timeline (newest first, at most maxItems) with totals and one bucket per
// calendar day for the `days` days ending on `today`. Day buckets count
// posts published, interactions detected, and likes and comments made that day.
func SummariseInteractions(today time.Time, days int, src InteractionSources, maxItems int) InteractionSummary {
	sum := InteractionSummary{Daily: Daily(today, days, nil)}
	index := dayIndex(sum.Daily)
	items := make([]Interaction, 0, len(src.Posts)+len(src.Replies)+len(src.Events)+len(src.Activity))
	items = appendPostInteractions(&sum, index, src.Posts, items)
	items = appendEventInteractions(&sum, index, src, items)
	items = appendThreadReplyInteractions(&sum, index, src, items)
	items = appendActivityInteractions(&sum, index, src.Activity, items)
	sort.SliceStable(items, func(i, j int) bool { return items[i].At.After(items[j].At) })
	if maxItems > 0 && len(items) > maxItems {
		items = items[:maxItems]
	}
	sum.Items = items
	return sum
}

func dayIndex(daily []DayBucket) map[string]int {
	index := make(map[string]int, len(daily))
	for i, d := range daily {
		index[d.Date] = i
	}
	return index
}

func addToDay(sum *InteractionSummary, index map[string]int, at time.Time, posts uint32, c Counts) {
	if i, ok := index[DateKey(at)]; ok {
		sum.Daily[i].Posts += posts
		sum.Daily[i].Counts = sum.Daily[i].Counts.Add(c)
	}
}

func appendPostInteractions(sum *InteractionSummary, index map[string]int, posts []*db.Post, items []Interaction) []Interaction {
	for _, p := range posts {
		at := p.CreatedAt
		if p.ScheduledAt.Valid {
			at = p.ScheduledAt.Time
		}
		sum.Posts++
		addToDay(sum, index, at, 1, Counts{})
		items = append(items, Interaction{
			Kind:        InteractionPost,
			At:          at,
			PostID:      p.ID,
			PostContent: p.Content,
			PostURL:     p.PostURL.String,
		})
	}
	return items
}

func appendEventInteractions(sum *InteractionSummary, index map[string]int, src InteractionSources, items []Interaction) []Interaction {
	for _, e := range src.Events {
		c := Counts{Likes: e.Likes, Reposts: e.Reposts, Replies: e.Replies, Quotes: e.Quotes}
		if src.RepliesFromThreads {
			// Counted from the threads below instead.
			c.Replies = 0
		}
		sum.Counts = sum.Counts.Add(c)
		addToDay(sum, index, e.DetectedAt, 0, c)
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
	return items
}

func appendThreadReplyInteractions(sum *InteractionSummary, index map[string]int, src InteractionSources, items []Interaction) []Interaction {
	if !src.RepliesFromThreads {
		return items
	}
	for _, r := range src.Replies {
		if !r.PostedAt.Valid {
			continue
		}
		one := Counts{Replies: 1}
		sum.Counts = sum.Counts.Add(one)
		addToDay(sum, index, r.PostedAt.Time, 0, one)
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
	return items
}

func appendActivityInteractions(sum *InteractionSummary, index map[string]int, activity []*db.AccountActivity, items []Interaction) []Interaction {
	for _, a := range activity {
		it, ok := activityInteraction(sum, index, a)
		if !ok {
			continue
		}
		items = append(items, it)
	}
	return items
}

func activityInteraction(sum *InteractionSummary, index map[string]int, a *db.AccountActivity) (Interaction, bool) {
	it := Interaction{
		At:          a.CreatedAt,
		PostContent: a.PostSnippet,
		PostURL:     a.PostURL,
		PostAuthor:  a.PostAuthor,
	}
	switch a.Kind {
	case db.AccountActivityLiked:
		sum.LikesGiven++
		it.Kind = InteractionLiked
		if i, ok := index[DateKey(a.CreatedAt)]; ok {
			sum.Daily[i].LikesGiven++
		}
	case db.AccountActivityCommented:
		sum.CommentsMade++
		it.Kind = InteractionCommented
		it.Comment = a.CommentContent.String
		it.CommentURL = a.CommentURL
		if i, ok := index[DateKey(a.CreatedAt)]; ok {
			sum.Daily[i].CommentsMade++
		}
	default:
		return Interaction{}, false
	}
	return it, true
}
