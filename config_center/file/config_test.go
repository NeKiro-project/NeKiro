package file

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

func TestReaderConfigRequiresEveryExplicitBound(t *testing.T) {
	root := t.TempDir()
	valid := ReaderConfig{Root: root, MaxPayloadBytes: 1, SubscriptionBuffer: 1}
	if err := validateReaderConfig(valid); err != nil {
		t.Fatalf("valid reader config rejected: %v", err)
	}
	for _, config := range []ReaderConfig{
		{Root: root, MaxPayloadBytes: 1},
		{Root: root, SubscriptionBuffer: 1},
		{Root: root, MaxPayloadBytes: -1, SubscriptionBuffer: 1},
		{Root: "relative", MaxPayloadBytes: 1, SubscriptionBuffer: 1},
		{Root: root + string(filepath.Separator), MaxPayloadBytes: 1, SubscriptionBuffer: 1},
	} {
		if err := validateReaderConfig(config); err == nil || !errors.Is(err, configcenter.ErrInvalid) {
			t.Errorf("reader config %#v error = %v, want invalid", config, err)
		}
	}
}

func TestPublisherConfigRequiresExplicitRegularMode(t *testing.T) {
	root := t.TempDir()
	if err := validatePublisherConfig(PublisherConfig{Root: root, MaxPayloadBytes: 1, FileMode: testPublisherMode()}); err != nil {
		t.Fatalf("valid publisher config rejected: %v", err)
	}
	for _, mode := range []fs.FileMode{0, os.ModeDir | 0o755, os.ModeSetuid | 0o600} {
		if err := validatePublisherConfig(PublisherConfig{Root: root, MaxPayloadBytes: 1, FileMode: mode}); err == nil || !errors.Is(err, configcenter.ErrInvalid) {
			t.Errorf("mode %#o error = %v, want invalid", mode, err)
		}
	}
}

func TestOpenPinnedRootRejectsInvalidRootState(t *testing.T) {
	root := t.TempDir()
	pinned, _, err := openPinnedRoot(root, configcenter.OperationObserve)
	if err != nil {
		t.Fatalf("open valid root: %v", err)
	}
	if err := pinned.Close(); err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(root, "missing")
	if _, _, err := openPinnedRoot(missing, configcenter.OperationObserve); err == nil || !errors.Is(err, configcenter.ErrUnavailable) {
		t.Fatalf("missing root error = %v, want unavailable", err)
	}
	filePath := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := openPinnedRoot(filePath, configcenter.OperationObserve); err == nil || !errors.Is(err, configcenter.ErrInvalid) {
		t.Fatalf("file root error = %v, want invalid", err)
	}

	symlink := filepath.Join(root, "root-link")
	if err := os.Symlink(root, symlink); err != nil {
		if os.IsPermission(err) {
			t.Skip("test account cannot create a symlink")
		}
		t.Fatal(err)
	}
	if _, _, err := openPinnedRoot(symlink, configcenter.OperationObserve); err == nil || !errors.Is(err, configcenter.ErrInvalid) {
		t.Fatalf("symlink root error = %v, want invalid", err)
	}
}

func TestOpenPinnedRootRejectsObservedIdentitySubstitution(t *testing.T) {
	root := configuredRoot(t)
	operations := productionFileOperations()
	operations.afterInitialRootLstat = func() {
		replaceConfiguredRoot(t, root)
	}

	pinned, identity, err := openPinnedRootWithOperations(root, configcenter.OperationObserve, operations)
	if pinned != nil || identity != nil {
		t.Fatalf("substituted root returned pinned state: root=%v identity=%v", pinned, identity)
	}
	if err == nil || !errors.Is(err, configcenter.ErrUnsafeState) {
		t.Fatalf("substituted root error = %v, want unsafe_state", err)
	}
}

func TestOpenPinnedRootMapsObservedNonDirectorySubstitutionToUnsafeState(t *testing.T) {
	root := configuredRoot(t)
	operations := productionFileOperations()
	operations.afterInitialRootLstat = func() {
		replaceConfiguredRootWithFile(t, root)
	}

	pinned, identity, err := openPinnedRootWithOperations(root, configcenter.OperationObserve, operations)
	if pinned != nil || identity != nil {
		t.Fatalf("non-directory substitution returned pinned state: root=%v identity=%v", pinned, identity)
	}
	if err == nil || !errors.Is(err, configcenter.ErrUnsafeState) {
		t.Fatalf("non-directory substitution error = %v, want unsafe_state", err)
	}
}

func TestRootIdentityCodeDistinguishesReplacementAndAbsence(t *testing.T) {
	root := configuredRoot(t)
	pinned, identity, err := openPinnedRoot(root, configcenter.OperationObserve)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pinned.Close() }()

	operations := productionFileOperations()
	if code := rootIdentityCode(root, pinned, identity, operations); code != "" {
		t.Fatalf("original root identity code = %q, want success", code)
	}
	baseLstat := operations.lstat
	replacement := filepath.Join(filepath.Dir(root), "replacement")
	if err := os.Mkdir(replacement, 0o700); err != nil {
		t.Fatal(err)
	}
	lost := false
	replaced := false
	operations.lstat = func(path string) (fs.FileInfo, error) {
		if path == root && lost {
			return nil, fs.ErrNotExist
		}
		if path == root && replaced {
			return baseLstat(replacement)
		}
		return baseLstat(path)
	}
	if runtime.GOOS == "windows" {
		replaced = true
	} else {
		replaceConfiguredRoot(t, root)
	}
	if code := rootIdentityCode(root, pinned, identity, operations); code != configcenter.CodeUnsafeState {
		t.Fatalf("replacement root identity code = %q, want unsafe_state", code)
	}
	if runtime.GOOS == "windows" {
		replaced = false
		lost = true
	} else if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if code := rootIdentityCode(root, pinned, identity, operations); code != configcenter.CodeUnavailable {
		t.Fatalf("missing root identity code = %q, want unavailable", code)
	}
}

func testPublisherMode() fs.FileMode {
	if runtime.GOOS == "windows" {
		return 0o666
	}
	return 0o640
}

func testWritablePublisherMode() fs.FileMode {
	if runtime.GOOS == "windows" {
		return 0o666
	}
	return 0o600
}

func configuredRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "configured")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func replaceConfiguredRoot(t *testing.T, root string) {
	t.Helper()
	previous := filepath.Join(filepath.Dir(root), "previous")
	if err := os.Rename(root, previous); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
}
