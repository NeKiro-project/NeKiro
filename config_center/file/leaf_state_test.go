package file

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

func TestLeafAccessPermissionOnRegularFileRemainsUnauthorized(t *testing.T) {
	root := t.TempDir()
	leaf := "alpha.value"
	if err := os.WriteFile(filepath.Join(root, leaf), []byte("value"), 0o600); err != nil {
		t.Fatal(err)
	}
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	missing, mapped := mapLeafAccessFailure(key, configcenter.OperationNext, root, leaf, fs.ErrPermission)
	if missing || !errors.Is(mapped, configcenter.ErrUnauthorized) {
		t.Fatalf("regular-file permission result missing=%v error=%v, want unauthorized", missing, mapped)
	}
}
