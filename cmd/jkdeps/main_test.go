package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dh-kam/jkdeps/internal/mixedgraph"
	"github.com/dh-kam/jkdeps/internal/testutil"
)

func writeJavaSample(t *testing.T, root, relativePath, body string) string {
	t.Helper()
	path := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir java package: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write sample java file: %v", err)
	}
	return path
}

func TestRunNoArgsPrintsUsage(t *testing.T) {
	exitCode := run([]string{})
	if exitCode != 2 {
		t.Fatalf("run() with no args = %d, want 2", exitCode)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	exitCode := run([]string{"does-not-exist"})
	if exitCode != 2 {
		t.Fatalf("run(does-not-exist) = %d, want 2", exitCode)
	}
}

func TestRunHelpReturnsZero(t *testing.T) {
	exitCode := run([]string{"help"})
	if exitCode != 0 {
		t.Fatalf("run(help) = %d, want 0", exitCode)
	}
	exitCode = run([]string{"-h"})
	if exitCode != 0 {
		t.Fatalf("run(-h) = %d, want 0", exitCode)
	}
	exitCode = run([]string{"--help"})
	if exitCode != 0 {
		t.Fatalf("run(--help) = %d, want 0", exitCode)
	}
}

func TestRunHelpRoutesToSubcommand(t *testing.T) {
	exitCode, out, stderr := runWithCapturedStdout(t, []string{"help", "graph"})
	if exitCode != 0 {
		t.Fatalf("run(help graph) = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected help on stdout only, stderr=%q", stderr)
	}
	if !strings.Contains(out, "Usage of graph:") {
		t.Fatalf("expected graph flag help, got: %s", out)
	}
	if !strings.Contains(out, "-group-by string") {
		t.Fatalf("expected graph flags in help output, got: %s", out)
	}
}

func TestRunSubcommandHelpWritesToStdout(t *testing.T) {
	exitCode, out, stderr := runWithCapturedStdout(t, []string{"graph", "--help"})
	if exitCode != 0 {
		t.Fatalf("run(graph --help) = %d, want 0", exitCode)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected subcommand help on stdout only, stderr=%q", stderr)
	}
	if !strings.Contains(out, "Usage of graph:") {
		t.Fatalf("expected graph flag help, got: %s", out)
	}
}

func TestRunSmokeParseRejectsInvalidGrammar(t *testing.T) {
	root := t.TempDir()
	javaPath := filepath.Join(root, "App.java")
	if err := os.WriteFile(javaPath, []byte("public class App {}"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	exitCode := run([]string{
		"smoke-parse",
		"--repo", root,
		"--workers", "1",
		"--java-grammar", "java99",
		"--max-errors", "1",
	})
	if exitCode != 1 {
		t.Fatalf("run(smoke-parse --java-grammar invalid) = %d, want 1", exitCode)
	}
}

func TestRunSmokeParseWithKtsInclusionDefaultsAndCanDisable(t *testing.T) {
	root := t.TempDir()
	ktsPath := filepath.Join(root, "script.kts")
	buildScriptPath := filepath.Join(root, "build.gradle.kts")
	if err := os.WriteFile(ktsPath, []byte("val tool: Int = 1"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.WriteFile(buildScriptPath, []byte("val x = 1"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"smoke-parse",
		"--repo", root,
		"--workers", "1",
	})
	if exitCode != 0 {
		t.Fatalf("run(smoke-parse default) = %d, want 0", exitCode)
	}
	if !strings.Contains(out, "Parse:        parsed=2 failed=0") {
		t.Fatalf("unexpected default smoke-parse counts: %s", out)
	}

	exitCode, out, _ = runWithCapturedStdout(t, []string{
		"smoke-parse",
		"--repo", root,
		"--workers", "1",
		"--include-kts=true",
	})
	if exitCode != 0 {
		t.Fatalf("run(smoke-parse --include-kts=true) = %d, want 0", exitCode)
	}
	if !strings.Contains(out, "Parse:        parsed=2 failed=0") {
		t.Fatalf("unexpected include-kts=true smoke-parse counts: %s", out)
	}

	exitCode, out, _ = runWithCapturedStdout(t, []string{
		"smoke-parse",
		"--repo", root,
		"--workers", "1",
		"--include-kts=false",
	})
	if exitCode != 0 {
		t.Fatalf("run(smoke-parse --include-kts=false) = %d, want 0", exitCode)
	}
	if !strings.Contains(out, "Parse:        parsed=0 failed=0") {
		t.Fatalf("unexpected include-kts=false smoke-parse counts: %s", out)
	}
}

func runWithCapturedStdout(t *testing.T, args []string) (int, string, string) {
	return testutil.CaptureOutput(t, func() int {
		return run(args)
	})
}

func TestRunGraphRejectsInvalidGroupBy(t *testing.T) {
	root := t.TempDir()
	exitCode := run([]string{
		"graph",
		"--repo", root,
		"--group-by", "package-or-dir",
		"--workers", "1",
	})
	if exitCode != 2 {
		t.Fatalf("run(graph --group-by invalid) = %d, want 2", exitCode)
	}
}

func TestRunGraphRejectsInvalidJavaGrammar(t *testing.T) {
	root := t.TempDir()
	exitCode := run([]string{
		"graph",
		"--repo", root,
		"--java-grammar", "java99",
		"--workers", "1",
	})
	if exitCode != 1 {
		t.Fatalf("run(graph --java-grammar invalid) = %d, want 1", exitCode)
	}
}

func TestRunGraphPrintsJSONWithoutArtifacts(t *testing.T) {
	root := t.TempDir()
	aPath := filepath.Join(root, "src", "main", "java", "com", "sample", "a", "Main.java")
	bPath := filepath.Join(root, "src", "main", "java", "com", "sample", "b", "Helper.java")
	if err := os.MkdirAll(filepath.Dir(aPath), 0o755); err != nil {
		t.Fatalf("mkdir for a package: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(bPath), 0o755); err != nil {
		t.Fatalf("mkdir for b package: %v", err)
	}
	if err := os.WriteFile(aPath, []byte(`package com.sample.a;

import com.sample.b.Helper;

public class Main {}`), 0o644); err != nil {
		t.Fatalf("write a file: %v", err)
	}
	if err := os.WriteFile(bPath, []byte(`package com.sample.b;

public class Helper {}`), 0o644); err != nil {
		t.Fatalf("write b file: %v", err)
	}

	exitCode, output, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--json",
	})
	if exitCode != 0 {
		t.Fatalf("run(graph --json) = %d, want 0", exitCode)
	}

	var graph mixedgraph.Graph
	if err := json.Unmarshal([]byte(output), &graph); err != nil {
		t.Fatalf("unmarshal graph json: %v output=%q", err, output)
	}
	if graph.GroupBy != mixedgraph.GroupByPackage {
		t.Fatalf("unexpected group_by: %s", graph.GroupBy)
	}
	if len(graph.Nodes) != 2 {
		t.Fatalf("unexpected node count: got=%d want=2", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("unexpected edge count: got=%d want=1", len(graph.Edges))
	}
}

func TestRunGraphJSONPrintsToStdoutAndWritesFileWhenOutSet(t *testing.T) {
	root := t.TempDir()
	aPath := filepath.Join(root, "src", "main", "java", "com", "sample", "a", "Main.java")
	bPath := filepath.Join(root, "src", "main", "java", "com", "sample", "b", "Helper.java")
	if err := os.MkdirAll(filepath.Dir(aPath), 0o755); err != nil {
		t.Fatalf("mkdir for a package: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(bPath), 0o755); err != nil {
		t.Fatalf("mkdir for b package: %v", err)
	}
	if err := os.WriteFile(aPath, []byte(`package com.sample.a;

import com.sample.b.Helper;

public class Main {}`), 0o644); err != nil {
		t.Fatalf("write a file: %v", err)
	}
	if err := os.WriteFile(bPath, []byte(`package com.sample.b;

public class Helper {}`), 0o644); err != nil {
		t.Fatalf("write b file: %v", err)
	}

	outBase := filepath.Join(root, "artifacts", "graph")
	exitCode, stdout, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--json",
		"--out", outBase,
	})
	if exitCode != 0 {
		t.Fatalf("run(graph --json --out) = %d, want 0", exitCode)
	}

	var fromStdout mixedgraph.Graph
	if err := json.Unmarshal([]byte(stdout), &fromStdout); err != nil {
		t.Fatalf("unmarshal stdout graph json: %v output=%q", err, stdout)
	}

	jsonPath := outBase + ".json"
	payload, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	var fromFile mixedgraph.Graph
	if err := json.Unmarshal(payload, &fromFile); err != nil {
		t.Fatalf("unmarshal file graph json: %v", err)
	}
	if len(fromStdout.Nodes) != len(fromFile.Nodes) || len(fromStdout.Edges) != len(fromFile.Edges) {
		t.Fatalf("stdout/file graph mismatch: stdout=%+v file=%+v", fromStdout, fromFile)
	}
}

func TestRunGraphFiltersByIncludeAndExcludePrefix(t *testing.T) {
	root := t.TempDir()
	aPath := filepath.Join(root, "src", "main", "java", "com", "sample", "a", "Main.java")
	bPath := filepath.Join(root, "src", "main", "java", "com", "sample", "b", "Util.java")
	cPath := filepath.Join(root, "src", "main", "java", "org", "sample", "c", "Third.java")
	if err := os.MkdirAll(filepath.Dir(aPath), 0o755); err != nil {
		t.Fatalf("mkdir for a package: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(bPath), 0o755); err != nil {
		t.Fatalf("mkdir for b package: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(cPath), 0o755); err != nil {
		t.Fatalf("mkdir for c package: %v", err)
	}
	if err := os.WriteFile(aPath, []byte(`package com.sample.a;

import com.sample.b.Util;
import org.sample.c.Third;

public class Main {}`), 0o644); err != nil {
		t.Fatalf("write a file: %v", err)
	}
	if err := os.WriteFile(bPath, []byte(`package com.sample.b;

public class Util {}`), 0o644); err != nil {
		t.Fatalf("write b file: %v", err)
	}
	if err := os.WriteFile(cPath, []byte(`package org.sample.c;

public class Third {}`), 0o644); err != nil {
		t.Fatalf("write c file: %v", err)
	}

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--json",
		"--include-prefix", "com.sample.",
	})
	if exitCode != 0 {
		t.Fatalf("run(graph --json --include-prefix) = %d, want 0. output=%s", exitCode, out)
	}

	var includeGraph mixedgraph.Graph
	if err := json.Unmarshal([]byte(out), &includeGraph); err != nil {
		t.Fatalf("unmarshal include graph json: %v output=%q", err, out)
	}
	if len(includeGraph.Nodes) != 2 {
		t.Fatalf("include-prefix should keep only matching package nodes; got %d", len(includeGraph.Nodes))
	}
	if len(includeGraph.Edges) != 1 {
		t.Fatalf("include-prefix should remove non-matching edges; got %d", len(includeGraph.Edges))
	}
	fromNode, toNode := includeGraph.Nodes[0].Name, includeGraph.Nodes[1].Name
	if fromNode != "com.sample.a" || toNode != "com.sample.b" {
		t.Fatalf("unexpected include-prefix nodes: %q, %q", fromNode, toNode)
	}
	if includeGraph.Edges[0].Count != 1 {
		t.Fatalf("expected matching-edge count=1, got=%d", includeGraph.Edges[0].Count)
	}

	exitCode, out, _ = runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--json",
		"--exclude-prefix", "com.sample.",
	})
	if exitCode != 0 {
		t.Fatalf("run(graph --json --exclude-prefix) = %d, want 0. output=%s", exitCode, out)
	}
	var excludeGraph mixedgraph.Graph
	if err := json.Unmarshal([]byte(out), &excludeGraph); err != nil {
		t.Fatalf("unmarshal exclude graph json: %v output=%q", err, out)
	}
	if len(excludeGraph.Nodes) != 1 {
		t.Fatalf("exclude-prefix should keep non-matching nodes only; got=%d nodes: %#v", len(excludeGraph.Nodes), excludeGraph.Nodes)
	}
	if excludeGraph.Nodes[0].Name != "org.sample.c" {
		t.Fatalf("unexpected kept node name: %q", excludeGraph.Nodes[0].Name)
	}
	if len(excludeGraph.Edges) != 0 {
		t.Fatalf("exclude-prefix should remove edges touching excluded nodes; got=%d", len(excludeGraph.Edges))
	}
}

func TestRunGraphFiltersByMinEdgeCount(t *testing.T) {
	root := t.TempDir()
	aPath := filepath.Join(root, "src", "main", "java", "com", "sample", "a", "Main.java")
	bPath := filepath.Join(root, "src", "main", "java", "com", "sample", "b", "Util.java")
	cPath := filepath.Join(root, "src", "main", "java", "com", "sample", "c", "Helper.java")
	if err := os.MkdirAll(filepath.Dir(aPath), 0o755); err != nil {
		t.Fatalf("mkdir for a package: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(bPath), 0o755); err != nil {
		t.Fatalf("mkdir for b package: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(cPath), 0o755); err != nil {
		t.Fatalf("mkdir for c package: %v", err)
	}
	if err := os.WriteFile(aPath, []byte(`package com.sample.a;

import com.sample.b.Util;
import com.sample.c.Helper;

public class Main {}`), 0o644); err != nil {
		t.Fatalf("write a file: %v", err)
	}
	if err := os.WriteFile(bPath, []byte(`package com.sample.b;

public class Util {}`), 0o644); err != nil {
		t.Fatalf("write b file: %v", err)
	}
	if err := os.WriteFile(cPath, []byte(`package com.sample.c;

public class Helper {}`), 0o644); err != nil {
		t.Fatalf("write c file: %v", err)
	}

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--json",
		"--min-edge-count", "2",
	})
	if exitCode != 0 {
		t.Fatalf("run(graph --json --min-edge-count=2) = %d, want 0. output=%s", exitCode, out)
	}
	var graph mixedgraph.Graph
	if err := json.Unmarshal([]byte(out), &graph); err != nil {
		t.Fatalf("unmarshal min-edge graph json: %v output=%q", err, out)
	}
	if len(graph.Edges) != 0 {
		t.Fatalf("min-edge filter should drop all edges with count < 2; got=%d", len(graph.Edges))
	}
	if len(graph.Nodes) != 0 {
		t.Fatalf("min-edge filter without matches should keep no nodes; got=%d", len(graph.Nodes))
	}
}

func TestRunGraphCanParseKotlinScriptsAndBuildScriptsByDefault(t *testing.T) {
	root := t.TempDir()
	kotlinFile := filepath.Join(root, "src", "main", "kotlin", "sample", "Tool.kt")
	ktsFile := filepath.Join(root, "src", "main", "kotlin", "sample", "utility.kts")
	buildScript := filepath.Join(root, "build.gradle.kts")
	graphOut := filepath.Join(root, "out", "mixed", "graph")

	if err := os.MkdirAll(filepath.Dir(kotlinFile), 0o755); err != nil {
		t.Fatalf("mkdir kotlin package: %v", err)
	}
	if err := os.WriteFile(kotlinFile, []byte(`package sample

class Tool`), 0o644); err != nil {
		t.Fatalf("write kotlin file: %v", err)
	}
	if err := os.WriteFile(ktsFile, []byte(`val answer = 42`), 0o644); err != nil {
		t.Fatalf("write kts file: %v", err)
	}
	if err := os.WriteFile(buildScript, []byte(`plugins {}`), 0o644); err != nil {
		t.Fatalf("write build script: %v", err)
	}

	_, out, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--out", graphOut,
	})
	if !strings.Contains(out, "Files:        total=2") {
		t.Fatalf("expected only kotlin + .kts to be parsed by default, got: %s", out)
	}

	_, out, _ = runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--include-kts",
		"--include-build-scripts",
		"--out", filepath.Join(root, "out", "mixed-with-build", "graph"),
	})
	if !strings.Contains(out, "Files:        total=3") {
		t.Fatalf("expected kotlin + .kts + build script when enabled, got: %s", out)
	}
}

func TestRunGraphCanIncludeBuildScriptsWithoutKts(t *testing.T) {
	root := t.TempDir()
	buildScript := filepath.Join(root, "build.gradle.kts")
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatalf("create scripts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "tool.kts"), []byte(`val answer = 42`), 0o644); err != nil {
		t.Fatalf("write kts file: %v", err)
	}
	if err := os.WriteFile(buildScript, []byte(`plugins {}`), 0o644); err != nil {
		t.Fatalf("write build script: %v", err)
	}

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--include-kts=false",
		"--include-build-scripts",
		"--json",
	})
	if exitCode != 0 {
		t.Fatalf("run(graph include-build-scripts without kts) = %d, want 0. output=%s", exitCode, out)
	}
	var graph mixedgraph.Graph
	if err := json.Unmarshal([]byte(out), &graph); err != nil {
		t.Fatalf("unmarshal build-script graph json: %v output=%q", err, out)
	}
	if graph.GroupBy != mixedgraph.GroupByPackage {
		t.Fatalf("unexpected group_by: %s", graph.GroupBy)
	}
	if len(graph.Nodes) != 0 {
		t.Fatalf("expected build-script-only graph without package declarations to keep 0 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Fatalf("expected no edges from package-less build scripts, got %d", len(graph.Edges))
	}
}

func TestRunGraphRespectsFailOnError(t *testing.T) {
	root := t.TempDir()
	validJava := filepath.Join(root, "src", "main", "java", "sample", "Valid.java")
	invalidJava := filepath.Join(root, "src", "main", "java", "sample", "Invalid.java")
	if err := os.MkdirAll(filepath.Dir(validJava), 0o755); err != nil {
		t.Fatalf("mkdir java package: %v", err)
	}
	if err := os.WriteFile(validJava, []byte("package sample;\n\npublic class Valid {}"), 0o644); err != nil {
		t.Fatalf("write valid java file: %v", err)
	}
	if err := os.WriteFile(invalidJava, []byte("package sample;\n\npublic class Invalid { int x = ; }"), 0o644); err != nil {
		t.Fatalf("write invalid java file: %v", err)
	}

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--out", filepath.Join(root, "graph-no-fail"),
	})
	if exitCode != 0 {
		t.Fatalf("run(graph without --fail-on-error) = %d, want 0. output=%s", exitCode, out)
	}
	if !strings.Contains(out, "Parse:        parsed=1 failed=1") {
		t.Fatalf("expected one parsed and one failed file, got: %s", out)
	}

	exitCode, out, _ = runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--fail-on-error",
		"--out", filepath.Join(root, "graph-fail"),
	})
	if exitCode != 1 {
		t.Fatalf("run(graph with --fail-on-error) = %d, want 1. output=%s", exitCode, out)
	}

	exitCode, out, _ = runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--fail-on-error",
		"--json",
	})
	if exitCode != 1 {
		t.Fatalf("run(graph with --fail-on-error --json) = %d, want 1. output=%s", exitCode, out)
	}
}

func TestRunGraphCreatesOutParentDirectories(t *testing.T) {
	root := t.TempDir()
	kotlinFile := filepath.Join(root, "src", "main", "kotlin", "sample", "Tool.kt")
	if err := os.MkdirAll(filepath.Dir(kotlinFile), 0o755); err != nil {
		t.Fatalf("mkdir kotlin package: %v", err)
	}
	if err := os.WriteFile(kotlinFile, []byte(`package sample`), 0o644); err != nil {
		t.Fatalf("write kotlin file: %v", err)
	}

	outRoot := filepath.Join(root, "nested", "artifact")
	outJSON := outRoot + ".json"
	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--out", outRoot,
	})
	if exitCode != 0 {
		t.Fatalf("run(graph with nested out path) = %d, want 0, output=%s", exitCode, out)
	}
	if _, err := os.Stat(outRoot + ".html"); err != nil {
		t.Fatalf("expected html artifact missing: %v", err)
	}
	if _, err := os.Stat(outJSON); err != nil {
		t.Fatalf("expected json artifact missing: %v", err)
	}
}

func TestRunGraphInvalidCPUProfilePathReturnsError(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), `package sample;

public class Sample {}`)
	t.Setenv("JKDEPS_CPU_PROFILE", filepath.Join(root, "missing", "dir", "jkdeps.cpuprofile"))
	exitCode, out, stderr := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--out", filepath.Join(root, "graph"),
	})
	if exitCode != 1 {
		t.Fatalf("run(graph with invalid cpu profile path) = %d, want 1. output=%s", exitCode, out)
	}
	if !strings.Contains(stderr, "profile init failed") {
		t.Fatalf("expected profile initialization failure on stderr, got: %s", stderr)
	}
}

func TestRunGraphInvalidInventoryPathReturnsError(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), `package sample;

public class Sample {}`)
	inventoryPath := filepath.Join(root, "missing", "inventory.json")
	exitCode, out, stderr := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--inventory", inventoryPath,
		"--out", filepath.Join(root, "graph"),
	})
	if exitCode != 1 {
		t.Fatalf("run(graph with invalid inventory path) = %d, want 1. output=%s", exitCode, out)
	}
	if !strings.Contains(stderr, "load inventory:") {
		t.Fatalf("expected load inventory failure on stderr, got: %s", stderr)
	}
}

func TestRunGraphWithValidInventory(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), `package sample;

public class Sample {}`)
	inventoryPath := filepath.Join(root, "inventory.json")
	inventory := `{"packages":[{"package":"com.example.demo","count":1}],"symbols":["com.example.Symbol"]}`
	if err := os.WriteFile(inventoryPath, []byte(inventory), 0o644); err != nil {
		t.Fatalf("write inventory: %v", err)
	}

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--inventory", inventoryPath,
		"--out", filepath.Join(root, "graph"),
	})
	if exitCode != 0 {
		t.Fatalf("run(graph with inventory) = %d, want 0. output=%s", exitCode, out)
	}
	if !strings.Contains(out, "Inventory:    files=1 packages=1 symbols=1") {
		t.Fatalf("expected inventory summary, got: %s", out)
	}
}

func TestRunGraphWithInvalidInventoryJSONReturnsError(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), `package sample;

public class Sample {}`)
	inventoryPath := filepath.Join(root, "inventory.json")
	if err := os.WriteFile(inventoryPath, []byte(`{"packages":[`), 0o644); err != nil {
		t.Fatalf("write broken inventory: %v", err)
	}

	exitCode, out, stderr := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--inventory", inventoryPath,
		"--out", filepath.Join(root, "graph"),
	})
	if exitCode != 1 {
		t.Fatalf("run(graph with invalid inventory json) = %d, want 1. output=%s", exitCode, out)
	}
	if !strings.Contains(stderr, "load inventory:") {
		t.Fatalf("expected load inventory failure on stderr, got: %s", stderr)
	}
}

func TestRunGraphDeduplicatesInventoryPaths(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), `package sample;

public class Sample {}`)
	inventoryPath := filepath.Join(root, "inventory.json")
	inventory := `{"packages":[{"package":"com.example.demo","count":1}],"symbols":["com.example.Symbol"]}`
	if err := os.WriteFile(inventoryPath, []byte(inventory), 0o644); err != nil {
		t.Fatalf("write inventory: %v", err)
	}

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--inventory", inventoryPath + "," + inventoryPath,
		"--out", filepath.Join(root, "graph"),
	})
	if exitCode != 0 {
		t.Fatalf("run(graph with duplicated inventory) = %d, want 0. output=%s", exitCode, out)
	}
	if !strings.Contains(out, "Inventory:    files=1 packages=1 symbols=1") {
		t.Fatalf("expected deduplicated inventory summary, got: %s", out)
	}
}

func TestRunGraphAcceptsDuplicateInventoryFlags(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), `package sample;

public class Sample {}`)

	inventoryA := filepath.Join(root, "inventory-a.json")
	inventoryB := filepath.Join(root, "inventory-b.json")
	inventory := `{"packages":[{"package":"com.example.demo","count":1}],"symbols":["com.example.Symbol"]}`
	if err := os.WriteFile(inventoryA, []byte(inventory), 0o644); err != nil {
		t.Fatalf("write inventory A: %v", err)
	}
	if err := os.WriteFile(inventoryB, []byte(inventory), 0o644); err != nil {
		t.Fatalf("write inventory B: %v", err)
	}

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--inventory", inventoryA,
		"--inventory", inventoryB,
		"--out", filepath.Join(root, "graph"),
	})
	if exitCode != 0 {
		t.Fatalf("run(graph with repeated inventory flags) = %d, want 0. output=%s", exitCode, out)
	}
	if !strings.Contains(out, "Inventory:    files=2 packages=1 symbols=1") {
		t.Fatalf("expected repeated inventory summary, got: %s", out)
	}
}

func TestRunGraphInvalidMemProfilePathReturnsError(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), `package sample;

public class Sample {}`)
	t.Setenv("JKDEPS_MEM_PROFILE", filepath.Join(root, "missing", "dir", "jkdeps.memprofile"))
	exitCode, out, stderr := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--out", filepath.Join(root, "graph"),
	})
	if exitCode != 1 {
		t.Fatalf("run(graph with invalid mem profile path) = %d, want 1. output=%s", exitCode, out)
	}
	if !strings.Contains(stderr, "profile init failed") {
		t.Fatalf("expected profile initialization failure on stderr, got: %s", stderr)
	}
}

func TestRunGraphWritesMemProfileFile(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), `package sample;

public class Sample {}`)
	memProfilePath := filepath.Join(root, "jkdeps.memprofile")
	t.Setenv("JKDEPS_MEM_PROFILE", memProfilePath)
	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--out", filepath.Join(root, "graph"),
	})
	if exitCode != 0 {
		t.Fatalf("run(graph with mem profile) = %d, want 0. output=%s", exitCode, out)
	}
	if _, err := os.Stat(memProfilePath); err != nil {
		t.Fatalf("expected mem profile file missing: %v", err)
	}
}

func TestRunGraphWritesCPUAndMemProfileFiles(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), `package sample;

public class Sample {}`)
	cpuProfilePath := filepath.Join(root, "jkdeps.cpuprofile")
	memProfilePath := filepath.Join(root, "jkdeps.memprofile")
	t.Setenv("JKDEPS_CPU_PROFILE", cpuProfilePath)
	t.Setenv("JKDEPS_MEM_PROFILE", memProfilePath)
	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--out", filepath.Join(root, "graph"),
	})
	if exitCode != 0 {
		t.Fatalf("run(graph with cpu+mem profiles) = %d, want 0. output=%s", exitCode, out)
	}
	if _, err := os.Stat(cpuProfilePath); err != nil {
		t.Fatalf("expected cpu profile file missing: %v", err)
	}
	if _, err := os.Stat(memProfilePath); err != nil {
		t.Fatalf("expected mem profile file missing: %v", err)
	}
}

func TestRunSmokeParseLimitsErrorsByMaxErrors(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 3; i++ {
		_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "sample", "Invalid"+string(rune('A'+i))+".java"), "public class Invalid { int x = ; }")
	}

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"smoke-parse",
		"--repo", root,
		"--workers", "1",
		"--max-errors", "1",
	})
	if exitCode != 1 {
		t.Fatalf("run(smoke-parse --max-errors=1) = %d, want 1", exitCode)
	}
	if !strings.Contains(out, "Parse:        parsed=0 failed=3") {
		t.Fatalf("unexpected smoke-parse summary for max errors limit: %s", out)
	}
	errorLines := strings.Count(out, "\n- ")
	if errorLines != 1 {
		t.Fatalf("expected only max-errors errors to print (1), got %d in output: %s", errorLines, out)
	}
}

func TestRunSmokeParseInvalidCPUProfilePathReturnsError(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), `package sample;

public class Sample {}`)
	t.Setenv("JKDEPS_CPU_PROFILE", filepath.Join(root, "missing", "dir", "jkdeps.cpuprofile"))
	exitCode, out, stderr := runWithCapturedStdout(t, []string{
		"smoke-parse",
		"--repo", root,
		"--workers", "1",
	})
	if exitCode != 1 {
		t.Fatalf("run(smoke-parse with invalid cpu profile path) = %d, want 1. output=%s", exitCode, out)
	}
	if !strings.Contains(stderr, "profile init failed") {
		t.Fatalf("expected profile initialization failure on stderr, got: %s", stderr)
	}
}

func TestRunSmokeParseWritesCPUAndMemProfileFiles(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), `package sample;

public class Sample {}`)
	cpuProfilePath := filepath.Join(root, "jkdeps.cpuprofile")
	memProfilePath := filepath.Join(root, "jkdeps.memprofile")
	t.Setenv("JKDEPS_CPU_PROFILE", cpuProfilePath)
	t.Setenv("JKDEPS_MEM_PROFILE", memProfilePath)
	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"smoke-parse",
		"--repo", root,
		"--workers", "1",
		"--fail-on-error=false",
	})
	if exitCode != 0 {
		t.Fatalf("run(smoke-parse with cpu+mem profiles) = %d, want 0. output=%s", exitCode, out)
	}
	if _, err := os.Stat(cpuProfilePath); err != nil {
		t.Fatalf("expected cpu profile file missing: %v", err)
	}
	if _, err := os.Stat(memProfilePath); err != nil {
		t.Fatalf("expected mem profile file missing: %v", err)
	}
}

func TestRunGraphRespectsLenientSyntax(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Invalid.java"), "public class Invalid { int x = ; }")

	exitCode, strictOut, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--out", filepath.Join(root, "strict-graph"),
	})
	if exitCode != 0 {
		t.Fatalf("run(graph without lenient) = %d, want 0", exitCode)
	}
	if !strings.Contains(strictOut, "Parse:        parsed=0 failed=1") {
		t.Fatalf("unexpected strict parse-status: %s", strictOut)
	}

	exitCode, lenientOut, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--lenient",
		"--out", filepath.Join(root, "lenient-graph"),
	})
	if exitCode != 0 {
		t.Fatalf("run(graph with lenient) = %d, want 0", exitCode)
	}
	if !strings.Contains(lenientOut, "Parse:        parsed=1 failed=0") {
		t.Fatalf("unexpected lenient parse-status: %s", lenientOut)
	}
}

func TestRunGraphRejectsInvalidFileTimeout(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), "package sample;\n\npublic class Sample {}")

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"graph",
		"--repo", root,
		"--workers", "1",
		"--file-timeout", "invalid-duration",
		"--out", filepath.Join(root, "graph"),
	})
	if exitCode != 2 {
		t.Fatalf("run(graph with invalid file-timeout) = %d, want 2, output=%s", exitCode, out)
	}
}

func TestRunSmokeParseRejectsInvalidFileTimeout(t *testing.T) {
	root := t.TempDir()
	_ = writeJavaSample(t, root, filepath.Join("src", "main", "java", "Sample.java"), "public class Sample {}")

	exitCode, out, _ := runWithCapturedStdout(t, []string{
		"smoke-parse",
		"--repo", root,
		"--workers", "1",
		"--file-timeout", "invalid-duration",
	})
	if exitCode != 2 {
		t.Fatalf("run(smoke-parse with invalid file-timeout) = %d, want 2, output=%s", exitCode, out)
	}
}
