package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

type Envelope struct {
	Version         int    `json:"version"`
	Profile         string `json:"profile"`
	KeyID           string `json:"key_id"`
	WrapNonce       string `json:"wrap_nonce,omitempty"`
	DataNonce       string `json:"data_nonce"`
	WrappedDataKey  string `json:"wrapped_data_key"`
	Ciphertext      string `json:"ciphertext"`
	EphemeralPublic string `json:"ephemeral_public,omitempty"`
}

func EncryptEnvelope(plaintext []byte, profile Profile, keyID string, publicKey []byte, aad []byte) ([]byte, error) {
	dataKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dataKey); err != nil {
		return nil, fmt.Errorf("create data key: %w", err)
	}

	envelope := Envelope{Version: 1, Profile: string(profile), KeyID: keyID}

	switch profile {
	case ProfileNIST:
		wrapped, err := wrapNIST(dataKey, publicKey)
		if err != nil {
			return nil, err
		}
		envelope.WrappedDataKey = base64.StdEncoding.EncodeToString(wrapped)

		aesCipher, err := aes.NewCipher(dataKey)
		if err != nil {
			return nil, fmt.Errorf("init aes: %w", err)
		}
		gcm, err := cipher.NewGCM(aesCipher)
		if err != nil {
			return nil, fmt.Errorf("init gcm: %w", err)
		}
		nonce := make([]byte, gcm.NonceSize())
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return nil, fmt.Errorf("create data nonce: %w", err)
		}
		ciphertext := gcm.Seal(nil, nonce, plaintext, aad)
		envelope.DataNonce = base64.StdEncoding.EncodeToString(nonce)
		envelope.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
	case ProfileModern:
		ephPriv := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, ephPriv); err != nil {
			return nil, fmt.Errorf("create ephemeral key: %w", err)
		}
		ephPriv[0] &= 248
		ephPriv[31] = (ephPriv[31] & 127) | 64

		ephPub, err := curve25519.X25519(ephPriv, curve25519.Basepoint)
		if err != nil {
			return nil, fmt.Errorf("derive ephemeral public key: %w", err)
		}
		shared, err := curve25519.X25519(ephPriv, publicKey)
		if err != nil {
			return nil, fmt.Errorf("derive shared key: %w", err)
		}

		wrapKey := make([]byte, 32)
		kdf := hkdf.New(sha256.New, shared, nil, append([]byte("nest-cli-wrap"), aad...))
		if _, err := io.ReadFull(kdf, wrapKey); err != nil {
			return nil, fmt.Errorf("derive wrapping key: %w", err)
		}

		wrapAEAD, err := chacha20poly1305.NewX(wrapKey)
		if err != nil {
			return nil, fmt.Errorf("init wrapping aead: %w", err)
		}
		wrapNonce := make([]byte, wrapAEAD.NonceSize())
		if _, err := io.ReadFull(rand.Reader, wrapNonce); err != nil {
			return nil, fmt.Errorf("create wrap nonce: %w", err)
		}
		wrapped := wrapAEAD.Seal(nil, wrapNonce, dataKey, aad)

		dataAEAD, err := chacha20poly1305.NewX(dataKey)
		if err != nil {
			return nil, fmt.Errorf("init data aead: %w", err)
		}
		dataNonce := make([]byte, dataAEAD.NonceSize())
		if _, err := io.ReadFull(rand.Reader, dataNonce); err != nil {
			return nil, fmt.Errorf("create data nonce: %w", err)
		}
		ciphertext := dataAEAD.Seal(nil, dataNonce, plaintext, aad)

		envelope.WrapNonce = base64.StdEncoding.EncodeToString(wrapNonce)
		envelope.DataNonce = base64.StdEncoding.EncodeToString(dataNonce)
		envelope.WrappedDataKey = base64.StdEncoding.EncodeToString(wrapped)
		envelope.EphemeralPublic = base64.StdEncoding.EncodeToString(ephPub)
		envelope.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
	default:
		return nil, fmt.Errorf("unsupported profile %q", profile)
	}

	return json.Marshal(envelope)
}

func DecryptEnvelope(envelopeBytes []byte, privateKey []byte, aad []byte) ([]byte, Envelope, error) {
	var envelope Envelope
	if err := json.Unmarshal(envelopeBytes, &envelope); err != nil {
		return nil, Envelope{}, fmt.Errorf("decode envelope: %w", err)
	}

	profile, err := ParseProfile(envelope.Profile)
	if err != nil {
		return nil, Envelope{}, err
	}

	wrapped, err := base64.StdEncoding.DecodeString(envelope.WrappedDataKey)
	if err != nil {
		return nil, Envelope{}, fmt.Errorf("decode wrapped key: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, Envelope{}, fmt.Errorf("decode ciphertext: %w", err)
	}

	var dataKey []byte
	switch profile {
	case ProfileNIST:
		dataKey, err = unwrapNIST(wrapped, privateKey)
		if err != nil {
			return nil, Envelope{}, err
		}
		nonce, err := base64.StdEncoding.DecodeString(envelope.DataNonce)
		if err != nil {
			return nil, Envelope{}, fmt.Errorf("decode data nonce: %w", err)
		}
		aesCipher, err := aes.NewCipher(dataKey)
		if err != nil {
			return nil, Envelope{}, fmt.Errorf("init aes: %w", err)
		}
		gcm, err := cipher.NewGCM(aesCipher)
		if err != nil {
			return nil, Envelope{}, fmt.Errorf("init gcm: %w", err)
		}
		plaintext, err := gcm.Open(nil, nonce, ciphertext, aad)
		if err != nil {
			return nil, Envelope{}, fmt.Errorf("decrypt payload: %w", err)
		}
		return plaintext, envelope, nil
	case ProfileModern:
		ephemeralPub, err := base64.StdEncoding.DecodeString(envelope.EphemeralPublic)
		if err != nil {
			return nil, Envelope{}, fmt.Errorf("decode ephemeral pubkey: %w", err)
		}
		wrapNonce, err := base64.StdEncoding.DecodeString(envelope.WrapNonce)
		if err != nil {
			return nil, Envelope{}, fmt.Errorf("decode wrap nonce: %w", err)
		}
		shared, err := curve25519.X25519(privateKey, ephemeralPub)
		if err != nil {
			return nil, Envelope{}, fmt.Errorf("derive shared key: %w", err)
		}
		wrapKey := make([]byte, 32)
		kdf := hkdf.New(sha256.New, shared, nil, append([]byte("nest-cli-wrap"), aad...))
		if _, err := io.ReadFull(kdf, wrapKey); err != nil {
			return nil, Envelope{}, fmt.Errorf("derive wrapping key: %w", err)
		}
		wrapAEAD, err := chacha20poly1305.NewX(wrapKey)
		if err != nil {
			return nil, Envelope{}, fmt.Errorf("init wrapping aead: %w", err)
		}
		dataKey, err = wrapAEAD.Open(nil, wrapNonce, wrapped, aad)
		if err != nil {
			return nil, Envelope{}, fmt.Errorf("unwrap data key: %w", err)
		}
		dataNonce, err := base64.StdEncoding.DecodeString(envelope.DataNonce)
		if err != nil {
			return nil, Envelope{}, fmt.Errorf("decode data nonce: %w", err)
		}
		dataAEAD, err := chacha20poly1305.NewX(dataKey)
		if err != nil {
			return nil, Envelope{}, fmt.Errorf("init data aead: %w", err)
		}
		plaintext, err := dataAEAD.Open(nil, dataNonce, ciphertext, aad)
		if err != nil {
			return nil, Envelope{}, fmt.Errorf("decrypt payload: %w", err)
		}
		return plaintext, envelope, nil
	default:
		return nil, Envelope{}, fmt.Errorf("unsupported profile %q", profile)
	}
}

func wrapNIST(dataKey []byte, publicKeyPEM []byte) ([]byte, error) {
	block, _ := pem.Decode(publicKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("decode public key pem")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	rsaPub, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
	}
	wrapped, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, dataKey, []byte("nest-cli-env"))
	if err != nil {
		return nil, fmt.Errorf("wrap data key: %w", err)
	}
	return wrapped, nil
}

func unwrapNIST(wrapped []byte, privateKeyPEM []byte) ([]byte, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("decode private key pem")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	rsaPriv, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}
	dataKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, rsaPriv, wrapped, []byte("nest-cli-env"))
	if err != nil {
		return nil, fmt.Errorf("unwrap data key: %w", err)
	}
	return dataKey, nil
}
