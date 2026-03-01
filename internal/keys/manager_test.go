package keys

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/platanist/nest-cli/internal/crypto"
)

func TestManagerListEmpty(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	list, err := manager.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}

func TestManagerGenerateAndLoadPrivate(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	rec, err := manager.Generate(crypto.ProfileModern, "passphrase")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if rec.ID == "" || rec.Fingerprint == "" {
		t.Fatalf("record missing required fields: %+v", rec)
	}

	list, err := manager.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected exactly one key record, got %d", len(list))
	}

	found, err := manager.Find(rec.ID)
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if found.ID != rec.ID {
		t.Fatalf("find returned wrong key: %q", found.ID)
	}

	location, err := manager.Location(rec.ID)
	if err != nil {
		t.Fatalf("location failed: %v", err)
	}
	if location == "" {
		t.Fatal("location must not be empty")
	}

	private, loadedRec, err := manager.LoadPrivate(rec.ID, "passphrase")
	if err != nil {
		t.Fatalf("load private failed: %v", err)
	}
	if len(private) == 0 {
		t.Fatal("private key bytes should not be empty")
	}
	if loadedRec.ID != rec.ID {
		t.Fatalf("load private returned wrong record: %q", loadedRec.ID)
	}
}

func TestManagerLoadPrivateWrongPassphrase(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("new manager failed: %v", err)
	}

	rec, err := manager.Generate(crypto.ProfileModern, "correct")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	_, _, err = manager.LoadPrivate(rec.ID, "wrong")
	if err == nil || !strings.Contains(err.Error(), "decrypt private key blob") {
		t.Fatalf("expected wrong passphrase error, got: %v", err)
	}
}
