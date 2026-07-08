package mixedgraph

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestParseRepositoryDrainsWorkersOnReadError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission semantics differ on Windows")
	}

	root := t.TempDir()
	readablePath := filepath.Join(root, "Readable.java")
	if err := os.WriteFile(readablePath, []byte("class Readable {}"), 0o644); err != nil {
		t.Fatalf("write readable java: %v", err)
	}

	inaccessiblePath := filepath.Join(root, "Inaccessible.java")
	if err := os.WriteFile(inaccessiblePath, []byte("class Broken {}"), 0o000); err != nil {
		t.Fatalf("write inaccessible java: %v", err)
	}
	if err := os.Chmod(inaccessiblePath, 0o000); err != nil {
		t.Fatalf("chmod inaccessible java: %v", err)
	}

	before := runtime.NumGoroutine()
	for range 20 {
		_, err := ParseRepository(root, ParseOptions{Workers: 2})
		if err == nil {
			t.Fatalf("expected read error, got nil")
		}
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after > before+4 {
		t.Fatalf("goroutines grew unexpectedly after repeated read errors: before=%d after=%d", before, after)
	}
}
