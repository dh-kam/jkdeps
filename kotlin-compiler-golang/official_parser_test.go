package kotlincompilergolang

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestParseJarLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	existingJar := filepath.Join(dir, "existing.jar")
	existingText := filepath.Join(dir, "ignore.txt")
	if err := os.WriteFile(existingJar, []byte("jar"), 0o644); err != nil {
		t.Fatalf("write existing jar: %v", err)
	}
	if err := os.WriteFile(existingText, []byte("txt"), 0o644); err != nil {
		t.Fatalf("write existing txt: %v", err)
	}

	out := parseJarLines(strings.Join([]string{
		"",
		"   ",
		existingText,
		existingJar,
		filepath.Join(dir, "missing.jar"),
		"--comment",
		filepath.Join(dir, "UPPER.JAR"),
	}, "\n"))

	if len(out) != 1 {
		t.Fatalf("parseJarLines() len=%d want=1 got=%v", len(out), out)
	}
	if out[0] != existingJar {
		t.Fatalf("parseJarLines()[0]=%q want=%q", out[0], existingJar)
	}
}

func TestOfficialShouldIncludeFile(t *testing.T) {
	t.Parallel()

	repoDir := filepath.Join("test", "repo", "src")
	cases := []struct {
		name                string
		path                string
		includeKTS          bool
		includeBuildScripts bool
		want                bool
	}{
		{
			name: "kotlin source always included",
			path: filepath.Join(repoDir, "Sample.kt"),
			want: true,
		},
		{
			name: "kts ignored when disabled",
			path: filepath.Join(repoDir, "sample.kts"),
			want: false,
		},
		{
			name:       "kts included when enabled",
			path:       filepath.Join(repoDir, "sample.kts"),
			includeKTS: true,
			want:       true,
		},
		{
			name:       "build gradle kts excluded by default",
			path:       filepath.Join(repoDir, "build.gradle.kts"),
			includeKTS: true,
			want:       false,
		},
		{
			name:                "build gradle kts included when includeBuildScripts enabled",
			path:                filepath.Join(repoDir, "build.gradle.kts"),
			includeKTS:          false,
			includeBuildScripts: true,
			want:                true,
		},
		{
			name:                "regular kts still excluded when only includeBuildScripts enabled",
			path:                filepath.Join(repoDir, "sample.kts"),
			includeKTS:          false,
			includeBuildScripts: true,
			want:                false,
		},
		{
			name:                "non kotlin extension excluded",
			path:                filepath.Join(repoDir, "README.md"),
			includeKTS:          true,
			includeBuildScripts: true,
			want:                false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := officialShouldIncludeFile(tc.path, tc.includeKTS, tc.includeBuildScripts)
			if got != tc.want {
				t.Fatalf("officialShouldIncludeFile(%q, includeKTS=%v, includeBuildScripts=%v) = %v want %v", tc.path, tc.includeKTS, tc.includeBuildScripts, got, tc.want)
			}
		})
	}
}

func TestOfficialDiagnostics(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		errorCount int
		messages   []string
		want       int
		wantFirst  string
		wantLast   string
	}{
		{
			name:       "negative or zero returns empty",
			errorCount: 0,
			messages:   []string{"should be ignored"},
			want:       0,
		},
		{
			name:       "pads to requested error count",
			errorCount: 3,
			messages:   []string{"  ", "actual issue"},
			want:       3,
			wantFirst:  "kotlin parse error",
			wantLast:   "kotlin parse error",
		},
		{
			name:       "keeps provided diagnostics and trims",
			errorCount: 1,
			messages:   []string{"  trailing space   "},
			want:       1,
			wantFirst:  "trailing space",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := officialDiagnostics(tc.errorCount, tc.messages)
			if len(got) != tc.want {
				t.Fatalf("officialDiagnostics(%d, ... ) len=%d want=%d", tc.errorCount, len(got), tc.want)
			}
			if tc.want == 0 {
				return
			}
			if got[0].Message != tc.wantFirst {
				t.Fatalf("officialDiagnostics() first message=%q want=%q", got[0].Message, tc.wantFirst)
			}
			if tc.want >= 2 && got[len(got)-1].Message != tc.wantLast {
				t.Fatalf("officialDiagnostics() last message=%q want=%q", got[len(got)-1].Message, tc.wantLast)
			}
			for i := range got {
				if got[i].Severity != SeverityError {
					t.Fatalf("officialDiagnostics()[%d].Severity=%q want=%q", i, got[i].Severity, SeverityError)
				}
			}
		})
	}
}

func TestOfficialDeclarationsToTopLevel(t *testing.T) {
	t.Parallel()

	got := officialDeclarationsToTopLevel([]officialSnapshotDecl{
		{Kind: "interface", Name: "Zeta"},
		{Kind: "class", Name: "Beta"},
		{Kind: "invalid", Name: "Skip"},
		{Kind: "class", Name: "Alpha"},
		{Kind: "typealias", Name: "T"},
		{Kind: "object", Name: ""},
		{Kind: "function", Name: "main"},
		{Kind: "property", Name: "value"},
		{Kind: "object", Name: "Omega"},
	})

	want := []TopLevelDeclaration{
		{Kind: DeclClass, Name: "Alpha"},
		{Kind: DeclClass, Name: "Beta"},
		{Kind: DeclFunction, Name: "main"},
		{Kind: DeclInterface, Name: "Zeta"},
		{Kind: DeclObject, Name: "Omega"},
		{Kind: DeclProperty, Name: "value"},
		{Kind: DeclTypeAlias, Name: "T"},
	}
	if len(got) != len(want) {
		t.Fatalf("officialDeclarationsToTopLevel() len=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i].Kind != want[i].Kind || got[i].Name != want[i].Name {
			t.Fatalf("officialDeclarationsToTopLevel()[%d]=%+v want=%+v", i, got[i], want[i])
		}
	}
}

func TestMaxInt(t *testing.T) {
	t.Parallel()

	if got := maxInt(10, 2); got != 10 {
		t.Fatalf("maxInt(10, 2) = %d want 10", got)
	}
	if got := maxInt(-1, 0); got != 0 {
		t.Fatalf("maxInt(-1, 0) = %d want 0", got)
	}
	if got := maxInt(7, 7); got != 7 {
		t.Fatalf("maxInt(7, 7) = %d want 7", got)
	}
}

func TestCompileOfficialSnapshotClassesMissingSourceReturnsError(t *testing.T) {
	toolDir := t.TempDir()
	_, err := compileOfficialSnapshotClasses([]string{}, toolDir)
	if err == nil {
		t.Fatalf("compileOfficialSnapshotClasses(%q) expected error", toolDir)
	}
	if !strings.Contains(err.Error(), "official snapshot source missing") {
		t.Fatalf("compileOfficialSnapshotClasses() error = %v, want contain official snapshot source missing", err)
	}
}

func TestCompileOfficialSnapshotClassesSkipsCompileWhenClassUpToDate(t *testing.T) {
	toolDir := t.TempDir()
	sourceRoot := filepath.Join(toolDir, "kotlin-official-parity")
	classDir := filepath.Join(toolDir, ".kcg-official-classes")
	sourcePath := filepath.Join(sourceRoot, "KotlinOfficialSnapshot.java")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source root: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("class KotlinOfficialSnapshot {}"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	classFile := filepath.Join(classDir, "KotlinOfficialSnapshot.class")
	if err := os.MkdirAll(classDir, 0o755); err != nil {
		t.Fatalf("mkdir class dir: %v", err)
	}
	if err := os.WriteFile(classFile, []byte("class"), 0o644); err != nil {
		t.Fatalf("write class file: %v", err)
	}

	now := time.Now()
	if err := os.Chtimes(sourcePath, now, now); err != nil {
		t.Fatalf("chtimes source: %v", err)
	}
	if err := os.Chtimes(classFile, now, now.Add(time.Second)); err != nil {
		t.Fatalf("chtimes class: %v", err)
	}

	got, err := compileOfficialSnapshotClasses([]string{}, toolDir)
	if err != nil {
		t.Fatalf("compileOfficialSnapshotClasses() unexpected error: %v", err)
	}
	if got != classDir {
		t.Fatalf("compileOfficialSnapshotClasses() = %q want %q", got, classDir)
	}
}

func TestCompileOfficialSnapshotClassesReturnsCompileErrorWhenCompileFails(t *testing.T) {
	toolDir := t.TempDir()
	sourceRoot := filepath.Join(toolDir, "kotlin-official-parity")
	sourcePath := filepath.Join(sourceRoot, "KotlinOfficialSnapshot.java")
	classDir := filepath.Join(toolDir, ".kcg-official-classes")
	classFile := filepath.Join(classDir, "KotlinOfficialSnapshot.class")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source root: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("class KotlinOfficialSnapshot { syntax-error"), 0o644); err != nil {
		t.Fatalf("write invalid source: %v", err)
	}
	if err := os.MkdirAll(classDir, 0o755); err != nil {
		t.Fatalf("mkdir class dir: %v", err)
	}
	if err := os.WriteFile(classFile, []byte("old"), 0o644); err != nil {
		t.Fatalf("write old class file: %v", err)
	}
	now := time.Now()
	if err := os.Chtimes(sourcePath, now, now); err != nil {
		t.Fatalf("chtimes source: %v", err)
	}
	if err := os.Chtimes(classFile, now.Add(-2*time.Second), now.Add(-2*time.Second)); err != nil {
		t.Fatalf("chtimes class: %v", err)
	}

	_, err := compileOfficialSnapshotClasses([]string{}, toolDir)
	if err == nil {
		t.Fatalf("compileOfficialSnapshotClasses() expected compile error")
	}
	if !strings.Contains(err.Error(), "compile official snapshot") {
		t.Fatalf("compileOfficialSnapshotClasses() error = %v, want compile official snapshot", err)
	}
}

func TestLoadOfficialSnapshotInvalidJSONReturnsError(t *testing.T) {
	repoRoot := t.TempDir()
	toolsDir := t.TempDir()
	scriptsDir := filepath.Clean(filepath.Join(toolsDir, "..", "scripts"))
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}

	embeddableJar := filepath.Join(toolsDir, "kotlin-compiler-embeddable.jar")
	runtimeJar := filepath.Join(toolsDir, "kotlin-stdlib.jar")
	if err := os.WriteFile(embeddableJar, []byte(""), 0o644); err != nil {
		t.Fatalf("write embeddable jar: %v", err)
	}
	if err := os.WriteFile(runtimeJar, []byte(""), 0o644); err != nil {
		t.Fatalf("write runtime jar: %v", err)
	}

	fetchEmbeddable := "#!/bin/sh\nprintf '%s\n' " + strconv.Quote(embeddableJar) + "\n"
	fetchRuntime := "#!/bin/sh\nprintf '%s\n' " + strconv.Quote(runtimeJar) + "\n"
	embeddableScript := filepath.Join(scriptsDir, "fetch_kotlin_compiler_embeddable.sh")
	runtimeScript := filepath.Join(scriptsDir, "fetch_kotlin_compiler_runtime_jars.sh")
	if err := os.WriteFile(embeddableScript, []byte(fetchEmbeddable), 0o755); err != nil {
		t.Fatalf("write embeddable script: %v", err)
	}
	if err := os.WriteFile(runtimeScript, []byte(fetchRuntime), 0o755); err != nil {
		t.Fatalf("write runtime script: %v", err)
	}

	sourceRoot := filepath.Join(toolsDir, "kotlin-official-parity")
	sourcePath := filepath.Join(sourceRoot, "KotlinOfficialSnapshot.java")
	classDir := filepath.Join(toolsDir, ".kcg-official-classes")
	classFile := filepath.Join(classDir, "KotlinOfficialSnapshot.class")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source root: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("class KotlinOfficialSnapshot {}"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.MkdirAll(classDir, 0o755); err != nil {
		t.Fatalf("mkdir class dir: %v", err)
	}
	if err := os.WriteFile(classFile, []byte("class"), 0o644); err != nil {
		t.Fatalf("write class file: %v", err)
	}
	now := time.Now()
	if err := os.Chtimes(sourcePath, now, now); err != nil {
		t.Fatalf("chtimes source: %v", err)
	}
	if err := os.Chtimes(classFile, now.Add(time.Second), now.Add(time.Second)); err != nil {
		t.Fatalf("chtimes class: %v", err)
	}

	javaScriptDir := filepath.Join(toolsDir, "bin")
	if err := os.MkdirAll(javaScriptDir, 0o755); err != nil {
		t.Fatalf("mkdir java script dir: %v", err)
	}
	javaScript := filepath.Join(javaScriptDir, "java")
	script := "#!/bin/sh\n\nout=\"\"\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = \"--out\" ]; then\n    shift\n    out=$1\n    break\n  fi\n  shift\ndone\nif [ -z \"$out\" ]; then\n  echo \"missing --out\" >&2\n  exit 2\nfi\nprintf 'not-json' > \"$out\"\n"
	if err := os.WriteFile(javaScript, []byte(script), 0o755); err != nil {
		t.Fatalf("write java script: %v", err)
	}

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", javaScriptDir+string(filepath.ListSeparator)+origPath)
	t.Setenv("JKTDEPS_TOOLS_DIR", toolsDir)

	comp := &Compiler{}
	_, err := comp.loadOfficialSnapshot(repoRoot)
	if err == nil {
		t.Fatalf("loadOfficialSnapshot() expected error")
	}
	if !strings.Contains(err.Error(), "parse official snapshot JSON") {
		t.Fatalf("loadOfficialSnapshot() error = %v, want parse official snapshot JSON", err)
	}
}

func TestLoadOfficialSnapshotReturnsParsedSnapshot(t *testing.T) {
	repoRoot := t.TempDir()
	toolsDir := t.TempDir()
	scriptsDir := filepath.Clean(filepath.Join(toolsDir, "..", "scripts"))
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}

	embeddableJar := filepath.Join(toolsDir, "kotlin-compiler-embeddable.jar")
	runtimeJar := filepath.Join(toolsDir, "kotlin-stdlib.jar")
	if err := os.WriteFile(embeddableJar, []byte(""), 0o644); err != nil {
		t.Fatalf("write embeddable jar: %v", err)
	}
	if err := os.WriteFile(runtimeJar, []byte(""), 0o644); err != nil {
		t.Fatalf("write runtime jar: %v", err)
	}

	fetchEmbeddable := "#!/bin/sh\nprintf '%s\n' " + strconv.Quote(embeddableJar) + "\n"
	fetchRuntime := "#!/bin/sh\nprintf '%s\n' " + strconv.Quote(runtimeJar) + "\n"
	if err := os.WriteFile(filepath.Join(scriptsDir, "fetch_kotlin_compiler_embeddable.sh"), []byte(fetchEmbeddable), 0o755); err != nil {
		t.Fatalf("write embeddable fetch script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "fetch_kotlin_compiler_runtime_jars.sh"), []byte(fetchRuntime), 0o755); err != nil {
		t.Fatalf("write runtime fetch script: %v", err)
	}

	sourceRoot := filepath.Join(toolsDir, "kotlin-official-parity")
	sourcePath := filepath.Join(sourceRoot, "KotlinOfficialSnapshot.java")
	classDir := filepath.Join(toolsDir, ".kcg-official-classes")
	classFile := filepath.Join(classDir, "KotlinOfficialSnapshot.class")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source root: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("class KotlinOfficialSnapshot {}"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.MkdirAll(classDir, 0o755); err != nil {
		t.Fatalf("mkdir class dir: %v", err)
	}
	if err := os.WriteFile(classFile, []byte("class"), 0o644); err != nil {
		t.Fatalf("write class file: %v", err)
	}
	now := time.Now()
	if err := os.Chtimes(sourcePath, now, now); err != nil {
		t.Fatalf("chtimes source: %v", err)
	}
	if err := os.Chtimes(classFile, now.Add(time.Second), now.Add(time.Second)); err != nil {
		t.Fatalf("chtimes class: %v", err)
	}

	javaScriptDir := filepath.Join(toolsDir, "bin")
	if err := os.MkdirAll(javaScriptDir, 0o755); err != nil {
		t.Fatalf("mkdir java script dir: %v", err)
	}
	javaScript := filepath.Join(javaScriptDir, "java")
	javaScriptBody := `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--out" ]; then
    shift
    out="$1"
    break
  fi
  shift
done
if [ -z "$out" ]; then
  echo "missing --out" >&2
  exit 2
fi
cat > "$out" <<JSON
{"files":[
  {"path":"src/main/kotlin/Main.kt","package_name":"demo","imports":["  kotlin.String ","kotlin.collections.List"],"declarations":[{"kind":"class","name":"Main"},{"kind":"kts","name":"Ignored"},{"kind":"function","name":"run"}],"error_count":1,"errors":["   ","oops"]},
  {"path":"src/main/kotlin/app.kts","package_name":"demo","imports":["kotlin.io.*"],"declarations":[{"kind":"function","name":"entry"}],"error_count":0,"errors":[]}
]}
JSON
`
	if err := os.WriteFile(javaScript, []byte(javaScriptBody), 0o755); err != nil {
		t.Fatalf("write java script: %v", err)
	}

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", javaScriptDir+string(filepath.ListSeparator)+origPath)
	t.Setenv("JKTDEPS_TOOLS_DIR", toolsDir)

	comp := &Compiler{}
	snapshot, err := comp.loadOfficialSnapshot(repoRoot)
	if err != nil {
		t.Fatalf("loadOfficialSnapshot() unexpected error: %v", err)
	}
	if len(snapshot.Files) != 2 {
		t.Fatalf("loadOfficialSnapshot() len=%d want=2", len(snapshot.Files))
	}
	if snapshot.Files[0].Path != "src/main/kotlin/Main.kt" {
		t.Fatalf("snapshot.Files[0].Path=%q", snapshot.Files[0].Path)
	}
	if snapshot.Files[0].ErrorCount != 1 || snapshot.Files[0].Errors[1] != "oops" {
		t.Fatalf("snapshot file diagnostics mismatch: %+v", snapshot.Files[0])
	}
}

func TestParseRepositoryWithEmbeddableAppliesFiltersAndCounts(t *testing.T) {
	repoRoot := t.TempDir()
	toolsDir := t.TempDir()
	scriptsDir := filepath.Clean(filepath.Join(toolsDir, "..", "scripts"))
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}

	embeddableJar := filepath.Join(toolsDir, "kotlin-compiler-embeddable.jar")
	runtimeJar := filepath.Join(toolsDir, "kotlin-stdlib.jar")
	if err := os.WriteFile(embeddableJar, []byte(""), 0o644); err != nil {
		t.Fatalf("write embeddable jar: %v", err)
	}
	if err := os.WriteFile(runtimeJar, []byte(""), 0o644); err != nil {
		t.Fatalf("write runtime jar: %v", err)
	}
	fetchEmbeddable := "#!/bin/sh\nprintf '%s\n' " + strconv.Quote(embeddableJar) + "\n"
	fetchRuntime := "#!/bin/sh\nprintf '%s\n' " + strconv.Quote(runtimeJar) + "\n"
	if err := os.WriteFile(filepath.Join(scriptsDir, "fetch_kotlin_compiler_embeddable.sh"), []byte(fetchEmbeddable), 0o755); err != nil {
		t.Fatalf("write embeddable fetch script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "fetch_kotlin_compiler_runtime_jars.sh"), []byte(fetchRuntime), 0o755); err != nil {
		t.Fatalf("write runtime fetch script: %v", err)
	}

	sourceRoot := filepath.Join(toolsDir, "kotlin-official-parity")
	sourcePath := filepath.Join(sourceRoot, "KotlinOfficialSnapshot.java")
	classDir := filepath.Join(toolsDir, ".kcg-official-classes")
	classFile := filepath.Join(classDir, "KotlinOfficialSnapshot.class")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source root: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("class KotlinOfficialSnapshot {}"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.MkdirAll(classDir, 0o755); err != nil {
		t.Fatalf("mkdir class dir: %v", err)
	}
	if err := os.WriteFile(classFile, []byte("class"), 0o644); err != nil {
		t.Fatalf("write class file: %v", err)
	}
	now := time.Now()
	if err := os.Chtimes(sourcePath, now, now); err != nil {
		t.Fatalf("chtimes source: %v", err)
	}
	if err := os.Chtimes(classFile, now.Add(time.Second), now.Add(time.Second)); err != nil {
		t.Fatalf("chtimes class: %v", err)
	}

	javaScriptDir := filepath.Join(toolsDir, "bin")
	if err := os.MkdirAll(javaScriptDir, 0o755); err != nil {
		t.Fatalf("mkdir java script dir: %v", err)
	}
	javaScript := filepath.Join(javaScriptDir, "java")
	javaScriptBody := `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--out" ]; then
    shift
    out="$1"
    break
  fi
  shift
done
if [ -z "$out" ]; then
  echo "missing --out" >&2
  exit 2
fi
cat > "$out" <<JSON
{"files":[
  {"path":"src/main/kotlin/Main.kt","package_name":"demo","imports":["kotlin.String","kotlin.collections.List"],"declarations":[{"kind":"class","name":"Main"},{"kind":"invalid","name":"X"}],"error_count":0,"errors":[]},
  {"path":"src/main/kotlin/Script.kts","package_name":"demo","imports":[],"declarations":[{"kind":"function","name":"scriptEntry"}],"error_count":0,"errors":[]},
  {"path":"build.gradle.kts","package_name":"build","imports":[],"declarations":[{"kind":"class","name":"BuildScript"}],"error_count":0,"errors":[]},
  {"path":"src/main/kotlin/Broken.kt","package_name":"demo","imports":[],"declarations":[{"kind":"function","name":"b"}],"error_count":2,"errors":["a","b"]}
]}
JSON
`
	if err := os.WriteFile(javaScript, []byte(javaScriptBody), 0o755); err != nil {
		t.Fatalf("write java script: %v", err)
	}

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", javaScriptDir+string(filepath.ListSeparator)+origPath)
	t.Setenv("JKTDEPS_TOOLS_DIR", toolsDir)

	comp := New(Config{
		Workers:          1,
		MaxErrorsPerFile: 10,
		IncludeKTS:       false,
		LenientSyntax:    false,
	})
	comp.config.ParseBackend = ParseBackendEmbeddable
	result, err := comp.ParseRepository(repoRoot)
	if err != nil {
		t.Fatalf("ParseRepository(embeddable) unexpected error: %v", err)
	}

	if result.Root != repoRoot {
		t.Fatalf("ParseRepository().Root=%q want %q", result.Root, repoRoot)
	}
	if result.TotalFiles != 2 {
		t.Fatalf("ParseRepository() total files=%d want=2", result.TotalFiles)
	}
	if result.ParsedFiles != 1 {
		t.Fatalf("ParseRepository() parsed=%d want=1", result.ParsedFiles)
	}
	if result.FailedFiles != 1 {
		t.Fatalf("ParseRepository() failed=%d want=1", result.FailedFiles)
	}
	if len(result.Files) != 2 {
		t.Fatalf("ParseRepository() file count=%d want=2", len(result.Files))
	}

	filesByPath := map[string]FileUnit{}
	for _, file := range result.Files {
		filesByPath[file.Path] = file
	}

	mainPath := filepath.Clean(filepath.Join(repoRoot, "src/main/kotlin/Main.kt"))
	brokenPath := filepath.Clean(filepath.Join(repoRoot, "src/main/kotlin/Broken.kt"))
	mainFile, ok := filesByPath[mainPath]
	if !ok {
		t.Fatalf("ParseRepository() missing main file at %q in %+v", mainPath, result.Files)
	}
	brokenFile, ok := filesByPath[brokenPath]
	if !ok {
		t.Fatalf("ParseRepository() missing broken file at %q in %+v", brokenPath, result.Files)
	}

	if !mainFile.Parsed || len(mainFile.Declarations) != 1 {
		t.Fatalf("main file invalid: %+v", mainFile)
	}
	if mainFile.Declarations[0].Kind != DeclClass || mainFile.Declarations[0].Name != "Main" {
		t.Fatalf("main file declaration=%+v", mainFile.Declarations)
	}
	if len(mainFile.Imports) != 2 || mainFile.Imports[0] != "kotlin.String" || mainFile.Imports[1] != "kotlin.collections.List" {
		t.Fatalf("main file imports=%v", mainFile.Imports)
	}
	if brokenFile.Parsed {
		t.Fatalf("broken file should be unparsed: %+v", brokenFile)
	}
}

func TestResolveEmbeddableJar(t *testing.T) {
	toolsDir := t.TempDir()
	scriptsDir := filepath.Clean(filepath.Join(toolsDir, "..", "scripts"))
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}

	validJar := filepath.Join(toolsDir, "kotlin-compiler-embeddable.jar")
	invalidJar := filepath.Join(toolsDir, "not-a-jar.txt")
	if err := os.WriteFile(validJar, []byte(""), 0o644); err != nil {
		t.Fatalf("write valid jar: %v", err)
	}
	if err := os.WriteFile(invalidJar, []byte(""), 0o644); err != nil {
		t.Fatalf("write invalid extension file: %v", err)
	}

	script := "#!/bin/sh\nprintf '%s\n' " + strconv.Quote(invalidJar) + "\n" + "printf '%s\n' " + strconv.Quote(validJar) + "\n"
	if err := os.WriteFile(filepath.Join(scriptsDir, "fetch_kotlin_compiler_embeddable.sh"), []byte(script), 0o755); err != nil {
		t.Fatalf("write embeddable script: %v", err)
	}

	got, err := resolveEmbeddableJar(toolsDir)
	if err != nil {
		t.Fatalf("resolveEmbeddableJar() unexpected error: %v", err)
	}
	if got != validJar {
		t.Fatalf("resolveEmbeddableJar() = %q want %q", got, validJar)
	}
}

func TestResolveRuntimeJars(t *testing.T) {
	toolsDir := t.TempDir()
	scriptsDir := filepath.Clean(filepath.Join(toolsDir, "..", "scripts"))
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}

	jarOne := filepath.Join(toolsDir, "a.jar")
	jarTwo := filepath.Join(toolsDir, "b.jar")
	missingJar := filepath.Join(toolsDir, "missing.jar")
	nonJar := filepath.Join(toolsDir, "c.txt")
	if err := os.WriteFile(jarOne, []byte(""), 0o644); err != nil {
		t.Fatalf("write jar one: %v", err)
	}
	if err := os.WriteFile(jarTwo, []byte(""), 0o644); err != nil {
		t.Fatalf("write jar two: %v", err)
	}
	if err := os.WriteFile(nonJar, []byte(""), 0o644); err != nil {
		t.Fatalf("write non-jar file: %v", err)
	}

	script := "#!/bin/sh\nprintf '%s\n' " + strconv.Quote(jarOne) + "\n" + "printf '%s\n' " + strconv.Quote(nonJar) + "\n" + "printf '%s\n' " + strconv.Quote(missingJar) + "\n" + "printf '%s\n' " + strconv.Quote(jarTwo) + "\n"
	if err := os.WriteFile(filepath.Join(scriptsDir, "fetch_kotlin_compiler_runtime_jars.sh"), []byte(script), 0o755); err != nil {
		t.Fatalf("write runtime script: %v", err)
	}

	got, err := resolveRuntimeJars(toolsDir)
	if err != nil {
		t.Fatalf("resolveRuntimeJars() unexpected error: %v", err)
	}
	want := []string{jarOne, jarTwo}
	if len(got) != len(want) {
		t.Fatalf("resolveRuntimeJars() len=%d want=%d got=%v", len(got), len(want), got)
	}
	if got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("resolveRuntimeJars()=%v want=%v", got, want)
	}
}

func TestResolveEmbeddableJarReturnsErrorWhenNoJarsFound(t *testing.T) {
	toolsDir := t.TempDir()
	scriptsDir := filepath.Clean(filepath.Join(toolsDir, "..", "scripts"))
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}
	script := "#!/bin/sh\nprintf '%s\n' /tmp/not-a-jar.txt\nprintf '%s\n' /tmp/missing.jar\n"
	if err := os.WriteFile(filepath.Join(scriptsDir, "fetch_kotlin_compiler_embeddable.sh"), []byte(script), 0o755); err != nil {
		t.Fatalf("write embeddable script: %v", err)
	}

	if _, err := resolveEmbeddableJar(toolsDir); err == nil {
		t.Fatalf("resolveEmbeddableJar() expected error")
	} else if !strings.Contains(err.Error(), "could not resolve kotlin-compiler-embeddable") {
		t.Fatalf("resolveEmbeddableJar() error = %v", err)
	}
}

func TestResolveRuntimeJarsReturnsErrorWhenNoJarsFound(t *testing.T) {
	toolsDir := t.TempDir()
	scriptsDir := filepath.Clean(filepath.Join(toolsDir, "..", "scripts"))
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}
	script := "#!/bin/sh\nprintf '%s\n' /tmp/not-a-jar.txt\nprintf '%s\n' /tmp/missing.jar\n"
	if err := os.WriteFile(filepath.Join(scriptsDir, "fetch_kotlin_compiler_runtime_jars.sh"), []byte(script), 0o755); err != nil {
		t.Fatalf("write runtime script: %v", err)
	}

	if _, err := resolveRuntimeJars(toolsDir); err == nil {
		t.Fatalf("resolveRuntimeJars() expected error")
	} else if !strings.Contains(err.Error(), "could not resolve kotlin runtime jars") {
		t.Fatalf("resolveRuntimeJars() error = %v", err)
	}
}

func TestRunCommand(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		if err := runCommand("echo", "ok"); err != nil {
			t.Fatalf("runCommand(echo ok) unexpected error: %v", err)
		}
	})

	t.Run("failure propagatesWrappedError", func(t *testing.T) {
		if err := runCommand("false"); err == nil {
			t.Fatalf("runCommand(false) expected error")
		} else if !strings.Contains(err.Error(), "command \"false\" failed") {
			t.Fatalf("runCommand(false) error mismatch: %v", err)
		}
	})
}

func TestRunCommandOutput(t *testing.T) {
	got, err := runCommandOutput("echo", "hello")
	if err != nil {
		t.Fatalf("runCommandOutput(echo hello) unexpected error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("runCommandOutput(echo hello) = %q want %q", got, "hello")
	}

	t.Run("failure returnsWrappedError", func(t *testing.T) {
		if _, err := runCommandOutput("false"); err == nil {
			t.Fatalf("runCommandOutput(false) expected error")
		} else if !strings.Contains(err.Error(), "command \"false\" failed") {
			t.Fatalf("runCommandOutput(false) error mismatch: %v", err)
		}
	})
}

func TestEmbeddableToolsDir(t *testing.T) {
	toolsDir := filepath.Join(t.TempDir(), "tools")
	t.Setenv("JKTDEPS_TOOLS_DIR", "  "+toolsDir+"  ")

	c := &Compiler{}
	got, err := c.embeddableToolsDir()
	if err != nil {
		t.Fatalf("embeddableToolsDir() unexpected error: %v", err)
	}
	if got != toolsDir {
		t.Fatalf("embeddableToolsDir() = %q want %q", got, toolsDir)
	}
}

func TestFindRepoRootFromCWD(t *testing.T) {
	repoRoot := t.TempDir()
	moduleRoot := filepath.Join(repoRoot, "module")
	subDir := filepath.Join(moduleRoot, "a", "b")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir sub dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("module github.com/dh-kam/jkdeps\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("chdir to subdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	got, err := findRepoRootFromCWD()
	if err != nil {
		t.Fatalf("findRepoRootFromCWD() unexpected error: %v", err)
	}
	if got != moduleRoot {
		t.Fatalf("findRepoRootFromCWD() = %q want %q", got, moduleRoot)
	}
}

func TestFindRepoRootFromCWDReturnsErrorWhenNotInModule(t *testing.T) {
	repoRoot := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("chdir to temp root: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if _, err := findRepoRootFromCWD(); err == nil {
		t.Fatalf("findRepoRootFromCWD() expected error")
	}
}
