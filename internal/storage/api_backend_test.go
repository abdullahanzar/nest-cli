package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIBackendHealthCheck(t *testing.T) {
	t.Run("healthy endpoint", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/health" {
				t.Fatalf("unexpected health path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"ok":true}`)
		}))
		defer server.Close()

		backend := newAPIBackend(server.URL, "")
		status, err := backend.HealthCheck(context.Background())
		if err != nil {
			t.Fatalf("health check failed: %v", err)
		}
		if !strings.Contains(status, "api: ") {
			t.Fatalf("unexpected health status: %s", status)
		}
	})

	t.Run("non-success status returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, "down")
		}))
		defer server.Close()

		backend := newAPIBackend(server.URL, "")
		_, err := backend.HealthCheck(context.Background())
		if err == nil || !strings.Contains(err.Error(), "health status=503") {
			t.Fatalf("expected status code in error, got: %v", err)
		}
	})
}

func TestAPIBackendRevokeUnsupported(t *testing.T) {
	backend := newAPIBackend("https://example.com", "token")
	_, err := backend.RevokeKey(context.Background(), "key-1")
	if err == nil || !strings.Contains(err.Error(), "not supported in api mode") {
		t.Fatalf("expected unsupported revoke error, got: %v", err)
	}
}
