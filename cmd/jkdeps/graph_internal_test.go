package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime/pprof"
	"strings"
	"testing"

	"github.com/dh-kam/jkdeps/internal/cliutil"
	"github.com/dh-kam/jkdeps/internal/flagutil"
	"github.com/dh-kam/jkdeps/internal/mixedgraph"
)

func TestGraphOutputPathsBasic(t *testing.T) {
	html, json := cliutil.GraphOutputPaths("", "jkdeps-mixed-graph")
	if html != "jkdeps-mixed-graph.html" {
		t.Fatalf("empty base html=%q want %q", html, "jkdeps-mixed-graph.html")
	}
	if json != "jkdeps-mixed-graph.json" {
		t.Fatalf("empty base json=%q want %q", json, "jkdeps-mixed-graph.json")
	}

	html, json = cliutil.GraphOutputPaths("custom.html", "jkdeps-mixed-graph")
	if html != "custom.html" || json != "custom.json" {
		t.Fatalf("html output mismatch for .html input: html=%q json=%q", html, json)
	}

	html, json = cliutil.GraphOutputPaths("custom.json", "jkdeps-mixed-graph")
	if html != "custom.html" || json != "custom.json" {
		t.Fatalf("html output mismatch for .json input: html=%q json=%q", html, json)
	}

	html, json = cliutil.GraphOutputPaths("artifact", "jkdeps-mixed-graph")
	if html != "artifact.html" || json != "artifact.json" {
		t.Fatalf("html output mismatch for plain input: html=%q json=%q", html, json)
	}

	html, json = cliutil.GraphOutputPaths("   ", "jkdeps-mixed-graph")
	if html != "jkdeps-mixed-graph.html" || json != "jkdeps-mixed-graph.json" {
		t.Fatalf("whitespace base should default: html=%q json=%q", html, json)
	}
}

func TestUniqueStrings(t *testing.T) {
	got := flagutil.UniqueStrings([]string{"a", "b", "a", " ", "", "b", "c"})
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UniqueStrings mismatch: got=%v want=%v", got, want)
	}

	if got := flagutil.UniqueStrings([]string{}); len(got) != 0 {
		t.Fatalf("UniqueStrings(empty) mismatch: got=%v want=[]", got)
	}
}

func TestStringListFlagSet(t *testing.T) {
	var flagValue stringListFlag
	if err := flagValue.Set("a, b,, c "); err != nil {
		t.Fatalf("set error: %v", err)
	}
	if got := strings.Join(flagValue, "|"); got != "a|b|c" {
		t.Fatalf("stringListFlag value=%q want %q", got, "a|b|c")
	}
}

func TestBuildGraphHTML(t *testing.T) {
	html := buildGraphHTML("artifact.json")
	if !strings.Contains(html, `"artifact.json"`) {
		t.Fatalf("buildGraphHTML expected quoted json file path, got %q", html)
	}
}

func TestWriteGraphArtifactsCreatesFiles(t *testing.T) {
	root := t.TempDir()
	graph := mixedgraph.Graph{
		Root:    root,
		GroupBy: mixedgraph.GroupByPackage,
		Nodes:   []mixedgraph.Node{{ID: 1, Name: "a", Kind: mixedgraph.NodeInternal, InDegree: 0, OutDegree: 0}},
		Edges:   []mixedgraph.Edge{},
	}

	htmlPath := filepath.Join(root, "out", "graph.html")
	jsonPath := filepath.Join(root, "out", "graph.json")
	if err := writeGraphArtifacts(htmlPath, jsonPath, graph); err != nil {
		t.Fatalf("writeGraphArtifacts() = %v", err)
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
	graph := mixedgraph.Graph{}
	err := writeGraphArtifacts(filepath.Join(conflict, "graph.html"), filepath.Join(root, "ok.json"), graph)
	if err == nil {
		t.Fatal("expected writeGraphArtifacts to fail when html parent is a file")
	}
}

func TestStartGraphProfilingCleansUpAfterFailure(t *testing.T) {
	cpuPath := filepath.Join(t.TempDir(), "jkdeps-cpu.pprof")
	brokenMemPath := filepath.Join(t.TempDir(), "missing-dir", "jkdeps-mem.pprof")

	t.Setenv("JKDEPS_CPU_PROFILE", cpuPath)
	t.Setenv("JKDEPS_MEM_PROFILE", brokenMemPath)

	err := startGraphProfiling()
	if err == nil {
		t.Fatal("expected startGraphProfiling to fail when memory profile path directory does not exist")
	}

	retryPath := filepath.Join(t.TempDir(), "jkdeps-cpu-retry.pprof")
	retryFile, err := os.Create(retryPath)
	if err != nil {
		t.Fatalf("create retry cpu profile: %v", err)
	}
	defer retryFile.Close()

	if err := pprof.StartCPUProfile(retryFile); err != nil {
		t.Fatalf("expected graph profile to be stopped on failure path, got start error: %v", err)
	}
	pprof.StopCPUProfile()
}

func TestStartGraphProfilingNoProfilesConfiguredIsNoop(t *testing.T) {
	t.Setenv("JKDEPS_CPU_PROFILE", "")
	t.Setenv("JKDEPS_MEM_PROFILE", "")

	if err := startGraphProfiling(); err != nil {
		t.Fatalf("startGraphProfiling = %v, want nil", err)
	}
	stopGraphProfiling()
}
