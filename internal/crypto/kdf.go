package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

type privateBlob struct {
	Version    int    `json:"version"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func EncryptPrivateBlob(passphrase string, plaintext []byte) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("create salt: %w", err)
	}

	key := argon2.IDKey([]byte(passphrase), salt, 3, 64*1024, 4, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("init aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("create nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, []byte("nest-cli-private-key"))
	blob := privateBlob{
		Version:    1,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}

	return json.Marshal(blob)
}

func DecryptPrivateBlob(passphrase string, blobBytes []byte) ([]byte, error) {
	var blob privateBlob
	if err := json.Unmarshal(blobBytes, &blob); err != nil {
		return nil, fmt.Errorf("decode private key blob: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(blob.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(blob.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(blob.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	key := argon2.IDKey([]byte(passphrase), salt, 3, 64*1024, 4, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("init aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init gcm: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte("nest-cli-private-key"))
	if err != nil {
		return nil, fmt.Errorf("decrypt private key blob: %w", err)
	}

	return plaintext, nil
}
