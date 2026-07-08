package porting_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	guavaRef  = "9857e70cf51a341ebb41dd2f0b8d3354f6a9d869"
	corRef    = "b11abdf01d4d5db85247ab365abc72efc7b95062"
	validSHAA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	validSHAB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("invalid repo root: %v", err)
	}
	return root
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json (%s): %v", path, err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write json (%s): %v", path, err)
	}
}

func writeText(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file (%s): %v", path, err)
	}
}

func seedCandidateArtifacts(t *testing.T, outDir, inventorySHA string) {
	t.Helper()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir out dir: %v", err)
	}

	acceptance := map[string]any{
		"parse": map[string]any{
			"failed_files":           0,
			"files_with_diagnostics": 1,
			"total_diagnostics":      1,
		},
		"resolve": map[string]any{
			"unresolved_imports": 0,
		},
		"graph": map[string]any{
			"unknown_nodes": 0,
		},
	}
	for _, rel := range []string{
		"kotlin-common-acceptance.json",
		"kotlin-js-acceptance-lenient.json",
		"kotlin-jvm-acceptance-lenient.json",
		"kotlin-core-acceptance-lenient.json",
	} {
		writeJSON(t, filepath.Join(outDir, rel), acceptance)
	}

	resolve := map[string]any{"unresolved_imports": 0}
	for _, rel := range []string{
		"kotlin-common-resolve.json",
		"kotlin-js-resolve-lenient.json",
		"kotlin-jvm-resolve-lenient.json",
		"kotlin-core-resolve-lenient.json",
	} {
		writeJSON(t, filepath.Join(outDir, rel), resolve)
	}

	writeJSON(t, filepath.Join(outDir, "porting-run-metadata.json"), map[string]any{
		"sample_refs": map[string]any{
			"guava":              guavaRef,
			"kotlinx.coroutines": corRef,
		},
		"inventory": map[string]any{
			"path":   "/tmp/runtime-index.json",
			"sha256": inventorySHA,
		},
	})

	writeText(t, filepath.Join(outDir, "guava_smoke_parse.log"), "Parse:        parsed=53 failed=0 success=100.00%\n")
	writeText(t, filepath.Join(outDir, "guava_graph_filtered.log"), "Parse:        parsed=53 failed=0\n")
	writeText(t, filepath.Join(outDir, "guava_stress_smoke_strict.log"), "Parse:        parsed=84 failed=0 success=100.00%\n")
	writeText(t, filepath.Join(outDir, "guava_stress_graph_strict.log"), "Parse:        parsed=84 failed=0\n")
	writeText(
		t,
		filepath.Join(outDir, "kotlin_core_mixed_graph_lenient.log"),
		"Files: total=699 java=1 kotlin=698\nParse:        parsed=699 failed=0\n",
	)
	writeText(
		t,
		filepath.Join(outDir, "kotlin_core_mixed_dir_graph_lenient.log"),
		"Files: total=699 java=1 kotlin=698\nParse:        parsed=699 failed=0\n",
	)
}

func setRunFlags(t *testing.T, outDir string, flags map[string]any) {
	t.Helper()
	metadataPath := filepath.Join(outDir, "porting-run-metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	metadata["run_flags"] = flags
	writeJSON(t, metadataPath, metadata)
}

func seedParityReport(t *testing.T, outDir, status string, overrides map[string]int) {
	t.Helper()
	summary := map[string]any{
		"official_total_files":        7,
		"go_total_files":              7,
		"files_compared":              7,
		"missing_in_go_files":         0,
		"missing_in_official_files":   0,
		"parse_status_mismatch_files": 0,
		"package_mismatch_files":      0,
		"import_mismatch_files":       0,
		"declaration_mismatch_files":  0,
	}
	thresholds := map[string]any{
		"max_missing_in_go":         0,
		"max_missing_in_official":   0,
		"max_parse_status_mismatch": 0,
		"max_package_mismatch":      0,
		"max_import_mismatch":       0,
		"max_declaration_mismatch":  0,
	}
	for key, value := range overrides {
		summary[key] = value
	}
	writeJSON(t, filepath.Join(outDir, "kotlin-official-parity.json"), map[string]any{
		"status":     status,
		"thresholds": thresholds,
		"summary":    summary,
	})
}

func baselineDoc(inventorySHA string) map[string]any {
	acceptanceEntry := map[string]any{
		"failed_files":                          0,
		"unresolved_imports":                    0,
		"unknown_nodes":                         0,
		"files_with_diagnostics":                1,
		"total_diagnostics":                     1,
		"max_regression_files_with_diagnostics": 0,
		"max_regression_total_diagnostics":      0,
	}
	return map[string]any{
		"sample_refs": map[string]any{
			"guava":              guavaRef,
			"kotlinx.coroutines": corRef,
		},
		"runtime_inventory": map[string]any{
			"sha256": inventorySHA,
		},
		"acceptance": map[string]any{
			"kotlin-common-acceptance.json":       acceptanceEntry,
			"kotlin-js-acceptance-lenient.json":   acceptanceEntry,
			"kotlin-jvm-acceptance-lenient.json":  acceptanceEntry,
			"kotlin-core-acceptance-lenient.json": acceptanceEntry,
		},
		"mixed_graph": map[string]any{
			"log_file":            "kotlin_core_mixed_graph_lenient.log",
			"min_java_files":      1,
			"min_kotlin_files":    1,
			"require_failed_zero": true,
		},
	}
}

func runScript(t *testing.T, root, relScript string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmdArgs := append([]string{relScript}, args...)
	cmd := exec.CommandContext(ctx, "bash", cmdArgs...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestVerifyPortingCompletionJSONOutPass(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	jsonOut := filepath.Join(outDir, "verify.json")
	seedCandidateArtifacts(t, outDir, validSHAA)

	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "verify_porting_completion.sh"),
		outDir,
		"--json-out",
		jsonOut,
	)
	if err != nil {
		t.Fatalf("verify script failed: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Porting completion check PASSED") {
		t.Fatalf("unexpected verify output:\n%s", out)
	}

	var report map[string]any
	data, readErr := os.ReadFile(jsonOut)
	if readErr != nil {
		t.Fatalf("read verify json: %v", readErr)
	}
	if unmarshalErr := json.Unmarshal(data, &report); unmarshalErr != nil {
		t.Fatalf("unmarshal verify json: %v", unmarshalErr)
	}
	if got, _ := report["status"].(string); got != "PASSED" {
		t.Fatalf("status mismatch: got %q", got)
	}
}

func TestVerifyPortingCompletionRejectsInvalidInventorySHA(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	seedCandidateArtifacts(t, outDir, "invalid-sha")

	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "verify_porting_completion.sh"),
		outDir,
	)
	if err == nil {
		t.Fatalf("verify unexpectedly succeeded\noutput:\n%s", out)
	}
	if !strings.Contains(out, "inventory.sha256 must be 64-char sha256") {
		t.Fatalf("unexpected verify failure output:\n%s", out)
	}
}

func TestVerifyPortingCompletionRequiresParityReportWhenEnabled(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	seedCandidateArtifacts(t, outDir, validSHAA)
	setRunFlags(t, outDir, map[string]any{
		"run_kotlin_official_parity": "1",
	})

	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "verify_porting_completion.sh"),
		outDir,
	)
	if err == nil {
		t.Fatalf("verify unexpectedly succeeded\noutput:\n%s", out)
	}
	if !strings.Contains(out, "kotlin-official-parity.json") {
		t.Fatalf("expected parity artifact error, got:\n%s", out)
	}
}

func TestVerifyPortingCompletionValidatesParityReportWhenEnabled(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	seedCandidateArtifacts(t, outDir, validSHAA)
	setRunFlags(t, outDir, map[string]any{
		"run_kotlin_official_parity": "1",
	})
	seedParityReport(t, outDir, "PASSED", nil)

	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "verify_porting_completion.sh"),
		outDir,
	)
	if err != nil {
		t.Fatalf("verify failed unexpectedly: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "kotlin-official-parity.json") {
		t.Fatalf("expected parity check output, got:\n%s", out)
	}
}

func TestComparePortingBaselineFailsOnRuntimeInventoryMismatch(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	seedCandidateArtifacts(t, outDir, validSHAB)

	baselinePath := filepath.Join(outDir, "baseline.json")
	reportMD := filepath.Join(outDir, "report.md")
	reportJSON := filepath.Join(outDir, "report.json")
	writeJSON(t, baselinePath, baselineDoc(validSHAA))

	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "compare_porting_baseline.sh"),
		outDir,
		baselinePath,
		reportMD,
		reportJSON,
	)
	if err == nil {
		t.Fatalf("compare unexpectedly succeeded\noutput:\n%s", out)
	}

	var report map[string]any
	data, readErr := os.ReadFile(reportJSON)
	if readErr != nil {
		t.Fatalf("read compare report json: %v", readErr)
	}
	if unmarshalErr := json.Unmarshal(data, &report); unmarshalErr != nil {
		t.Fatalf("unmarshal compare report json: %v", unmarshalErr)
	}
	if got, _ := report["status"].(string); got != "FAILED" {
		t.Fatalf("status mismatch: got %q", got)
	}
	errorsList, ok := report["errors"].([]any)
	if !ok || len(errorsList) == 0 {
		t.Fatalf("compare errors missing: %#v", report["errors"])
	}
	found := false
	for _, item := range errorsList {
		text, _ := item.(string)
		if strings.Contains(text, "runtime inventory sha mismatch") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing runtime inventory mismatch error: %v", errorsList)
	}
}

func TestComparePortingBaselinePassesOnRuntimeInventoryMatch(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	seedCandidateArtifacts(t, outDir, validSHAA)

	baselinePath := filepath.Join(outDir, "baseline.json")
	reportMD := filepath.Join(outDir, "report.md")
	reportJSON := filepath.Join(outDir, "report.json")
	writeJSON(t, baselinePath, baselineDoc(validSHAA))

	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "compare_porting_baseline.sh"),
		outDir,
		baselinePath,
		reportMD,
		reportJSON,
	)
	if err != nil {
		t.Fatalf("compare script failed: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Porting baseline compare PASSED") {
		t.Fatalf("unexpected compare output:\n%s", out)
	}

	var report map[string]any
	data, readErr := os.ReadFile(reportJSON)
	if readErr != nil {
		t.Fatalf("read compare report json: %v", readErr)
	}
	if unmarshalErr := json.Unmarshal(data, &report); unmarshalErr != nil {
		t.Fatalf("unmarshal compare report json: %v", unmarshalErr)
	}
	if got, _ := report["status"].(string); got != "PASSED" {
		t.Fatalf("status mismatch: got %q", got)
	}
	runtimeInventory, ok := report["runtime_inventory"].(map[string]any)
	if !ok {
		t.Fatalf("runtime_inventory missing: %#v", report["runtime_inventory"])
	}
	candidate, ok := runtimeInventory["candidate"].(map[string]any)
	if !ok {
		t.Fatalf("runtime_inventory.candidate missing: %#v", runtimeInventory["candidate"])
	}
	if got, _ := candidate["sha256"].(string); got != validSHAA {
		t.Fatalf("candidate runtime sha mismatch: got %q", got)
	}
}

func TestVerifyPortingCompletionJSONStdoutFirstLine(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	seedCandidateArtifacts(t, outDir, validSHAA)

	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "verify_porting_completion.sh"),
		outDir,
		"--json",
	)
	if err != nil {
		t.Fatalf("verify script failed: %v\noutput:\n%s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 {
		t.Fatal("verify output is empty")
	}
	var payload map[string]any
	if unmarshalErr := json.Unmarshal([]byte(lines[0]), &payload); unmarshalErr != nil {
		t.Fatalf("failed to parse json line: %v\noutput:\n%s", unmarshalErr, out)
	}
	if got, _ := payload["status"].(string); got != "PASSED" {
		t.Fatalf("json status mismatch: got %q", got)
	}
}

func TestUpdatePortingBaselineBuildsFile(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	seedCandidateArtifacts(t, outDir, validSHAA)

	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "update_porting_baseline.sh"),
		outDir,
		baselinePath,
	)
	if err != nil {
		t.Fatalf("update baseline script failed: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "updated baseline:") {
		t.Fatalf("unexpected update output:\n%s", out)
	}

	var baseline map[string]any
	data, readErr := os.ReadFile(baselinePath)
	if readErr != nil {
		t.Fatalf("read baseline: %v", readErr)
	}
	if unmarshalErr := json.Unmarshal(data, &baseline); unmarshalErr != nil {
		t.Fatalf("unmarshal baseline: %v", unmarshalErr)
	}

	sampleRefs, ok := baseline["sample_refs"].(map[string]any)
	if !ok {
		t.Fatalf("baseline sample_refs missing: %#v", baseline["sample_refs"])
	}
	if got, _ := sampleRefs["guava"].(string); got != guavaRef {
		t.Fatalf("sample_refs.guava mismatch: %q", got)
	}

	runtimeInventory, ok := baseline["runtime_inventory"].(map[string]any)
	if !ok {
		t.Fatalf("baseline runtime_inventory missing: %#v", baseline["runtime_inventory"])
	}
	if got, _ := runtimeInventory["sha256"].(string); got != validSHAA {
		t.Fatalf("runtime_inventory.sha256 mismatch: %q", got)
	}

	acceptance, ok := baseline["acceptance"].(map[string]any)
	if !ok {
		t.Fatalf("baseline acceptance missing: %#v", baseline["acceptance"])
	}
	if _, exists := acceptance["kotlin-core-acceptance-lenient.json"]; !exists {
		t.Fatalf("baseline acceptance missing core entry: %#v", acceptance)
	}
}

func TestUpdatePortingBaselineCheckModePassesWhenSynced(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	seedCandidateArtifacts(t, outDir, validSHAA)

	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	writeJSON(t, baselinePath, map[string]any{
		"baseline_date": "2000-01-02",
		"artifact_dir":  "/stable/artifact-dir",
		"sample_refs": map[string]any{
			"guava":              guavaRef,
			"kotlinx.coroutines": corRef,
		},
		"runtime_inventory": map[string]any{
			"sha256": validSHAA,
		},
		"acceptance": map[string]any{
			"kotlin-common-acceptance.json": map[string]any{
				"failed_files":                          0,
				"unresolved_imports":                    0,
				"unknown_nodes":                         0,
				"files_with_diagnostics":                1,
				"total_diagnostics":                     1,
				"max_regression_files_with_diagnostics": 3,
				"max_regression_total_diagnostics":      7,
			},
			"kotlin-js-acceptance-lenient.json": map[string]any{
				"failed_files":                          0,
				"unresolved_imports":                    0,
				"unknown_nodes":                         0,
				"files_with_diagnostics":                1,
				"total_diagnostics":                     1,
				"max_regression_files_with_diagnostics": 0,
				"max_regression_total_diagnostics":      0,
			},
			"kotlin-jvm-acceptance-lenient.json": map[string]any{
				"failed_files":                          0,
				"unresolved_imports":                    0,
				"unknown_nodes":                         0,
				"files_with_diagnostics":                1,
				"total_diagnostics":                     1,
				"max_regression_files_with_diagnostics": 0,
				"max_regression_total_diagnostics":      0,
			},
			"kotlin-core-acceptance-lenient.json": map[string]any{
				"failed_files":                          0,
				"unresolved_imports":                    0,
				"unknown_nodes":                         0,
				"files_with_diagnostics":                1,
				"total_diagnostics":                     1,
				"max_regression_files_with_diagnostics": 0,
				"max_regression_total_diagnostics":      0,
			},
		},
		"mixed_graph": map[string]any{
			"log_file":            "kotlin_core_mixed_graph_lenient.log",
			"min_java_files":      1,
			"min_kotlin_files":    1,
			"require_failed_zero": true,
		},
	})

	_, err := runScript(
		t,
		root,
		filepath.Join("scripts", "update_porting_baseline.sh"),
		outDir,
		baselinePath,
	)
	if err != nil {
		t.Fatalf("initial baseline update failed: %v", err)
	}

	out, checkErr := runScript(
		t,
		root,
		filepath.Join("scripts", "update_porting_baseline.sh"),
		outDir,
		baselinePath,
		"--check",
	)
	if checkErr != nil {
		t.Fatalf("check mode failed unexpectedly: %v\noutput:\n%s", checkErr, out)
	}
	if !strings.Contains(out, "baseline is up to date:") {
		t.Fatalf("unexpected check output:\n%s", out)
	}

	var baseline map[string]any
	data, readErr := os.ReadFile(baselinePath)
	if readErr != nil {
		t.Fatalf("read baseline: %v", readErr)
	}
	if unmarshalErr := json.Unmarshal(data, &baseline); unmarshalErr != nil {
		t.Fatalf("unmarshal baseline: %v", unmarshalErr)
	}
	if got, _ := baseline["baseline_date"].(string); got != "2000-01-02" {
		t.Fatalf("baseline_date should be preserved, got %q", got)
	}
	if got, _ := baseline["artifact_dir"].(string); got != "/stable/artifact-dir" {
		t.Fatalf("artifact_dir should be preserved, got %q", got)
	}
	acceptance, ok := baseline["acceptance"].(map[string]any)
	if !ok {
		t.Fatalf("acceptance missing: %#v", baseline["acceptance"])
	}
	common, ok := acceptance["kotlin-common-acceptance.json"].(map[string]any)
	if !ok {
		t.Fatalf("common acceptance missing: %#v", acceptance["kotlin-common-acceptance.json"])
	}
	if got, _ := common["max_regression_files_with_diagnostics"].(float64); got != 3 {
		t.Fatalf("max_regression_files_with_diagnostics should be preserved, got %v", got)
	}
	if got, _ := common["max_regression_total_diagnostics"].(float64); got != 7 {
		t.Fatalf("max_regression_total_diagnostics should be preserved, got %v", got)
	}
}

func TestUpdatePortingBaselineCheckModeFailsWhenStale(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	seedCandidateArtifacts(t, outDir, validSHAA)

	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	writeJSON(t, baselinePath, baselineDoc(validSHAB))

	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "update_porting_baseline.sh"),
		outDir,
		baselinePath,
		"--check",
	)
	if err == nil {
		t.Fatalf("check mode unexpectedly passed\noutput:\n%s", out)
	}
	if !strings.Contains(out, "baseline is out of date:") {
		t.Fatalf("unexpected stale check output:\n%s", out)
	}

	generatedPath := baselinePath + ".generated"
	if _, statErr := os.Stat(generatedPath); statErr != nil {
		t.Fatalf("expected generated baseline file to exist: %v", statErr)
	}
}

func TestPortingAuditScriptPassesAndWritesAuditJSON(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	seedCandidateArtifacts(t, outDir, validSHAA)

	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	artifactDir := "/stable/artifact-dir"
	writeJSON(t, baselinePath, map[string]any{
		"baseline_date": "2000-01-02",
		"artifact_dir":  artifactDir,
		"sample_refs": map[string]any{
			"guava":              guavaRef,
			"kotlinx.coroutines": corRef,
		},
		"runtime_inventory": map[string]any{
			"sha256": validSHAA,
		},
		"acceptance": map[string]any{
			"kotlin-common-acceptance.json": map[string]any{
				"failed_files":                          0,
				"unresolved_imports":                    0,
				"unknown_nodes":                         0,
				"files_with_diagnostics":                1,
				"total_diagnostics":                     1,
				"max_regression_files_with_diagnostics": 0,
				"max_regression_total_diagnostics":      0,
			},
			"kotlin-js-acceptance-lenient.json": map[string]any{
				"failed_files":                          0,
				"unresolved_imports":                    0,
				"unknown_nodes":                         0,
				"files_with_diagnostics":                1,
				"total_diagnostics":                     1,
				"max_regression_files_with_diagnostics": 0,
				"max_regression_total_diagnostics":      0,
			},
			"kotlin-jvm-acceptance-lenient.json": map[string]any{
				"failed_files":                          0,
				"unresolved_imports":                    0,
				"unknown_nodes":                         0,
				"files_with_diagnostics":                1,
				"total_diagnostics":                     1,
				"max_regression_files_with_diagnostics": 0,
				"max_regression_total_diagnostics":      0,
			},
			"kotlin-core-acceptance-lenient.json": map[string]any{
				"failed_files":                          0,
				"unresolved_imports":                    0,
				"unknown_nodes":                         0,
				"files_with_diagnostics":                1,
				"total_diagnostics":                     1,
				"max_regression_files_with_diagnostics": 0,
				"max_regression_total_diagnostics":      0,
			},
		},
		"mixed_graph": map[string]any{
			"log_file":            "kotlin_core_mixed_graph_lenient.log",
			"min_java_files":      1,
			"min_kotlin_files":    1,
			"require_failed_zero": true,
		},
	})

	auditJSON := filepath.Join(outDir, "porting-audit.json")
	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "porting_audit.sh"),
		outDir,
		baselinePath,
		auditJSON,
	)
	if err != nil {
		t.Fatalf("porting audit failed: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "porting audit PASSED:") {
		t.Fatalf("unexpected porting audit output:\n%s", out)
	}

	var report map[string]any
	data, readErr := os.ReadFile(auditJSON)
	if readErr != nil {
		t.Fatalf("read audit json: %v", readErr)
	}
	if unmarshalErr := json.Unmarshal(data, &report); unmarshalErr != nil {
		t.Fatalf("unmarshal audit json: %v", unmarshalErr)
	}
	if got, _ := report["status"].(string); got != "PASSED" {
		t.Fatalf("audit status mismatch: %q", got)
	}
	syncCheck, ok := report["sync_check"].(map[string]any)
	if !ok {
		t.Fatalf("sync_check missing: %#v", report["sync_check"])
	}
	if got, _ := syncCheck["status"].(string); got != "PASSED" {
		t.Fatalf("sync_check status mismatch: %q", got)
	}
	verify, ok := report["verify"].(map[string]any)
	if !ok {
		t.Fatalf("verify payload missing: %#v", report["verify"])
	}
	if got, _ := verify["status"].(string); got != "PASSED" {
		t.Fatalf("verify status mismatch in audit payload: %q", got)
	}
}

func TestPortingAuditScriptFailsAndWritesAuditJSONOnCompareFailure(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	seedCandidateArtifacts(t, outDir, validSHAA)

	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	writeJSON(t, baselinePath, baselineDoc(validSHAB))

	auditJSON := filepath.Join(outDir, "porting-audit.json")
	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "porting_audit.sh"),
		outDir,
		baselinePath,
		auditJSON,
	)
	if err == nil {
		t.Fatalf("porting audit unexpectedly passed\noutput:\n%s", out)
	}

	var report map[string]any
	data, readErr := os.ReadFile(auditJSON)
	if readErr != nil {
		t.Fatalf("read audit json: %v", readErr)
	}
	if unmarshalErr := json.Unmarshal(data, &report); unmarshalErr != nil {
		t.Fatalf("unmarshal audit json: %v", unmarshalErr)
	}
	if got, _ := report["status"].(string); got != "FAILED" {
		t.Fatalf("audit status mismatch: %q", got)
	}
	if got, _ := report["failed_stage"].(string); got != "compare" {
		t.Fatalf("audit failed_stage mismatch: %q", got)
	}
}

func TestPortingAuditScriptFailsAndWritesAuditJSONOnVerifyFailure(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	// Deliberately do not seed required artifacts so verify stage fails first.
	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	writeJSON(t, baselinePath, baselineDoc(validSHAA))

	auditJSON := filepath.Join(outDir, "porting-audit.json")
	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "porting_audit.sh"),
		outDir,
		baselinePath,
		auditJSON,
	)
	if err == nil {
		t.Fatalf("porting audit unexpectedly passed\noutput:\n%s", out)
	}

	var report map[string]any
	data, readErr := os.ReadFile(auditJSON)
	if readErr != nil {
		t.Fatalf("read audit json: %v", readErr)
	}
	if unmarshalErr := json.Unmarshal(data, &report); unmarshalErr != nil {
		t.Fatalf("unmarshal audit json: %v", unmarshalErr)
	}
	if got, _ := report["status"].(string); got != "FAILED" {
		t.Fatalf("audit status mismatch: %q", got)
	}
	if got, _ := report["failed_stage"].(string); got != "verify" {
		t.Fatalf("audit failed_stage mismatch: %q", got)
	}
	verify, ok := report["verify"].(map[string]any)
	if !ok {
		t.Fatalf("verify payload missing: %#v", report["verify"])
	}
	if got, _ := verify["status"].(string); got != "FAILED" {
		t.Fatalf("verify status mismatch in audit payload: %q", got)
	}
}

func TestPortingAuditScriptFailsAndWritesAuditJSONOnBaselineSyncFailure(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	seedCandidateArtifacts(t, outDir, validSHAA)

	// Compare can still pass when max_regression_* fields are omitted (defaults to 0),
	// but update_porting_baseline --check will fail due to exact payload mismatch.
	baselinePath := filepath.Join(t.TempDir(), "baseline-missing-max-regression.json")
	writeJSON(t, baselinePath, map[string]any{
		"baseline_date": "2000-01-02",
		"artifact_dir":  outDir,
		"sample_refs": map[string]any{
			"guava":              guavaRef,
			"kotlinx.coroutines": corRef,
		},
		"runtime_inventory": map[string]any{
			"sha256": validSHAA,
		},
		"acceptance": map[string]any{
			"kotlin-common-acceptance.json": map[string]any{
				"failed_files":           0,
				"unresolved_imports":     0,
				"unknown_nodes":          0,
				"files_with_diagnostics": 1,
				"total_diagnostics":      1,
			},
			"kotlin-js-acceptance-lenient.json": map[string]any{
				"failed_files":           0,
				"unresolved_imports":     0,
				"unknown_nodes":          0,
				"files_with_diagnostics": 1,
				"total_diagnostics":      1,
			},
			"kotlin-jvm-acceptance-lenient.json": map[string]any{
				"failed_files":           0,
				"unresolved_imports":     0,
				"unknown_nodes":          0,
				"files_with_diagnostics": 1,
				"total_diagnostics":      1,
			},
			"kotlin-core-acceptance-lenient.json": map[string]any{
				"failed_files":           0,
				"unresolved_imports":     0,
				"unknown_nodes":          0,
				"files_with_diagnostics": 1,
				"total_diagnostics":      1,
			},
		},
		"mixed_graph": map[string]any{
			"log_file":            "kotlin_core_mixed_graph_lenient.log",
			"min_java_files":      1,
			"min_kotlin_files":    1,
			"require_failed_zero": true,
		},
	})

	auditJSON := filepath.Join(outDir, "porting-audit.json")
	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "porting_audit.sh"),
		outDir,
		baselinePath,
		auditJSON,
	)
	if err == nil {
		t.Fatalf("porting audit unexpectedly passed\noutput:\n%s", out)
	}

	var report map[string]any
	data, readErr := os.ReadFile(auditJSON)
	if readErr != nil {
		t.Fatalf("read audit json: %v", readErr)
	}
	if unmarshalErr := json.Unmarshal(data, &report); unmarshalErr != nil {
		t.Fatalf("unmarshal audit json: %v", unmarshalErr)
	}
	if got, _ := report["status"].(string); got != "FAILED" {
		t.Fatalf("audit status mismatch: %q", got)
	}
	if got, _ := report["failed_stage"].(string); got != "baseline_sync" {
		t.Fatalf("audit failed_stage mismatch: %q", got)
	}
	syncCheck, ok := report["sync_check"].(map[string]any)
	if !ok {
		t.Fatalf("sync_check missing: %#v", report["sync_check"])
	}
	if got, _ := syncCheck["status"].(string); got != "FAILED" {
		t.Fatalf("sync_check status mismatch: %q", got)
	}
}

func TestAppendPortingStepSummaryWritesSections(t *testing.T) {
	root := repoRoot(t)
	artifactDir := t.TempDir()
	summaryPath := filepath.Join(t.TempDir(), "summary.md")

	writeJSON(t, filepath.Join(artifactDir, "porting-audit.json"), map[string]any{
		"status":        "PASSED",
		"failed_stage":  "",
		"out_dir":       artifactDir,
		"baseline_file": "docs/porting-baseline.json",
		"sync_check": map[string]any{
			"status": "PASSED",
		},
	})
	writeJSON(t, filepath.Join(artifactDir, "porting-completion-verify.json"), map[string]any{
		"status":  "PASSED",
		"out_dir": artifactDir,
		"checks": []any{
			"check-1",
		},
		"errors": []any{},
	})
	writeJSON(t, filepath.Join(artifactDir, "porting-baseline-compare.json"), map[string]any{
		"status":    "PASSED",
		"candidate": artifactDir,
		"baseline":  "docs/porting-baseline.json",
		"checks": []any{
			"cmp-1",
		},
		"errors": []any{},
	})

	out, err := runScript(
		t,
		root,
		filepath.Join("scripts", "append_porting_step_summary.sh"),
		artifactDir,
		summaryPath,
	)
	if err != nil {
		t.Fatalf("append summary script failed: %v\noutput:\n%s", err, out)
	}

	data, readErr := os.ReadFile(summaryPath)
	if readErr != nil {
		t.Fatalf("read summary: %v", readErr)
	}
	text := string(data)
	for _, marker := range []string{
		"### Porting Audit",
		"### Porting Completion Verify",
		"### Porting Baseline Compare",
		"check-1",
		"cmp-1",
	} {
		if !strings.Contains(text, marker) {
			t.Fatalf("summary missing marker %q:\n%s", marker, text)
		}
	}
}
