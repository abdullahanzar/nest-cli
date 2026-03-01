package storage

import (
	"strings"
	"testing"

	"github.com/platanist/nest-cli/internal/config"
)

func TestNewBackendValidation(t *testing.T) {
	t.Run("api mode requires api url", func(t *testing.T) {
		_, err := NewBackend("api-origin", config.Origin{Mode: config.ModeAPI}, "token")
		if err == nil || !strings.Contains(err.Error(), "requires api_base_url") {
			t.Fatalf("expected missing api url error, got: %v", err)
		}
	})

	t.Run("mongo mode requires mongo uri", func(t *testing.T) {
		t.Setenv("NEST_MONGO_URI_MONGO_ORIGIN", "")
		t.Setenv("NEST_MONGO_URI", "")
		_, err := NewBackend("mongo-origin", config.Origin{Mode: config.ModeMongo}, "")
		if err == nil || !strings.Contains(err.Error(), "requires mongo_uri") {
			t.Fatalf("expected missing mongo uri error, got: %v", err)
		}
	})

	t.Run("unknown mode falls back to api when api url is present", func(t *testing.T) {
		backend, err := NewBackend("compat", config.Origin{Mode: "legacy", APIBaseURL: "https://example.com"}, "token")
		if err != nil {
			t.Fatalf("expected api fallback success, got: %v", err)
		}
		if backend == nil {
			t.Fatal("backend should not be nil")
		}
	})
}
