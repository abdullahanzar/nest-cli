//go:build integration

package storage

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/platanist/nest-cli/internal/api"
	"github.com/platanist/nest-cli/internal/config"
	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMongoBackendIntegrationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	originName := "origin"
	uri := startMongoForTest(t)
	origin := config.Origin{
		Mode:          config.ModeMongo,
		MongoURI:      uri,
		MongoDatabase: "nest_cli_integration",
	}

	backend, err := NewBackend(originName, origin, "")
	if err != nil {
		t.Fatalf("create mongo backend failed: %v", err)
	}

	ctx := context.Background()
	registerResp, err := backend.RegisterKey(ctx, api.RegisterKeyRequest{
		KeyID:       "key-1",
		Profile:     "modern",
		Fingerprint: "fp-1",
		PublicKey:   "pub-1",
	})
	if err != nil {
		t.Fatalf("register key failed: %v", err)
	}
	if !registerResp.Status {
		t.Fatalf("expected successful register response: %+v", registerResp)
	}

	listResp, err := backend.ListRemoteKeys(ctx)
	if err != nil {
		t.Fatalf("list keys failed: %v", err)
	}
	if !listResp.Status || len(listResp.Keys) == 0 {
		t.Fatalf("expected non-empty remote key list: %+v", listResp)
	}

	firstPush, err := backend.PushSecret(ctx, api.PushSecretRequest{
		Origin:       originName,
		Application:  "my-app",
		Envelope:     "encrypted-v1",
		Profile:      "modern",
		KeyID:        "key-1",
		Fingerprint:  "fp-1",
		ChecksumSHA:  "checksum-1",
		ContentBytes: 18,
	})
	if err != nil {
		t.Fatalf("first push failed: %v", err)
	}
	if firstPush.Version != 1 {
		t.Fatalf("expected version 1, got %d", firstPush.Version)
	}

	secondPush, err := backend.PushSecret(ctx, api.PushSecretRequest{
		Origin:       originName,
		Application:  "my-app",
		Envelope:     "encrypted-v2",
		Profile:      "modern",
		KeyID:        "key-1",
		Fingerprint:  "fp-1",
		ChecksumSHA:  "checksum-2",
		ContentBytes: 19,
	})
	if err != nil {
		t.Fatalf("second push failed: %v", err)
	}
	if secondPush.Version != 2 {
		t.Fatalf("expected version 2, got %d", secondPush.Version)
	}

	pullResp, err := backend.PullSecret(ctx, api.PullSecretRequest{
		Origin:      originName,
		Application: "my-app",
		KeyID:       "key-1",
		Fingerprint: "fp-1",
	})
	if err != nil {
		t.Fatalf("pull secret failed: %v", err)
	}
	if pullResp.Version != 2 || pullResp.Envelope != "encrypted-v2" {
		t.Fatalf("unexpected pull response: %+v", pullResp)
	}

	health, err := backend.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	if !strings.Contains(health, "mongo:") {
		t.Fatalf("unexpected health output: %s", health)
	}

	revokeResp, err := backend.RevokeKey(ctx, "key-1")
	if err != nil {
		t.Fatalf("revoke key failed: %v", err)
	}
	if !revokeResp.Status {
		t.Fatalf("expected revoke success, got: %+v", revokeResp)
	}

	_, err = backend.PullSecret(ctx, api.PullSecretRequest{
		Origin:      originName,
		Application: "my-app",
		KeyID:       "key-1",
		Fingerprint: "fp-1",
	})
	if err == nil || !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("expected revoked key pull failure, got: %v", err)
	}
}

func startMongoForTest(t *testing.T) string {
	t.Helper()

	ctx := context.Background()
	request := testcontainers.ContainerRequest{
		Image:        "mongo:7",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForListeningPort("27017/tcp").WithStartupTimeout(90 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start mongo container failed: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("resolve container host failed: %v", err)
	}
	port, err := container.MappedPort(ctx, "27017/tcp")
	if err != nil {
		t.Fatalf("resolve mapped port failed: %v", err)
	}

	return fmt.Sprintf("mongodb://%s:%s", host, port.Port())
}
