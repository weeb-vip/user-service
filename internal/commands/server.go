package commands

import (
	"github.com/weeb-vip/user"

	"github.com/spf13/cobra"
)

func configureServerCommand(rootCmd *cobra.Command) {
	var serverCmd = &cobra.Command{
		Use:   "server",
		Short: "manipulate server",
	}

	var serverStartCmd = &cobra.Command{
		Use:   "start",
		Short: "start listening to requests",
		RunE:  startServer,
	}

	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(serverStartCmd)
}

func startServer(cmd *cobra.Command, args []string) error {
	return user.StartServer()
}
