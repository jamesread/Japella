package commands

import (
	"fmt"
	"os"

	"github.com/jamesread/japella/internal/utils"
	"github.com/spf13/cobra"
)

func newAdminPasswordResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "passwordreset <username>",
		Short: "Reset a user's password",
		Long:  "Reset a local user's password to a randomly generated value and print it.",
		Args:  cobra.ExactArgs(1),
		RunE:  runAdminPasswordReset,
	}
}

func runAdminPasswordReset(cmd *cobra.Command, args []string) error {
	username := args[0]

	database, err := openDB()
	if err != nil {
		return err
	}

	user := database.GetUserByUsername(username)
	if user == nil {
		return fmt.Errorf("user not found: %s", username)
	}

	password, err := utils.GeneratePassword()
	if err != nil {
		return fmt.Errorf("generate password: %w", err)
	}

	hash, err := utils.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := database.UpdateUserPassword(user.ID, hash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Password reset for user %q\nNew password: %s\n", username, password)
	return nil
}
