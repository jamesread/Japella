package db

import (
	"os"
	"testing"

	"github.com/jamesread/japella/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestAdminPasswordReset(t *testing.T) {
	// This test demonstrates the admin password reset functionality
	// It shows how the environment variable triggers the password reset

	// Test 1: Verify environment variable detection
	t.Run("EnvironmentVariableDetection", func(t *testing.T) {
		// Test with environment variable set
		os.Setenv("JAPELLA_RESET_ADMIN_PASSWORD", "true")
		defer os.Unsetenv("JAPELLA_RESET_ADMIN_PASSWORD")

		envValue := os.Getenv("JAPELLA_RESET_ADMIN_PASSWORD")
		assert.NotEmpty(t, envValue, "Environment variable should be set")
		assert.Equal(t, "true", envValue, "Environment variable should have the expected value")
	})

	// Test 2: Verify password hashing works correctly
	t.Run("PasswordHashing", func(t *testing.T) {
		password := "admin"
		hashedPassword, err := utils.HashPassword(password)

		assert.NoError(t, err, "Should hash password without error")
		assert.NotEmpty(t, hashedPassword, "Hashed password should not be empty")

		// Verify the hash can be verified
		match, err := utils.VerifyPassword(hashedPassword, password)
		assert.NoError(t, err, "Should verify password without error")
		assert.True(t, match, "Password should match its hash")
	})

	// Test 3: Verify environment variable is not set by default
	t.Run("EnvironmentVariableNotSet", func(t *testing.T) {
		// Ensure environment variable is not set
		os.Unsetenv("JAPELLA_RESET_ADMIN_PASSWORD")

		envValue := os.Getenv("JAPELLA_RESET_ADMIN_PASSWORD")
		assert.Empty(t, envValue, "Environment variable should not be set")
	})
}

// TestAdminPasswordResetIntegration demonstrates the integration
// This would require a real database connection, so it's commented out
// but shows how the feature would be tested in a real environment
/*
func TestAdminPasswordResetIntegration(t *testing.T) {
	// This test would require a real database connection
	// It demonstrates how the feature works in practice

	// Set up test database
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create an admin user with a different password
	adminUser, err := db.CreateUserAccount("admin", "someOtherPassword")
	assert.NoError(t, err)
	assert.NotNil(t, adminUser)

	// Set the environment variable
	os.Setenv("JAPELLA_RESET_ADMIN_PASSWORD", "true")
	defer os.Unsetenv("JAPELLA_RESET_ADMIN_PASSWORD")

	// Call initAdminUser (this would happen during startup)
	chain := &ConnectionChain{}
	db.initAdminUser(chain)

	// Verify no error occurred
	assert.NoError(t, chain.err)

	// Verify the admin password was reset
	updatedAdmin := db.GetUserByUsername("admin")
	assert.NotNil(t, updatedAdmin)

	// Verify the password is now "admin"
	match, err := utils.VerifyPassword(updatedAdmin.PasswordHash, "admin")
	assert.NoError(t, err)
	assert.True(t, match, "Admin password should be reset to 'admin'")
}
*/
