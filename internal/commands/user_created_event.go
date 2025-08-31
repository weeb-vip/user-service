package commands

import (
	"github.com/weeb-vip/user/handlers"

	"github.com/spf13/cobra"
)

func configureUserCreatedEventCommand(rootCmd *cobra.Command) {
	var eventingCmd = &cobra.Command{
		Use:   "eventing",
		Short: "manipulate eventing",
	}

	var userCreatedStartCmd = &cobra.Command{
		Use:   "user-created",
		Short: "start listening to events",
		RunE:  startUserCreatedEventing,
	}

	rootCmd.AddCommand(eventingCmd)
	eventingCmd.AddCommand(userCreatedStartCmd)
}

func startUserCreatedEventing(cmd *cobra.Command, args []string) error {
	return handlers.UserCreatedEventing()
}
