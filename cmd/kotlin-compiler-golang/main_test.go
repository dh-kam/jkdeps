package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dh-kam/jkdeps/internal/cliutil"
	"github.com/dh-kam/jkdeps/internal/flagutil"
	"github.com/dh-kam/jkdeps/internal/testutil"
	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

func runWithCapturedOutput(t *testing.T, args []string) (int, string, string) {
	return testutil.CaptureOutput(t, func() int {
		return run(args)
	})
}

func TestResolvePathAgainstRepo(t *testing.T) {

	const repoRoot = "/tmp/repo-root"

	cases := []struct {
		name      string
		input     string
		repoRoot  string
		want      string
		allowErr  bool
		trimEmpty bool
	}{
		{
			name:     "absolute path passes through",
			input:    "/other/path/file.kt",
			repoRoot: repoRoot,
			want:     "/other/path/file.kt",
		},
		{
			name:     "relative path joins repo root",
			input:    "src/main/kotlin/A.kt",
			repoRoot: repoRoot,
			want:     filepath.Clean(filepath.Join(repoRoot, "src/main/kotlin/A.kt")),
		},
		{
			name:     "empty string returns empty",
			input:    "   ",
			repoRoot: repoRoot,
			want:     "",
		},
		{
			name:      "relative with empty repo root uses cwd absolute",
			input:     ".",
			repoRoot:  "",
			allowErr:  false,
			trimEmpty: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolvePathAgainstRepo(tc.input, tc.repoRoot)
			if !tc.allowErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.trimEmpty {
				got = strings.TrimSpace(got)
			}
			if tc.name == "relative with empty repo root uses cwd absolute" {
				if got == "" {
					t.Fatalf("expected absolute path, got empty")
				}
				if !filepath.IsAbs(got) {
					t.Fatalf("expected absolute path, got %q", got)
				}
				return
			}
			if got != tc.want {
				t.Fatalf("resolvePathAgainstRepo(%q, %q) = %q, want %q", tc.input, tc.repoRoot, got, tc.want)
			}
		})
	}
}

func TestResolvePathsForCompile(t *testing.T) {

	repoRoot := "/tmp/repo-root"
	got, err := resolvePathsForCompile([]string{"  src/main/kotlin ", "", "build", "/abs/path"}, repoRoot)
	if err != nil {
		t.Fatalf("resolvePathsForCompile() unexpected error: %v", err)
	}

	want := []string{
		filepath.Clean(filepath.Join(repoRoot, "src/main/kotlin")),
		filepath.Clean(filepath.Join(repoRoot, "build")),
		filepath.Clean("/abs/path"),
	}
	if len(got) != len(want) {
		t.Fatalf("resolvePathsForCompile() len=%d want=%d, got=%v", len(got), len(want), got)
	}
	for i, wantPath := range want {
		if got[i] != wantPath {
			t.Fatalf("resolvePathsForCompile()[%d]=%q want=%q", i, got[i], wantPath)
		}
	}
}

func TestNormalizeExcludePaths(t *testing.T) {

	base := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(base); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWd)
	})

	wantRel := filepath.Clean(filepath.Join(base, "generated"))
	got := normalizeExcludePaths([]string{
		"generated",
		"",
		"generated/",
		"   .   ",
		"",
		"  generated/./nested",
	})
	expected := []string{
		wantRel,
		base,
		filepath.Clean(filepath.Join(base, "generated", "nested")),
	}

	if len(got) != len(expected) {
		t.Fatalf("normalizeExcludePaths() len=%d want=%d got=%v", len(got), len(expected), got)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("normalizeExcludePaths()[%d]=%q want=%q", i, got[i], expected[i])
		}
	}
}

func TestIsExcludedPath(t *testing.T) {
	t.Run("basic exclusion patterns", func(t *testing.T) {
		exclude := map[string]struct{}{
			filepath.Clean("/tmp/project/build"):   {},
			filepath.Clean("/tmp/project/.gradle"): {},
		}

		cases := []struct {
			path string
			want bool
		}{
			{path: "/tmp/project/build", want: true},
			{path: "/tmp/project/build/classes", want: true},
			{path: "/tmp/project/.gradle/caches", want: true},
			{path: "/tmp/project/building", want: false},
			{path: "/tmp/other", want: false},
		}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.path, func(t *testing.T) {
				got := isExcludedPath(tc.path, exclude)
				if got != tc.want {
					t.Fatalf("isExcludedPath(%q) = %v, want %v", tc.path, got, tc.want)
				}
			})
		}
	})

	t.Run("empty exclude map", func(t *testing.T) {
		exclude := map[string]struct{}{}
		if got := isExcludedPath("/tmp/project/build", exclude); got {
			t.Fatalf("isExcludedPath with empty map should return false, got true")
		}
	})

	t.Run("empty prefix in exclude map is ignored", func(t *testing.T) {
		exclude := map[string]struct{}{
			"": {}, // empty prefix should be skipped
			filepath.Clean("/tmp/project/build"): {},
		}
		// Should still match the non-empty prefix
		if got := isExcludedPath("/tmp/project/build", exclude); !got {
			t.Fatalf("isExcludedPath should still match non-empty prefix, got false")
		}
		// Non-matching path should return false
		if got := isExcludedPath("/tmp/other", exclude); got {
			t.Fatalf("isExcludedPath with non-matching path should return false, got true")
		}
	})

	t.Run("relative path handling", func(t *testing.T) {
		// Create a temporary directory structure for testing
		tmpDir := t.TempDir()
		buildDir := filepath.Join(tmpDir, "build")
		if err := os.MkdirAll(buildDir, 0o755); err != nil {
			t.Fatalf("mkdir build: %v", err)
		}

		// Change to tmpDir so relative paths work
		origWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		defer os.Chdir(origWd)
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		exclude := map[string]struct{}{
			filepath.Clean(buildDir): {},
		}

		// Test with relative path
		if got := isExcludedPath("build", exclude); !got {
			t.Fatalf("isExcludedPath with relative path should match, got false")
		}
	})
}

func TestIsKotlinSourcePath(t *testing.T) {
	cases := []struct {
		name         string
		path         string
		includeKTS   bool
		includeBuild bool
		want         bool
	}{
		{
			name:         "kt file",
			path:         filepath.FromSlash("src/main/sample/Source.KT"),
			includeKTS:   false,
			includeBuild: false,
			want:         true,
		},
		{
			name:         "kts ignored by default",
			path:         filepath.FromSlash("build.gradle.kts"),
			includeKTS:   true,
			includeBuild: false,
			want:         false,
		},
		{
			name:         "settings build script ignored by default",
			path:         filepath.FromSlash("settings.gradle.kts"),
			includeKTS:   true,
			includeBuild: false,
			want:         false,
		},
		{
			name:         "kts excluded when includeKTS false",
			path:         filepath.FromSlash("foo.kts"),
			includeKTS:   false,
			includeBuild: false,
			want:         false,
		},
		{
			name:         "kts included when includeKTS true",
			path:         filepath.FromSlash("foo.kts"),
			includeKTS:   true,
			includeBuild: false,
			want:         true,
		},
		{
			name:         "build script included when include-build-scripts true and include-kts false",
			path:         filepath.FromSlash("build.gradle.kts"),
			includeKTS:   false,
			includeBuild: true,
			want:         true,
		},
		{
			name:         "regular kts still excluded when include-kts false",
			path:         filepath.FromSlash("tool.kts"),
			includeKTS:   false,
			includeBuild: true,
			want:         false,
		},
		{
			name:         "build scripts included when flag set",
			path:         filepath.FromSlash("build.gradle.kts"),
			includeKTS:   true,
			includeBuild: true,
			want:         true,
		},
		{
			name:         "non kotlin file",
			path:         filepath.FromSlash("README.md"),
			includeKTS:   true,
			includeBuild: true,
			want:         false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := isKotlinSourcePath(tc.path, tc.includeKTS, tc.includeBuild)
			if got != tc.want {
				t.Fatalf("isKotlinSourcePath(%q, includeKTS=%v, includeBuildScripts=%v) = %v want %v", tc.path, tc.includeKTS, tc.includeBuild, got, tc.want)
			}
		})
	}
}

func TestIsTopLevelCommand(t *testing.T) {
	cases := []struct {
		name    string
		command string
		want    bool
	}{
		{name: "parse", command: "parse", want: true},
		{name: "symbols", command: "symbols", want: true},
		{name: "compile", command: "compile", want: true},
		{name: "acceptance", command: "acceptance", want: true},
		{name: "help", command: "help", want: false},
		{name: "flag", command: "-h", want: false},
		{name: "unknown", command: "does-not-exist", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := isTopLevelCommand(tc.command); got != tc.want {
				t.Fatalf("isTopLevelCommand(%q) = %v want %v", tc.command, got, tc.want)
			}
		})
	}
}

func TestIsHelpAlias(t *testing.T) {
	cases := []struct {
		name      string
		candidate string
		want      bool
	}{
		{name: "double dash help", candidate: "--help", want: true},
		{name: "single dash help", candidate: "-h", want: true},
		{name: "help command", candidate: "help", want: true},
		{name: "normal command", candidate: "parse", want: false},
		{name: "flag", candidate: "-help", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := isHelpAlias(tc.candidate); got != tc.want {
				t.Fatalf("isHelpAlias(%q) = %v want %v", tc.candidate, got, tc.want)
			}
		})
	}
}

func TestNormalizeHelpInvocationArgs(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "single help alias remains unchanged",
			in:   []string{"--help"},
			want: []string{"--help"},
		},
		{
			name: "single non-help remains unchanged",
			in:   []string{"compile"},
			want: []string{"compile"},
		},
		{
			name: "non-help first arg remains unchanged even with aliases later",
			in:   []string{"compile", "-h", "help", "parse"},
			want: []string{"compile", "-h", "help", "parse"},
		},
		{
			name: "double help alias with unknown command falls back",
			in:   []string{"--help", "--help"},
			want: []string{"--help"},
		},
		{
			name: "double dash help for known command",
			in:   []string{"--help", "--help", "parse"},
			want: []string{"parse", "--help"},
		},
		{
			name: "mixed aliases with args",
			in:   []string{"-h", "--help", "symbols", "--repo", "."},
			want: []string{"symbols", "--help", "--repo", "."},
		},
		{
			name: "single alias then help keyword",
			in:   []string{"-h", "help", "compile"},
			want: []string{"compile", "--help"},
		},
		{
			name: "single dash then double dash help and command",
			in:   []string{"-h", "--help", "parse", "--repo", "."},
			want: []string{"parse", "--help", "--repo", "."},
		},
		{
			name: "help alias then command should be unchanged",
			in:   []string{"help", "parse"},
			want: []string{"help", "parse"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeHelpInvocationArgs(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("normalizeHelpInvocationArgs(%v) len=%d want=%d", tc.in, len(got), len(tc.want))
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("normalizeHelpInvocationArgs(%v)[%d]=%q want=%q", tc.in, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestBuildTopLevelCommandSet(t *testing.T) {
	got := cliutil.BuildCommandSet([]string{"parse", "symbols", "symbols", "compile", "help"})
	if _, ok := got["parse"]; !ok {
		t.Fatalf("expected parse in set")
	}
	if _, ok := got["symbols"]; !ok {
		t.Fatalf("expected symbols in set")
	}
	if _, ok := got["compile"]; !ok {
		t.Fatalf("expected compile in set")
	}
	if _, ok := got["help"]; !ok {
		t.Fatalf("expected help in set")
	}
	if len(got) != 4 {
		t.Fatalf("buildTopLevelCommandSet() should deduplicate entries, got len=%d", len(got))
	}
}

func TestBuildTopLevelCommandSetNilOrEmpty(t *testing.T) {
	if got := cliutil.BuildCommandSet(nil); len(got) != 0 {
		t.Fatalf("buildTopLevelCommandSet(nil) = %d, want 0", len(got))
	}
}

func TestCollectKotlinFilesForCompile(t *testing.T) {

	root := t.TempDir()
	mustWriteCompileTestFile(t, filepath.Join(root, "src", "main", "kotlin", "App.kt"))
	mustWriteCompileTestFile(t, filepath.Join(root, "scripts", "tool.kts"))
	mustWriteCompileTestFile(t, filepath.Join(root, "build.gradle.kts"))
	mustWriteCompileTestFile(t, filepath.Join(root, "settings.gradle.kts"))
	mustWriteCompileTestFile(t, filepath.Join(root, "custom-excluded", "Hidden.kt"))
	mustWriteCompileTestFile(t, filepath.Join(root, "build", "generated", "Skip.kt"))
	mustWriteCompileTestFile(t, filepath.Join(root, "out", "Skip.kt"))
	mustWriteCompileTestFile(t, filepath.Join(root, "target", "Skip.kt"))
	mustWriteCompileTestFile(t, filepath.Join(root, "node_modules", "Skip.kt"))

	got, err := collectKotlinFilesForCompile(
		root,
		[]string{filepath.Join(root, "custom-excluded")},
		false,
		true,
	)
	if err != nil {
		t.Fatalf("collectKotlinFilesForCompile() unexpected error: %v", err)
	}

	want := []string{
		filepath.Join(root, "build.gradle.kts"),
		filepath.Join(root, "settings.gradle.kts"),
		filepath.Join(root, "src", "main", "kotlin", "App.kt"),
	}
	if len(got) != len(want) {
		t.Fatalf("collectKotlinFilesForCompile() len=%d want=%d got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("collectKotlinFilesForCompile()[%d]=%q want=%q", i, got[i], want[i])
		}
	}

	t.Run("error case: non-existent path", func(t *testing.T) {
		// Pass a non-existent path that should trigger an error
		_, err := collectKotlinFilesForCompile(
			"/nonexistent/path/that/does/not/exist",
			nil,
			false,
			false,
		)
		if err == nil {
			t.Fatal("collectKotlinFilesForCompile() with non-existent path should return error")
		}
	})
}

func TestCollectKotlinFilesForCompilePaths(t *testing.T) {

	root := t.TempDir()
	dirA := filepath.Join(root, "moduleA")
	dirB := filepath.Join(root, "moduleB")
	singleFile := filepath.Join(root, "standalone.kts")

	mustWriteCompileTestFile(t, filepath.Join(dirA, "A.kt"))
	mustWriteCompileTestFile(t, filepath.Join(dirA, "script.kts"))
	mustWriteCompileTestFile(t, filepath.Join(dirB, "B.kt"))
	mustWriteCompileTestFile(t, singleFile)

	got, err := collectKotlinFilesForCompilePaths(
		[]string{dirA, singleFile, dirA, dirB},
		[]string{filepath.Join(dirB, "B.kt")},
		true,
		false,
	)
	if err != nil {
		t.Fatalf("collectKotlinFilesForCompilePaths() unexpected error: %v", err)
	}

	want := []string{
		filepath.Join(dirA, "A.kt"),
		filepath.Join(dirA, "script.kts"),
		singleFile,
	}
	if len(got) != len(want) {
		t.Fatalf("collectKotlinFilesForCompilePaths() len=%d want=%d got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("collectKotlinFilesForCompilePaths()[%d]=%q want=%q", i, got[i], want[i])
		}
	}
}

func mustWriteCompileTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte("class Sample\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestCollectExtraCompilerArgs(t *testing.T) {

	path := filepath.Join(t.TempDir(), "args.txt")
	content := strings.Join([]string{
		"",
		"   ",
		"# comment",
		"--jvm-target 1.9",
		"--xjsr305=strict",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write args file: %v", err)
	}

	args, err := collectExtraCompilerArgs([]string{path})
	if err != nil {
		t.Fatalf("collectExtraCompilerArgs() unexpected error: %v", err)
	}
	if len(args) != 3 {
		t.Fatalf("collectExtraCompilerArgs() len=%d want=3; got=%v", len(args), args)
	}
	if got := args[0]; got != "--jvm-target" {
		t.Fatalf("collectExtraCompilerArgs()[0]=%q want=%q", got, "--jvm-target")
	}
	if got := args[1]; got != "1.9" {
		t.Fatalf("collectExtraCompilerArgs()[1]=%q want=%q", got, "1.9")
	}
	if got := args[2]; got != "--xjsr305=strict" {
		t.Fatalf("collectExtraCompilerArgs()[2]=%q want=%q", got, "--xjsr305=strict")
	}
}

func TestUniqueStringsTrimsAndDedupsPreserveOrder(t *testing.T) {

	in := []string{"  a", "a", "", " b ", "a", "c", "  ", "b"}
	got := flagutil.UniqueStrings(in)
	want := []string{"a", "b", "c"}

	if len(got) != len(want) {
		t.Fatalf("uniqueStrings() len=%d want=%d got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("uniqueStrings()[%d]=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestSplitArgLine(t *testing.T) {

	tests := []struct {
		name    string
		line    string
		want    []string
		wantErr bool
	}{
		{name: "blank line", line: "   ", want: nil},
		{name: "comment line", line: "# comment", want: nil},
		{name: "simple args", line: "--jvm-target 17", want: []string{"--jvm-target", "17"}},
		{name: "inline comment", line: "--flag value  # trailing", want: []string{"--flag", "value"}},
		{name: "unterminated quote", line: "--flag \"oops", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := splitArgLine(tc.line)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("splitArgLine(%q) expected error, got nil", tc.line)
				}
				return
			}
			if err != nil {
				t.Fatalf("splitArgLine(%q) unexpected error: %v", tc.line, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("splitArgLine(%q) len=%d want=%d got=%v", tc.line, len(got), len(tc.want), got)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("splitArgLine(%q)[%d]=%q want=%q", tc.line, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestCollectExtraCompilerArgsSkipsBOMAndCRLF(t *testing.T) {

	path := filepath.Join(t.TempDir(), "args.txt")
	content := strings.Join([]string{
		"\ufeff--jvm-target 1.8",
		"   # comment",
		"",
		"--xjsr305=strict",
	}, "\r\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write args file: %v", err)
	}

	args, err := collectExtraCompilerArgs([]string{path})
	if err != nil {
		t.Fatalf("collectExtraCompilerArgs() unexpected error: %v", err)
	}
	if len(args) != 3 {
		t.Fatalf("collectExtraCompilerArgs() len=%d want=3; got=%v", len(args), args)
	}
	if got := args[0]; got != "--jvm-target" {
		t.Fatalf("collectExtraCompilerArgs()[0]=%q want=%q", got, "--jvm-target")
	}
	if got := args[1]; got != "1.8" {
		t.Fatalf("collectExtraCompilerArgs()[1]=%q want=%q", got, "1.8")
	}
	if got := args[2]; got != "--xjsr305=strict" {
		t.Fatalf("collectExtraCompilerArgs()[2]=%q want=%q", got, "--xjsr305=strict")
	}
}

func TestCollectExtraCompilerArgsSupportsQuotedValues(t *testing.T) {

	path := filepath.Join(t.TempDir(), "quoted.args")
	content := strings.Join([]string{
		`--option "hello world"`,
		`-P 'plugin:com.example.enabled=true'`,
		`--empty ""`,
		`--hash token # inline comment`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write args file: %v", err)
	}

	args, err := collectExtraCompilerArgs([]string{path})
	if err != nil {
		t.Fatalf("collectExtraCompilerArgs() unexpected error: %v", err)
	}
	if len(args) != 8 {
		t.Fatalf("collectExtraCompilerArgs() len=%d want=8; got=%v", len(args), args)
	}
	expected := []string{
		"--option",
		"hello world",
		"-P",
		"plugin:com.example.enabled=true",
		"--empty",
		"",
		"--hash",
		"token",
	}
	for i, want := range expected {
		if args[i] != want {
			t.Fatalf("collectExtraCompilerArgs()[%d]=%q want=%q", i, args[i], want)
		}
	}
}

func TestCollectExtraCompilerArgsRejectsUnterminatedQuote(t *testing.T) {

	path := filepath.Join(t.TempDir(), "bad.args")
	content := "--option \"unterminated\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write args file: %v", err)
	}

	if _, err := collectExtraCompilerArgs([]string{path}); err == nil {
		t.Fatalf("collectExtraCompilerArgs() expected error for unterminated quote")
	}
}

func TestCollectExtraCompilerArgsLongLine(t *testing.T) {

	path := filepath.Join(t.TempDir(), "long.args")
	wideValue := strings.Repeat("a", 70000)
	content := "--language-version=" + wideValue + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write args file: %v", err)
	}

	args, err := collectExtraCompilerArgs([]string{path})
	if err != nil {
		t.Fatalf("collectExtraCompilerArgs() unexpected error: %v", err)
	}
	if len(args) != 1 {
		t.Fatalf("collectExtraCompilerArgs() len=%d want=1; got=%v", len(args), args)
	}
	want := "--language-version=" + wideValue
	if args[0] != want {
		t.Fatalf("collectExtraCompilerArgs()[0] length=%d want=%d", len(args[0]), len(want))
	}
}

func TestCollectExtraCompilerArgsNoNewlineAndEmptyFile(t *testing.T) {

	t.Run("no-newline-last-line", func(t *testing.T) {

		path := filepath.Join(t.TempDir(), "no-newline.args")
		if err := os.WriteFile(path, []byte("--jvm-target 1.8"), 0o644); err != nil {
			t.Fatalf("write args file: %v", err)
		}

		args, err := collectExtraCompilerArgs([]string{path})
		if err != nil {
			t.Fatalf("collectExtraCompilerArgs() unexpected error: %v", err)
		}
		if len(args) != 2 {
			t.Fatalf("collectExtraCompilerArgs() len=%d want=2; got=%v", len(args), args)
		}
		if args[0] != "--jvm-target" || args[1] != "1.8" {
			t.Fatalf("collectExtraCompilerArgs() = %v", args)
		}
	})

	t.Run("empty-file-only", func(t *testing.T) {

		path := filepath.Join(t.TempDir(), "empty.args")
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			t.Fatalf("write args file: %v", err)
		}

		args, err := collectExtraCompilerArgs([]string{path})
		if err != nil {
			t.Fatalf("collectExtraCompilerArgs() unexpected error: %v", err)
		}
		if len(args) != 0 {
			t.Fatalf("collectExtraCompilerArgs() len=%d want=0; got=%v", len(args), args)
		}
	})
}

func TestSplitArgLineComplexEscapesAndQuotes(t *testing.T) {

	tests := []struct {
		name string
		line string
		want []string
	}{
		{
			name: "escaped hash outside quotes",
			line: `-Dhash=a\#b --name value`,
			want: []string{"-Dhash=a#b", "--name", "value"},
		},
		{
			name: "escaped space",
			line: `--name\ with\ space`,
			want: []string{"--name with space"},
		},
		{
			name: "quoted hash preserved",
			line: `--path "a # b"`,
			want: []string{"--path", "a # b"},
		},
		{
			name: "comment after quoted token",
			line: `--path "a b" # trailing comment`,
			want: []string{"--path", "a b"},
		},
		{
			name: "unterminated quote",
			line: `--bad "unterminated`,
			want: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := splitArgLine(tc.line)
			if tc.want == nil {
				if err == nil {
					t.Fatalf("splitArgLine(%q) expected error", tc.line)
				}
				return
			}

			if err != nil {
				t.Fatalf("splitArgLine(%q) unexpected error: %v", tc.line, err)
			}
			if len(tokens) != len(tc.want) {
				t.Fatalf("splitArgLine(%q) len=%d want=%d", tc.line, len(tokens), len(tc.want))
			}
			for i := range tc.want {
				if tokens[i] != tc.want[i] {
					t.Fatalf("splitArgLine(%q)[%d]=%q want=%q", tc.line, i, tokens[i], tc.want[i])
				}
			}
		})
	}
}

func TestCollectExtraCompilerArgsSupportsEscapedSpaceInLine(t *testing.T) {

	path := filepath.Join(t.TempDir(), "escape.args")
	if err := os.WriteFile(path, []byte(`--name\ with\ space\ tab\#`), 0o644); err != nil {
		t.Fatalf("write args file: %v", err)
	}

	args, err := collectExtraCompilerArgs([]string{path})
	if err != nil {
		t.Fatalf("collectExtraCompilerArgs() unexpected error: %v", err)
	}
	if len(args) != 1 {
		t.Fatalf("collectExtraCompilerArgs() len=%d want=1; got=%v", len(args), args)
	}
	if args[0] != "--name with space tab#" {
		t.Fatalf("collectExtraCompilerArgs()[0]=%q want=%q", args[0], "--name with space tab#")
	}
}

func TestQuoteCommandArgs(t *testing.T) {

	got := quoteCommandArgs("kotlinc", []string{"--jvm-target", "1.8"})
	expected := "\"kotlinc\" \"--jvm-target\" \"1.8\""
	if got != expected {
		t.Fatalf("quoteCommandArgs() = %q want=%q", got, expected)
	}

	got = quoteCommandArgs("kotlin compiler", []string{"--out", "out dir", `a"b`})
	expected = "\"kotlin compiler\" \"--out\" \"out dir\" \"a\\\"b\""
	if got != expected {
		t.Fatalf("quoteCommandArgs(escaped) = %q want=%q", got, expected)
	}
}

func TestCollectExtraCompilerArgsHandlesCommentAndEscapedHash(t *testing.T) {

	path := filepath.Join(t.TempDir(), "comment-hash.args")
	content := strings.Join([]string{
		`--name value # inline comment`,
		`-Dhash=a\#b`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write args file: %v", err)
	}

	args, err := collectExtraCompilerArgs([]string{path})
	if err != nil {
		t.Fatalf("collectExtraCompilerArgs() unexpected error: %v", err)
	}
	if len(args) != 3 {
		t.Fatalf("collectExtraCompilerArgs() len=%d want=3; got=%v", len(args), args)
	}
	if args[0] != "--name" {
		t.Fatalf("collectExtraCompilerArgs()[0]=%q want=%q", args[0], "--name")
	}
	if args[1] != "value" {
		t.Fatalf("collectExtraCompilerArgs()[1]=%q want=%q", args[1], "value")
	}
	if args[2] != "-Dhash=a#b" {
		t.Fatalf("collectExtraCompilerArgs()[2]=%q want=%q", args[2], "-Dhash=a#b")
	}
}

func TestCollectKotlinFilesForCompilePathsSkipsGeneratedAndBuildDirs(t *testing.T) {

	repoRoot := t.TempDir()
	sources := filepath.Join(repoRoot, "src", "main", "kotlin")
	buildDir := filepath.Join(repoRoot, "build")
	outDir := filepath.Join(repoRoot, "out")
	targetDir := filepath.Join(repoRoot, "target")
	nodeModulesDir := filepath.Join(repoRoot, "node_modules")
	gitDir := filepath.Join(repoRoot, ".git")
	excludedDir := filepath.Join(repoRoot, "generated")

	for _, dir := range []string{sources, buildDir, outDir, targetDir, nodeModulesDir, gitDir, excludedDir, filepath.Join(repoRoot, "sub")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", dir, err)
		}
	}

	if err := os.WriteFile(filepath.Join(repoRoot, "src", "main", "kotlin", "A.kt"), []byte("package t\n\nfun f() {}"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "src", "main", "kotlin", "build.gradle.kts"), []byte("plugins {}"), 0o644); err != nil {
		t.Fatalf("write build script kts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "src", "main", "kotlin", "keep.kts"), []byte("fun g() {}"), 0o644); err != nil {
		t.Fatalf("write kts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "build", "ignored.kt"), []byte("package t\n\nfun ignored() {}"), 0o644); err != nil {
		t.Fatalf("write ignored build kt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "out", "ignored.kt"), []byte("package t\n\nfun ignored() {}"), 0o644); err != nil {
		t.Fatalf("write ignored out kt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "target", "ignored.kt"), []byte("package t\n\nfun ignored() {}"), 0o644); err != nil {
		t.Fatalf("write ignored target kt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "node_modules", "ignored.kt"), []byte("package t\n\nfun ignored() {}"), 0o644); err != nil {
		t.Fatalf("write ignored node_modules kt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".git", "ignored.kt"), []byte("package t\n\nfun ignored() {}"), 0o644); err != nil {
		t.Fatalf("write ignored git kt: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repoRoot, "generated", "ignored.kt"), []byte("package t\n\nfun ignored() {}"), 0o644); err != nil {
		t.Fatalf("write excluded kt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "sub", "active.kt"), []byte("package t\n\nfun active() {}"), 0o644); err != nil {
		t.Fatalf("write active kt: %v", err)
	}

	got, err := collectKotlinFilesForCompilePaths([]string{repoRoot}, []string{excludedDir}, true, false)
	if err != nil {
		t.Fatalf("collectKotlinFilesForCompilePaths() unexpected error: %v", err)
	}

	want := []string{
		filepath.Clean(filepath.Join(repoRoot, "src", "main", "kotlin", "A.kt")),
		filepath.Clean(filepath.Join(repoRoot, "src", "main", "kotlin", "keep.kts")),
		filepath.Clean(filepath.Join(repoRoot, "sub", "active.kt")),
	}
	if len(got) != len(want) {
		t.Fatalf("collect files len=%d want=%d got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("collect file[%d]=%q want=%q", i, got[i], want[i])
		}
	}

	gotNoKts, err := collectKotlinFilesForCompilePaths([]string{repoRoot}, []string{excludedDir}, false, false)
	if err != nil {
		t.Fatalf("collectKotlinFilesForCompilePaths() with no kts unexpected error: %v", err)
	}
	wantNoKts := []string{
		filepath.Clean(filepath.Join(repoRoot, "src", "main", "kotlin", "A.kt")),
		filepath.Clean(filepath.Join(repoRoot, "sub", "active.kt")),
	}
	if len(gotNoKts) != len(wantNoKts) {
		t.Fatalf("collect files w/out kts len=%d want=%d got=%v", len(gotNoKts), len(wantNoKts), gotNoKts)
	}
	for i := range wantNoKts {
		if gotNoKts[i] != wantNoKts[i] {
			t.Fatalf("collect kts-disabled file[%d]=%q want=%q", i, gotNoKts[i], wantNoKts[i])
		}
	}
}

func TestParseParserBackend(t *testing.T) {

	for _, tc := range []string{"antlr", "ANTLR", "embeddable", "EMBEDDABLE"} {
		if _, err := parseParserBackend(tc); err != nil {
			t.Fatalf("parseParserBackend(%q) unexpected error: %v", tc, err)
		}
	}

	if _, err := parseParserBackend("javac"); err == nil {
		t.Fatalf("parseParserBackend(\"javac\") should return error")
	}
}

func TestGraphOutputPaths(t *testing.T) {

	tests := []struct {
		name     string
		in       string
		wantHTML string
		wantJSON string
	}{
		{
			name:     "default base",
			in:       "",
			wantHTML: "jkdeps-graph.html",
			wantJSON: "jkdeps-graph.json",
		},
		{
			name:     "implicit suffixes",
			in:       filepath.FromSlash("tmp/graph"),
			wantHTML: filepath.FromSlash("tmp/graph.html"),
			wantJSON: filepath.FromSlash("tmp/graph.json"),
		},
		{
			name:     "html input",
			in:       filepath.FromSlash("tmp/graph.html"),
			wantHTML: filepath.FromSlash("tmp/graph.html"),
			wantJSON: filepath.FromSlash("tmp/graph.json"),
		},
		{
			name:     "json input",
			in:       filepath.FromSlash("tmp/graph.json"),
			wantHTML: filepath.FromSlash("tmp/graph.html"),
			wantJSON: filepath.FromSlash("tmp/graph.json"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotHTML, gotJSON := cliutil.GraphOutputPaths(tc.in, "jkdeps-graph")
			if gotHTML != tc.wantHTML {
				t.Fatalf("graphOutputPaths(%q) html=%q want=%q", tc.in, gotHTML, tc.wantHTML)
			}
			if gotJSON != tc.wantJSON {
				t.Fatalf("graphOutputPaths(%q) json=%q want=%q", tc.in, gotJSON, tc.wantJSON)
			}
		})
	}
}

func TestBuildGraphHTML(t *testing.T) {
	output := buildGraphHTML("my graph.json")
	if !strings.Contains(output, "\"my graph.json\"") {
		t.Fatalf("buildGraphHTML() should inject quoted data file, got: %q", output)
	}
	if strings.Contains(output, "__DATA_FILE__") {
		t.Fatalf("buildGraphHTML() should replace template token")
	}
}

func TestWriteGraphArtifactsCreatesFiles(t *testing.T) {
	root := t.TempDir()
	graph := kcg.WebGraph{
		Nodes: []kcg.WebGraphNode{{ID: 1, Name: "sample", Kind: kcg.WebGraphNodeInternal}},
	}

	htmlPath := filepath.Join(root, "out", "graph.html")
	jsonPath := filepath.Join(root, "out", "graph.json")
	if err := writeGraphArtifacts(htmlPath, jsonPath, graph); err != nil {
		t.Fatalf("writeGraphArtifacts() unexpected error: %v", err)
	}

	if _, err := os.Stat(htmlPath); err != nil {
		t.Fatalf("expected html artifact: %v", err)
	}
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("expected json artifact: %v", err)
	}
}

func TestWriteGraphArtifactsFailsIfParentIsFile(t *testing.T) {
	root := t.TempDir()
	conflict := filepath.Join(root, "out")
	if err := os.WriteFile(conflict, []byte("blocked"), 0o644); err != nil {
		t.Fatalf("write conflict file: %v", err)
	}

	err := writeGraphArtifacts(filepath.Join(conflict, "graph.html"), filepath.Join(root, "ok.json"), kcg.WebGraph{})
	if err == nil {
		t.Fatal("expected writeGraphArtifacts to fail when html parent is a file")
	}
}

func TestPrintUsage(t *testing.T) {
	var b bytes.Buffer
	printUsage(&b)
	output := b.String()
	if !strings.Contains(output, "Usage:") {
		t.Fatalf("printUsage output missing usage header: %q", output)
	}
	if !strings.Contains(output, "kotlin-compiler-golang parse [flags]") {
		t.Fatalf("printUsage output missing parse command: %q", output)
	}
	if !strings.Contains(output, "kotlin-compiler-golang acceptance [flags]") {
		t.Fatalf("printUsage output missing acceptance command: %q", output)
	}
}

func TestWriteGraphArtifactsWritesFiles(t *testing.T) {

	outBase := t.TempDir()
	htmlPath := filepath.Join(outBase, "artifacts", "graph.html")
	jsonPath := filepath.Join(outBase, "artifacts", "graph.json")
	graph := kcg.WebGraph{
		Root: "/tmp/repo",
		Nodes: []kcg.WebGraphNode{
			{
				ID:        1,
				Name:      "a",
				Kind:      kcg.WebGraphNodeInternal,
				InDegree:  0,
				OutDegree: 1,
			},
		},
		Edges: []kcg.WebGraphEdge{
			{FromID: 1, ToID: 1, Count: 2},
		},
	}

	if err := writeGraphArtifacts(htmlPath, jsonPath, graph); err != nil {
		t.Fatalf("writeGraphArtifacts() unexpected error: %v", err)
	}

	payload, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read JSON artifact: %v", err)
	}
	var got kcg.WebGraph
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal JSON artifact: %v", err)
	}
	if got.Root != graph.Root {
		t.Fatalf("graph.Root = %q want %q", got.Root, graph.Root)
	}
	if len(got.Nodes) != len(graph.Nodes) {
		t.Fatalf("graph.Nodes len=%d want=%d", len(got.Nodes), len(graph.Nodes))
	}

	htmlPayload, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read HTML artifact: %v", err)
	}
	if !strings.Contains(string(htmlPayload), "\"graph.json\"") {
		t.Fatalf("html output missing data file reference: %s", string(htmlPayload))
	}
}

func TestCollectKotlinFilesForCompilePathsRespectsExcludesAndKtsSettings(t *testing.T) {

	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src")
	if err := os.MkdirAll(filepath.Join(srcDir, "child"), 0o755); err != nil {
		t.Fatalf("mkdir child dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "tmp"), 0o755); err != nil {
		t.Fatalf("mkdir tmp dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "build", "out"), 0o755); err != nil {
		t.Fatalf("mkdir build dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "keep.kts"), []byte("fun g() {}\n"), 0o644); err != nil {
		t.Fatalf("write kts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "build.gradle.kts"), []byte("plugins {}\n"), 0o644); err != nil {
		t.Fatalf("write build kts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "settings.gradle.kts"), []byte("rootProject.name = \"x\"\n"), 0o644); err != nil {
		t.Fatalf("write settings kts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "child", "B.kt"), []byte("package t\n\nfun h() {}\n"), 0o644); err != nil {
		t.Fatalf("write child kt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "tmp", "C.kt"), []byte("package t\n\nfun i() {}\n"), 0o644); err != nil {
		t.Fatalf("write excluded kt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "build", "out", "D.kt"), []byte("package t\n\nfun j() {}\n"), 0o644); err != nil {
		t.Fatalf("write build output kt: %v", err)
	}

	gotIncludeKts, err := collectKotlinFilesForCompilePaths([]string{srcDir}, []string{filepath.Join(srcDir, "tmp")}, true, false)
	if err != nil {
		t.Fatalf("collectKotlinFilesForCompilePaths() with kts include unexpected error: %v", err)
	}

	wantIncludeKts := []string{
		filepath.Clean(filepath.Join(srcDir, "A.kt")),
		filepath.Clean(filepath.Join(srcDir, "child", "B.kt")),
		filepath.Clean(filepath.Join(srcDir, "keep.kts")),
	}
	if len(gotIncludeKts) != len(wantIncludeKts) {
		t.Fatalf("with includeKTS=true got %d files, want %d; got=%v", len(gotIncludeKts), len(wantIncludeKts), gotIncludeKts)
	}
	for i := range wantIncludeKts {
		if gotIncludeKts[i] != wantIncludeKts[i] {
			t.Fatalf("with includeKTS=true file[%d]=%q want=%q", i, gotIncludeKts[i], wantIncludeKts[i])
		}
	}

	gotNoKts, err := collectKotlinFilesForCompilePaths([]string{srcDir}, []string{filepath.Join(srcDir, "tmp")}, false, false)
	if err != nil {
		t.Fatalf("collectKotlinFilesForCompilePaths() without kts unexpected error: %v", err)
	}
	wantNoKts := []string{
		filepath.Clean(filepath.Join(srcDir, "A.kt")),
		filepath.Clean(filepath.Join(srcDir, "child", "B.kt")),
	}
	if len(gotNoKts) != len(wantNoKts) {
		t.Fatalf("with includeKTS=false got %d files, want %d; got=%v", len(gotNoKts), len(wantNoKts), gotNoKts)
	}
	for i := range wantNoKts {
		if gotNoKts[i] != wantNoKts[i] {
			t.Fatalf("with includeKTS=false file[%d]=%q want=%q", i, gotNoKts[i], wantNoKts[i])
		}
	}

	gotBuildScriptsOnly, err := collectKotlinFilesForCompilePaths([]string{srcDir}, []string{filepath.Join(srcDir, "tmp")}, false, true)
	if err != nil {
		t.Fatalf("collectKotlinFilesForCompilePaths() with build scripts but no kts unexpected error: %v", err)
	}
	wantBuildScriptsOnly := []string{
		filepath.Clean(filepath.Join(srcDir, "A.kt")),
		filepath.Clean(filepath.Join(srcDir, "build.gradle.kts")),
		filepath.Clean(filepath.Join(srcDir, "child", "B.kt")),
		filepath.Clean(filepath.Join(srcDir, "settings.gradle.kts")),
	}
	if len(gotBuildScriptsOnly) != len(wantBuildScriptsOnly) {
		t.Fatalf("with includeKTS=false includeBuildScripts=true got %d files, want %d; got=%v", len(gotBuildScriptsOnly), len(wantBuildScriptsOnly), gotBuildScriptsOnly)
	}
	for i := range wantBuildScriptsOnly {
		if gotBuildScriptsOnly[i] != wantBuildScriptsOnly[i] {
			t.Fatalf("with includeKTS=false includeBuildScripts=true file[%d]=%q want=%q", i, gotBuildScriptsOnly[i], wantBuildScriptsOnly[i])
		}
	}
}

func TestCollectKotlinFilesForCompilePathsMissingPathReturnsError(t *testing.T) {
	repoRoot := t.TempDir()
	missing := filepath.Join(repoRoot, "does-not-exist")
	if _, err := collectKotlinFilesForCompilePaths([]string{missing}, nil, true, false); err == nil {
		t.Fatalf("collectKotlinFilesForCompilePaths(%q) should return error", missing)
	}
}

func TestRunCompileMissingArgFileReturnsError(t *testing.T) {

	repoRoot := t.TempDir()
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	_ = rd
	os.Stderr = w
	defer func() {
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := runCompile([]string{
		"--repo", repoRoot,
		"--arg-file", filepath.FromSlash("does-not-exist.args"),
		"--source", filepath.FromSlash("src/main/kotlin"),
	})
	_ = w.Close()
	if exitCode != 1 {
		t.Fatalf("runCompile() = %d, want 1", exitCode)
	}
}

func TestRunCompileMissingArgFileThroughRunReturnsError(t *testing.T) {
	repoRoot := t.TempDir()
	exitCode := run([]string{
		"compile",
		"--repo", repoRoot,
		"--arg-file", filepath.FromSlash("does-not-exist.args"),
		"--source", filepath.FromSlash("src/main/kotlin"),
	})
	if exitCode != 1 {
		t.Fatalf("run(compile --arg-file missing) = %d, want 1", exitCode)
	}
}

func TestRunCompileInvalidFlagReturnsUsage(t *testing.T) {
	repoRoot := t.TempDir()
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	_ = rd
	os.Stderr = w
	defer func() {
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := runCompile([]string{
		"--repo", repoRoot,
		"--does-not-exist",
	})
	_ = w.Close()

	if exitCode != 2 {
		t.Fatalf("runCompile() = %d, want 2", exitCode)
	}
}

func TestRunCompileMissingKotlincReturnsFailureByDefaultViaRun(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	exitCode := runCompile([]string{
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--kotlinc", filepath.FromSlash("/does-not-exist-kotlinc"),
	})
	if exitCode != 1 {
		t.Fatalf("runCompile() = %d, want 1", exitCode)
	}
}

func TestRunCompileMissingKotlincWithFailOnErrorFalseReturnsSuccessViaRun(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	exitCode := runCompile([]string{
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--kotlinc", filepath.FromSlash("/does-not-exist-kotlinc"),
		"--fail-on-error=false",
	})
	if exitCode != 0 {
		t.Fatalf("runCompile() = %d, want 0", exitCode)
	}
}

func TestRunCompileNoSourcesReturnsError(t *testing.T) {

	repoRoot := t.TempDir()
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	_ = rd
	os.Stderr = w
	defer func() {
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := runCompile([]string{
		"--repo", repoRoot,
		"--source", filepath.FromSlash("empty"),
	})
	_ = w.Close()
	if exitCode != 1 {
		t.Fatalf("runCompile() = %d, want 1", exitCode)
	}
}

func TestRunCompileNoSourcesViaRunReturnsError(t *testing.T) {
	repoRoot := t.TempDir()
	exitCode := run([]string{
		"compile",
		"--repo", repoRoot,
		"--source", filepath.FromSlash("empty"),
	})
	if exitCode != 1 {
		t.Fatalf("run(compile --source empty) = %d, want 1", exitCode)
	}
}

func TestRunCompileMissingKotlincReturnsFailureByDefault(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	exitCode := run([]string{
		"compile",
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--kotlinc", filepath.FromSlash("/does-not-exist-kotlinc"),
	})
	if exitCode != 1 {
		t.Fatalf("run(compile --kotlinc missing) = %d, want 1", exitCode)
	}
}

func TestRunCompileMissingKotlincWithFailOnErrorFalseReturnsSuccess(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	exitCode := run([]string{
		"compile",
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--kotlinc", filepath.FromSlash("/does-not-exist-kotlinc"),
		"--fail-on-error=false",
	})
	if exitCode != 0 {
		t.Fatalf("run(compile --kotlinc missing with fail-on-error=false) = %d, want 0", exitCode)
	}
}

func TestRunCompileSourceExcludeOutViaRun(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	excludeDir := filepath.Join(srcDir, "exclude")
	if err := os.MkdirAll(excludeDir, 0o755); err != nil {
		t.Fatalf("mkdir source dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun a() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(excludeDir, "B.kt"), []byte("package t\n\nfun b() {}\n"), 0o644); err != nil {
		t.Fatalf("write excluded source: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{
		"compile",
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--exclude", filepath.FromSlash("src/main/kotlin/exclude"),
		"--dry-run",
		"--out", filepath.FromSlash(filepath.Join(repoRoot, "out")),
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(compile source+exclude+out) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, filepath.FromSlash(filepath.Join(repoRoot, "out"))) {
		t.Fatalf("dry-run output missing out path: %q", output)
	}
}

func TestRunCompileDryRunArgFilePluginAndClasspath(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	libDir := filepath.Join(repoRoot, "lib")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir source dirs: %v", err)
	}
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("mkdir lib dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "libs.jar"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write classpath jar: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "plugin.jar"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write plugin jar: %v", err)
	}
	argFile := filepath.Join(repoRoot, "compile.args")
	if err := os.WriteFile(argFile, []byte("--language-version 1.9\n-P pluginArg=1\n"), 0o644); err != nil {
		t.Fatalf("write arg-file: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{
		"compile",
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--classpath", filepath.FromSlash("lib/libs.jar"),
		"--plugin", filepath.FromSlash("plugin.jar"),
		"--arg-file", filepath.FromSlash("compile.args"),
		"--dry-run",
		"--kotlinc", filepath.FromSlash("echo"),
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(compile --dry-run with arg-file/plugin/classpath) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "--language-version") || !strings.Contains(output, "1.9") {
		t.Fatalf("dry-run output missing arg-file content: %q", output)
	}
	if !strings.Contains(output, "-Xplugin="+filepath.Join(repoRoot, "plugin.jar")) {
		t.Fatalf("dry-run output missing plugin path: %q", output)
	}
	if !strings.Contains(output, "-cp") {
		t.Fatalf("dry-run output missing -cp: %q", output)
	}
	if !strings.Contains(output, filepath.Join(libDir, "libs.jar")) {
		t.Fatalf("dry-run output missing classpath path: %q", output)
	}
}

func TestRunUnknownCommandReturnsUsage(t *testing.T) {

	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	_ = rd
	os.Stderr = w
	defer func() {
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"does-not-exist"})
	_ = w.Close()

	if exitCode != 2 {
		t.Fatalf("run() = %d, want 2", exitCode)
	}
}

func TestRunTopLevelHelpIncludesUsageAndCommands(t *testing.T) {
	for _, flagArg := range []string{"--help", "-h", "help"} {
		t.Run(flagArg, func(t *testing.T) {
			origStdout := os.Stdout
			origStderr := os.Stderr
			rd, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("create pipe: %v", err)
			}
			if flagArg == "help" {
				os.Stdout = w
				os.Stderr = origStderr
			} else {
				os.Stdout = w
				os.Stderr = w
			}
			defer func() {
				os.Stdout = origStdout
				os.Stderr = origStderr
				_ = rd.Close()
			}()

			args := []string{flagArg}
			if flagArg == "help" {
				args = []string{"help"}
			}
			exitCode := run(args)
			_ = w.Close()

			if exitCode != 0 {
				t.Fatalf("run(%q) = %d, want 0", flagArg, exitCode)
			}

			var b bytes.Buffer
			if _, err := io.Copy(&b, rd); err != nil {
				t.Fatalf("read pipe: %v", err)
			}
			got := b.String()
			if !strings.Contains(got, "Usage:") {
				t.Fatalf("help output missing Usage: %q", got)
			}
			for _, command := range topLevelCommands {
				var expected string
				if command == "compile" {
					expected = "  kotlin-compiler-golang compile [flags] [-- <extra kotlinc args>]"
				} else {
					expected = "  kotlin-compiler-golang " + command + " [flags]"
				}
				if !strings.Contains(got, expected) {
					t.Fatalf("help output missing command %q: got=%q", command, got)
				}
			}
		})
	}
}

func TestRunSubcommandHelpWritesToStdout(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "direct", args: []string{"parse", "--help"}, want: "Usage of parse:"},
		{name: "help command", args: []string{"help", "parse"}, want: "Usage of parse:"},
		{name: "double dash route", args: []string{"--help", "symbols"}, want: "Usage of symbols:"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exitCode, stdout, stderr := runWithCapturedOutput(t, tc.args)
			if exitCode != 0 {
				t.Fatalf("run(%v) = %d, want 0", tc.args, exitCode)
			}
			if strings.TrimSpace(stderr) != "" {
				t.Fatalf("expected help on stdout only, stderr=%q", stderr)
			}
			if !strings.Contains(stdout, tc.want) {
				t.Fatalf("stdout = %q, want substring %q", stdout, tc.want)
			}
		})
	}
}

func TestRunParseCommandDelegates(t *testing.T) {

	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("fun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}

	exitCode := run([]string{
		"parse",
		"--repo", repoRoot,
		"--json",
	})
	if exitCode != 0 {
		t.Fatalf("run(parse) = %d, want 0", exitCode)
	}
}

func TestRunParseJSONOutput(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	exitCode := run([]string{
		"parse",
		"--repo", repoRoot,
		"--json",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(parse --json) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	var result kcg.RepositoryResult
	if err := json.Unmarshal(b.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal parse json: %v", err)
	}
	if result.Root != repoRoot {
		t.Fatalf("parse json root=%q want=%q", result.Root, repoRoot)
	}
	if result.TotalFiles != 1 {
		t.Fatalf("parse json total_files=%d want=1", result.TotalFiles)
	}
}

func TestRunParseSummaryOutputUsesUnifiedLabels(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}

	exitCode, stdout, stderr := runWithCapturedOutput(t, []string{
		"parse",
		"--repo", repoRoot,
	})
	if exitCode != 0 {
		t.Fatalf("run(parse) = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{"Root:", "Files:", "Parse:", "Duration:"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want substring %q", stdout, want)
		}
	}
}

func TestRunParseSummaryOutputUsesFailuresHeader(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("class 12345 {\n"), 0o644); err != nil {
		t.Fatalf("write invalid kt: %v", err)
	}

	exitCode, stdout, stderr := runWithCapturedOutput(t, []string{
		"parse",
		"--repo", repoRoot,
		"--fail-on-error=false",
	})
	if exitCode != 0 {
		t.Fatalf("run(parse) = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Failures:") {
		t.Fatalf("stdout = %q, want failures header", stdout)
	}
}

func TestRunParseIncludeBuildScriptsWithoutKts(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "tool.kts"), []byte("val answer = 42\n"), 0o644); err != nil {
		t.Fatalf("write tool kts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "build.gradle.kts"), []byte("plugins {}\n"), 0o644); err != nil {
		t.Fatalf("write build script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "settings.gradle.kts"), []byte("rootProject.name = \"sample\"\n"), 0o644); err != nil {
		t.Fatalf("write settings script: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	t.Cleanup(func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	})

	exitCode := run([]string{
		"parse",
		"--repo", repoRoot,
		"--json",
		"--include-kts=false",
		"--include-build-scripts",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(parse --include-kts=false --include-build-scripts --json) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	var includeBuildOnly kcg.RepositoryResult
	if err := json.Unmarshal(b.Bytes(), &includeBuildOnly); err != nil {
		t.Fatalf("unmarshal parse json: %v", err)
	}
	if includeBuildOnly.TotalFiles != 3 {
		t.Fatalf("include-build-scripts without kts total_files=%d want=3", includeBuildOnly.TotalFiles)
	}

	// reset capture for default behavior check
	rd2, w2, err := os.Pipe()
	if err != nil {
		t.Fatalf("create second pipe: %v", err)
	}
	os.Stdout = w2
	os.Stderr = w2
	exitCode = run([]string{
		"parse",
		"--repo", repoRoot,
		"--json",
	})
	if err := w2.Close(); err != nil {
		t.Fatalf("close second pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(parse with defaults --json) = %d, want 0", exitCode)
	}

	var b2 bytes.Buffer
	if _, err := io.Copy(&b2, rd2); err != nil {
		t.Fatalf("read second pipe: %v", err)
	}
	var defaultResult kcg.RepositoryResult
	if err := json.Unmarshal(b2.Bytes(), &defaultResult); err != nil {
		t.Fatalf("unmarshal default parse json: %v", err)
	}
	if defaultResult.TotalFiles != 2 {
		t.Fatalf("default parse total_files=%d want=2 (kt + .kts)", defaultResult.TotalFiles)
	}
}

func TestRunSymbolsJSONOutput(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nclass A\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	exitCode := run([]string{
		"symbols",
		"--repo", repoRoot,
		"--json",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(symbols --json) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(b.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal symbols json: %v", err)
	}
	if len(result) == 0 {
		t.Fatalf("symbols json should contain fields, got empty object")
	}
}

func TestRunGraphJSONOutput(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nclass A\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	exitCode := run([]string{
		"graph",
		"--repo", repoRoot,
		"--json",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(graph --json) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	var graph kcg.WebGraph
	if err := json.Unmarshal(b.Bytes(), &graph); err != nil {
		t.Fatalf("unmarshal graph json: %v", err)
	}
	if graph.Root != repoRoot {
		t.Fatalf("graph json root=%q want=%q", graph.Root, repoRoot)
	}
	if len(graph.Nodes) == 0 {
		t.Fatalf("graph json nodes should not be empty")
	}
}

func TestRunDepsJSONOutput(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nclass A\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	exitCode := run([]string{
		"deps",
		"--repo", repoRoot,
		"--json",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(deps --json) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(b.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal deps json: %v", err)
	}
	if _, ok := result["nodes"]; !ok {
		t.Fatalf("deps json should contain nodes")
	}
}

func TestRunResolveJSONOutput(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nimport java.util.List\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	exitCode := run([]string{
		"resolve",
		"--repo", repoRoot,
		"--json",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(resolve --json) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	var report kcg.ResolutionReport
	if err := json.Unmarshal(b.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal resolve json: %v", err)
	}
	if report.TotalImports < 1 {
		t.Fatalf("resolve report should include at least one import, got %d", report.TotalImports)
	}
}

func TestRunResolveSummaryOutputUsesUnifiedLabels(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nimport java.util.List\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}

	exitCode, stdout, stderr := runWithCapturedOutput(t, []string{
		"resolve",
		"--repo", repoRoot,
	})
	if exitCode != 0 {
		t.Fatalf("run(resolve) = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{"Root:", "Imports:", "Resolved:", "Parse:", "Duration:"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want substring %q", stdout, want)
		}
	}
}

func TestRunResolveSummaryOutputUsesUnifiedSectionHeaders(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nimport missing.pkg.Type\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}

	exitCode, stdout, stderr := runWithCapturedOutput(t, []string{
		"resolve",
		"--repo", repoRoot,
	})
	if exitCode != 0 {
		t.Fatalf("run(resolve) = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Unresolved Imports:") {
		t.Fatalf("stdout = %q, want unresolved imports section header", stdout)
	}
}

func TestRunDepsSummaryOutputUsesTopEdgesHeader(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package a\n\nimport b.B\n\nclass A(val b: B)\n"), 0o644); err != nil {
		t.Fatalf("write A.kt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "B.kt"), []byte("package b\n\nclass B\n"), 0o644); err != nil {
		t.Fatalf("write B.kt: %v", err)
	}

	exitCode, stdout, stderr := runWithCapturedOutput(t, []string{
		"deps",
		"--repo", repoRoot,
	})
	if exitCode != 0 {
		t.Fatalf("run(deps) = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Top Edges:") {
		t.Fatalf("stdout = %q, want top edges header", stdout)
	}
}

func TestRunGraphWritesArtifacts(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nclass A\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}

	outBase := filepath.Join(repoRoot, "out", "graph-report")
	htmlPath := outBase + ".html"
	jsonPath := outBase + ".json"

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{
		"graph",
		"--repo", repoRoot,
		"--out", outBase,
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(graph --out %s) = %d, want 0", outBase, exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, filepath.Base(htmlPath)) {
		t.Fatalf("graph output should mention html path in %q", output)
	}
	if !strings.Contains(output, filepath.Base(jsonPath)) {
		t.Fatalf("graph output should mention json path in %q", output)
	}
	for _, want := range []string{"Root:", "Nodes:", "Edges:", "Parse:", "Duration:", "Output:", "Viewer:"} {
		if !strings.Contains(output, want) {
			t.Fatalf("graph output = %q, want substring %q", output, want)
		}
	}

	htmlPayload, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read html artifact: %v", err)
	}
	if len(htmlPayload) == 0 {
		t.Fatalf("html artifact is empty: %q", htmlPath)
	}
	jsonPayload, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read json artifact: %v", err)
	}
	if len(jsonPayload) == 0 {
		t.Fatalf("json artifact is empty: %q", jsonPath)
	}

	var graph kcg.WebGraph
	if err := json.Unmarshal(jsonPayload, &graph); err != nil {
		t.Fatalf("unmarshal graph artifact json: %v", err)
	}
	if graph.Root != repoRoot {
		t.Fatalf("graph artifact root=%q want=%q", graph.Root, repoRoot)
	}
}

func TestRunCompileHelp(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"compile", "--help"})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(compile --help) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if !strings.Contains(b.String(), "Usage of compile:") {
		t.Fatalf("compile --help output missing compile flag usage: %q", b.String())
	}
}

func TestRunCompileDryRunViaRun(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{
		"compile",
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--dry-run",
		"--out", filepath.FromSlash(filepath.Join(repoRoot, "out")),
		"--kotlinc", filepath.FromSlash("echo"),
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(compile --dry-run) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "kotlin-compiler-golang compile -> echo") {
		t.Fatalf("dry-run output missing compiler line: %q", output)
	}
}

func TestRunCompileDryRunViaRunPreservesArgumentOrder(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	libDir := filepath.Join(repoRoot, "lib")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("mkdir lib dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "compile.args"), []byte("--language-version 1.9\n"), 0o644); err != nil {
		t.Fatalf("write arg file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "ksp-plugin.jar"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write plugin jar: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "libs.jar"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write classpath jar: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{
		"compile",
		"--repo", filepath.FromSlash(repoRoot),
		"--source", filepath.FromSlash(filepath.Join("src", "main", "kotlin")),
		"--jvm-target", "1.8",
		"--include-runtime",
		"--classpath", filepath.FromSlash(filepath.Join("lib", "libs.jar")),
		"--plugin", filepath.FromSlash("ksp-plugin.jar"),
		"--arg-file", filepath.FromSlash("compile.args"),
		"--dry-run",
		"--kotlinc", filepath.FromSlash("echo"),
		"--",
		"-P",
		"plugin:com.test.x=true",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(compile --dry-run) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()

	plugin := filepath.Clean(filepath.Join(repoRoot, "ksp-plugin.jar"))
	classpath := filepath.Clean(filepath.Join(libDir, "libs.jar"))
	source := filepath.Clean(filepath.Join(srcDir, "A.kt"))

	idxJVM := strings.Index(output, "\"-jvm-target\"")
	idxJVMVersion := strings.Index(output, "\"1.8\"")
	idxIncludeRuntime := strings.Index(output, "\"-include-runtime\"")
	idxClasspath := strings.Index(output, "\"-cp\"")
	idxPlugin := strings.Index(output, "\"-Xplugin="+plugin)
	idxD := strings.Index(output, "\"-d\"")
	idxSource := strings.Index(output, "\""+source+"\"")
	idxArgFile := strings.Index(output, "\"--language-version\"")
	idxFriend := strings.Index(output, "\"1.9\"")
	idxPassthrough := strings.Index(output, "\"-P\"")

	if idxJVM < 0 || idxJVMVersion < 0 || idxIncludeRuntime < 0 || idxClasspath < 0 || idxPlugin < 0 || idxD < 0 || idxSource < 0 || idxArgFile < 0 || idxFriend < 0 || idxPassthrough < 0 {
		t.Fatalf("required dry-run tokens missing. output=%s", output)
	}
	if idxJVM > idxJVMVersion {
		t.Fatalf("jvm-target flag should come before its version token. output=%s", output)
	}
	if idxJVMVersion > idxIncludeRuntime {
		t.Fatalf("jvm-target should come before include-runtime. output=%s", output)
	}
	if idxIncludeRuntime > idxClasspath {
		t.Fatalf("include-runtime should come before classpath. output=%s", output)
	}
	if idxClasspath > idxPlugin {
		t.Fatalf("classpath should come before plugin path flags. output=%s", output)
	}
	if idxPlugin > idxD {
		t.Fatalf("plugin flags should come before -d. output=%s", output)
	}
	if idxD > idxSource {
		t.Fatalf("-d should come before source list. output=%s", output)
	}
	if idxSource > idxArgFile || idxSource > idxFriend {
		t.Fatalf("arg-file tokens should come after source list. output=%s", output)
	}
	if idxArgFile > idxPassthrough || idxFriend > idxPassthrough {
		t.Fatalf("arg-file tokens should come before passthrough args. output=%s", output)
	}
	if !strings.Contains(output, plugin) {
		t.Fatalf("dry-run output missing resolved plugin path: %s", output)
	}
	if !strings.Contains(output, classpath) {
		t.Fatalf("dry-run output missing resolved classpath path: %s", output)
	}
}

func TestRunCommandsRejectInvalidParserBackend(t *testing.T) {

	repoRoot := t.TempDir()
	commands := []string{"parse", "symbols", "deps", "resolve", "graph"}

	for _, command := range commands {
		command := command
		t.Run(command, func(t *testing.T) {

			exitCode := run([]string{
				command,
				"--repo", repoRoot,
				"--parser-backend", "javac",
			})
			if exitCode != 2 {
				t.Fatalf("run(%s --parser-backend javac) = %d, want 2", command, exitCode)
			}
		})
	}
}

func TestRunSubcommandHelpReturnsUsage(t *testing.T) {
	commands := []string{"parse", "symbols", "deps", "resolve", "graph", "acceptance", "compile"}

	for _, command := range commands {
		command := command
		t.Run(command, func(t *testing.T) {
			origStdout := os.Stdout
			origStderr := os.Stderr
			rd, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("create pipe: %v", err)
			}
			os.Stdout = w
			os.Stderr = w
			defer func() {
				os.Stdout = origStdout
				os.Stderr = origStderr
				_ = rd.Close()
			}()

			exitCode := run([]string{command, "--help"})
			_ = w.Close()

			if exitCode != 0 {
				t.Fatalf("run(%s --help) = %d, want 0", command, exitCode)
			}

			var b bytes.Buffer
			if _, err := io.Copy(&b, rd); err != nil {
				t.Fatalf("read pipe: %v", err)
			}
			got := b.String()
			if !strings.Contains(got, "Usage of "+command+":") {
				t.Fatalf("help output missing Usage of %s:, got=%q", command, got)
			}
		})
	}
}

func TestRunHelpSubcommandInvokesSubcommandUsage(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"help", "parse"})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(help, parse) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, "Usage of parse:") {
		t.Fatalf("help parse output missing parse usage: %q", got)
	}
}

func TestRunDashHSubcommandInvokesSubcommandUsage(t *testing.T) {
	t.Run("dash-h", func(t *testing.T) {
		origStdout := os.Stdout
		origStderr := os.Stderr
		rd, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("create pipe: %v", err)
		}
		os.Stdout = w
		os.Stderr = w
		defer func() {
			os.Stdout = origStdout
			os.Stderr = origStderr
			_ = rd.Close()
		}()

		exitCode := run([]string{"-h", "parse"})
		_ = w.Close()
		if exitCode != 0 {
			t.Fatalf("run(-h, parse) = %d, want 0", exitCode)
		}

		var b bytes.Buffer
		if _, err := io.Copy(&b, rd); err != nil {
			t.Fatalf("read pipe: %v", err)
		}
		if !strings.Contains(b.String(), "Usage of parse:") {
			t.Fatalf("help parse output missing parse usage: %q", b.String())
		}
	})

	t.Run("dash-dash-help", func(t *testing.T) {
		origStdout := os.Stdout
		origStderr := os.Stderr
		rd, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("create pipe: %v", err)
		}
		os.Stdout = w
		os.Stderr = w
		defer func() {
			os.Stdout = origStdout
			os.Stderr = origStderr
			_ = rd.Close()
		}()

		exitCode := run([]string{"--help", "symbols"})
		_ = w.Close()
		if exitCode != 0 {
			t.Fatalf("run(--help, symbols) = %d, want 0", exitCode)
		}

		var b bytes.Buffer
		if _, err := io.Copy(&b, rd); err != nil {
			t.Fatalf("read pipe: %v", err)
		}
		if !strings.Contains(b.String(), "Usage of symbols:") {
			t.Fatalf("help symbols output missing symbols usage: %q", b.String())
		}
	})
}

func TestRunTopLevelHelpAndUnknownSecondArgDoesNotRecurse(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"--help", "--help"})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(--help, --help) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if !strings.Contains(b.String(), "Usage:") {
		t.Fatalf("expected top-level usage output, got=%q", b.String())
	}
}

func TestRunDashHelpDashHelpShowsUsageOrSubcommandHelp(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"--help", "--help", "parse"})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(--help, --help, parse) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, "Usage of parse:") && !strings.Contains(got, "Usage:") {
		t.Fatalf("expected help output, got=%q", got)
	}
}

func TestRunDashHelpDashHelpKnownCommandShowsSubcommandHelp(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"--help", "--help", "parse"})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(--help, --help, parse) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if !strings.Contains(b.String(), "Usage of parse:") {
		t.Fatalf("expected parse usage, got=%q", b.String())
	}
}

func TestRunHelpAliasAliasKnownCommandShowsSubcommandHelp(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "dash-h-dash-h-parse", args: []string{"-h", "-h", "parse"}},
		{name: "dash-h-help-parse", args: []string{"-h", "help", "parse"}},
		{name: "help-dash-help-parse", args: []string{"help", "-h", "parse"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			origStdout := os.Stdout
			origStderr := os.Stderr
			rd, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("create pipe: %v", err)
			}
			os.Stdout = w
			os.Stderr = w

			exitCode := run(tc.args)
			if err := w.Close(); err != nil {
				t.Fatalf("close pipe write end: %v", err)
			}
			if exitCode != 0 {
				os.Stdout = origStdout
				os.Stderr = origStderr
				t.Fatalf("run(%v) = %d, want 0", tc.args, exitCode)
			}

			var b bytes.Buffer
			if _, err := io.Copy(&b, rd); err != nil {
				t.Fatalf("read pipe: %v", err)
			}
			_ = rd.Close()
			output := b.String()
			if !strings.Contains(output, "Usage of parse:") {
				os.Stdout = origStdout
				os.Stderr = origStderr
				t.Fatalf("expected parse usage, got=%q", output)
			}
			os.Stdout = origStdout
			os.Stderr = origStderr
		})
	}
}

func TestRunHelpAliasChainKnownCommandWithExtraArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "help dash-h command with flag",
			args: []string{"help", "-h", "parse", "--repo", "."},
			want: "Usage of parse:",
		},
		{
			name: "dash-h double-h command with parse arg",
			args: []string{"-h", "--help", "parse", "--repo", "."},
			want: "Usage of parse:",
		},
		{
			name: "dash-h help command with unknown token",
			args: []string{"--help", "help", "parse", "--does-not-exist"},
			want: "Usage of parse:",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			origStdout := os.Stdout
			origStderr := os.Stderr
			rd, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("create pipe: %v", err)
			}
			os.Stdout = w
			os.Stderr = w

			exitCode := run(tc.args)
			_ = w.Close()
			if exitCode != 0 {
				os.Stdout = origStdout
				os.Stderr = origStderr
				t.Fatalf("run(%v) = %d, want 0", tc.args, exitCode)
			}

			var b bytes.Buffer
			if _, err := io.Copy(&b, rd); err != nil {
				os.Stdout = origStdout
				os.Stderr = origStderr
				t.Fatalf("read pipe: %v", err)
			}
			_ = rd.Close()
			output := b.String()
			if !strings.Contains(output, tc.want) {
				os.Stdout = origStdout
				os.Stderr = origStderr
				t.Fatalf("expected %q, got=%q", tc.want, output)
			}
			os.Stdout = origStdout
			os.Stderr = origStderr
		})
	}
}

func TestRunHelpAliasChainAcceptanceWithExtraArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "help dash-h acceptance with flags",
			args: []string{"help", "-h", "acceptance", "--max-failed-files", "0"},
			want: "Usage of acceptance:",
		},
		{
			name: "dash-h dash-help acceptance with extra arg",
			args: []string{"-h", "--help", "acceptance", "--max-unresolved-imports", "0"},
			want: "Usage of acceptance:",
		},
		{
			name: "double dash help acceptance with json flag",
			args: []string{"--help", "help", "acceptance", "--json"},
			want: "Usage of acceptance:",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			origStdout := os.Stdout
			origStderr := os.Stderr
			rd, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("create pipe: %v", err)
			}
			os.Stdout = w
			os.Stderr = w

			exitCode := run(tc.args)
			_ = w.Close()
			if exitCode != 0 {
				os.Stdout = origStdout
				os.Stderr = origStderr
				t.Fatalf("run(%v) = %d, want 0", tc.args, exitCode)
			}

			var b bytes.Buffer
			if _, err := io.Copy(&b, rd); err != nil {
				os.Stdout = origStdout
				os.Stderr = origStderr
				t.Fatalf("read pipe: %v", err)
			}
			_ = rd.Close()
			output := b.String()
			if !strings.Contains(output, tc.want) {
				os.Stdout = origStdout
				os.Stderr = origStderr
				t.Fatalf("expected %q, got=%q", tc.want, output)
			}
			os.Stdout = origStdout
			os.Stderr = origStderr
		})
	}
}

func TestIsBuildArtifactDir(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"", false},
		{".git", true},
		{"build", true},
		{"out", true},
		{"target", true},
		{"node_modules", true},
		{"src", false},
		{"settings.gradle.kts", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := isBuildArtifactDir(tc.name)
			if got != tc.want {
				t.Fatalf("isBuildArtifactDir(%q) = %v want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestRunDashHelpDashHelpUnknownCommandFallsBackToTopLevelUsage(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"--help", "--help", "does-not-exist"})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(--help, --help, does-not-exist) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if !strings.Contains(b.String(), "Usage:") {
		t.Fatalf("expected top-level usage output, got=%q", b.String())
	}
}

func TestRunHelpHelpAcceptanceStillShowsAcceptanceUsage(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"--help", "help", "acceptance"})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(help, help, acceptance) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, "Usage of acceptance:") {
		t.Fatalf("expected acceptance usage, got=%q", got)
	}
}

func TestRunHelpFlagsReturnZero(t *testing.T) {

	cases := []struct {
		name string
		args []string
	}{
		{name: "dash-h", args: []string{"-h"}},
		{name: "double-dash-help", args: []string{"--help"}},
		{name: "help command", args: []string{"help"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			origStdout := os.Stdout
			origStderr := os.Stderr
			rd, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("create pipe: %v", err)
			}
			os.Stdout = w
			os.Stderr = w
			defer func() {
				os.Stdout = origStdout
				os.Stderr = origStderr
				_ = rd.Close()
			}()

			exitCode := run(tc.args)
			_ = w.Close()
			if exitCode != 0 {
				t.Fatalf("run(%q) = %d, want 0", tc.name, exitCode)
			}
			var b bytes.Buffer
			if _, err := io.Copy(&b, rd); err != nil {
				t.Fatalf("read pipe: %v", err)
			}
			if !strings.Contains(b.String(), "Usage:") {
				t.Fatalf("expected usage output for %q, got=%q", tc.name, b.String())
			}
		})
	}
}

func TestRunDashHelpWithoutSubcommandDoesNotRecurse(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"-h", "does-not-exist"})
	_ = w.Close()
	if exitCode != 2 {
		t.Fatalf("run(-h, does-not-exist) = %d, want 2", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "unknown command") {
		t.Fatalf("expected unknown command for -h + does-not-exist, got=%q", output)
	}
}

func TestRunHelpWithUnknownSubcommandDoesNotRecurse(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"--help", "does-not-exist"})
	_ = w.Close()
	if exitCode != 2 {
		t.Fatalf("run(--help, does-not-exist) = %d, want 2", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if !strings.Contains(b.String(), "unknown command") {
		t.Fatalf("expected unknown command for --help + does-not-exist, got=%q", b.String())
	}
}

func TestRunHelpSubcommandUnknownCommandReturnsUsage(t *testing.T) {
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"help", "does-not-exist"})
	_ = w.Close()
	if exitCode != 2 {
		t.Fatalf("run(help, does-not-exist) = %d, want 2", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, "unknown command") {
		t.Fatalf("expected unknown command message, got=%q", got)
	}
}

func TestRunHelpHelpUnknownSubcommandReturnsTopLevelUsage(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stderr = origStderr
		os.Stdout = origStdout
		_ = rd.Close()
	}()

	exitCode := run([]string{"--help", "help", "does-not-exist"})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(--help, help, does-not-exist) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if !strings.Contains(b.String(), "Usage:") {
		t.Fatalf("expected top-level usage, got=%q", b.String())
	}
}

func TestRunDashHelpHelpUnknownSubcommandReturnsTopLevelUsage(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stderr = origStderr
		os.Stdout = origStdout
		_ = rd.Close()
	}()

	exitCode := run([]string{"-h", "help", "does-not-exist"})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(-h, help, does-not-exist) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if !strings.Contains(b.String(), "Usage:") {
		t.Fatalf("expected top-level usage, got=%q", b.String())
	}
}

func TestRunHelpAliasCombinationsWithUnknownSubcommandReturnsTopLevelUsage(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "help --help unknown", args: []string{"help", "--help", "does-not-exist"}},
		{name: "help -h unknown", args: []string{"help", "-h", "does-not-exist"}},
		{name: "dash-h --help unknown", args: []string{"-h", "--help", "does-not-exist"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			origStdout := os.Stdout
			origStderr := os.Stderr
			rd, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("create pipe: %v", err)
			}
			os.Stdout = w
			os.Stderr = w
			defer func() {
				os.Stdout = origStdout
				os.Stderr = origStderr
				_ = rd.Close()
			}()

			exitCode := run(tc.args)
			_ = w.Close()
			if exitCode != 0 {
				t.Fatalf("run(%v) = %d, want 0", tc.args, exitCode)
			}

			var b bytes.Buffer
			if _, err := io.Copy(&b, rd); err != nil {
				t.Fatalf("read pipe: %v", err)
			}
			if !strings.Contains(b.String(), "Usage:") {
				t.Fatalf("expected top-level usage, got=%q", b.String())
			}
		})
	}
}

func TestRunHelpAliasUnknownChainFallsBackToTopLevelUsage(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "help -h unknown", args: []string{"help", "-h", "does-not-exist"}},
		{name: "help --help unknown", args: []string{"help", "--help", "does-not-exist"}},
		{name: "dash-h dash-h unknown", args: []string{"-h", "-h", "does-not-exist"}},
		{name: "double-dash-help unknown", args: []string{"--help", "--help", "does-not-exist"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			origStdout := os.Stdout
			origStderr := os.Stderr
			rd, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("create pipe: %v", err)
			}
			os.Stdout = w
			os.Stderr = w
			defer func() {
				os.Stdout = origStdout
				os.Stderr = origStderr
				_ = rd.Close()
			}()

			exitCode := run(tc.args)
			_ = w.Close()
			if exitCode != 0 {
				t.Fatalf("run(%v) = %d, want 0", tc.args, exitCode)
			}

			var b bytes.Buffer
			if _, err := io.Copy(&b, rd); err != nil {
				t.Fatalf("read pipe: %v", err)
			}
			if !strings.Contains(b.String(), "Usage:") {
				t.Fatalf("expected top-level usage, got=%q", b.String())
			}
		})
	}
}

func TestRunHelpSubcommandWithExtraUnknownArgStillShowsUsage(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"help", "parse", "--does-not-exist"})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(help, parse, --does-not-exist) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if !strings.Contains(b.String(), "Usage of parse:") {
		t.Fatalf("expected parse usage, got=%q", b.String())
	}
}

func TestRunDashHelpHelpCommand(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"--help", "help", "parse"})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(--help, help, parse) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, "Usage of parse:") {
		t.Fatalf("expected parse usage via normalized help-help parse invocation, got=%q", got)
	}
}

func TestRunHelpSubcommandWithExtraArgStillReturnsSubcommandUsage(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"help", "parse", "--repo", "."})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(help, parse, --repo, .) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, "Usage of parse:") {
		t.Fatalf("expected parse usage, got=%q", got)
	}
}

func TestRunDashHSubcommandWithExtraArgStillReturnsSubcommandUsage(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{"-h", "parse", "--repo", "."})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(-h, parse, --repo, .) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, "Usage of parse:") {
		t.Fatalf("expected parse usage, got=%q", got)
	}
}

func TestRunCommandMissingRepoReturnsError(t *testing.T) {

	repoRoot := filepath.Join(t.TempDir(), "missing", "repo")
	commands := []string{"parse", "symbols", "deps", "resolve", "graph"}

	for _, command := range commands {
		command := command
		t.Run(command, func(t *testing.T) {

			exitCode := run([]string{
				command,
				"--repo", repoRoot,
			})
			if exitCode != 1 {
				t.Fatalf("run(%s --repo=%s) = %d, want 1", command, repoRoot, exitCode)
			}
		})
	}
}

func TestRunNoArgsReturnsUsage(t *testing.T) {

	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	_ = rd
	os.Stderr = w
	defer func() {
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{})
	_ = w.Close()
	if exitCode != 2 {
		t.Fatalf("run() = %d, want 2", exitCode)
	}
}

func TestRunCompileDryRunPassesThroughUnknownArgs(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	libDir := filepath.Join(repoRoot, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("mkdir lib dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	argFile := filepath.Join(repoRoot, "compile.args")
	if err := os.WriteFile(argFile, []byte("--language-version=1.9\n"), 0o644); err != nil {
		t.Fatalf("write arg file: %v", err)
	}
	classpath := filepath.Join(libDir, "libs.jar")
	if err := os.WriteFile(classpath, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write classpath file: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()
	defer func() {
		_ = rd.Close()
	}()

	exitCode := runCompile([]string{
		"--repo", repoRoot,
		"--arg-file", filepath.FromSlash("compile.args"),
		"--classpath", filepath.FromSlash(filepath.Join("lib", "libs.jar")),
		"--jvm-target", "1.8",
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--dry-run",
		"--",
		"-P",
		"plugin:com.test.x=true",
		"-Xjsr305=strict",
	})
	if exitCode != 0 {
		_ = w.Close()
		t.Fatalf("runCompile() = %d, want 0", exitCode)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "\"-P\"") {
		t.Fatalf("dry-run output missing passthrough -P: %s", output)
	}
	if !strings.Contains(output, "\"plugin:com.test.x=true\"") {
		t.Fatalf("dry-run output missing passthrough plugin option: %s", output)
	}
	if !strings.Contains(output, "\"-Xjsr305=strict\"") {
		t.Fatalf("dry-run output missing passthrough argument: %s", output)
	}
	if !strings.Contains(output, "--language-version=1.9") {
		t.Fatalf("dry-run output missing arg-file content: %s", output)
	}
	if !strings.Contains(output, "libs.jar") {
		t.Fatalf("dry-run output missing resolved classpath: %s", output)
	}

	idxCompilerArg := strings.Index(output, "\"--language-version=1.9\"")
	idxPassthrough := strings.Index(output, "\"-P\"")
	if idxCompilerArg < 0 {
		t.Fatalf("expected compiler arg from arg-file in output: %s", output)
	}
	if idxPassthrough < 0 {
		t.Fatalf("expected passthrough -P in output: %s", output)
	}
	if idxPassthrough < idxCompilerArg {
		t.Fatalf("expected passthrough args to be appended after compiler args. output=%s", output)
	}
}

func TestRunCompileDryRunOrderWithResolvedBuildArgsAndPassthrough(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	libDir := filepath.Join(repoRoot, "lib")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("mkdir lib dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "compile.args"), []byte(`--language-version 1.9 --friend-modules "module one"`), 0o644); err != nil {
		t.Fatalf("write arg file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "ksp-plugin.jar"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "libs.jar"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write classpath: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()
	defer func() {
		_ = rd.Close()
	}()

	exitCode := runCompile([]string{
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--arg-file", filepath.FromSlash("compile.args"),
		"--classpath", filepath.FromSlash(filepath.Join("lib", "libs.jar")),
		"--plugin", filepath.FromSlash("ksp-plugin.jar"),
		"--jvm-target", "1.8",
		"--dry-run",
		"--",
		"-P",
		"plugin:com.test.x=true",
		"-Xjsr305=strict",
	})
	if exitCode != 0 {
		_ = w.Close()
		t.Fatalf("runCompile() = %d, want 0", exitCode)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()

	plugin := filepath.Clean(filepath.Join(repoRoot, "ksp-plugin.jar"))
	classpath := filepath.Clean(filepath.Join(repoRoot, "lib", "libs.jar"))
	sourceFile := filepath.Clean(filepath.Join(srcDir, "A.kt"))
	if !strings.Contains(output, "\"--language-version\"") || !strings.Contains(output, "\"1.9\"") {
		t.Fatalf("dry-run output missing arg-file tokens: %s", output)
	}
	if !strings.Contains(output, "\"module one\"") {
		t.Fatalf("dry-run output missing quoted arg-file token value: %s", output)
	}
	if !strings.Contains(output, plugin) {
		t.Fatalf("dry-run output missing resolved plugin path: %s", output)
	}
	if !strings.Contains(output, classpath) {
		t.Fatalf("dry-run output missing resolved classpath path: %s", output)
	}
	if !strings.Contains(output, sourceFile) {
		t.Fatalf("dry-run output missing source file: %s", output)
	}
	if !strings.Contains(output, "\"-P\"") {
		t.Fatalf("dry-run output missing passthrough -P: %s", output)
	}
	if !strings.Contains(output, "\"plugin:com.test.x=true\"") {
		t.Fatalf("dry-run output missing passthrough plugin option: %s", output)
	}
	if !strings.Contains(output, "\"-Xjsr305=strict\"") {
		t.Fatalf("dry-run output missing passthrough argument: %s", output)
	}

	idxClasspath := strings.Index(output, "\"-cp\"")
	idxPlugin := strings.Index(output, "\"-Xplugin="+plugin)
	idxD := strings.Index(output, "\"-d\"")
	idxSource := strings.Index(output, "\""+sourceFile+"\"")
	idxArgFile := strings.Index(output, "\"--language-version\"")
	idxFriend := strings.Index(output, "\"module one\"")
	idxPassthrough := strings.Index(output, "\"-P\"")
	if idxClasspath < 0 || idxPlugin < 0 || idxD < 0 || idxSource < 0 || idxArgFile < 0 || idxFriend < 0 || idxPassthrough < 0 {
		t.Fatalf("required dry-run tokens missing: output=%s", output)
	}

	if idxPlugin < idxClasspath {
		t.Fatalf("plugin should appear after classpath: %s", output)
	}
	if idxD < idxClasspath || idxD < idxPlugin {
		t.Fatalf("destination flag should appear after classpath+plugin: %s", output)
	}
	if idxSource < idxD {
		t.Fatalf("sources should appear after -d: %s", output)
	}
	if idxArgFile < idxD || idxFriend < idxD {
		t.Fatalf("arg-file args should appear before source files: %s", output)
	}
	if idxPassthrough < idxSource {
		t.Fatalf("passthrough args should be appended after source/compiled args: %s", output)
	}
	if idxPassthrough < idxFriend {
		t.Fatalf("passthrough args should come after arg-file tokens: %s", output)
	}
}

func TestRunCompileArgFileWithoutTrailingNewline(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	argFile := filepath.Join(repoRoot, "compile.args")
	if err := os.WriteFile(argFile, []byte(`--language-version 1.9 --friend-modules "module one"`), 0o644); err != nil {
		t.Fatalf("write arg file: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()
	defer func() {
		_ = rd.Close()
	}()

	exitCode := runCompile([]string{
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--arg-file", filepath.FromSlash("compile.args"),
		"--dry-run",
		"--",
		"-P",
		"plugin:com.test.x=true",
	})
	if exitCode != 0 {
		_ = w.Close()
		t.Fatalf("runCompile() = %d, want 0", exitCode)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "\"--language-version\"") {
		t.Fatalf("dry-run output missing arg-file token: %s", output)
	}
	if !strings.Contains(output, "\"1.9\"") {
		t.Fatalf("dry-run output missing arg-file value: %s", output)
	}
	if !strings.Contains(output, "\"module one\"") {
		t.Fatalf("dry-run output missing quoted arg-file value: %s", output)
	}
	if !strings.Contains(output, "\"-P\"") {
		t.Fatalf("dry-run output missing passthrough -P: %s", output)
	}
	if !strings.Contains(output, "\"plugin:com.test.x=true\"") {
		t.Fatalf("dry-run output missing passthrough plugin: %s", output)
	}

	idxArg := strings.Index(output, "\"module one\"")
	idxPassthrough := strings.Index(output, "\"-P\"")
	if idxArg < 0 {
		t.Fatalf("expected module one in output: %s", output)
	}
	if idxPassthrough < idxArg {
		t.Fatalf("expected passthrough args after arg-file args. output=%s", output)
	}
}

func TestRunCompileListSourcesResolvesRepoRelativeInputs(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "compile.args"), []byte("--jvm-target=1.9\n"), 0o644); err != nil {
		t.Fatalf("write arg file: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()
	defer func() {
		_ = rd.Close()
	}()

	exitCode := runCompile([]string{
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--list-sources",
	})
	if exitCode != 0 {
		_ = w.Close()
		_ = rd.Close()
		t.Fatalf("runCompile() = %d, want 0", exitCode)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if expected := filepath.Clean(filepath.Join(repoRoot, "src", "main", "kotlin", "A.kt")); !strings.Contains(b.String(), expected) {
		t.Fatalf("list-sources output missing expected source path %q. output=%s", expected, b.String())
	}
}

func TestRunCompileListSourcesIncludesBuildScriptsWithoutKts(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "tool.kts"), []byte("val answer = 42\n"), 0o644); err != nil {
		t.Fatalf("write kts source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "build.gradle.kts"), []byte("plugins {}\n"), 0o644); err != nil {
		t.Fatalf("write build script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "settings.gradle.kts"), []byte("rootProject.name = \"sample\"\n"), 0o644); err != nil {
		t.Fatalf("write settings script: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{
		"compile",
		"--repo", repoRoot,
		"--list-sources",
		"--include-kts=false",
		"--include-build-scripts",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(compile --list-sources --include-kts=false --include-build-scripts) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "sources=3") {
		t.Fatalf("expected 3 sources when include-kts=false and include-build-scripts=true, got output: %s", output)
	}
	expected := []string{
		filepath.Clean(filepath.Join(srcDir, "A.kt")),
		filepath.Clean(filepath.Join(repoRoot, "build.gradle.kts")),
		filepath.Clean(filepath.Join(repoRoot, "settings.gradle.kts")),
	}
	for _, path := range expected {
		if !strings.Contains(output, path) {
			t.Fatalf("list-sources output missing expected source %q. output=%s", path, output)
		}
	}
	if strings.Contains(output, filepath.Clean(filepath.Join(srcDir, "tool.kts"))) {
		t.Fatalf("list-sources should not include ordinary .kts when --include-kts=false. output=%s", output)
	}
}

func TestRunCompileListSourcesExcludesExplicitFile(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	keepPath := filepath.Join(srcDir, "Keep.kt")
	excludedPath := filepath.Join(srcDir, "Skip.kt")
	if err := os.WriteFile(keepPath, []byte("package t\n\nfun keep() {}\n"), 0o644); err != nil {
		t.Fatalf("write keep source: %v", err)
	}
	if err := os.WriteFile(excludedPath, []byte("package t\n\nfun skip() {}\n"), 0o644); err != nil {
		t.Fatalf("write excluded source: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{
		"compile",
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--exclude", filepath.FromSlash(filepath.Join("src", "main", "kotlin", "Skip.kt")),
		"--list-sources",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(compile --list-sources --exclude file) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, keepPath) {
		t.Fatalf("list-sources output missing kept source %q. output=%s", keepPath, output)
	}
	if strings.Contains(output, excludedPath) {
		t.Fatalf("list-sources output should exclude %q. output=%s", excludedPath, output)
	}
}

func TestRunCompileListSourcesWithSingleFileSource(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	fileA := filepath.Join(srcDir, "A.kt")
	fileB := filepath.Join(srcDir, "B.kt")
	if err := os.WriteFile(fileA, []byte("package t\n\nfun a() {}\n"), 0o644); err != nil {
		t.Fatalf("write source A: %v", err)
	}
	if err := os.WriteFile(fileB, []byte("package t\n\nfun b() {}\n"), 0o644); err != nil {
		t.Fatalf("write source B: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{
		"compile",
		"--repo", repoRoot,
		"--source", filepath.FromSlash(filepath.Join("src", "main", "kotlin", "B.kt")),
		"--list-sources",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(compile --source file --list-sources) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "sources=1") {
		t.Fatalf("expected exactly one source, got output: %s", output)
	}
	if !strings.Contains(output, fileB) {
		t.Fatalf("list-sources output missing selected file %q. output=%s", fileB, output)
	}
	if strings.Contains(output, fileA) {
		t.Fatalf("list-sources output should not include non-selected file %q. output=%s", fileA, output)
	}
}

func TestRunCompileDryRunUsesResolvedBuildArgs(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	libDir := filepath.Join(repoRoot, "lib")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("mkdir lib dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "compile.args"), []byte("--language-version=1.9\n"), 0o644); err != nil {
		t.Fatalf("write arg file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "ksp-plugin.jar"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "libs.jar"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write classpath file: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()
	defer func() {
		_ = rd.Close()
	}()

	exitCode := runCompile([]string{
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--classpath", filepath.FromSlash("lib/libs.jar"),
		"--plugin", filepath.FromSlash("ksp-plugin.jar"),
		"--arg-file", filepath.FromSlash("compile.args"),
		"--dry-run",
	})
	if exitCode != 0 {
		_ = w.Close()
		t.Fatalf("runCompile() = %d, want 0", exitCode)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}

	var out bytes.Buffer
	if _, err := io.Copy(&out, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}

	output := out.String()
	wantPlugin := filepath.Clean(filepath.Join(repoRoot, "ksp-plugin.jar"))
	wantClasspath := filepath.Clean(filepath.Join(repoRoot, "lib", "libs.jar"))

	if !strings.Contains(output, wantPlugin) {
		t.Fatalf("dry-run output missing resolved plugin path %q. output=%s", wantPlugin, output)
	}
	if !strings.Contains(output, wantClasspath) {
		t.Fatalf("dry-run output missing resolved classpath path %q. output=%s", wantClasspath, output)
	}
	if !strings.Contains(output, "--language-version=1.9") {
		t.Fatalf("dry-run output missing arg-file content. output=%s", output)
	}
}

func TestRunCompileDryRunDeduplicatesPluginAndClasspathInputs(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "plugin.jar"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}
	libDir := filepath.Join(repoRoot, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("mkdir lib dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "libs.jar"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write lib: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = rd.Close()
	}()

	exitCode := run([]string{
		"compile",
		"--repo", filepath.FromSlash(repoRoot),
		"--source", filepath.FromSlash(filepath.Join("src", "main", "kotlin")),
		"--plugin", filepath.FromSlash("plugin.jar"),
		"--plugin", filepath.FromSlash("plugin.jar"),
		"--plugin", filepath.FromSlash("plugins/../plugin.jar"),
		"--classpath", filepath.FromSlash(filepath.Join("lib", "libs.jar")),
		"--classpath", filepath.FromSlash(filepath.Join("lib", "libs.jar")),
		"--dry-run",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(compile --dry-run) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()

	plugin := filepath.Clean(filepath.Join(repoRoot, "plugin.jar"))
	classpath := filepath.Clean(filepath.Join(libDir, "libs.jar"))
	pluginToken := "\"-Xplugin=" + plugin + "\""
	classpathToken := "\"-cp\" \"" + classpath + "\""

	if got := strings.Count(output, pluginToken); got != 1 {
		t.Fatalf("expected plugin path once, got=%d output=%s", got, output)
	}
	if got := strings.Count(output, classpathToken); got != 1 {
		t.Fatalf("expected classpath token once, got=%d output=%s", got, output)
	}
	if !strings.Contains(output, plugin) {
		t.Fatalf("dry-run output missing plugin path %s", output)
	}
	if !strings.Contains(output, classpath) {
		t.Fatalf("dry-run output missing classpath path %s", output)
	}
}

func TestRunCompileDryRunResolvesAndJoinsClasspathEntries(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	libDir := filepath.Join(repoRoot, "lib")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("mkdir lib dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nfun f() {}\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "a.jar"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write classpath jar: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "b.jar"), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write classpath jar: %v", err)
	}

	classpath := filepath.Join("lib", "a.jar") + "," + filepath.Join("lib", "b.jar")
	expectedClasspath := filepath.Join(repoRoot, "lib", "a.jar") + string(filepath.ListSeparator) + filepath.Join(repoRoot, "lib", "b.jar")

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()
	defer func() {
		_ = rd.Close()
	}()

	exitCode := runCompile([]string{
		"--repo", repoRoot,
		"--source", filepath.FromSlash("src/main/kotlin"),
		"--classpath", classpath,
		"--classpath", filepath.FromSlash("lib/a.jar"),
		"--dry-run",
	})
	if exitCode != 0 {
		_ = w.Close()
		t.Fatalf("runCompile() = %d, want 0", exitCode)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	expected := `"-cp" "` + expectedClasspath + `"`
	if !strings.Contains(output, expected) {
		t.Fatalf("classpath output mismatch. expected=%q output=%s", expected, output)
	}
}

func TestRunSymbolsHandlesInvalidParserBackend(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "A.kt"), []byte("package t\n\nclass A\n"), 0o644); err != nil {
		t.Fatalf("write kt: %v", err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	exitCode := run([]string{
		"symbols",
		"--repo", repoRoot,
		"--parser-backend", "invalid",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 2 {
		t.Fatalf("run(symbols --parser-backend invalid) = %d, want 2", exitCode)
	}

	_ = rd.Close()
}

func TestLoadExternalIndices(t *testing.T) {
	t.Run("empty paths returns empty index", func(t *testing.T) {
		index, err := loadExternalIndices([]string{})
		if err != nil {
			t.Fatalf("loadExternalIndices([]) error = %v, want nil", err)
		}
		if len(index.Packages) != 0 || len(index.Symbols) != 0 {
			t.Fatalf("empty paths should return empty index, got packages=%d symbols=%d",
				len(index.Packages), len(index.Symbols))
		}
	})

	t.Run("non-existent file returns error", func(t *testing.T) {
		_, err := loadExternalIndices([]string{"/nonexistent/inventory.json"})
		if err == nil {
			t.Fatal("loadExternalIndices with non-existent file should return error")
		}
	})

	t.Run("valid inventory file loads successfully", func(t *testing.T) {
		// Create a temporary inventory file
		tmpDir := t.TempDir()
		inventoryPath := filepath.Join(tmpDir, "inventory.json")

		// Create a minimal valid inventory JSON using EmbeddableInventory format
		inventory := struct {
			JarPath  string                     `json:"jar_path"`
			Packages []kcg.InventoryPackage     `json:"packages"`
			Symbols  []string                   `json:"symbols"`
		}{
			JarPath: "test.jar",
			Packages: []kcg.InventoryPackage{
				{Package: "test.package"},
			},
			Symbols: []string{
				"test.package.TestSymbol",
			},
		}

		data, err := json.Marshal(inventory)
		if err != nil {
			t.Fatalf("marshal inventory: %v", err)
		}
		if err := os.WriteFile(inventoryPath, data, 0o644); err != nil {
			t.Fatalf("write inventory file: %v", err)
		}

		// Load the inventory
		loaded, err := loadExternalIndices([]string{inventoryPath})
		if err != nil {
			t.Fatalf("loadExternalIndices error = %v, want nil", err)
		}

		// Verify the loaded content
		if len(loaded.Packages) != 1 {
			t.Fatalf("expected 1 package, got %d", len(loaded.Packages))
		}
		if _, ok := loaded.Packages["test.package"]; !ok {
			t.Fatalf("expected package 'test.package' not found in %v", loaded.Packages)
		}
		if len(loaded.Symbols) != 1 {
			t.Fatalf("expected 1 symbol, got %d", len(loaded.Symbols))
		}
		if _, ok := loaded.Symbols["test.package.TestSymbol"]; !ok {
			t.Fatalf("expected symbol 'test.package.TestSymbol' not found in %v", loaded.Symbols)
		}
	})
}

func TestCreateCompiler(t *testing.T) {
	t.Run("creates compiler with default configuration", func(t *testing.T) {
		compiler := createCompiler(
			4,    // workers
			10,   // maxErrors
			true, // includeKTS
			false, // includeBuildScripts
			false, // lenient
			0,    // fileTimeout (0 means no timeout)
			kcg.ParseBackendANTLR,
		)

		if compiler == nil {
			t.Fatal("createCompiler returned nil")
		}
	})

	t.Run("creates compiler with all options enabled", func(t *testing.T) {
		compiler := createCompiler(
			8,     // workers
			100,   // maxErrors
			true,  // includeKTS
			true,  // includeBuildScripts
			true,  // lenient
			1000000, // fileTimeout (1ms)
			kcg.ParseBackendEmbeddable,
		)

		if compiler == nil {
			t.Fatal("createCompiler returned nil")
		}
	})

	t.Run("creates compiler with embeddable backend", func(t *testing.T) {
		compiler := createCompiler(
			1,    // workers
			1,    // maxErrors
			false, // includeKTS
			false, // includeBuildScripts
			false, // lenient
			0,    // fileTimeout
			kcg.ParseBackendEmbeddable,
		)

		if compiler == nil {
			t.Fatal("createCompiler returned nil")
		}
	})
}
