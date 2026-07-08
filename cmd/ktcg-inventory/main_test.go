package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/dh-kam/jkdeps/internal/testutil"
)

func TestPrintUsage(t *testing.T) {
	var out bytes.Buffer
	printUsage(&out)

	text := out.String()
	if !strings.Contains(text, "Usage:") {
		t.Fatalf("printUsage output missing usage header: %q", text)
	}
	if !strings.Contains(text, "ktcg-inventory --jar <path>") {
		t.Fatalf("printUsage output missing command usage: %q", text)
	}
	if !strings.Contains(text, "-jar value") {
		t.Fatalf("printUsage output missing jar flag: %q", text)
	}
}

func TestRunUsagePaths(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantCode     int
		wantStdout   string
		wantStderr   string
		wantEmptyErr bool
		wantEmptyOut bool
	}{
		{name: "no args", args: nil, wantCode: 2, wantStderr: "Usage:", wantEmptyOut: true},
		{name: "help command", args: []string{"help"}, wantCode: 0, wantStdout: "Usage:", wantEmptyErr: true},
		{name: "short help", args: []string{"-h"}, wantCode: 0, wantStdout: "Usage:", wantEmptyErr: true},
		{name: "long help", args: []string{"--help"}, wantCode: 0, wantStdout: "Usage:", wantEmptyErr: true},
		{name: "help flag after option", args: []string{"--symbols=false", "--help"}, wantCode: 0, wantStdout: "Usage of ktcg-inventory:", wantEmptyErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runWithCapturedOutput(t, tc.args)
			if code != tc.wantCode {
				t.Fatalf("run(%v) = %d, want %d", tc.args, code, tc.wantCode)
			}
			if tc.wantStdout != "" && !strings.Contains(stdout, tc.wantStdout) {
				t.Fatalf("stdout = %q, want substring %q", stdout, tc.wantStdout)
			}
			if tc.wantStderr != "" && !strings.Contains(stderr, tc.wantStderr) {
				t.Fatalf("stderr = %q, want substring %q", stderr, tc.wantStderr)
			}
			if tc.wantEmptyErr && strings.TrimSpace(stderr) != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if tc.wantEmptyOut && strings.TrimSpace(stdout) != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
		})
	}
}

func TestUniquePathsTrimsDeduplicatesAndPreservesOrder(t *testing.T) {
	in := []string{" a.jar ", "a.jar", "", "b.jar", "  ", "b.jar", "c.jar"}
	got := uniquePaths(in)
	want := []string{"a.jar", "b.jar", "c.jar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniquePaths() = %v, want %v", got, want)
	}
}

func TestClassifyEntry(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantKind  entryKind
		wantEntry string
	}{
		{name: "class", in: "kotlin/collections/List.class", wantKind: entryClass, wantEntry: "kotlin/collections/List"},
		{name: "builtins", in: "kotlin/kotlin.kotlin_builtins", wantKind: entryBuiltins, wantEntry: "kotlin/kotlin"},
		{name: "metadata", in: "pkg/foo.kotlin_metadata", wantKind: entryMetadata, wantEntry: "pkg/foo"},
		{name: "kjsm", in: "pkg/foo.kjsm", wantKind: entryMetadata, wantEntry: "pkg/foo"},
		{name: "knm", in: "pkg/foo.knm", wantKind: entryMetadata, wantEntry: "pkg/foo"},
		{name: "skip", in: "META-INF/MANIFEST.MF", wantKind: entrySkip, wantEntry: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotKind, gotEntry := classifyEntry(tc.in)
			if gotKind != tc.wantKind || gotEntry != tc.wantEntry {
				t.Fatalf("classifyEntry(%q) = (%v, %q), want (%v, %q)", tc.in, gotKind, gotEntry, tc.wantKind, tc.wantEntry)
			}
		})
	}
}

func TestClassPathToSymbol(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "kotlin/collections/List", want: "kotlin.collections.List"},
		{in: "kotlin/collections/Builders$Builder", want: "kotlin.collections.Builders.Builder"},
		{in: "kotlin/package-info", want: ""},
		{in: "module-info", want: ""},
	}

	for _, tc := range tests {
		if got := classPathToSymbol(tc.in); got != tc.want {
			t.Fatalf("classPathToSymbol(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPackageFromEntry(t *testing.T) {
	tests := []struct {
		name string
		kind entryKind
		in   string
		want string
	}{
		{name: "class package", kind: entryClass, in: "kotlin/collections/List", want: "kotlin.collections"},
		{name: "root class package", kind: entryClass, in: "TopLevel", want: ""},
		{name: "builtins package", kind: entryBuiltins, in: "kotlin/kotlin", want: "kotlin"},
		{name: "metadata package", kind: entryMetadata, in: "native/pkg/Foo", want: "native.pkg"},
		{name: "klib linkdata package", kind: entryMetadata, in: "default/linkdata/package_kotlinx.coroutines/0_knm", want: "kotlinx.coroutines"},
		{name: "empty", kind: entryClass, in: "", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := packageFromEntry(tc.kind, tc.in); got != tc.want {
				t.Fatalf("packageFromEntry(%v, %q) = %q, want %q", tc.kind, tc.in, got, tc.want)
			}
		})
	}
}

func TestBuildInventoryCountsPackagesAndFiltersSyntheticSymbols(t *testing.T) {
	root := t.TempDir()
	jarPath := filepath.Join(root, "sample.jar")
	if err := writeTestJar(jarPath, map[string]string{
		"kotlin/collections/List.class":           "",
		"kotlin/collections/List$Companion.class": "",
		"module-info.class":                       "",
		"kotlin/package-info.class":               "",
		"kotlin/kotlin.kotlin_builtins":           "",
		"default/linkdata/package_sample/0.knm":   "",
	}); err != nil {
		t.Fatalf("write test jar: %v", err)
	}

	inv, err := buildInventory([]string{jarPath}, true)
	if err != nil {
		t.Fatalf("buildInventory() error = %v", err)
	}
	if inv.JarPath != jarPath {
		t.Fatalf("JarPath mismatch: got=%q want=%q", inv.JarPath, jarPath)
	}
	if inv.ClassFiles != 4 {
		t.Fatalf("ClassFiles mismatch: got=%d want=4", inv.ClassFiles)
	}
	if inv.TopLevelClasses != 3 {
		t.Fatalf("TopLevelClasses mismatch: got=%d want=3", inv.TopLevelClasses)
	}
	if inv.BuiltinsFiles != 1 {
		t.Fatalf("BuiltinsFiles mismatch: got=%d want=1", inv.BuiltinsFiles)
	}
	if inv.MetadataFiles != 1 {
		t.Fatalf("MetadataFiles mismatch: got=%d want=1", inv.MetadataFiles)
	}
	if !slices.Contains(inv.Symbols, "kotlin.collections.List") {
		t.Fatalf("expected exported symbol list to include kotlin.collections.List, got=%v", inv.Symbols)
	}
	if slices.Contains(inv.Symbols, "module-info") || slices.Contains(inv.Symbols, "kotlin.package-info") {
		t.Fatalf("expected synthetic symbols to be filtered, got=%v", inv.Symbols)
	}
}

func TestRunPrintsInventoryJSONToStdout(t *testing.T) {
	root := t.TempDir()
	jarPath := filepath.Join(root, "sample.jar")
	if err := writeTestJar(jarPath, map[string]string{
		"kotlin/collections/List.class": "",
	}); err != nil {
		t.Fatalf("write test jar: %v", err)
	}

	code, stdout, stderr := runWithCapturedOutput(t, []string{"--jar", jarPath})
	if code != 0 {
		t.Fatalf("run(--jar) = %d, want 0", code)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "\"jar_path\":") || !strings.Contains(stdout, "\"class_files\": 1") {
		t.Fatalf("stdout = %q, want inventory json", stdout)
	}
}

func TestRunWritesOutputFile(t *testing.T) {
	root := t.TempDir()
	jarPath := filepath.Join(root, "sample.jar")
	if err := writeTestJar(jarPath, map[string]string{
		"kotlin/collections/List.class": "",
	}); err != nil {
		t.Fatalf("write test jar: %v", err)
	}

	outPath := filepath.Join(root, "nested", "inventory.json")
	code, stdout, stderr := runWithCapturedOutput(t, []string{"--jar", jarPath, "--out", outPath})
	if code != 0 {
		t.Fatalf("run(--jar, --out) = %d, want 0", code)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "wrote "+outPath) {
		t.Fatalf("stdout = %q, want write confirmation", stdout)
	}

	payload, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(payload), "\"jar_path\":") || !strings.Contains(string(payload), "\"class_files\": 1") {
		t.Fatalf("output file = %q, want inventory json", string(payload))
	}
}

func TestRunReportsBuildInventoryError(t *testing.T) {
	missingJar := filepath.Join(t.TempDir(), "missing.jar")

	code, stdout, stderr := runWithCapturedOutput(t, []string{"--jar", missingJar})
	if code != 1 {
		t.Fatalf("run(--jar missing) = %d, want 1", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "build inventory:") {
		t.Fatalf("stderr = %q, want build inventory prefix", stderr)
	}
}

func TestRunReportsWriteOutputError(t *testing.T) {
	root := t.TempDir()
	jarPath := filepath.Join(root, "sample.jar")
	if err := writeTestJar(jarPath, map[string]string{
		"kotlin/collections/List.class": "",
	}); err != nil {
		t.Fatalf("write test jar: %v", err)
	}

	conflict := filepath.Join(root, "blocked")
	if err := os.WriteFile(conflict, []byte("x"), 0o644); err != nil {
		t.Fatalf("write conflict: %v", err)
	}

	code, stdout, stderr := runWithCapturedOutput(t, []string{"--jar", jarPath, "--out", filepath.Join(conflict, "inventory.json")})
	if code != 1 {
		t.Fatalf("run(--jar, --out conflict) = %d, want 1", code)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "write output:") {
		t.Fatalf("stderr = %q, want write output prefix", stderr)
	}
}

func writeTestJar(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, body := range files {
		entry, err := w.Create(name)
		if err != nil {
			_ = w.Close()
			return err
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			_ = w.Close()
			return err
		}
	}
	return w.Close()
}

func runWithCapturedOutput(t *testing.T, args []string) (int, string, string) {
	return testutil.CaptureOutput(t, func() int {
		return run(args)
	})
}
