package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

type LoginRequest struct {
	Email  string `json:"email"`
	APIKey string `json:"apiKey"`
}

type LoginResponse struct {
	Status           bool   `json:"status"`
	Reason           string `json:"reason"`
	Token            string `json:"token"`
	TokenType        string `json:"tokenType"`
	ExpiresInSeconds int    `json:"expiresInSeconds"`
}

type RegisterKeyRequest struct {
	KeyID       string `json:"key_id"`
	Profile     string `json:"profile"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
}

type RegisterKeyResponse struct {
	Status bool   `json:"status"`
	Reason string `json:"reason"`
}

type RemoteKey struct {
	KeyID       string `json:"keyId"`
	Profile     string `json:"profile"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   any    `json:"publicKey"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	LastUsedAt  string `json:"lastUsedAt"`
	RevokedAt   string `json:"revokedAt"`
}

type ListKeysResponse struct {
	Status bool        `json:"status"`
	Reason string      `json:"reason"`
	Keys   []RemoteKey `json:"keys"`
}

type PushSecretRequest struct {
	Origin       string `json:"origin"`
	Application  string `json:"application"`
	Envelope     string `json:"envelope"`
	Profile      string `json:"profile"`
	KeyID        string `json:"key_id"`
	Fingerprint  string `json:"fingerprint"`
	ChecksumSHA  string `json:"checksum_sha256"`
	ContentBytes int    `json:"content_bytes"`
}

type PushSecretResponse struct {
	Version int `json:"version"`
}

type PullSecretRequest struct {
	Origin      string `json:"origin"`
	Application string `json:"application"`
	KeyID       string `json:"key_id"`
	Fingerprint string `json:"fingerprint"`
}

type PullSecretResponse struct {
	Envelope    string `json:"envelope"`
	Version     int    `json:"version"`
	KeyID       string `json:"key_id"`
	Profile     string `json:"profile"`
	Fingerprint string `json:"fingerprint"`
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTP:    &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) PushSecret(req PushSecretRequest) (PushSecretResponse, error) {
	var response PushSecretResponse
	if err := c.postJSON("/api/cli/secrets/push", req, &response); err != nil {
		return PushSecretResponse{}, err
	}
	return response, nil
}

func (c *Client) PullSecret(req PullSecretRequest) (PullSecretResponse, error) {
	var response PullSecretResponse
	if err := c.postJSON("/api/cli/secrets/pull", req, &response); err != nil {
		return PullSecretResponse{}, err
	}
	return response, nil
}

func (c *Client) Login(req LoginRequest) (LoginResponse, error) {
	var response LoginResponse
	if err := c.postJSON("/api/cli/auth/login", req, &response); err != nil {
		return LoginResponse{}, err
	}
	return response, nil
}

func (c *Client) RegisterKey(req RegisterKeyRequest) (RegisterKeyResponse, error) {
	var response RegisterKeyResponse
	if err := c.postJSON("/api/cli/keys/register", req, &response); err != nil {
		return RegisterKeyResponse{}, err
	}
	return response, nil
}

func (c *Client) ListRemoteKeys() (ListKeysResponse, error) {
	httpReq, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/cli/keys/list", nil)
	if err != nil {
		return ListKeysResponse{}, fmt.Errorf("build request: %w", err)
	}
	if c.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	}

	res, err := c.HTTP.Do(httpReq)
	if err != nil {
		return ListKeysResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return ListKeysResponse{}, fmt.Errorf("read response body: %w", err)
	}

	if res.StatusCode >= 300 {
		return ListKeysResponse{}, fmt.Errorf("api error: status=%d body=%s", res.StatusCode, strings.TrimSpace(string(resBody)))
	}

	var response ListKeysResponse
	if err := json.Unmarshal(resBody, &response); err != nil {
		return ListKeysResponse{}, fmt.Errorf("decode response body: %w", err)
	}

	return response, nil
}

func (c *Client) postJSON(path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request body: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	}

	res, err := c.HTTP.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if res.StatusCode >= 300 {
		return fmt.Errorf("api error: status=%d body=%s", res.StatusCode, strings.TrimSpace(string(resBody)))
	}

	if len(bytes.TrimSpace(resBody)) == 0 || out == nil {
		return nil
	}

	if err := json.Unmarshal(resBody, out); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}

	return nil
}
