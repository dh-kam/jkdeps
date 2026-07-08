package parser

import (
	"testing"

	ast "github.com/dh-kam/jkdeps/internal/ast"
)

// TestInMemoryFileReader tests the InMemoryFileReader implementation
func TestInMemoryFileReader(t *testing.T) {
	files := map[string][]byte{
		"/test/File.java": []byte("package test; class File {}"),
		"/test/Other.kt":  []byte("package test; class Other"),
	}

	reader := NewInMemoryFileReader(files)

	t.Run("Read existing file", func(t *testing.T) {
		content, err := reader.Read("/test/File.java")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "package test; class File {}"
		if string(content) != expected {
			t.Errorf("expected %q, got %q", expected, string(content))
		}
	})

	t.Run("Read non-existent file", func(t *testing.T) {
		_, err := reader.Read("/nonexistent.java")
		if err == nil {
			t.Fatal("expected error for non-existent file, got nil")
		}
	})

	t.Run("Exists returns true for existing file", func(t *testing.T) {
		if !reader.Exists("/test/File.java") {
			t.Error("expected Exists to return true")
		}
	})

	t.Run("Exists returns false for non-existent file", func(t *testing.T) {
		if reader.Exists("/nonexistent.java") {
			t.Error("expected Exists to return false")
		}
	})

	t.Run("IsDir always returns false", func(t *testing.T) {
		if reader.IsDir("/test/File.java") {
			t.Error("expected IsDir to return false")
		}
	})
}

// TestANTLRParserWithCustomReader tests dependency injection with custom reader
func TestANTLRParserWithCustomReader(t *testing.T) {
	sourceCode := []byte("package test; public class Test {}")

	files := map[string][]byte{
		"/test/Test.java": sourceCode,
	}

	reader := NewInMemoryFileReader(files)
	parser := NewANTLRParserWithReader(JavaGrammar20, ast.LanguageJava, reader)

	result, err := parser.ParseFile("/test/Test.java", ast.ParseOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got diagnostics: %v", result.Diagnostics)
	}
}

// TestOSFileReader tests the OSFileReader implementation
func TestOSFileReader(t *testing.T) {
	reader := NewOSFileReader()

	t.Run("Implements SourceReader interface", func(t *testing.T) {
		var _ ast.SourceReader = reader
	})

	t.Run("Exists for current directory", func(t *testing.T) {
		if !reader.Exists(".") {
			t.Error("expected current directory to exist")
		}
	})

	t.Run("IsDir for current directory", func(t *testing.T) {
		if !reader.IsDir(".") {
			t.Error("expected current directory to be a directory")
		}
	})

	t.Run("Read existing file", func(t *testing.T) {
		// Read this test file itself
		content, err := reader.Read("file_reader_test.go")
		if err != nil {
			t.Fatalf("unexpected error reading test file: %v", err)
		}
		if len(content) == 0 {
			t.Error("expected non-empty content")
		}
		// Verify it's actually Go code
		if !contains(content, []byte("package parser")) {
			t.Error("expected file content to contain 'package parser'")
		}
	})

	t.Run("Read non-existent file", func(t *testing.T) {
		_, err := reader.Read("/nonexistent_file_xyz_123.java")
		if err == nil {
			t.Fatal("expected error for non-existent file, got nil")
		}
	})

	t.Run("Exists returns false for non-existent file", func(t *testing.T) {
		if reader.Exists("/nonexistent_file_xyz_123.java") {
			t.Error("expected Exists to return false for non-existent file")
		}
	})

	t.Run("IsDir returns false for file", func(t *testing.T) {
		if reader.IsDir("file_reader_test.go") {
			t.Error("expected IsDir to return false for regular file")
		}
	})
}

func contains(content []byte, substr []byte) bool {
	return len(content) >= len(substr) && indexOf(content, substr) >= 0
}

func indexOf(content []byte, substr []byte) int {
	for i := 0; i <= len(content)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if content[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
