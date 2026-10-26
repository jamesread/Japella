// Package feedauthor decides whether a feed post was written by the social
// account it was fetched through, or by someone else.
package feedauthor

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// handleParts splits "@alice@example.social", "alice@example.social" or
// "alice" into a lowercase local part and (possibly empty) domain.
func handleParts(s string) (local, domain string) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "@")
	if i := strings.Index(s, "@"); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

func sameHandle(a, b string) bool {
	al, ad := handleParts(a)
	bl, bd := handleParts(b)
	if al == "" || al != bl {
		return false
	}
	// A bare username matches any domain: platforms omit it for local accounts.
	return ad == "" || bd == "" || ad == bd
}

// IsOwnPost reports whether a post's author is the account itself, comparing
// the account's identity with the author's handle and, failing that, name.
// Posts with no author details are treated as the account's own.
func IsOwnPost(accountIdentity, authorHandle, authorName string) bool {
	if strings.TrimSpace(authorHandle) == "" && strings.TrimSpace(authorName) == "" {
		return true
	}
	if authorHandle != "" && sameHandle(accountIdentity, authorHandle) {
		return true
	}
	return authorHandle == "" && sameHandle(accountIdentity, authorName)
}

// PostKey identifies a post on its platform: its URL, or its remote id when
// there is no URL. It returns "" when there is neither.
func PostKey(remoteURL, remoteID string) string {
	if u := strings.TrimSpace(remoteURL); u != "" {
		return "url:" + u
	}
	if id := strings.TrimSpace(remoteID); id != "" {
		return "id:" + id
	}
	return ""
}

// KeyHash is the stored form of a PostKey.
func KeyHash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
