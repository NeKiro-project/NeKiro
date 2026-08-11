package usage_test

import (
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestQuickstartGoBlocksRemainPortableAndSynchronized(t *testing.T) {
	english := readGoBlocks(t, filepath.Join("..", "..", "README.md"))
	chinese := readGoBlocks(t, filepath.Join("..", "..", "README.zh-CN.md"))
	if len(english) != 2 {
		t.Fatalf("English Quickstart Go blocks = %d, want 2", len(english))
	}
	if !reflect.DeepEqual(english, chinese) {
		t.Fatal("English and Chinese Quickstart Go blocks differ")
	}
	for index, source := range english {
		if _, err := format.Source([]byte(source)); err != nil {
			t.Fatalf("Quickstart Go block %d does not parse: %v", index+1, err)
		}
		if strings.Contains(source, "NeKiro-Samples/internal/nacosregistration") {
			t.Fatalf("Quickstart Go block %d imports the Samples-internal registration adapter", index+1)
		}
		if !strings.Contains(source, "nekiro-sdk-go/agent/registration/nacos") {
			t.Fatalf("Quickstart Go block %d does not use the public SDK registration package", index+1)
		}
	}
}

func readGoBlocks(t *testing.T, path string) []string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	var blocks []string
	var current []string
	inBlock := false
	for _, line := range lines {
		switch {
		case !inBlock && line == "```go":
			inBlock = true
			current = nil
		case inBlock && line == "```":
			blocks = append(blocks, strings.Join(current, "\n")+"\n")
			inBlock = false
		case inBlock:
			current = append(current, line)
		}
	}
	if inBlock {
		t.Fatalf("unterminated Go block in %s", path)
	}
	return blocks
}
