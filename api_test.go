package jkdeps_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dh-kam/jkdeps"
)

func TestAnalyzeFacade(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "src/main/kotlin/sample/kt/KThing.kt", "package sample.kt\n\nclass KThing\n")
	writeFile(t, root, "src/main/java/sample/JThing.java", "package sample;\n\nimport sample.kt.KThing;\n\npublic class JThing {}\n")

	report, err := jkdeps.Analyze(context.Background(), root, jkdeps.Options{
		Parse: jkdeps.ParseOptions{Workers: 2},
		Graph: jkdeps.GraphOptions{GroupBy: jkdeps.GroupByPackage},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if report.Repository.TotalFiles != 2 {
		t.Fatalf("TotalFiles = %d, want 2", report.Repository.TotalFiles)
	}
	if report.Repository.ParsedFiles != 2 {
		t.Fatalf("ParsedFiles = %d, want 2", report.Repository.ParsedFiles)
	}
	if len(report.Graph.Nodes) < 2 {
		t.Fatalf("graph node count = %d, want at least 2", len(report.Graph.Nodes))
	}
	if report.Dependencies.TotalDependencies == 0 {
		t.Fatal("expected at least one dependency")
	}

	filtered := report.Graph.Filter(jkdeps.GraphFilter{IncludePrefix: []string{"sample"}})
	if len(filtered.Nodes) == 0 {
		t.Fatal("expected filtered graph to keep sample nodes")
	}
}

func TestParseRepositoryCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := jkdeps.ParseRepository(ctx, t.TempDir(), jkdeps.ParseOptions{})
	if err != context.Canceled {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := jkdeps.DefaultOptions()
	if opts.Parse.JavaGrammar != jkdeps.JavaGrammarDefault {
		t.Fatalf("JavaGrammar = %q, want %q", opts.Parse.JavaGrammar, jkdeps.JavaGrammarDefault)
	}
	if opts.Parse.JavaParseMode != jkdeps.JavaParseModeHeaderOnly {
		t.Fatalf("JavaParseMode = %q, want %q", opts.Parse.JavaParseMode, jkdeps.JavaParseModeHeaderOnly)
	}
	if opts.Parse.KotlinScripts != jkdeps.KotlinScriptsRegular {
		t.Fatalf("KotlinScripts = %q, want %q", opts.Parse.KotlinScripts, jkdeps.KotlinScriptsRegular)
	}
	if opts.Graph.GroupBy != jkdeps.GroupByPackage {
		t.Fatalf("GroupBy = %q, want %q", opts.Graph.GroupBy, jkdeps.GroupByPackage)
	}
}

func ExampleAnalyze() {
	root, err := os.MkdirTemp("", "jkdeps-example-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)

	writeExampleFile(root, "src/main/kotlin/sample/kt/KThing.kt", "package sample.kt\n\nclass KThing\n")
	writeExampleFile(root, "src/main/java/sample/JThing.java", "package sample;\n\nimport sample.kt.KThing;\n\npublic class JThing {}\n")

	report, err := jkdeps.Analyze(context.Background(), root, jkdeps.Options{})
	if err != nil {
		panic(err)
	}

	fmt.Printf("files=%d parsed=%d graph_nodes=%d unresolved=%d\n",
		report.Repository.TotalFiles,
		report.Repository.ParsedFiles,
		len(report.Graph.Nodes),
		len(report.Dependencies.UnresolvedImports),
	)

	// Output:
	// files=2 parsed=2 graph_nodes=2 unresolved=0
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	writeExampleFile(root, rel, content)
}

func writeExampleFile(root, rel, content string) {
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		panic(err)
	}
}
