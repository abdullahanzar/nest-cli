package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/platanist/nest-cli/internal/api"
)

type apiBackend struct {
	client  *api.Client
	baseURL string
}

func newAPIBackend(baseURL string, token string) Backend {
	return &apiBackend{
		client:  api.New(baseURL, token),
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (b *apiBackend) PushSecret(_ context.Context, req api.PushSecretRequest) (api.PushSecretResponse, error) {
	return b.client.PushSecret(req)
}

func (b *apiBackend) PullSecret(_ context.Context, req api.PullSecretRequest) (api.PullSecretResponse, error) {
	return b.client.PullSecret(req)
}

func (b *apiBackend) RegisterKey(_ context.Context, req api.RegisterKeyRequest) (api.RegisterKeyResponse, error) {
	return b.client.RegisterKey(req)
}

func (b *apiBackend) RevokeKey(_ context.Context, keyID string) (api.RegisterKeyResponse, error) {
	return api.RegisterKeyResponse{}, fmt.Errorf("key revoke is not supported in api mode yet (key=%s)", keyID)
}

func (b *apiBackend) ListRemoteKeys(_ context.Context) (api.ListKeysResponse, error) {
	return b.client.ListRemoteKeys()
}

func (b *apiBackend) HealthCheck(_ context.Context) (string, error) {
	req, err := http.NewRequest(http.MethodGet, b.baseURL+"/api/health", nil)
	if err != nil {
		return "", fmt.Errorf("build health request: %w", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("health request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("read health response: %w", err)
	}
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("health status=%d body=%s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	return fmt.Sprintf("api: %s (%d)", b.baseURL, res.StatusCode), nil
}
