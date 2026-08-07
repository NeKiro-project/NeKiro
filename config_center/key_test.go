package configcenter

import (
	"errors"
	"strings"
	"testing"
)

func TestParseKeyStrictGrammarAndBounds(t *testing.T) {
	maxKey := strings.Repeat("a", 32) + "/" + strings.Repeat("b", 32) + "/" + strings.Repeat("c", 32) + "/" + strings.Repeat("d", 32) + "/" + strings.Repeat("e", 28)
	if len(maxKey) != maxKeyLength {
		t.Fatalf("max key fixture length = %d, want %d", len(maxKey), maxKeyLength)
	}

	valid := []string{
		"a",
		"A0._:-",
		"alpha/B2/last-1",
		strings.Repeat("z", maxKeySegmentLength),
		maxKey,
	}
	for _, value := range valid {
		key, err := ParseKey(value)
		if err != nil {
			t.Fatalf("ParseKey(%q): %v", value, err)
		}
		if !key.Valid() || key.String() != value {
			t.Fatalf("ParseKey(%q) did not preserve the exact valid value", value)
		}
	}

	invalid := []string{
		"",
		"/a",
		"a/",
		"a//b",
		".",
		"..",
		"a/.",
		"a/..",
		" a",
		"a ",
		"a b",
		"a?b",
		"-a",
		"_a",
		"é",
		strings.Repeat("a", maxKeySegmentLength+1),
		maxKey + "a",
	}
	for _, value := range invalid {
		_, err := ParseKey(value)
		if err == nil || !errors.Is(err, ErrInvalid) {
			t.Errorf("ParseKey(%q) error = %v, want invalid", value, err)
		}
	}

	var zero Key
	if zero.Valid() {
		t.Fatal("zero Key is unexpectedly valid")
	}
}

func TestParseProviderIDStrictGrammarAndBounds(t *testing.T) {
	maxProvider := "a" + strings.Repeat("b", maxProviderIDLength-1)
	valid := []string{"file", "file.v1-_:", maxProvider}
	for _, value := range valid {
		provider, err := ParseProviderID(value)
		if err != nil {
			t.Fatalf("ParseProviderID(%q): %v", value, err)
		}
		if !provider.Valid() || provider.String() != value {
			t.Fatalf("ParseProviderID(%q) did not preserve the exact value", value)
		}
	}

	invalid := []string{"", "/", "file/other", " file", "file ", "-file", "_file", "é", maxProvider + "x"}
	for _, value := range invalid {
		_, err := ParseProviderID(value)
		if err == nil || !errors.Is(err, ErrInvalid) {
			t.Errorf("ParseProviderID(%q) error = %v, want invalid", value, err)
		}
	}
}
