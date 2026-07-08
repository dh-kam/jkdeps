package kotlincompilergolang

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAppliesDefaultsOnce(t *testing.T) {
	compiler := New(Config{})

	if compiler.config.Workers != 1 {
		t.Fatalf("New(Config{}).config.Workers = %d, want 1", compiler.config.Workers)
	}
	if compiler.config.MaxErrorsPerFile != 10 {
		t.Fatalf("New(Config{}).config.MaxErrorsPerFile = %d, want 10", compiler.config.MaxErrorsPerFile)
	}
	if compiler.config.ParseBackend != ParseBackendDefault {
		t.Fatalf("New(Config{}).config.ParseBackend = %q, want %q", compiler.config.ParseBackend, ParseBackendDefault)
	}
}

func TestRepositoryWorkerCount(t *testing.T) {
	compiler := New(Config{Workers: 4})

	tests := []struct {
		files int
		want  int
	}{
		{files: 0, want: 0},
		{files: 1, want: 1},
		{files: 2, want: 2},
		{files: 10, want: 4},
	}

	for _, tc := range tests {
		if got := compiler.repositoryWorkerCount(tc.files); got != tc.want {
			t.Fatalf("repositoryWorkerCount(%d) = %d, want %d", tc.files, got, tc.want)
		}
	}
}

func TestMergeTopLevelDeclarations(t *testing.T) {
	left := []TopLevelDeclaration{
		{Kind: DeclClass, Name: "App", Line: 1},
		{Kind: DeclFunction, Name: "run", Line: 2},
	}
	right := []TopLevelDeclaration{
		{Kind: DeclFunction, Name: "run", Line: 20},
		{Kind: DeclObject, Name: "Holder", Line: 3},
	}

	got := mergeTopLevelDeclarations(left, right)
	if len(got) != 3 {
		t.Fatalf("mergeTopLevelDeclarations() len = %d, want 3", len(got))
	}
	if got[0].Kind != left[0].Kind || got[0].Name != left[0].Name || got[1].Kind != left[1].Kind || got[1].Name != left[1].Name {
		t.Fatalf("mergeTopLevelDeclarations() should preserve left declarations first: %+v", got)
	}
	if got[2].Kind != DeclObject || got[2].Name != "Holder" {
		t.Fatalf("mergeTopLevelDeclarations() last = %+v, want Holder object", got[2])
	}
}

func TestMergeTopLevelDeclarationsHandlesEmptyInputs(t *testing.T) {
	right := []TopLevelDeclaration{{Kind: DeclClass, Name: "OnlyRight", Line: 1}}

	if got := mergeTopLevelDeclarations(nil, nil); got != nil {
		t.Fatalf("mergeTopLevelDeclarations(nil, nil) = %+v, want nil", got)
	}
	got := mergeTopLevelDeclarations(nil, right)
	if len(got) != 1 || got[0].Kind != right[0].Kind || got[0].Name != right[0].Name {
		t.Fatalf("mergeTopLevelDeclarations(nil, right) = %+v, want copy of right", got)
	}
	got[0].Name = "Changed"
	if right[0].Name != "OnlyRight" {
		t.Fatalf("mergeTopLevelDeclarations(nil, right) should copy right slice, right=%+v", right)
	}
	left := []TopLevelDeclaration{{Kind: DeclFunction, Name: "left", Line: 2}}
	got = mergeTopLevelDeclarations(left, nil)
	if len(got) != 1 || got[0].Kind != left[0].Kind || got[0].Name != left[0].Name {
		t.Fatalf("mergeTopLevelDeclarations(left, nil) = %+v, want left", got)
	}
}

func TestSyntaxErrorListenerAddPanicAndDiagnostics(t *testing.T) {
	listener := newSyntaxErrorListener("/tmp/Broken.kt", 2)
	listener.addPanic(os.ErrInvalid)
	listener.addPanic(os.ErrPermission)
	listener.addPanic(os.ErrNotExist)

	got := listener.Diagnostics()
	if len(got) != 2 {
		t.Fatalf("Diagnostics() len = %d, want 2", len(got))
	}
	if got[0].Severity != SeverityError || got[0].Path != "/tmp/Broken.kt" {
		t.Fatalf("Diagnostics()[0] = %+v, want error severity and path", got[0])
	}
	got[0].Message = "changed"
	if listener.Diagnostics()[0].Message != os.ErrInvalid.Error() {
		t.Fatal("Diagnostics() should return a copy")
	}
}

func TestParseRepositoryUnitReadError(t *testing.T) {
	repoRoot := t.TempDir()
	brokenPath := filepath.Join(repoRoot, "Broken.kt")
	if err := os.Symlink(filepath.Join(repoRoot, "missing", "Broken.kt"), brokenPath); err != nil {
		t.Fatalf("symlink broken kotlin file: %v", err)
	}

	compiler := New(Config{Workers: 1})
	unit := compiler.parseRepositoryUnit(brokenPath)
	if unit.Parsed {
		t.Fatal("parseRepositoryUnit() broken file should be marked failed")
	}
	if len(unit.Diagnostics) != 1 {
		t.Fatalf("parseRepositoryUnit() diagnostics len = %d, want 1", len(unit.Diagnostics))
	}
	if unit.Diagnostics[0].Severity != SeverityError {
		t.Fatalf("parseRepositoryUnit() severity = %q, want %q", unit.Diagnostics[0].Severity, SeverityError)
	}
}

func TestAppendRepositoryUnit(t *testing.T) {
	result := newRepositoryResult("/repo", 2)

	appendRepositoryUnit(&result, FileUnit{Path: "/repo/A.kt", Parsed: true})
	appendRepositoryUnit(&result, FileUnit{Path: "/repo/B.kt", Parsed: false})

	if len(result.Files) != 2 {
		t.Fatalf("appendRepositoryUnit() files len = %d, want 2", len(result.Files))
	}
	if result.ParsedFiles != 1 || result.FailedFiles != 1 {
		t.Fatalf("appendRepositoryUnit() parsed=%d failed=%d, want 1/1", result.ParsedFiles, result.FailedFiles)
	}
}

func TestParseRepositoryCollectsReadErrorsAsFailedFiles(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, "src", "main", "kotlin", "App.kt"), "package sample\nclass App\n")

	brokenPath := filepath.Join(repoRoot, "src", "main", "kotlin", "Broken.kt")
	if err := os.Symlink(filepath.Join(repoRoot, "missing", "Broken.kt"), brokenPath); err != nil {
		t.Fatalf("symlink broken kotlin file: %v", err)
	}

	compiler := New(Config{Workers: 2, IncludeKTS: true})
	result, err := compiler.ParseRepository(repoRoot)
	if err != nil {
		t.Fatalf("ParseRepository() unexpected error: %v", err)
	}
	if result.TotalFiles != 2 {
		t.Fatalf("ParseRepository().TotalFiles = %d, want 2", result.TotalFiles)
	}
	if result.ParsedFiles != 1 || result.FailedFiles != 1 {
		t.Fatalf("ParseRepository() parsed=%d failed=%d, want 1/1", result.ParsedFiles, result.FailedFiles)
	}
	if len(result.Files) != 2 {
		t.Fatalf("ParseRepository().Files len = %d, want 2", len(result.Files))
	}

	byPath := map[string]FileUnit{}
	for _, unit := range result.Files {
		byPath[unit.Path] = unit
	}
	brokenUnit, ok := byPath[brokenPath]
	if !ok {
		t.Fatalf("ParseRepository() missing broken file unit in %+v", result.Files)
	}
	if brokenUnit.Parsed {
		t.Fatal("ParseRepository() broken file should be marked failed")
	}
	if len(brokenUnit.Diagnostics) != 1 {
		t.Fatalf("broken file diagnostics len = %d, want 1", len(brokenUnit.Diagnostics))
	}
	if brokenUnit.Diagnostics[0].Severity != SeverityError {
		t.Fatalf("broken file severity = %q, want %q", brokenUnit.Diagnostics[0].Severity, SeverityError)
	}
}

func TestParseRepositoryWithEmptyDirectory(t *testing.T) {
	emptyDir := t.TempDir()
	// No Kotlin files in the directory

	compiler := New(Config{Workers: 1, IncludeKTS: false, IncludeBuildScripts: false})
	result, err := compiler.ParseRepository(emptyDir)
	if err != nil {
		t.Fatalf("ParseRepository(empty) unexpected error: %v", err)
	}

	if result.TotalFiles != 0 {
		t.Fatalf("ParseRepository(empty).TotalFiles = %d, want 0", result.TotalFiles)
	}
	if result.ParsedFiles != 0 || result.FailedFiles != 0 {
		t.Fatalf("ParseRepository(empty) parsed=%d failed=%d, want 0/0", result.ParsedFiles, result.FailedFiles)
	}
	if len(result.Files) != 0 {
		t.Fatalf("ParseRepository(empty).Files len = %d, want 0", len(result.Files))
	}
	// Duration should still be recorded
	if result.Duration == 0 {
		t.Fatal("ParseRepository(empty).Duration should be non-zero")
	}
}

func TestParseRepositoryWithNonExistentRoot(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does-not-exist")

	compiler := New(Config{Workers: 1})
	_, err := compiler.ParseRepository(nonExistent)
	if err == nil {
		t.Fatal("ParseRepository(non-existent) should return error")
	}
}
