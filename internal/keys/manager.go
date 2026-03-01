package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/platanist/nest-cli/internal/crypto"
	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/curve25519"
)

const serviceName = "nest-cli"

type Record struct {
	ID          string    `json:"id"`
	Profile     string    `json:"profile"`
	Fingerprint string    `json:"fingerprint"`
	Public      string    `json:"public"`
	Backend     string    `json:"backend"`
	FilePath    string    `json:"file_path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Manager struct {
	keysDir   string
	indexPath string
}

func NewManager(baseConfigPath string) (*Manager, error) {
	baseDir := filepath.Dir(baseConfigPath)
	keysDir := filepath.Join(baseDir, "keys")
	if err := os.MkdirAll(keysDir, 0o700); err != nil {
		return nil, fmt.Errorf("create keys directory: %w", err)
	}

	return &Manager{keysDir: keysDir, indexPath: filepath.Join(keysDir, "index.json")}, nil
}

func (m *Manager) Generate(profile crypto.Profile, passphrase string) (Record, error) {
	id := uuid.NewString()
	var publicBytes []byte
	var privateBytes []byte

	switch profile {
	case crypto.ProfileNIST:
		key, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return Record{}, fmt.Errorf("generate rsa key: %w", err)
		}
		pubPKIX, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			return Record{}, fmt.Errorf("marshal rsa public key: %w", err)
		}
		privatePKCS8, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return Record{}, fmt.Errorf("marshal rsa private key: %w", err)
		}
		publicBytes = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubPKIX})
		privateBytes = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privatePKCS8})
	case crypto.ProfileModern:
		priv := make([]byte, 32)
		if _, err := rand.Read(priv); err != nil {
			return Record{}, fmt.Errorf("generate x25519 private key: %w", err)
		}
		priv[0] &= 248
		priv[31] = (priv[31] & 127) | 64
		pub, err := curve25519.X25519(priv, curve25519.Basepoint)
		if err != nil {
			return Record{}, fmt.Errorf("derive x25519 public key: %w", err)
		}
		publicBytes = pub
		privateBytes = priv
	default:
		return Record{}, fmt.Errorf("unsupported profile %q", profile)
	}

	sum := sha256.Sum256(publicBytes)
	rec := Record{
		ID:          id,
		Profile:     string(profile),
		Fingerprint: hex.EncodeToString(sum[:]),
		Public:      base64.StdEncoding.EncodeToString(publicBytes),
		CreatedAt:   time.Now().UTC(),
	}

	blob, err := crypto.EncryptPrivateBlob(passphrase, privateBytes)
	if err != nil {
		return Record{}, err
	}

	if err := keyring.Set(serviceName, id, string(blob)); err == nil {
		rec.Backend = "keyring"
	} else {
		filePath := filepath.Join(m.keysDir, id+".enc")
		if err := os.WriteFile(filePath, blob, 0o600); err != nil {
			return Record{}, fmt.Errorf("store encrypted private key in file: %w", err)
		}
		rec.Backend = "file"
		rec.FilePath = filePath
	}

	records, err := m.List()
	if err != nil {
		return Record{}, err
	}
	records = append(records, rec)
	if err := m.writeIndex(records); err != nil {
		return Record{}, err
	}

	return rec, nil
}

func (m *Manager) List() ([]Record, error) {
	b, err := os.ReadFile(m.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Record{}, nil
		}
		return nil, fmt.Errorf("read key index: %w", err)
	}
	var records []Record
	if err := json.Unmarshal(b, &records); err != nil {
		return nil, fmt.Errorf("decode key index: %w", err)
	}
	return records, nil
}

func (m *Manager) Location(keyID string) (string, error) {
	records, err := m.List()
	if err != nil {
		return "", err
	}
	for _, rec := range records {
		if rec.ID != keyID {
			continue
		}
		if rec.Backend == "keyring" {
			return "os-keyring (service=nest-cli, key=" + keyID + ")", nil
		}
		if rec.FilePath != "" {
			return rec.FilePath, nil
		}
		return rec.Backend, nil
	}
	return "", fmt.Errorf("key %q not found", keyID)
}

func (m *Manager) Find(keyID string) (Record, error) {
	records, err := m.List()
	if err != nil {
		return Record{}, err
	}
	for _, rec := range records {
		if rec.ID == keyID {
			return rec, nil
		}
	}
	return Record{}, fmt.Errorf("key %q not found", keyID)
}

func (m *Manager) LoadPrivate(keyID string, passphrase string) ([]byte, Record, error) {
	rec, err := m.Find(keyID)
	if err != nil {
		return nil, Record{}, err
	}

	var blob []byte
	if rec.Backend == "keyring" {
		value, err := keyring.Get(serviceName, keyID)
		if err != nil {
			return nil, Record{}, fmt.Errorf("load key from keyring: %w", err)
		}
		blob = []byte(value)
	} else {
		if rec.FilePath == "" {
			return nil, Record{}, fmt.Errorf("key %q missing file path", keyID)
		}
		blob, err = os.ReadFile(rec.FilePath)
		if err != nil {
			return nil, Record{}, fmt.Errorf("read encrypted key file: %w", err)
		}
	}

	privateKey, err := crypto.DecryptPrivateBlob(passphrase, blob)
	if err != nil {
		return nil, Record{}, err
	}

	return privateKey, rec, nil
}

func (m *Manager) writeIndex(records []Record) error {
	b, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("encode key index: %w", err)
	}
	if err := os.WriteFile(m.indexPath, b, 0o600); err != nil {
		return fmt.Errorf("write key index: %w", err)
	}
	return nil
}
