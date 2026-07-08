package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

func TestValidateAcceptanceThresholds(t *testing.T) {
	report := kcg.AcceptanceReport{
		Parse: kcg.AcceptanceParseMetrics{
			TotalFiles:           10,
			ParsedFiles:          5,
			FailedFiles:          5,
			FailureRate:          50.0,
			FilesWithDiagnostics: 1,
			TotalDiagnostics:     2,
		},
		Resolve: kcg.AcceptanceResolveMetrics{
			UnresolvedImports: 3,
			ResolvedRate:      75.0,
		},
	}

	if err := validateAcceptanceThresholds(report, 0, -1, -1, -1, -1, -1); err == nil {
		t.Fatalf("validateAcceptanceThresholds() should fail when failed files exceed max")
	}
	if err := validateAcceptanceThresholds(report, -1, -1, -1, -1, 80.0, -1); err == nil {
		t.Fatalf("validateAcceptanceThresholds() should fail when resolved rate below min")
	}
	if err := validateAcceptanceThresholds(report, -1, -1, -1, -1, -1, 10.0); err == nil {
		t.Fatalf("validateAcceptanceThresholds() should fail when parse failure rate exceeds max")
	}
	if err := validateAcceptanceThresholds(report, 5, 3, 2, 3, 75.0, 100.0); err != nil {
		t.Fatalf("validateAcceptanceThresholds() unexpected error: %v", err)
	}
}

func TestWriteAcceptanceReportWritesJSONAndCreatesDir(t *testing.T) {
	tempDir := t.TempDir()
	reportPath := filepath.Join(tempDir, "out", "acceptance", "report.json")
	report := kcg.AcceptanceReport{}

	if err := writeAcceptanceReport(reportPath, report); err != nil {
		t.Fatalf("writeAcceptanceReport() unexpected error: %v", err)
	}

	payload, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report file: %v", err)
	}

	var got kcg.AcceptanceReport
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal report JSON: %v", err)
	}
}

func TestWriteAcceptanceReportFailsWhenParentIsFile(t *testing.T) {
	tempDir := t.TempDir()
	conflict := filepath.Join(tempDir, "blocked")
	if err := os.WriteFile(conflict, []byte("x"), 0o644); err != nil {
		t.Fatalf("write conflict: %v", err)
	}

	err := writeAcceptanceReport(filepath.Join(conflict, "report.json"), kcg.AcceptanceReport{})
	if err == nil {
		t.Fatal("expected writeAcceptanceReport to fail when parent path is a file")
	}
}

func TestRunAcceptanceInvalidInventoryReturnsError(t *testing.T) {
	repoRoot := t.TempDir()
	exitCode := runAcceptance([]string{
		"--repo", repoRoot,
		"--inventory", filepath.FromSlash("does-not-exist-inventory.json"),
		"--out", filepath.FromSlash(filepath.Join(repoRoot, "out", "acc.json")),
	})
	if exitCode != 1 {
		t.Fatalf("runAcceptance() = %d, want 1", exitCode)
	}
}

func TestRunAcceptanceInvalidRepoReturnsError(t *testing.T) {
	repoRoot := filepath.Join(t.TempDir(), "not", "exists", "yet")
	exitCode := runAcceptance([]string{
		"--repo", repoRoot,
	})
	if exitCode != 1 {
		t.Fatalf("runAcceptance() = %d, want 1", exitCode)
	}
}

func TestRunAcceptanceWriteReportFailureReturnsError(t *testing.T) {
	repoRoot := t.TempDir()
	exitCode := runAcceptance([]string{
		"--repo", repoRoot,
		"--out", filepath.FromSlash("."),
	})
	if exitCode != 1 {
		t.Fatalf("runAcceptance() = %d, want 1", exitCode)
	}
}

func TestRunAcceptanceOutPathAsDirectoryReturnsError(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package sample\n\nclass A\n"), 0o644); err != nil {
		t.Fatalf("write sample kt: %v", err)
	}

	exitCode := runAcceptance([]string{
		"--repo", repoRoot,
		"--out", repoRoot,
	})
	if exitCode != 1 {
		t.Fatalf("runAcceptance() = %d, want 1", exitCode)
	}
}

func TestRunAcceptanceInvalidParserBackendReturnsError(t *testing.T) {
	repoRoot := t.TempDir()
	exitCode := runAcceptance([]string{
		"--repo", repoRoot,
		"--parser-backend", "javac",
	})
	if exitCode != 2 {
		t.Fatalf("runAcceptance() = %d, want 2", exitCode)
	}
}

func TestRunAcceptanceInvalidFlagReturnsUsage(t *testing.T) {
	repoRoot := t.TempDir()
	exitCode := runAcceptance([]string{
		"--repo", repoRoot,
		"--does-not-exist",
	})
	if exitCode != 2 {
		t.Fatalf("runAcceptance() = %d, want 2", exitCode)
	}
}

func TestRunAcceptanceThresholdViolationReturnsFailure(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("class 12345 {\n"), 0o644); err != nil {
		t.Fatalf("write invalid kt: %v", err)
	}

	exitCode := runAcceptance([]string{
		"--repo", repoRoot,
		"--max-failed-files", "0",
	})
	if exitCode != 1 {
		t.Fatalf("runAcceptance() = %d, want 1", exitCode)
	}
}

func TestRunAcceptanceThresholdViolationViaRunReturnsFailure(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("class 12345 {\n"), 0o644); err != nil {
		t.Fatalf("write invalid kt: %v", err)
	}

	exitCode := run([]string{
		"acceptance",
		"--repo", repoRoot,
		"--max-failed-files", "0",
	})
	if exitCode != 1 {
		t.Fatalf("run(acceptance --max-failed-files 0) = %d, want 1", exitCode)
	}
}

func TestRunAcceptanceHelp(t *testing.T) {
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

	exitCode := run([]string{"acceptance", "--help"})
	_ = w.Close()

	if exitCode != 0 {
		t.Fatalf("run(acceptance --help) = %d, want 0", exitCode)
	}
}

func TestRunAcceptanceUsageIncludesFlags(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	rd, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	_ = rd
	os.Stdout = w
	os.Stderr = w
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	exitCode := run([]string{"acceptance", "--help"})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(acceptance --help) = %d, want 0", exitCode)
	}
	var b bytes.Buffer
	if _, err := b.ReadFrom(rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "  -max-failed-files") {
		t.Fatalf("acceptance --help output missing -max-failed-files flag")
	}
	if !strings.Contains(output, "  -min-resolved-rate") {
		t.Fatalf("acceptance --help output missing -min-resolved-rate flag")
	}
}

func TestRunAcceptanceHelpWritesToStdout(t *testing.T) {
	exitCode, stdout, stderr := runWithCapturedOutput(t, []string{"acceptance", "--help"})
	if exitCode != 0 {
		t.Fatalf("run(acceptance --help) = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected acceptance help on stdout only, stderr=%q", stderr)
	}
	if !strings.Contains(stdout, "Usage of acceptance:") {
		t.Fatalf("stdout = %q, want acceptance usage", stdout)
	}
}

func TestRunAcceptanceJSONAndOutWritesFilesAndStdout(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package sample\n\nclass A\n"), 0o644); err != nil {
		t.Fatalf("write sample kt: %v", err)
	}
	reportPath := filepath.Join(repoRoot, "out", "acceptance.json")

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

	exitCode := runAcceptance([]string{
		"--repo", repoRoot,
		"--out", reportPath,
		"--json",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("runAcceptance() = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "\"parse\"") {
		t.Fatalf("acceptance --json output missing parse block: %q", output)
	}
	if !strings.Contains(output, "\"resolve\"") {
		t.Fatalf("acceptance --json output missing resolve block: %q", output)
	}
	if !strings.Contains(output, "\"symbols\"") {
		t.Fatalf("acceptance --json output missing symbols block: %q", output)
	}

	payload, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read acceptance report file: %v", err)
	}
	var got kcg.AcceptanceReport
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal acceptance report: %v", err)
	}
	if got.Root != repoRoot {
		t.Fatalf("report.Root = %q want %q", got.Root, repoRoot)
	}
}

func TestRunAcceptanceJSONOutput(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package sample\n\nclass A\n"), 0o644); err != nil {
		t.Fatalf("write sample kt: %v", err)
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

	exitCode := runAcceptance([]string{
		"--repo", repoRoot,
		"--json",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}

	if exitCode != 0 {
		t.Fatalf("runAcceptance() = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "\"parse\"") {
		t.Fatalf("acceptance --json output missing parse block: %q", output)
	}
	if !strings.Contains(output, "\"symbols\"") {
		t.Fatalf("acceptance --json output missing symbols block: %q", output)
	}
	if !strings.Contains(output, "\"resolve\"") {
		t.Fatalf("acceptance --json output missing resolve block: %q", output)
	}
}

func TestRunAcceptanceSummaryOutputUsesUnifiedLabels(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package sample\n\nclass A\n"), 0o644); err != nil {
		t.Fatalf("write sample kt: %v", err)
	}

	reportPath := filepath.Join(repoRoot, "out", "acceptance.json")
	exitCode, stdout, stderr := runWithCapturedOutput(t, []string{
		"acceptance",
		"--repo", repoRoot,
		"--out", reportPath,
	})
	if exitCode != 0 {
		t.Fatalf("run(acceptance --out) = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{"Root:", "Parse:", "Resolve:", "Symbols:", "Graph:", "Duration:", "Report:"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want substring %q", stdout, want)
		}
	}
}

func TestRunAcceptanceSummaryOutputUsesUnresolvedReasonsHeader(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package sample\n\nimport missing.pkg.Type\n\nclass A(val t: Type)\n"), 0o644); err != nil {
		t.Fatalf("write sample kt: %v", err)
	}

	exitCode, stdout, stderr := runWithCapturedOutput(t, []string{
		"acceptance",
		"--repo", repoRoot,
	})
	if exitCode != 0 {
		t.Fatalf("run(acceptance) = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Unresolved Reasons:") {
		t.Fatalf("stdout = %q, want unresolved reasons header", stdout)
	}
}

func TestRunAcceptanceCommandJSONRoutesThroughRun(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package sample\n\nclass A\n"), 0o644); err != nil {
		t.Fatalf("write sample kt: %v", err)
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
		"acceptance",
		"--repo", repoRoot,
		"--json",
	})
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe write end: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("run(acceptance --json) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "\"parse\"") {
		t.Fatalf("acceptance output missing parse block: %q", output)
	}
	if !strings.Contains(output, "\"symbols\"") {
		t.Fatalf("acceptance output missing symbols block: %q", output)
	}
	if !strings.Contains(output, "\"resolve\"") {
		t.Fatalf("acceptance output missing resolve block: %q", output)
	}
}

func TestRunAcceptanceInvalidFlagThroughRunReturnsUsage(t *testing.T) {
	repoRoot := t.TempDir()
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

	exitCode := run([]string{
		"acceptance",
		"--repo", repoRoot,
		"--does-not-exist",
	})
	_ = w.Close()
	if exitCode != 2 {
		t.Fatalf("run(acceptance --does-not-exist) = %d, want 2", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "flag provided but not defined: -does-not-exist") && !strings.Contains(output, "unknown flag: --does-not-exist") {
		t.Fatalf("expected unknown flag message, got: %q", output)
	}
}

func TestRunAcceptanceHelpThroughRunReturnsUsage(t *testing.T) {
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

	exitCode := run([]string{"acceptance", "--help"})
	_ = w.Close()
	if exitCode != 0 {
		t.Fatalf("run(acceptance --help) = %d, want 0", exitCode)
	}

	var b bytes.Buffer
	if _, err := io.Copy(&b, rd); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	output := b.String()
	if !strings.Contains(output, "Usage of acceptance:") {
		t.Fatalf("acceptance help output missing usage: %q", output)
	}
}

func TestRunAcceptanceMaxUnresolvedThroughRunReturnsFailure(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package sample\n\nimport unknown.foo.Bar\n\nclass A {\n\n"), 0o644); err != nil {
		t.Fatalf("write sample kt: %v", err)
	}

	exitCode := run([]string{
		"acceptance",
		"--repo", repoRoot,
		"--max-unresolved-imports", "0",
	})
	if exitCode != 1 {
		t.Fatalf("run(acceptance --max-unresolved-imports 0) = %d, want 1", exitCode)
	}
}

func TestRunAcceptanceMinResolvedRateThroughRunReturnsFailure(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package sample\n\nimport unknown.foo.Bar\n\nclass A {}\n"), 0o644); err != nil {
		t.Fatalf("write sample kt: %v", err)
	}

	exitCode := run([]string{
		"acceptance",
		"--repo", repoRoot,
		"--min-resolved-rate", "100",
	})
	if exitCode != 1 {
		t.Fatalf("run(acceptance --min-resolved-rate 100) = %d, want 1", exitCode)
	}
}

func TestRunAcceptanceMaxParseFailureRateThroughRunReturnsFailure(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("class 12345 {\n"), 0o644); err != nil {
		t.Fatalf("write invalid kt: %v", err)
	}

	exitCode := run([]string{
		"acceptance",
		"--repo", repoRoot,
		"--max-parse-failure-rate", "0",
	})
	if exitCode != 1 {
		t.Fatalf("run(acceptance --max-parse-failure-rate 0) = %d, want 1", exitCode)
	}
}

func TestRunAcceptanceMaxTotalDiagnosticsThroughRunReturnsFailure(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package sample\n\nclass A {\n"), 0o644); err != nil {
		t.Fatalf("write invalid kt: %v", err)
	}

	exitCode := run([]string{
		"acceptance",
		"--repo", repoRoot,
		"--max-total-diagnostics", "0",
	})
	if exitCode != 1 {
		t.Fatalf("run(acceptance --max-total-diagnostics 0) = %d, want 1", exitCode)
	}
}

func TestRunAcceptanceMaxFilesWithDiagnosticsThroughRunReturnsFailure(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package sample\n\nclass A {\n"), 0o644); err != nil {
		t.Fatalf("write invalid kt: %v", err)
	}

	exitCode := run([]string{
		"acceptance",
		"--repo", repoRoot,
		"--max-files-with-diagnostics", "0",
	})
	if exitCode != 1 {
		t.Fatalf("run(acceptance --max-files-with-diagnostics 0) = %d, want 1", exitCode)
	}
}

func TestRunAcceptanceCombinedThresholdsThroughRunReturnsFailure(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "A.kt"), []byte("package sample\n\nimport unknown.foo.Bar\n\nclass A {}\n"), 0o644); err != nil {
		t.Fatalf("write sample kt: %v", err)
	}

	exitCode := run([]string{
		"acceptance",
		"--repo", repoRoot,
		"--max-parse-failure-rate", "0",
		"--min-resolved-rate", "100",
	})
	if exitCode != 1 {
		t.Fatalf("run(acceptance threshold combination) = %d, want 1", exitCode)
	}
}
