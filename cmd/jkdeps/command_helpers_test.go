package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dh-kam/jkdeps/internal/mixedgraph"
	"github.com/dh-kam/jkdeps/internal/parser"
)

func TestParseGroupByFlag(t *testing.T) {
	group, err := parseGroupByFlag(" package ")
	if err != nil {
		t.Fatalf("parseGroupByFlag(package) error = %v", err)
	}
	if group != mixedgraph.GroupByPackage {
		t.Fatalf("group = %q, want %q", group, mixedgraph.GroupByPackage)
	}

	_, err = parseGroupByFlag("package-or-dir")
	if err == nil {
		t.Fatal("expected invalid group-by error, got nil")
	}
	if !strings.Contains(err.Error(), "expected package|dir") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadExternalIndexFlags(t *testing.T) {
	root := t.TempDir()
	inventoryPath := filepath.Join(root, "inventory.json")
	payload := []byte(`{"packages":[{"package":"com.example","count":1}],"symbols":["com.example.Symbol"]}`)
	if err := os.WriteFile(inventoryPath, payload, 0o644); err != nil {
		t.Fatalf("write inventory: %v", err)
	}

	paths, index, err := loadExternalIndexFlags(stringListFlag{" " + inventoryPath + " ", inventoryPath})
	if err != nil {
		t.Fatalf("loadExternalIndexFlags(...) = %v", err)
	}
	if len(paths) != 1 || paths[0] != inventoryPath {
		t.Fatalf("paths = %v, want [%s]", paths, inventoryPath)
	}
	if len(index.Packages) != 1 {
		t.Fatalf("len(index.Packages) = %d, want 1", len(index.Packages))
	}
	if len(index.Symbols) != 1 {
		t.Fatalf("len(index.Symbols) = %d, want 1", len(index.Symbols))
	}
}

func TestWriteMixedRepositorySummary(t *testing.T) {
	result := mixedgraph.RepositoryResult{
		Root:        "/repo",
		TotalFiles:  4,
		JavaFiles:   2,
		KotlinFiles: 2,
		ParsedFiles: 3,
		FailedFiles: 1,
		Duration:    250 * time.Millisecond,
	}

	var withGrammar bytes.Buffer
	writeMixedRepositorySummary(&withGrammar, result, parser.JavaGrammar("java20"), true)
	output := withGrammar.String()
	for _, want := range []string{
		"Root:",
		"/repo",
		"Java Grammar:",
		"java20",
		"Files:",
		"total=4 java=2 kotlin=2",
		"Parse:",
		"parsed=3 failed=1 success=75.00%",
		"Duration:",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("summary output missing %q in %q", want, output)
		}
	}

	var withoutGrammar bytes.Buffer
	writeMixedRepositorySummary(&withoutGrammar, result, "", false)
	output = withoutGrammar.String()
	if strings.Contains(output, "Java Grammar:") {
		t.Fatalf("unexpected Java Grammar line in %q", output)
	}
	if !strings.Contains(output, "parsed=3 failed=1") {
		t.Fatalf("missing parse summary in %q", output)
	}
	if strings.Contains(output, "success=") {
		t.Fatalf("unexpected success rate in %q", output)
	}
}

func TestWriteSlowParseFilesSummary(t *testing.T) {
	result := mixedgraph.RepositoryResult{
		Files: []mixedgraph.FileUnit{
			{Path: "/repo/b.kt", Relative: "b.kt", Parsed: true, Duration: 120 * time.Millisecond},
			{Path: "/repo/a.java", Relative: "a.java", Parsed: false, Duration: 2 * time.Second},
		},
	}

	var buf bytes.Buffer
	writeSlowParseFilesSummary(&buf, result, 2)
	out := buf.String()
	for _, want := range []string{
		"Slow Parse Files (top 2)",
		"2s failed a.java",
		"120ms parsed b.kt",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("slow parse summary missing %q in %q", want, out)
		}
	}
}
