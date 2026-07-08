package kotlincompilergolang

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadInventoryPackages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.json")
	payload := `{
  "jar_path": "/tmp/kotlin-compiler-embeddable.jar",
  "class_files": 2,
  "top_level_classes": 2,
  "builtins_files": 1,
  "packages": [
    {"package":"kotlin.coroutines","count": 10},
    {"package":"org.jetbrains.kotlin.psi","count": 20}
  ]
}`
	writeFixture(t, path, payload)

	packages, err := LoadInventoryPackages(path)
	if err != nil {
		t.Fatalf("LoadInventoryPackages returned error: %v", err)
	}
	if len(packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(packages))
	}
	if _, ok := packages["kotlin.coroutines"]; !ok {
		t.Fatalf("expected kotlin.coroutines package")
	}
	if _, ok := packages["org.jetbrains.kotlin.psi"]; !ok {
		t.Fatalf("expected org.jetbrains.kotlin.psi package")
	}
}

func TestLoadExternalIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.json")
	payload := `{
  "source_jars": ["/tmp/a.jar"],
  "class_files": 3,
  "top_level_classes": 2,
  "builtins_files": 1,
  "metadata_files": 2,
  "packages": [
    {"package":"kotlin.coroutines","count": 10},
    {"package":"kotlin.js","count": 5}
  ],
  "symbols": [
    "kotlin.js.Promise",
    "org.jetbrains.kotlin.psi.KtFile"
  ]
}`
	writeFixture(t, path, payload)

	index, err := LoadExternalIndex(path)
	if err != nil {
		t.Fatalf("LoadExternalIndex returned error: %v", err)
	}
	if !index.HasPackage("kotlin.coroutines") {
		t.Fatalf("expected kotlin.coroutines package")
	}
	if !index.HasSymbol("kotlin.js.Promise") {
		t.Fatalf("expected kotlin.js.Promise symbol")
	}
}

func TestLoadExternalIndices(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "inventory-a.json")
	pathB := filepath.Join(dir, "inventory-b.json")

	payloadA := `{
  "packages": [{"package":"kotlin.coroutines","count": 2}],
  "symbols": ["kotlin.js.Promise"]
}`
	payloadB := `{
  "packages": [{"package":"org.w3c.dom","count": 4}],
  "symbols": ["org.w3c.dom.Window"]
}`
	writeFixture(t, pathA, payloadA)
	writeFixture(t, pathB, payloadB)

	index, err := LoadExternalIndices([]string{pathA, pathB})
	if err != nil {
		t.Fatalf("LoadExternalIndices returned error: %v", err)
	}
	if len(index.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(index.Packages))
	}
	if len(index.Symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(index.Symbols))
	}
}

func writeFixture(t *testing.T, path, payload string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write inventory fixture: %v", err)
	}
}
