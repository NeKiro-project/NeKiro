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

func TestLeafStateAfterAccessFailureClassifiesNativePathStates(t *testing.T) {
	root := t.TempDir()
	regular := "regular.value"
	if err := os.WriteFile(filepath.Join(root, regular), []byte("value"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		leaf string
		want leafFailureState
	}{
		{name: "invalid UTF-16 path", leaf: "invalid\x00.value", want: leafFailureUnknown},
		{name: "missing leaf", leaf: "missing.value", want: leafFailureMissing},
		{name: "directory leaf", leaf: ".", want: leafFailureUnsafe},
		{name: "regular leaf", leaf: regular, want: leafFailureUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := leafStateAfterAccessFailure(root, test.leaf); got != test.want {
				t.Fatalf("leaf state = %v, want %v", got, test.want)
			}
		})
	}
}

func TestReaderGetClassifiesWindowsLstatAccessFailures(t *testing.T) {
	tests := []struct {
		name     string
		makeLeaf func(t *testing.T, path string)
		want     error
	}{
		{name: "missing replacement", want: configcenter.ErrMissing},
		{name: "regular file", makeLeaf: writeWindowsTestLeaf, want: configcenter.ErrUnauthorized},
		{name: "reparse point", makeLeaf: writeWindowsTestSymlink, want: configcenter.ErrUnsafeState},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader, key, path := openWindowsLeafTestReader(t)
			if test.makeLeaf != nil {
				test.makeLeaf(t, path)
			}
			reader.operations.rootLeafLstat = func(*os.Root, string) (fs.FileInfo, error) {
				return nil, fs.ErrPermission
			}
			if _, err := reader.Get(t.Context(), key); !errors.Is(err, test.want) {
				t.Fatalf("Get error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestReaderGetClassifiesWindowsOpenAccessFailures(t *testing.T) {
	tests := []struct {
		name       string
		removeLeaf bool
		want       error
	}{
		{name: "regular file", want: configcenter.ErrUnauthorized},
		{name: "removed replacement", removeLeaf: true, want: configcenter.ErrMissing},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader, key, path := openWindowsLeafTestReader(t)
			writeWindowsTestLeaf(t, path)
			reader.operations.rootLeafOpen = func(*os.Root, string) (*os.File, error) {
				if test.removeLeaf {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				}
				return nil, fs.ErrPermission
			}
			if _, err := reader.Get(t.Context(), key); !errors.Is(err, test.want) {
				t.Fatalf("Get error = %v, want %v", err, test.want)
			}
		})
	}
}

func openWindowsLeafTestReader(t *testing.T) (*Reader, configcenter.Key, string) {
	t.Helper()
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return reader, key, filepath.Join(root, leaf)
}

func writeWindowsTestLeaf(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("value"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeWindowsTestSymlink(t *testing.T, path string) {
	t.Helper()
	outside := filepath.Join(t.TempDir(), "outside")
	writeWindowsTestLeaf(t, outside)
	if err := os.Symlink(outside, path); err != nil {
		if os.IsPermission(err) {
			t.Skip("test account cannot create a symlink")
		}
		t.Fatal(err)
	}
}
