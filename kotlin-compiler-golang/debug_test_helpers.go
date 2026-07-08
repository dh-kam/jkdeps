package kotlincompilergolang

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const debugSampleBasePath = "/tmp/jkdeps-samples"

func debugSamplePath(parts ...string) string {
	all := make([]string, 0, len(parts)+1)
	all = append(all, debugSampleBasePath)
	all = append(all, parts...)
	return filepath.Join(all...)
}

func readSampleSourceOrSkip(t *testing.T, path string) []byte {
	t.Helper()
	resolved := resolveDebugSamplePath(path)
	src, err := os.ReadFile(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("sample file not found. requested=%s resolved=%s. Set KCG_SAMPLE_ROOT if samples live elsewhere.", path, resolved)
			return nil
		}
		t.Fatalf("read sample file %q: %v", resolved, err)
	}
	return src
}

func resolveDebugSamplePath(path string) string {
	path = filepath.Clean(path)
	root := strings.TrimSpace(os.Getenv("KCG_SAMPLE_ROOT"))
	if root == "" {
		return path
	}

	root = filepath.Clean(root)
	defaultRoot := filepath.Clean(debugSampleBasePath)
	rel, err := filepath.Rel(defaultRoot, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	if rel == "." {
		return root
	}
	return filepath.Join(root, rel)
}
