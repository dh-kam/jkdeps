package mixedgraph

import (
	"testing"
	"time"

	"github.com/dh-kam/jkdeps/internal/parser"
)

func TestRepositoryResultSlowestFilesOrdersAndLimits(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{
			{Path: "/repo/c.kt", Relative: "c.kt", Language: LangKotlin, Parsed: true, Duration: 150 * time.Millisecond},
			{Path: "/repo/a.java", Relative: "a.java", Language: LangJava, Parsed: false, Duration: 500 * time.Millisecond},
			{Path: "/repo/b.java", Relative: "b.java", Language: LangJava, Parsed: true, Duration: 250 * time.Millisecond},
			{Path: "/repo/d.java", Relative: "d.java", Language: LangJava, Parsed: true, Duration: 500 * time.Millisecond},
		},
	}

	got := result.SlowestFiles(3)
	if len(got) != 3 {
		t.Fatalf("len(SlowestFiles) = %d, want 3", len(got))
	}

	if got[0].Relative != "a.java" {
		t.Fatalf("got[0].Relative = %q, want a.java", got[0].Relative)
	}
	if got[1].Relative != "d.java" {
		t.Fatalf("got[1].Relative = %q, want d.java", got[1].Relative)
	}
	if got[2].Relative != "b.java" {
		t.Fatalf("got[2].Relative = %q, want b.java", got[2].Relative)
	}
	if got[0].Parsed {
		t.Fatalf("got[0].Parsed = true, want false")
	}
}

func TestRepositoryResultSlowestFilesHandlesZeroLimit(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{{Path: "/repo/a.java", Relative: "a.java", Duration: time.Second}},
	}

	if got := result.SlowestFiles(0); got != nil {
		t.Fatalf("SlowestFiles(0) = %v, want nil", got)
	}
}

func TestJavaParseModeValidation(t *testing.T) {
	if !JavaParseModeFull.IsValid() {
		t.Fatalf("JavaParseModeFull should be valid")
	}
	if !JavaParseModeHeaderOnly.IsValid() {
		t.Fatalf("JavaParseModeHeaderOnly should be valid")
	}
	if JavaParseMode("unknown").IsValid() {
		t.Fatalf("unexpected parse mode reported as valid")
	}
}

func TestParseOptionsWithDefaultsIncludesJavaParseMode(t *testing.T) {
	opts := ParseOptions{
		JavaGrammar: parser.JavaGrammar20,
	}

	got := opts.withDefaults()
	if got.JavaParseMode != JavaParseModeFull {
		t.Fatalf("withDefaults().JavaParseMode = %q, want %q", got.JavaParseMode, JavaParseModeFull)
	}
}
