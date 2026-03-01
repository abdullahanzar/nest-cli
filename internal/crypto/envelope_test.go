package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"

	"golang.org/x/crypto/curve25519"
)

func TestEncryptDecryptEnvelopeRoundTrip(t *testing.T) {
	plaintext := []byte("DB_USER=app\nDB_PASS=s3cret\n")
	aad := []byte("origin:app:key-1")

	t.Run("modern", func(t *testing.T) {
		priv, pub := mustX25519KeyPair(t)
		envelopeBytes, err := EncryptEnvelope(plaintext, ProfileModern, "key-1", pub, aad)
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}

		decrypted, envelope, err := DecryptEnvelope(envelopeBytes, priv, aad)
		if err != nil {
			t.Fatalf("decrypt failed: %v", err)
		}
		if string(decrypted) != string(plaintext) {
			t.Fatalf("plaintext mismatch: %q", string(decrypted))
		}
		if envelope.KeyID != "key-1" {
			t.Fatalf("unexpected key id: %q", envelope.KeyID)
		}
	})

	t.Run("nist", func(t *testing.T) {
		priv, pub := mustRSAKeyPairPEM(t)
		envelopeBytes, err := EncryptEnvelope(plaintext, ProfileNIST, "key-1", pub, aad)
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}

		decrypted, envelope, err := DecryptEnvelope(envelopeBytes, priv, aad)
		if err != nil {
			t.Fatalf("decrypt failed: %v", err)
		}
		if string(decrypted) != string(plaintext) {
			t.Fatalf("plaintext mismatch: %q", string(decrypted))
		}
		if envelope.Profile != string(ProfileNIST) {
			t.Fatalf("unexpected profile: %q", envelope.Profile)
		}
	})
}

func TestDecryptEnvelopeRejectsAADMismatchAndTamper(t *testing.T) {
	priv, pub := mustX25519KeyPair(t)
	originalAAD := []byte("origin:app:key-1")

	envelopeBytes, err := EncryptEnvelope([]byte("TOKEN=abc123"), ProfileModern, "key-1", pub, originalAAD)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	_, _, err = DecryptEnvelope(envelopeBytes, priv, []byte("origin:app:key-2"))
	if err == nil || !strings.Contains(err.Error(), "unwrap data key") {
		t.Fatalf("expected aad mismatch error, got: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(envelopeBytes, &env); err != nil {
		t.Fatalf("unmarshal envelope failed: %v", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		t.Fatalf("decode ciphertext failed: %v", err)
	}
	ciphertext[len(ciphertext)-1] ^= 0x01
	env.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
	tampered, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal tampered envelope failed: %v", err)
	}

	_, _, err = DecryptEnvelope(tampered, priv, originalAAD)
	if err == nil || !strings.Contains(err.Error(), "decrypt payload") {
		t.Fatalf("expected tamper detection error, got: %v", err)
	}
}

func TestPrivateBlobRoundTripAndWrongPassphrase(t *testing.T) {
	blob, err := EncryptPrivateBlob("correct", []byte("private-material"))
	if err != nil {
		t.Fatalf("encrypt private blob failed: %v", err)
	}

	plaintext, err := DecryptPrivateBlob("correct", blob)
	if err != nil {
		t.Fatalf("decrypt private blob failed: %v", err)
	}
	if string(plaintext) != "private-material" {
		t.Fatalf("unexpected plaintext: %q", string(plaintext))
	}

	_, err = DecryptPrivateBlob("wrong", blob)
	if err == nil || !strings.Contains(err.Error(), "decrypt private key blob") {
		t.Fatalf("expected wrong passphrase error, got: %v", err)
	}
}

func mustX25519KeyPair(t *testing.T) ([]byte, []byte) {
	t.Helper()
	priv := make([]byte, 32)
	if _, err := rand.Read(priv); err != nil {
		t.Fatalf("generate private key failed: %v", err)
	}
	priv[0] &= 248
	priv[31] = (priv[31] & 127) | 64
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		t.Fatalf("derive public key failed: %v", err)
	}
	return priv, pub
}

func mustRSAKeyPairPEM(t *testing.T) ([]byte, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key failed: %v", err)
	}
	pubPKIX, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal rsa public key failed: %v", err)
	}
	privatePKCS8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal rsa private key failed: %v", err)
	}
	pub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubPKIX})
	priv := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privatePKCS8})
	return priv, pub
}
