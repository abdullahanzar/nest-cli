package cmd

import (
	"fmt"

	"github.com/platanist/nest-cli/internal/config"
	"github.com/platanist/nest-cli/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nest",
	Short: "Secure .env push/pull CLI",
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		cfg, err := config.LoadOrCreate(app.ConfigPath)
		if err != nil {
			return err
		}
		app.Config = cfg
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&app.ConfigPath, "config", "", "Path to config file")
	rootCmd.Version = fmt.Sprintf("%s (%s %s)", version.Version, version.Commit, version.Date)

	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newKeysCmd())
	rootCmd.AddCommand(newPushCmd())
	rootCmd.AddCommand(newPullCmd())
}
