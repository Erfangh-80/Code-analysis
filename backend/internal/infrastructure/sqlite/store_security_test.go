package sqlite_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Erfangh-80/code-analysis/backend/internal/infrastructure/sqlite"
)

func TestStoreCreatesPrivateDatabaseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analyses.db")
	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}
	defer store.Close()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat SQLite database: %v", err)
	}
	if permissions := info.Mode().Perm(); permissions != 0o600 {
		t.Errorf("database permissions = %o, want 600", permissions)
	}
}
