package parser

import (
	"os"

	ast "github.com/dh-kam/jkdeps/internal/ast"
)

// OSFileReader implements ast.SourceReader using the OS file system
type OSFileReader struct{}

// NewOSFileReader creates a new OSFileReader
func NewOSFileReader() *OSFileReader {
	return &OSFileReader{}
}

// Read reads the content of a file
func (r *OSFileReader) Read(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Exists checks if a file exists
func (r *OSFileReader) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir checks if a path is a directory
func (r *OSFileReader) IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Ensure OSFileReader implements ast.SourceReader
var _ ast.SourceReader = (*OSFileReader)(nil)

// InMemoryFileReader implements ast.SourceReader for testing
type InMemoryFileReader struct {
	files map[string][]byte
}

// NewInMemoryFileReader creates a new InMemoryFileReader
func NewInMemoryFileReader(files map[string][]byte) *InMemoryFileReader {
	return &InMemoryFileReader{
		files: files,
	}
}

// Read reads the content of a file from memory
func (r *InMemoryFileReader) Read(path string) ([]byte, error) {
	content, ok := r.files[path]
	if !ok {
		return nil, &os.PathError{Op: "read", Path: path, Err: os.ErrNotExist}
	}
	return content, nil
}

// Exists checks if a file exists in memory
func (r *InMemoryFileReader) Exists(path string) bool {
	_, ok := r.files[path]
	return ok
}

// IsDir checks if a path is a directory (always false for in-memory)
func (r *InMemoryFileReader) IsDir(path string) bool {
	return false
}

// Ensure InMemoryFileReader implements ast.SourceReader
var _ ast.SourceReader = (*InMemoryFileReader)(nil)
