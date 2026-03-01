package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/platanist/nest-cli/internal/api"
	"github.com/platanist/nest-cli/internal/crypto"
	"github.com/platanist/nest-cli/internal/keys"
	"github.com/platanist/nest-cli/internal/storage"
	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "pull <origin> <application-name>",
		Short: "Download and decrypt remote .env from origin",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			originName := args[0]
			application := args[1]

			_, origin, err := resolveOrigin(originName)
			if err != nil {
				return err
			}
			if app.Config.ActiveKeyID == "" {
				return fmt.Errorf("no active key configured. Run: nest keys use <key-id>")
			}

			path, err := resolveConfigPath()
			if err != nil {
				return err
			}
			manager, err := keys.NewManager(path)
			if err != nil {
				return err
			}

			passphrase, err := promptPassphrase("passphrase to decrypt private key: ")
			if err != nil {
				return err
			}
			privateKey, rec, err := manager.LoadPrivate(app.Config.ActiveKeyID, passphrase)
			if err != nil {
				return err
			}

			backend, err := storage.NewBackend(originName, origin, app.Config.AuthToken)
			if err != nil {
				return err
			}

			response, err := backend.PullSecret(context.Background(), api.PullSecretRequest{
				Origin:      originName,
				Application: application,
				KeyID:       rec.ID,
				Fingerprint: rec.Fingerprint,
			})
			if err != nil {
				return err
			}

			envelopeBytes, err := base64.StdEncoding.DecodeString(response.Envelope)
			if err != nil {
				return fmt.Errorf("decode envelope: %w", err)
			}

			aad := []byte(originName + ":" + application + ":" + rec.ID)
			plaintext, envelope, err := crypto.DecryptEnvelope(envelopeBytes, privateKey, aad)
			if err != nil {
				return err
			}
			if envelope.KeyID != rec.ID {
				return fmt.Errorf("key mismatch: local key %q, envelope key %q", rec.ID, envelope.KeyID)
			}

			tempPath := outputPath + ".tmp"
			if err := os.WriteFile(tempPath, plaintext, 0o600); err != nil {
				return fmt.Errorf("write temp env file: %w", err)
			}
			if err := os.Rename(tempPath, outputPath); err != nil {
				return fmt.Errorf("replace env file atomically: %w", err)
			}

			fmt.Printf("pulled application=%s origin=%s version=%d output=%s\n", application, originName, response.Version, outputPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&outputPath, "out", ".env", "Output path for decrypted env file")
	return cmd
}
