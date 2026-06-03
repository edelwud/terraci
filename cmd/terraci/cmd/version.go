package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/cmd/terraci/internal/versionflow"
)

func newVersionCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			prepared, err := runflow.FromContext(cmd.Context())
			if err != nil {
				return err
			}
			return versionflow.Write(os.Stdout, versionflow.Build(versionflow.Metadata{
				Version: app.Version,
				Commit:  app.Commit,
				Date:    app.Date,
			}, prepared))
		},
	}
	runflow.MarkCommand(cmd, runflow.CommandPolicy{SkipConfig: true})
	return cmd
}
