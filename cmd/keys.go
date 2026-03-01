package cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/platanist/nest-cli/internal/api"
	"github.com/platanist/nest-cli/internal/config"
	"github.com/platanist/nest-cli/internal/crypto"
	"github.com/platanist/nest-cli/internal/keys"
	"github.com/platanist/nest-cli/internal/storage"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage asymmetric keys",
	}

	cmd.AddCommand(newKeysGenerateCmd())
	cmd.AddCommand(newKeysListCmd())
	cmd.AddCommand(newKeysLocationCmd())
	cmd.AddCommand(newKeysUseCmd())
	cmd.AddCommand(newKeysRegisterCmd())
	cmd.AddCommand(newKeysRevokeCmd())
	cmd.AddCommand(newKeysRemoteListCmd())

	return cmd
}

func newKeysGenerateCmd() *cobra.Command {
	var profileValue string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate and store a new asymmetric key",
		RunE: func(_ *cobra.Command, _ []string) error {
			profile, err := crypto.ParseProfile(profileValue)
			if err != nil {
				return err
			}

			passphrase, err := promptPassphrase("passphrase for private key: ")
			if err != nil {
				return err
			}

			path, err := resolveConfigPath()
			if err != nil {
				return err
			}

			manager, err := keys.NewManager(path)
			if err != nil {
				return err
			}

			rec, err := manager.Generate(profile, passphrase)
			if err != nil {
				return err
			}

			if app.Config.ActiveKeyID == "" {
				app.Config.ActiveKeyID = rec.ID
			}
			if err := config.Save(path, app.Config); err != nil {
				return err
			}

			fmt.Printf("generated key %s\n", rec.ID)
			fmt.Printf("profile: %s\n", rec.Profile)
			fmt.Printf("fingerprint: %s\n", rec.Fingerprint)
			fmt.Printf("backend: %s\n", rec.Backend)
			return nil
		},
	}

	cmd.Flags().StringVar(&profileValue, "profile", "modern", "Crypto profile (modern|nist)")
	return cmd
}

func newKeysListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List local key metadata",
		RunE: func(_ *cobra.Command, _ []string) error {
			path, err := resolveConfigPath()
			if err != nil {
				return err
			}
			manager, err := keys.NewManager(path)
			if err != nil {
				return err
			}
			records, err := manager.List()
			if err != nil {
				return err
			}
			for _, rec := range records {
				marker := " "
				if app.Config.ActiveKeyID == rec.ID {
					marker = "*"
				}
				fmt.Printf("%s %s profile=%s backend=%s fingerprint=%s\n", marker, rec.ID, rec.Profile, rec.Backend, rec.Fingerprint)
			}
			if len(records) == 0 {
				fmt.Println("no keys found")
			}
			return nil
		},
	}
}

func newKeysLocationCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "location [key-id]",
		Short: "Show where a private key is stored",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			keyID := app.Config.ActiveKeyID
			if len(args) == 1 {
				keyID = args[0]
			}
			if keyID == "" {
				return fmt.Errorf("key-id is required or configure active key")
			}
			path, err := resolveConfigPath()
			if err != nil {
				return err
			}
			manager, err := keys.NewManager(path)
			if err != nil {
				return err
			}
			location, err := manager.Location(keyID)
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", location)
			return nil
		},
	}
}

func newKeysUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use [key-id]",
		Short: "Set active key id",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			app.Config.ActiveKeyID = args[0]
			return saveConfig()
		},
	}
}

func newKeysRegisterCmd() *cobra.Command {
	var originName string

	cmd := &cobra.Command{
		Use:   "register [key-id]",
		Short: "Register a local key with remote origin",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			keyID := app.Config.ActiveKeyID
			if len(args) == 1 {
				keyID = args[0]
			}
			if keyID == "" {
				return fmt.Errorf("key-id is required or configure active key")
			}

			_, origin, err := resolveOrigin(originName)
			if err != nil {
				return err
			}

			path, err := resolveConfigPath()
			if err != nil {
				return err
			}
			manager, err := keys.NewManager(path)
			if err != nil {
				return err
			}

			rec, err := manager.Find(keyID)
			if err != nil {
				return err
			}

			backend, err := storage.NewBackend(originName, origin, app.Config.AuthToken)
			if err != nil {
				return err
			}

			response, err := backend.RegisterKey(context.Background(), api.RegisterKeyRequest{
				KeyID:       rec.ID,
				Profile:     rec.Profile,
				Fingerprint: rec.Fingerprint,
				PublicKey:   rec.Public,
			})
			if err != nil {
				return err
			}
			if !response.Status {
				reason := response.Reason
				if reason == "" {
					reason = "key registration failed"
				}
				return errors.New(reason)
			}

			fmt.Printf("registered key %s on origin\n", rec.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&originName, "origin", "", "Origin alias to register key against")
	return cmd
}

func newKeysRemoteListCmd() *cobra.Command {
	var originName string

	cmd := &cobra.Command{
		Use:   "remote-list",
		Short: "List keys registered on remote origin",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, origin, err := resolveOrigin(originName)
			if err != nil {
				return err
			}

			backend, err := storage.NewBackend(originName, origin, app.Config.AuthToken)
			if err != nil {
				return err
			}

			response, err := backend.ListRemoteKeys(context.Background())
			if err != nil {
				return err
			}
			if !response.Status {
				reason := response.Reason
				if reason == "" {
					reason = "remote key listing failed"
				}
				return errors.New(reason)
			}

			if len(response.Keys) == 0 {
				fmt.Println("no remote keys found")
				return nil
			}

			for _, key := range response.Keys {
				active := " "
				if app.Config.ActiveKeyID == key.KeyID {
					active = "*"
				}
				fmt.Printf("%s %s profile=%s fingerprint=%s updated=%s\n", active, key.KeyID, key.Profile, key.Fingerprint, key.UpdatedAt)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&originName, "origin", "", "Origin alias to query")
	return cmd
}

func newKeysRevokeCmd() *cobra.Command {
	var originName string

	cmd := &cobra.Command{
		Use:   "revoke [key-id]",
		Short: "Revoke a registered key for an origin (mongo mode)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			keyID := app.Config.ActiveKeyID
			if len(args) == 1 {
				keyID = args[0]
			}
			if keyID == "" {
				return fmt.Errorf("key-id is required or configure active key")
			}

			resolvedName, origin, err := resolveOrigin(originName)
			if err != nil {
				return err
			}

			backend, err := storage.NewBackend(resolvedName, origin, app.Config.AuthToken)
			if err != nil {
				return err
			}

			response, err := backend.RevokeKey(context.Background(), keyID)
			if err != nil {
				return err
			}
			if !response.Status {
				reason := response.Reason
				if reason == "" {
					reason = "key revocation failed"
				}
				return errors.New(reason)
			}

			fmt.Printf("revoked key %s on origin %s\n", keyID, resolvedName)
			return nil
		},
	}

	cmd.Flags().StringVar(&originName, "origin", "", "Origin alias to revoke key against")
	return cmd
}

func resolveConfigPath() (string, error) {
	if app.ConfigPath != "" {
		return app.ConfigPath, nil
	}
	return config.DefaultPath()
}

func promptPassphrase(label string) (string, error) {
	fmt.Print(label)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("read passphrase: %w", err)
	}
	return string(b), nil
}

func decodePublicKey(rec keys.Record) ([]byte, error) {
	pub, err := base64.StdEncoding.DecodeString(rec.Public)
	if err != nil {
		return nil, fmt.Errorf("decode stored public key for key %q: %w", rec.ID, err)
	}
	return pub, nil
}
