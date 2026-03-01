package storage

import (
	"context"

	"github.com/platanist/nest-cli/internal/api"
)

type Backend interface {
	PushSecret(ctx context.Context, req api.PushSecretRequest) (api.PushSecretResponse, error)
	PullSecret(ctx context.Context, req api.PullSecretRequest) (api.PullSecretResponse, error)
	RegisterKey(ctx context.Context, req api.RegisterKeyRequest) (api.RegisterKeyResponse, error)
	RevokeKey(ctx context.Context, keyID string) (api.RegisterKeyResponse, error)
	ListRemoteKeys(ctx context.Context) (api.ListKeysResponse, error)
	HealthCheck(ctx context.Context) (string, error)
}
