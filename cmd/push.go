package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/platanist/nest-cli/internal/api"
	"github.com/platanist/nest-cli/internal/crypto"
	"github.com/platanist/nest-cli/internal/keys"
	"github.com/platanist/nest-cli/internal/storage"
	"github.com/spf13/cobra"
)

func newPushCmd() *cobra.Command {
	var envPath string

	cmd := &cobra.Command{
		Use:   "push <origin> <application-name>",
		Short: "Encrypt and upload local .env to origin",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			originName := args[0]
			application := args[1]

			_, origin, err := resolveOrigin(originName)
			if err != nil {
				return err
			}
			if app.Config.ActiveKeyID == "" {
				return fmt.Errorf("no active key configured. Run: nest keys generate")
			}

			path, err := resolveConfigPath()
			if err != nil {
				return err
			}
			manager, err := keys.NewManager(path)
			if err != nil {
				return err
			}

			rec, err := manager.Find(app.Config.ActiveKeyID)
			if err != nil {
				return err
			}
			pub, err := decodePublicKey(rec)
			if err != nil {
				return err
			}

			envRaw, err := os.ReadFile(envPath)
			if err != nil {
				return fmt.Errorf("read env file %q: %w", envPath, err)
			}

			profile, err := crypto.ParseProfile(rec.Profile)
			if err != nil {
				return err
			}
			aad := []byte(originName + ":" + application + ":" + rec.ID)
			envelope, err := crypto.EncryptEnvelope(envRaw, profile, rec.ID, pub, aad)
			if err != nil {
				return err
			}

			hash := sha256.Sum256(envRaw)
			backend, err := storage.NewBackend(originName, origin, app.Config.AuthToken)
			if err != nil {
				return err
			}

			response, err := backend.PushSecret(context.Background(), api.PushSecretRequest{
				Origin:       originName,
				Application:  application,
				Envelope:     base64.StdEncoding.EncodeToString(envelope),
				Profile:      rec.Profile,
				KeyID:        rec.ID,
				Fingerprint:  rec.Fingerprint,
				ChecksumSHA:  hex.EncodeToString(hash[:]),
				ContentBytes: len(envRaw),
			})
			if err != nil {
				return err
			}

			fmt.Printf("pushed application=%s origin=%s version=%d key=%s\n", application, originName, response.Version, rec.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&envPath, "file", ".env", "Path to .env file")
	return cmd
}
