package postengagement

import (
	"sort"
	"time"
)

// Interactions is the number of user interactions (impressions excluded).
func (c Counts) Interactions() uint32 {
	return c.Likes + c.Reposts + c.Replies + c.Quotes
}

// Add returns the element-wise sum of two Counts.
func (c Counts) Add(o Counts) Counts {
	return Counts{
		Likes:       c.Likes + o.Likes,
		Reposts:     c.Reposts + o.Reposts,
		Replies:     c.Replies + o.Replies,
		Quotes:      c.Quotes + o.Quotes,
		Impressions: c.Impressions + o.Impressions,
	}
}

// PostSample is one post's publish time and latest counters.
type PostSample struct {
	PostID   uint32
	PostedAt time.Time
	Counts   Counts
}

// DayBucket aggregates the posts published on one calendar day (UTC).
type DayBucket struct {
	Date   string
	Posts  uint32
	Counts Counts
	// Likes and comments the account made (recent interactions only).
	LikesGiven   uint32
	CommentsMade uint32
}

// DateKey is the calendar day (UTC) a time falls on, as YYYY-MM-DD.
func DateKey(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

// Daily groups samples into one bucket per calendar day for the `days` days
// ending on `today`, oldest first. Samples outside the window are ignored.
func Daily(today time.Time, days int, samples []PostSample) []DayBucket {
	if days <= 0 {
		return []DayBucket{}
	}

	buckets := make([]DayBucket, days)
	index := make(map[string]int, days)
	start := today.UTC().AddDate(0, 0, -(days - 1))
	for i := 0; i < days; i++ {
		key := DateKey(start.AddDate(0, 0, i))
		buckets[i] = DayBucket{Date: key}
		index[key] = i
	}

	for _, s := range samples {
		i, ok := index[DateKey(s.PostedAt)]
		if !ok {
			continue
		}
		buckets[i].Posts++
		buckets[i].Counts = buckets[i].Counts.Add(s.Counts)
	}

	return buckets
}

// Total sums the counters of all samples.
func Total(samples []PostSample) Counts {
	var total Counts
	for _, s := range samples {
		total = total.Add(s.Counts)
	}
	return total
}

// SortByInteractions orders samples by interactions, then impressions, then
// most recent first. It sorts in place and returns the slice.
func SortByInteractions(samples []PostSample) []PostSample {
	sort.SliceStable(samples, func(i, j int) bool {
		a, b := samples[i], samples[j]
		if ai, bi := a.Counts.Interactions(), b.Counts.Interactions(); ai != bi {
			return ai > bi
		}
		if a.Counts.Impressions != b.Counts.Impressions {
			return a.Counts.Impressions > b.Counts.Impressions
		}
		return a.PostedAt.After(b.PostedAt)
	})
	return samples
}
