package runtime_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/4current/relayops/internal/runtime"
)

func TestDBPathCreatesAppDir(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("USERPROFILE", tmp) // harmless on non-Windows

	dbPath, err := runtime.DBPath()
	if err != nil {
		t.Fatalf("DBPath: %v", err)
	}

	wantDir := filepath.Join(tmp, ".relayops")
	if _, err := os.Stat(wantDir); err != nil {
		t.Fatalf("expected runtime dir %s to exist: %v", wantDir, err)
	}

	wantDB := filepath.Join(wantDir, "relayops.db")
	if dbPath != wantDB {
		t.Fatalf("expected dbPath=%s, got %s", wantDB, dbPath)
	}
}
