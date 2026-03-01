package cmd

import (
	"context"
	"fmt"

	"github.com/platanist/nest-cli/internal/storage"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var originName string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run local diagnostics",
		RunE: func(_ *cobra.Command, _ []string) error {
			resolvedName, origin, err := resolveOrigin(originName)
			if err != nil {
				return err
			}

			fmt.Printf("config: ok (%s)\n", resolvedName)
			backend, err := storage.NewBackend(resolvedName, origin, app.Config.AuthToken)
			if err != nil {
				return err
			}
			health, err := backend.HealthCheck(context.Background())
			if err != nil {
				return err
			}
			fmt.Println(health)
			return nil
		},
	}

	cmd.Flags().StringVar(&originName, "origin", "", "Origin alias to test")
	return cmd
}
