package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testPassword123"
	hashedPassword, err := HashPassword(password)

	assert.NoError(t, err, "Expected no error while hashing password")
	assert.NotEmpty(t, hashedPassword, "Expected a non-empty hashed password")

	match, err := VerifyPassword(hashedPassword, password)
	assert.NoError(t, err, "Expected no error while verifying password")
	assert.True(t, match, "Expected the password to match the hash")

	// Test with an incorrect password
	incorrectMatch, err := VerifyPassword(hashedPassword, "wrongPassword")
	assert.NoError(t, err, "Expected no error while verifying incorrect password")
	assert.False(t, incorrectMatch, "Expected the incorrect password to not match the hash")
}

func TestHashShortPassword(t *testing.T) {
	shortPassword := "?"
	hashedPassword, err := HashPassword(shortPassword)

	assert.NoError(t, err, "Expected no error while hashing short password")
	assert.NotEmpty(t, hashedPassword, "Expected a non-empty hashed password")

	match, err := VerifyPassword(hashedPassword, shortPassword)
	assert.NoError(t, err, "Expected no error while verifying short password")
	assert.True(t, match, "Expected the short password to match the hash")
}
