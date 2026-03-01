package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/platanist/nest-cli/internal/config"
)

func TestRootHelpIncludesCriticalCommandsAndFlags(t *testing.T) {
	withFreshApp(t)

	output, err := executeForTest(t, rootCmd, "--help")
	if err != nil {
		t.Fatalf("help should not fail: %v", err)
	}

	for _, want := range []string{"init", "auth", "doctor", "config", "keys", "push", "pull", "--config", "--version"} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output missing %q\n%s", want, output)
		}
	}
}

func TestInitFlagValidationAndEnvFallback(t *testing.T) {
	t.Run("rejects invalid mode", func(t *testing.T) {
		withFreshApp(t)
		app.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")

		_, err := executeForTest(t, newInitCmd(), "--origin", "origin", "--mode", "bad", "--mongo-uri", "mongodb://localhost:27017")
		if err == nil || !strings.Contains(err.Error(), "invalid --mode") {
			t.Fatalf("expected invalid mode error, got: %v", err)
		}
	})

	t.Run("api mode requires api url", func(t *testing.T) {
		withFreshApp(t)
		app.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")

		_, err := executeForTest(t, newInitCmd(), "--origin", "origin", "--mode", "api")
		if err == nil || !strings.Contains(err.Error(), "--api-url is required") {
			t.Fatalf("expected missing api-url error, got: %v", err)
		}
	})

	t.Run("mongo mode accepts env fallback", func(t *testing.T) {
		withFreshApp(t)
		app.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
		t.Setenv("NEST_MONGO_URI_ORIGIN", "mongodb://env-host:27017")

		_, err := executeForTest(t, newInitCmd(), "--origin", "origin", "--mode", "mongo")
		if err != nil {
			t.Fatalf("expected success with env fallback, got: %v", err)
		}
		if app.Config.DefaultOrigin != "origin" {
			t.Fatalf("default origin should be set, got %q", app.Config.DefaultOrigin)
		}
	})
}

func TestConfigSetOriginFlagValidation(t *testing.T) {
	t.Run("api mode requires api url", func(t *testing.T) {
		withFreshApp(t)
		app.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")

		_, err := executeForTest(t, newConfigSetOriginCmd(), "sample", "--mode", "api")
		if err == nil || !strings.Contains(err.Error(), "--api-url is required") {
			t.Fatalf("expected missing api-url error, got: %v", err)
		}
	})

	t.Run("mongo mode accepts env fallback", func(t *testing.T) {
		withFreshApp(t)
		app.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
		t.Setenv("NEST_MONGO_URI_SAMPLE", "mongodb://env-host:27017")

		_, err := executeForTest(t, newConfigSetOriginCmd(), "sample", "--mode", "mongo")
		if err != nil {
			t.Fatalf("expected success with env fallback, got: %v", err)
		}
		if _, ok := app.Config.Origins["sample"]; !ok {
			t.Fatalf("origin not saved in config")
		}
	})
}

func TestAuthLoginValidation(t *testing.T) {
	t.Run("requires email and api-key", func(t *testing.T) {
		withFreshApp(t)

		_, err := executeForTest(t, newAuthLoginCmd())
		if err == nil || !strings.Contains(err.Error(), "--email and --api-key are required") {
			t.Fatalf("expected validation error, got: %v", err)
		}
	})

	t.Run("rejects mongo mode origin", func(t *testing.T) {
		withFreshApp(t)
		app.Config.DefaultOrigin = "origin"
		app.Config.Origins["origin"] = config.Origin{Mode: config.ModeMongo, MongoURI: "mongodb://localhost:27017"}

		_, err := executeForTest(t, newAuthLoginCmd(), "--email", "u@example.com", "--api-key", "secret")
		if err == nil || !strings.Contains(err.Error(), "does not require auth login") {
			t.Fatalf("expected mongo mode rejection, got: %v", err)
		}
	})
}

func TestConfigSetProfileValidation(t *testing.T) {
	withFreshApp(t)
	app.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")

	_, err := executeForTest(t, newConfigSetProfileCmd(), "invalid")
	if err == nil || !strings.Contains(err.Error(), "unsupported crypto profile") {
		t.Fatalf("expected profile parse error, got: %v", err)
	}
}

func TestKeysGenerateValidation(t *testing.T) {
	withFreshApp(t)

	_, err := executeForTest(t, newKeysGenerateCmd(), "--profile", "invalid")
	if err == nil || !strings.Contains(err.Error(), "unsupported crypto profile") {
		t.Fatalf("expected invalid profile error, got: %v", err)
	}
}

func TestKeysLocationValidation(t *testing.T) {
	withFreshApp(t)

	_, err := executeForTest(t, newKeysLocationCmd())
	if err == nil || !strings.Contains(err.Error(), "key-id is required") {
		t.Fatalf("expected missing key-id error, got: %v", err)
	}
}

func TestKeysRegisterAndRevokeValidation(t *testing.T) {
	t.Run("register requires key id when no active key", func(t *testing.T) {
		withFreshApp(t)

		_, err := executeForTest(t, newKeysRegisterCmd())
		if err == nil || !strings.Contains(err.Error(), "key-id is required") {
			t.Fatalf("expected key-id requirement error, got: %v", err)
		}
	})

	t.Run("revoke requires key id when no active key", func(t *testing.T) {
		withFreshApp(t)

		_, err := executeForTest(t, newKeysRevokeCmd())
		if err == nil || !strings.Contains(err.Error(), "key-id is required") {
			t.Fatalf("expected key-id requirement error, got: %v", err)
		}
	})
}

func TestPushAndPullArgAndKeyValidation(t *testing.T) {
	t.Run("push validates exact args", func(t *testing.T) {
		withFreshApp(t)
		_, err := executeForTest(t, newPushCmd())
		if err == nil || !strings.Contains(err.Error(), "accepts 2 arg") {
			t.Fatalf("expected arg validation error, got: %v", err)
		}
	})

	t.Run("pull validates exact args", func(t *testing.T) {
		withFreshApp(t)
		_, err := executeForTest(t, newPullCmd())
		if err == nil || !strings.Contains(err.Error(), "accepts 2 arg") {
			t.Fatalf("expected arg validation error, got: %v", err)
		}
	})

	t.Run("push requires active key", func(t *testing.T) {
		withFreshApp(t)
		app.Config.Origins["origin"] = config.Origin{Mode: config.ModeMongo, MongoURI: "mongodb://localhost:27017"}

		_, err := executeForTest(t, newPushCmd(), "origin", "my-app")
		if err == nil || !strings.Contains(err.Error(), "no active key configured") {
			t.Fatalf("expected active key error, got: %v", err)
		}
	})

	t.Run("pull requires active key", func(t *testing.T) {
		withFreshApp(t)
		app.Config.Origins["origin"] = config.Origin{Mode: config.ModeMongo, MongoURI: "mongodb://localhost:27017"}

		_, err := executeForTest(t, newPullCmd(), "origin", "my-app")
		if err == nil || !strings.Contains(err.Error(), "no active key configured") {
			t.Fatalf("expected active key error, got: %v", err)
		}
	})
}

func TestResolveOriginValidationMatrix(t *testing.T) {
	t.Run("requires default origin when not supplied", func(t *testing.T) {
		withFreshApp(t)

		_, _, err := resolveOrigin("")
		if err == nil || !strings.Contains(err.Error(), "origin is required") {
			t.Fatalf("expected missing origin error, got: %v", err)
		}
	})

	t.Run("fails for unknown origin", func(t *testing.T) {
		withFreshApp(t)
		app.Config.DefaultOrigin = "unknown"

		_, _, err := resolveOrigin("")
		if err == nil || !strings.Contains(err.Error(), "not configured") {
			t.Fatalf("expected unknown origin error, got: %v", err)
		}
	})

	t.Run("fails for api origin without url", func(t *testing.T) {
		withFreshApp(t)
		app.Config.DefaultOrigin = "api"
		app.Config.Origins["api"] = config.Origin{Mode: config.ModeAPI}

		_, _, err := resolveOrigin("")
		if err == nil || !strings.Contains(err.Error(), "empty api_base_url") {
			t.Fatalf("expected missing api url error, got: %v", err)
		}
	})
}
