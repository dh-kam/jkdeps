package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadExternalIndexEmptyPaths(t *testing.T) {
	paths, index, err := loadExternalIndex(nil)
	if err != nil {
		t.Fatalf("loadExternalIndex(nil) error = %v", err)
	}
	if len(paths) != 0 {
		t.Fatalf("expected no normalized paths, got %v", paths)
	}
	if len(index.Packages) != 0 || len(index.Symbols) != 0 {
		t.Fatalf("expected empty external index, got packages=%d symbols=%d", len(index.Packages), len(index.Symbols))
	}
}

func TestLoadExternalIndexDeduplicatesAndLoadsInventory(t *testing.T) {
	root := t.TempDir()
	inventoryPath := filepath.Join(root, "inventory.json")
	payload := []byte(`{
  "packages": [
    {"package": "kotlin.collections", "count": 1},
    {"package": "kotlin.io", "count": 2}
  ],
  "symbols": [
    "kotlin.collections.List"
  ]
}`)
	if err := os.WriteFile(inventoryPath, payload, 0o644); err != nil {
		t.Fatalf("write inventory: %v", err)
	}

	paths, index, err := loadExternalIndex([]string{" " + inventoryPath + " ", inventoryPath})
	if err != nil {
		t.Fatalf("loadExternalIndex() error = %v", err)
	}
	wantPaths := []string{inventoryPath}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("normalized paths = %v, want %v", paths, wantPaths)
	}
	if !index.HasPackage("kotlin.collections") || !index.HasPackage("kotlin.io") {
		t.Fatalf("expected inventory packages to be loaded, got %v", index.PackageNames())
	}
	if !index.HasSymbol("kotlin.collections.List") {
		t.Fatalf("expected inventory symbol to be loaded, got %v", index.SymbolNames())
	}
}

func TestLoadExternalIndexReturnsLoadErrors(t *testing.T) {
	root := t.TempDir()
	brokenPath := filepath.Join(root, "broken.json")
	if err := os.WriteFile(brokenPath, []byte(`{"packages":[`), 0o644); err != nil {
		t.Fatalf("write broken inventory: %v", err)
	}

	if _, _, err := loadExternalIndex([]string{brokenPath}); err == nil {
		t.Fatal("expected inventory load error, got nil")
	}
}
