package api

import (
	"database/sql"
	"strings"

	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/feedauthor"
	"github.com/jamesread/japella/internal/keywordwatch"
)

const (
	activitySnippetRunes = 300
	activityAuthorRunes  = 120
	activityCommentRunes = 5000
)

// activityFor describes the post a like or comment was made on. Stored feed
// posts are described from the database; column posts only exist on the
// client, so their display fields are taken from the ref as plain text.
func (s *ControlApi) activityFor(sa *db.SocialAccount, ref *controlv1.PostRef) *db.AccountActivity {
	key := feedauthor.PostKey(ref.GetRemoteUrl(), ref.GetRemoteId())
	if key == "" {
		return nil
	}
	a := &db.AccountActivity{
		SocialAccountID: sa.ID,
		PostKeyHash:     feedauthor.KeyHash(key),
		PostURL:         safeSnapshotURL(ref.GetRemoteUrl()),
		PostAuthor:      ref.GetAuthorName(),
		PostSnippet:     ref.GetSnippet(),
	}
	s.fillActivityFromFeedEntry(a, sa.ID, ref.GetFeedPostId())
	a.PostAuthor = truncateRunes(strings.TrimSpace(keywordwatch.StripHTML(a.PostAuthor)), activityAuthorRunes)
	a.PostSnippet = truncateRunes(strings.TrimSpace(keywordwatch.StripHTML(a.PostSnippet)), activitySnippetRunes)
	return a
}

func feedEntryMatchesAccount(entry *db.Feed, socialAccountID uint32, err error) bool {
	return err == nil && entry != nil && entry.SocialAccountID == socialAccountID
}

func (s *ControlApi) fillActivityFromFeedEntry(a *db.AccountActivity, socialAccountID, feedPostID uint32) {
	if feedPostID == 0 {
		return
	}
	entry, err := s.DB.GetFeedEntryByID(feedPostID)
	if !feedEntryMatchesAccount(entry, socialAccountID, err) {
		return
	}
	a.PostAuthor = entry.AuthorName
	a.PostSnippet = entry.Content
	if u := safeSnapshotURL(entry.RemoteURL); u != "" {
		a.PostURL = u
	}
}

// recordLikeActivity keeps the account's recent interactions in step with a
// like or unlike. Failures are logged by the DB layer and don't fail the like.
func (s *ControlApi) recordLikeActivity(sa *db.SocialAccount, ref *controlv1.PostRef, liked bool) {
	a := s.activityFor(sa, ref)
	if a == nil {
		return
	}
	if liked {
		_ = s.DB.RecordAccountLike(a)
	} else {
		_ = s.DB.DeleteAccountLike(sa.ID, a.PostKeyHash)
	}
}

func (s *ControlApi) recordCommentActivity(sa *db.SocialAccount, ref *controlv1.PostRef, content string, reply *connector.PostReply) {
	a := s.activityFor(sa, ref)
	if a == nil {
		return
	}
	a.CommentContent = sql.NullString{String: truncateRunes(content, activityCommentRunes), Valid: true}
	if reply != nil {
		a.CommentURL = safeSnapshotURL(reply.RemoteURL)
	}
	_ = s.DB.RecordAccountComment(a)
}
