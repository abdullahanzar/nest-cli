package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientLogin(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/auth/login" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Email != "user@example.com" || req.APIKey != "abc123" {
			t.Fatalf("unexpected payload: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(LoginResponse{
			Status:           true,
			Token:            "token-1",
			TokenType:        "Bearer",
			ExpiresInSeconds: 900,
		})
	}))
	defer ts.Close()

	client := New(ts.URL, "")
	resp, err := client.Login(LoginRequest{Email: "user@example.com", APIKey: "abc123"})
	if err != nil {
		t.Fatalf("login error: %v", err)
	}
	if !resp.Status || resp.Token != "token-1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestClientRegisterKeyAndAuthHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/keys/register" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		var req RegisterKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.KeyID != "k1" || req.Fingerprint != "f1" || req.PublicKey != "pub" || req.Profile != "modern" {
			t.Fatalf("unexpected register payload: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(RegisterKeyResponse{Status: true, Reason: "ok"})
	}))
	defer ts.Close()

	client := New(ts.URL, "token-123")
	resp, err := client.RegisterKey(RegisterKeyRequest{
		KeyID:       "k1",
		Profile:     "modern",
		Fingerprint: "f1",
		PublicKey:   "pub",
	})
	if err != nil {
		t.Fatalf("register key error: %v", err)
	}
	if !resp.Status {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestClientPushPullPayloads(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/cli/secrets/push":
			var req PushSecretRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode push request: %v", err)
			}
			if req.Origin != "origin" || req.Application != "app" || req.KeyID != "k1" || req.Fingerprint != "fp" {
				t.Fatalf("unexpected push payload: %+v", req)
			}
			_ = json.NewEncoder(w).Encode(PushSecretResponse{Version: 7})
		case "/api/cli/secrets/pull":
			var req PullSecretRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode pull request: %v", err)
			}
			if req.KeyID != "k1" || req.Fingerprint != "fp" {
				t.Fatalf("missing strict key identity on pull: %+v", req)
			}
			_ = json.NewEncoder(w).Encode(PullSecretResponse{
				Envelope:    "env",
				Version:     7,
				KeyID:       "k1",
				Profile:     "modern",
				Fingerprint: "fp",
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client := New(ts.URL, "token-123")
	pushResp, err := client.PushSecret(PushSecretRequest{
		Origin:       "origin",
		Application:  "app",
		Envelope:     "payload",
		Profile:      "modern",
		KeyID:        "k1",
		Fingerprint:  "fp",
		ChecksumSHA:  "abc",
		ContentBytes: 100,
	})
	if err != nil {
		t.Fatalf("push error: %v", err)
	}
	if pushResp.Version != 7 {
		t.Fatalf("unexpected push response: %+v", pushResp)
	}

	pullResp, err := client.PullSecret(PullSecretRequest{Origin: "origin", Application: "app", KeyID: "k1", Fingerprint: "fp"})
	if err != nil {
		t.Fatalf("pull error: %v", err)
	}
	if pullResp.Version != 7 || pullResp.KeyID != "k1" {
		t.Fatalf("unexpected pull response: %+v", pullResp)
	}
}

func TestClientAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, "key mismatch")
	}))
	defer ts.Close()

	client := New(ts.URL, "token")
	_, err := client.PullSecret(PullSecretRequest{Origin: "origin", Application: "app", KeyID: "wrong", Fingerprint: "wrong"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status=409") {
		t.Fatalf("expected status in error, got: %v", err)
	}
}

func TestClientListRemoteKeys(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/cli/keys/list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-xyz" {
			t.Fatalf("unexpected auth header: %s", got)
		}

		_ = json.NewEncoder(w).Encode(ListKeysResponse{
			Status: true,
			Keys: []RemoteKey{
				{KeyID: "k1", Profile: "modern", Fingerprint: "fp1", UpdatedAt: "2026-03-01T00:00:00Z"},
			},
		})
	}))
	defer ts.Close()

	client := New(ts.URL, "token-xyz")
	resp, err := client.ListRemoteKeys()
	if err != nil {
		t.Fatalf("remote-list error: %v", err)
	}
	if !resp.Status || len(resp.Keys) != 1 || resp.Keys[0].KeyID != "k1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
