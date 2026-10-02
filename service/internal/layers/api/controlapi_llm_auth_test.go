package api

import (
	"database/sql"
	"testing"

	"github.com/jamesread/japella/internal/db"
)

func TestCanEditPostContentDraftOwner(t *testing.T) {
	s := &ControlApi{}
	post := &db.Post{
		State:             db.PostStateDraft,
		SubmittedByUserID: sql.NullInt32{Int32: 7, Valid: true},
	}
	if !s.canEditPostContent(7, post) {
		t.Fatal("draft submitter should edit")
	}
	if s.canEditPostContent(8, post) {
		t.Fatal("other user should not edit draft")
	}
}

func TestCanEditPostContentRejectsPublished(t *testing.T) {
	s := &ControlApi{}
	post := &db.Post{
		State:             db.PostStateCompleted,
		SubmittedByUserID: sql.NullInt32{Int32: 7, Valid: true},
	}
	if s.canEditPostContent(7, post) {
		t.Fatal("completed posts should not be editable via llm path")
	}
}
