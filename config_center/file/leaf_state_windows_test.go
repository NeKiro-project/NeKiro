//go:build windows

package file

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

func TestLeafAccessPermissionOnReparsePointIsUnsafeState(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	leaf := "alpha.value"
	path := filepath.Join(root, leaf)
	if err := os.Symlink(outside, path); err != nil {
		if os.IsPermission(err) {
			t.Skip("test account cannot create a symlink")
		}
		t.Fatal(err)
	}
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	for iteration := 0; iteration < 32; iteration++ {
		missing, mapped := mapLeafAccessFailure(key, configcenter.OperationNext, root, leaf, fs.ErrPermission)
		if missing || !errors.Is(mapped, configcenter.ErrUnsafeState) {
			t.Fatalf("iteration %d missing=%v mapped error = %v, want unsafe_state", iteration, missing, mapped)
		}
	}
}

func TestLeafAccessReplacementWindowCanRemainMissing(t *testing.T) {
	root := t.TempDir()
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	missing, mapped := mapLeafAccessFailure(key, configcenter.OperationNext, root, "alpha.value", fs.ErrPermission)
	if !missing || mapped != nil {
		t.Fatalf("replacement-window result missing=%v error=%v, want missing", missing, mapped)
	}
}
