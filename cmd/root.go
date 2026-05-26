package cmd

import (
	app "github.com/softcdata/testudo-server/cmd/app"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disaster",
		Short: "Disaster server",
	}
	cmd.AddCommand(app.NewServerCommand())

	return cmd
}
