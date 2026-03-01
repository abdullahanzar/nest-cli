package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.CryptoProfile != "modern" {
		t.Fatalf("unexpected default profile: %q", cfg.CryptoProfile)
	}
	if cfg.Origins == nil {
		t.Fatal("origins map must be initialized")
	}
}

func TestSaveLoadAndLoadOrCreate(t *testing.T) {
	t.Run("save and load round-trip", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nest", "config.yaml")
		cfg := Default()
		cfg.DefaultOrigin = "origin"
		cfg.Origins["origin"] = Origin{Mode: ModeMongo, MongoURI: "mongodb://localhost:27017"}

		if err := Save(path, cfg); err != nil {
			t.Fatalf("save failed: %v", err)
		}

		loaded, err := Load(path)
		if err != nil {
			t.Fatalf("load failed: %v", err)
		}
		if loaded.DefaultOrigin != "origin" {
			t.Fatalf("unexpected default origin: %q", loaded.DefaultOrigin)
		}
		if got := loaded.Origins["origin"].MongoURI; got != "mongodb://localhost:27017" {
			t.Fatalf("unexpected mongo URI: %q", got)
		}
	})

	t.Run("load-or-create initializes missing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "new", "config.yaml")
		cfg, err := LoadOrCreate(path)
		if err != nil {
			t.Fatalf("load-or-create failed: %v", err)
		}
		if cfg.CryptoProfile != "modern" {
			t.Fatalf("unexpected profile: %q", cfg.CryptoProfile)
		}
	})

	t.Run("load rejects invalid yaml", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.yaml")
		if err := Save(path, Default()); err != nil {
			t.Fatalf("initial save failed: %v", err)
		}
		if err := os.WriteFile(path, []byte("origins: ["), 0o600); err != nil {
			t.Fatalf("write malformed file failed: %v", err)
		}

		_, err := Load(path)
		if err == nil || !strings.Contains(err.Error(), "decode config file") {
			t.Fatalf("expected decode error, got: %v", err)
		}
	})
}

func TestResolveMongoURIAndNormalize(t *testing.T) {
	origin := "primary-origin"

	t.Run("prefers explicit config uri", func(t *testing.T) {
		t.Setenv("NEST_MONGO_URI_PRIMARY_ORIGIN", "mongodb://env-origin")
		t.Setenv("NEST_MONGO_URI", "mongodb://env-global")
		got := ResolveMongoURI(origin, Origin{MongoURI: " mongodb://configured "})
		if got != "mongodb://configured" {
			t.Fatalf("expected configured uri, got: %q", got)
		}
	})

	t.Run("falls back to origin env var", func(t *testing.T) {
		t.Setenv("NEST_MONGO_URI_PRIMARY_ORIGIN", "mongodb://env-origin")
		t.Setenv("NEST_MONGO_URI", "mongodb://env-global")
		got := ResolveMongoURI(origin, Origin{})
		if got != "mongodb://env-origin" {
			t.Fatalf("expected origin env uri, got: %q", got)
		}
	})

	t.Run("falls back to global env var", func(t *testing.T) {
		t.Setenv("NEST_MONGO_URI_PRIMARY_ORIGIN", "")
		t.Setenv("NEST_MONGO_URI", "mongodb://env-global")
		got := ResolveMongoURI(origin, Origin{})
		if got != "mongodb://env-global" {
			t.Fatalf("expected global env uri, got: %q", got)
		}
	})

	t.Run("normalizes origin name for env key", func(t *testing.T) {
		got := NormalizeOriginEnvKey("my.origin-1")
		if got != "MY_ORIGIN_1" {
			t.Fatalf("unexpected normalized key: %q", got)
		}
	})
}
