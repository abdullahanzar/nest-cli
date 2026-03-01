package cmd

import (
	"errors"
	"fmt"

	"github.com/platanist/nest-cli/internal/api"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate CLI sessions",
	}

	cmd.AddCommand(newAuthLoginCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var originName string
	var email string
	var apiKey string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login using email and apiKey, then store bearer token",
		RunE: func(_ *cobra.Command, _ []string) error {
			if email == "" || apiKey == "" {
				return fmt.Errorf("--email and --api-key are required")
			}

			resolvedName, origin, err := resolveOrigin(originName)
			if err != nil {
				return err
			}

			if origin.EffectiveMode() == "mongo" {
				return fmt.Errorf("origin %q is in mongo mode and does not require auth login", resolvedName)
			}

			client := api.New(origin.APIBaseURL, "")
			response, err := client.Login(api.LoginRequest{Email: email, APIKey: apiKey})
			if err != nil {
				return err
			}

			if !response.Status || response.Token == "" {
				reason := response.Reason
				if reason == "" {
					reason = "login failed"
				}
				return errors.New(reason)
			}

			app.Config.AuthToken = response.Token
			if err := saveConfig(); err != nil {
				return err
			}

			fmt.Printf("login successful: token stored (expires in %ds)\n", response.ExpiresInSeconds)
			return nil
		},
	}

	cmd.Flags().StringVar(&originName, "origin", "", "Origin alias to authenticate against")
	cmd.Flags().StringVar(&email, "email", "", "User email")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "User api key")

	return cmd
}
