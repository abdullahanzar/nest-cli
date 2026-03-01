package cmd

import (
	"fmt"

	"github.com/platanist/nest-cli/internal/config"
	"github.com/platanist/nest-cli/internal/crypto"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage nest-cli config",
	}
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigSetTokenCmd())
	cmd.AddCommand(newConfigSetProfileCmd())
	cmd.AddCommand(newConfigSetOriginCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show effective config summary",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("default_origin: %s\n", app.Config.DefaultOrigin)
			fmt.Printf("crypto_profile: %s\n", app.Config.CryptoProfile)
			fmt.Printf("active_key_id: %s\n", app.Config.ActiveKeyID)
			fmt.Printf("origins: %d\n", len(app.Config.Origins))
			for name, origin := range app.Config.Origins {
				fmt.Printf("- %s mode=%s\n", name, origin.EffectiveMode())
			}
		},
	}
}

func newConfigSetTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-token [token]",
		Short: "Set API auth token for CLI requests",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			app.Config.AuthToken = args[0]
			return saveConfig()
		},
	}
}

func newConfigSetProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-profile [modern|nist]",
		Short: "Set default crypto profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			profile, err := crypto.ParseProfile(args[0])
			if err != nil {
				return err
			}
			app.Config.CryptoProfile = string(profile)
			return saveConfig()
		},
	}
}

func newConfigSetOriginCmd() *cobra.Command {
	var mode string
	var apiURL string
	var mongoURI string
	var mongoDB string

	cmd := &cobra.Command{
		Use:   "set-origin [name]",
		Short: "Set or update origin storage settings",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			origin := args[0]

			resolvedMode := mode
			if resolvedMode == "" {
				switch {
				case mongoURI != "":
					resolvedMode = "mongo"
				case apiURL != "":
					resolvedMode = "api"
				default:
					resolvedMode = "mongo"
				}
			}

			if resolvedMode != "mongo" && resolvedMode != "api" {
				return fmt.Errorf("invalid --mode %q (allowed: mongo, api)", resolvedMode)
			}

			if resolvedMode == "api" && apiURL == "" {
				return fmt.Errorf("--api-url is required for api mode")
			}

			current := app.Config.Origins[origin]
			current.Mode = resolvedMode
			current.APIBaseURL = apiURL
			current.MongoURI = mongoURI
			current.MongoDatabase = mongoDB
			if resolvedMode == config.ModeMongo && config.ResolveMongoURI(origin, current) == "" {
				return fmt.Errorf("mongo mode requires --mongo-uri or env NEST_MONGO_URI_%s (or NEST_MONGO_URI)", config.NormalizeOriginEnvKey(origin))
			}
			app.Config.Origins[origin] = current
			if app.Config.DefaultOrigin == "" {
				app.Config.DefaultOrigin = origin
			}
			return saveConfig()
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "", "Storage mode for this origin (mongo|api). Defaults to mongo")
	cmd.Flags().StringVar(&apiURL, "api-url", "", "Origin API base URL")
	cmd.Flags().StringVar(&mongoURI, "mongo-uri", "", "Origin MongoDB URI for direct mode")
	cmd.Flags().StringVar(&mongoDB, "mongo-db", "", "Optional MongoDB database name override")
	return cmd
}
