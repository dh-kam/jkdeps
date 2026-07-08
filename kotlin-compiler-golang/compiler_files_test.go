package kotlincompilergolang

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectKotlinFiles_ExcludeBuildScriptsByDefault(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "src", "App.kt"), "package sample\nclass App\n")
	writeTestFile(t, filepath.Join(root, "scripts", "tool.kts"), "val x = 1\n")
	writeTestFile(t, filepath.Join(root, "build.gradle.kts"), "val y = 2\n")
	writeTestFile(t, filepath.Join(root, "settings.gradle.kts"), "val z = 3\n")

	files, err := collectKotlinFiles(root, true, false)
	if err != nil {
		t.Fatalf("collect files: %v", err)
	}

	got := map[string]struct{}{}
	for _, path := range files {
		got[filepath.Base(path)] = struct{}{}
	}
	if _, ok := got["App.kt"]; !ok {
		t.Fatalf("expected App.kt in collected files: %+v", files)
	}
	if _, ok := got["tool.kts"]; !ok {
		t.Fatalf("expected tool.kts in collected files: %+v", files)
	}
	if _, ok := got["build.gradle.kts"]; ok {
		t.Fatalf("did not expect build.gradle.kts when includeBuildScripts=false: %+v", files)
	}
	if _, ok := got["settings.gradle.kts"]; ok {
		t.Fatalf("did not expect settings.gradle.kts when includeBuildScripts=false: %+v", files)
	}
}

func TestCollectKotlinFiles_IncludeBuildScripts(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "src", "App.kt"), "package sample\nclass App\n")
	writeTestFile(t, filepath.Join(root, "build.gradle.kts"), "val y = 2\n")

	files, err := collectKotlinFiles(root, true, true)
	if err != nil {
		t.Fatalf("collect files: %v", err)
	}

	got := map[string]struct{}{}
	for _, path := range files {
		got[filepath.Base(path)] = struct{}{}
	}
	if _, ok := got["App.kt"]; !ok {
		t.Fatalf("expected App.kt in collected files: %+v", files)
	}
	if _, ok := got["build.gradle.kts"]; !ok {
		t.Fatalf("expected build.gradle.kts when includeBuildScripts=true: %+v", files)
	}
}

func TestCollectKotlinFiles_IncludeBuildScriptsWithoutIncludingKts(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "src", "App.kt"), "package sample\nclass App\n")
	writeTestFile(t, filepath.Join(root, "scripts", "tool.kts"), "val x = 1\n")
	writeTestFile(t, filepath.Join(root, "build.gradle.kts"), "val y = 2\n")
	writeTestFile(t, filepath.Join(root, "settings.gradle.kts"), "val z = 3\n")

	files, err := collectKotlinFiles(root, false, true)
	if err != nil {
		t.Fatalf("collect files: %v", err)
	}

	got := map[string]struct{}{}
	for _, path := range files {
		got[filepath.Base(path)] = struct{}{}
	}
	if _, ok := got["App.kt"]; !ok {
		t.Fatalf("expected App.kt in collected files: %+v", files)
	}
	if _, ok := got["build.gradle.kts"]; !ok {
		t.Fatalf("expected build.gradle.kts when includeBuildScripts=true and includeKTS=false: %+v", files)
	}
	if _, ok := got["settings.gradle.kts"]; !ok {
		t.Fatalf("expected settings.gradle.kts when includeBuildScripts=true and includeKTS=false: %+v", files)
	}
	if _, ok := got["tool.kts"]; ok {
		t.Fatalf("did not expect tool.kts when includeKTS=false: %+v", files)
	}
}

func writeTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
