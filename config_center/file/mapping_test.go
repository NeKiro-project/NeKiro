package file

import (
	"errors"
	"strings"
	"testing"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

func TestMapKeyIsFlatReversibleAndBounded(t *testing.T) {
	maxKeyValue := strings.Repeat("a", 32) + "/" + strings.Repeat("b", 32) + "/" + strings.Repeat("c", 32) + "/" + strings.Repeat("d", 32) + "/" + strings.Repeat("e", 28)
	key, err := configcenter.ParseKey(maxKeyValue)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if len(leaf) != MaxMappedLeafLength {
		t.Fatalf("mapped max leaf length = %d, want %d", len(leaf), MaxMappedLeafLength)
	}
	if strings.ContainsAny(leaf, `/\\`) {
		t.Fatalf("mapped leaf is not flat: %q", leaf)
	}
	recovered, err := UnmapLeaf(leaf)
	if err != nil || recovered != key {
		t.Fatalf("UnmapLeaf(MapKey(key)) = %q, %v", recovered.String(), err)
	}
	if recovered, ok := mappedKeyFromEventName(leaf); !ok || recovered != key {
		t.Fatalf("event mapping = %q, %v", recovered.String(), ok)
	}
}

func TestMappingRejectsInvalidKeysAndNonCanonicalLeaves(t *testing.T) {
	var invalid configcenter.Key
	if _, err := MapKey(invalid); err == nil || !errors.Is(err, configcenter.ErrInvalid) {
		t.Fatalf("invalid key mapping error = %v", err)
	}
	for _, leaf := range []string{
		"other",
		mappingPrefix + mappingSuffix,
		mappingPrefix + "=" + mappingSuffix,
		mappingPrefix + "YQ=" + mappingSuffix,
		mappingPrefix + "eB" + mappingSuffix,
	} {
		if _, err := UnmapLeaf(leaf); err == nil || !errors.Is(err, configcenter.ErrInvalid) {
			t.Errorf("UnmapLeaf(%q) error = %v, want invalid", leaf, err)
		}
	}
	if _, ok := mappedKeyFromEventName("unrelated.tmp"); ok {
		t.Fatal("unrelated event name was interpreted as a key")
	}
}
