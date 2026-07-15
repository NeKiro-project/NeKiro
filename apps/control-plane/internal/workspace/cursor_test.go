package workspace

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

func TestInstallationCursorRoundTripsOrderingPosition(t *testing.T) {
	position := InstallationPosition{
		InstalledAt:    time.Date(2026, 7, 15, 10, 0, 0, 123456000, time.FixedZone("CST", 8*60*60)),
		InstallationID: "installation-a",
	}
	cursor, err := EncodeInstallationCursor("workspace-a", 25, position)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeInstallationCursor(cursor, "workspace-a", 25)
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.InstalledAt.Equal(position.InstalledAt) || decoded.InstallationID != position.InstallationID {
		t.Fatalf("decoded position = %#v, want %#v", decoded, position)
	}
}

func TestInstallationCursorRejectsBindingAndPayloadViolations(t *testing.T) {
	position := InstallationPosition{InstalledAt: time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC), InstallationID: "installation-a"}
	cursor, err := EncodeInstallationCursor("workspace-a", 25, position)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name      string
		value     string
		workspace string
		limit     int
	}{
		{name: "workspace mismatch", value: cursor, workspace: "workspace-b", limit: 25},
		{name: "limit mismatch", value: cursor, workspace: "workspace-a", limit: 24},
		{name: "empty", value: "", workspace: "workspace-a", limit: 25},
		{name: "malformed", value: "not-base64", workspace: "workspace-a", limit: 25},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeInstallationCursor(test.value, test.workspace, test.limit); !errors.Is(err, ErrInvalid) {
				t.Fatalf("DecodeInstallationCursor error = %v, want invalid", err)
			}
		})
	}

	duplicate := base64.RawURLEncoding.EncodeToString([]byte(`{"version":1,"version":1,"workspaceId":"workspace-a","limit":25,"installedAt":"2026-07-15T10:00:00Z","installationId":"installation-a"}`))
	if _, err := DecodeInstallationCursor(duplicate, "workspace-a", 25); !errors.Is(err, ErrInvalid) {
		t.Fatalf("duplicate cursor error = %v, want invalid", err)
	}
	trailing := base64.RawURLEncoding.EncodeToString([]byte(`{"version":1,"workspaceId":"workspace-a","limit":25,"installedAt":"2026-07-15T10:00:00Z","installationId":"installation-a"} {}`))
	if _, err := DecodeInstallationCursor(trailing, "workspace-a", 25); !errors.Is(err, ErrInvalid) {
		t.Fatalf("trailing cursor error = %v, want invalid", err)
	}
}
