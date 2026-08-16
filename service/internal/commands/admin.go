package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newAdminCmd())
}

func newAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Administrative commands",
		Long:  "Run administrative commands against the Japella database (for use alongside a running instance).",
	}
	cmd.AddCommand(newAdminUserGroupAddCmd())
	cmd.AddCommand(newAdminPasswordResetCmd())
	return cmd
}

func newAdminUserGroupAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "usergroupadd <username> <usergroup>",
		Short: "Add a user to a user group",
		Args:  cobra.ExactArgs(2),
		RunE:  runAdminUserGroupAdd,
	}
}

func runAdminUserGroupAdd(cmd *cobra.Command, args []string) error {
	username := args[0]
	groupName := args[1]

	database, err := openDB()
	if err != nil {
		return err
	}

	user := database.GetUserByUsername(username)
	if user == nil {
		return fmt.Errorf("user not found: %s", username)
	}

	group := database.GetUserGroupByName(groupName)
	if group == nil {
		return fmt.Errorf("user group not found: %s", groupName)
	}

	if err := database.AddUserGroupMember(user.ID, group.ID); err != nil {
		return fmt.Errorf("add user to group: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Added user %q to group %q\n", username, groupName)
	return nil
}
