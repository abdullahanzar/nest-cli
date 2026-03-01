package cmd

import (
	"fmt"

	"github.com/platanist/nest-cli/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var origin string
	var mode string
	var apiBaseURL string
	var mongoURI string
	var mongoDB string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize nest-cli config",
		RunE: func(_ *cobra.Command, _ []string) error {
			if origin == "" {
				return fmt.Errorf("--origin is required")
			}

			resolvedMode := mode
			if resolvedMode == "" {
				switch {
				case mongoURI != "":
					resolvedMode = "mongo"
				case apiBaseURL != "":
					resolvedMode = "api"
				default:
					resolvedMode = "mongo"
				}
			}

			if resolvedMode != "mongo" && resolvedMode != "api" {
				return fmt.Errorf("invalid --mode %q (allowed: mongo, api)", resolvedMode)
			}

			if resolvedMode == "api" && apiBaseURL == "" {
				return fmt.Errorf("--api-url is required for api mode")
			}

			current := app.Config.Origins[origin]
			current.Mode = resolvedMode
			current.APIBaseURL = apiBaseURL
			current.MongoURI = mongoURI
			current.MongoDatabase = mongoDB
			if resolvedMode == config.ModeMongo && config.ResolveMongoURI(origin, current) == "" {
				return fmt.Errorf("mongo mode requires --mongo-uri or env NEST_MONGO_URI_%s (or NEST_MONGO_URI)", configNormalizeOriginForMessage(origin))
			}
			app.Config.Origins[origin] = current
			app.Config.DefaultOrigin = origin

			if err := saveConfig(); err != nil {
				return err
			}

			if resolvedMode == "mongo" {
				fmt.Printf("initialized origin %q in mongo mode\n", origin)
			} else {
				fmt.Printf("initialized origin %q in api mode with api %q\n", origin, apiBaseURL)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&origin, "origin", "origin", "Origin alias to configure")
	cmd.Flags().StringVar(&mode, "mode", "", "Storage mode for this origin (mongo|api). Defaults to mongo")
	cmd.Flags().StringVar(&apiBaseURL, "api-url", "", "Base URL for the CLI API")
	cmd.Flags().StringVar(&mongoURI, "mongo-uri", "", "MongoDB connection URI for direct storage mode")
	cmd.Flags().StringVar(&mongoDB, "mongo-db", "", "Optional MongoDB database name override")

	return cmd
}

func configNormalizeOriginForMessage(origin string) string {
	return config.NormalizeOriginEnvKey(origin)
}
