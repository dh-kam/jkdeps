package main

import (
	"flag"
	"testing"
	"time"

	"github.com/dh-kam/jkdeps/internal/mixedgraph"
	"github.com/dh-kam/jkdeps/internal/parser"
)

func TestAddMixedParseCommandFlagsDefaults(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flags := addMixedParseCommandFlags(fs)

	if flags.repo() != "." {
		t.Fatalf("flags.repo() = %q, want .", flags.repo())
	}
	if flags.grammar() != parser.JavaGrammarDefault {
		t.Fatalf("flags.grammar() = %q, want %q", flags.grammar(), parser.JavaGrammarDefault)
	}
	if flags.failOnErrorEnabled() {
		t.Fatalf("flags.failOnErrorEnabled() = true, want false")
	}
	if flags.topParseFilesCount() != 0 {
		t.Fatalf("flags.topParseFilesCount() = %d, want 0", flags.topParseFilesCount())
	}

	opts := flags.parseOptions()
	if opts.JavaGrammar != parser.JavaGrammarDefault {
		t.Fatalf("opts.JavaGrammar = %q, want %q", opts.JavaGrammar, parser.JavaGrammarDefault)
	}
	if opts.JavaParseMode != mixedgraph.JavaParseModeFull {
		t.Fatalf("opts.JavaParseMode = %q, want %q", opts.JavaParseMode, mixedgraph.JavaParseModeFull)
	}
	if !opts.IncludeKTS {
		t.Fatalf("opts.IncludeKTS = false, want true")
	}
	if opts.IncludeBuildScripts {
		t.Fatalf("opts.IncludeBuildScripts = true, want false")
	}
	if opts.MaxErrorsPerFile != 10 {
		t.Fatalf("opts.MaxErrorsPerFile = %d, want 10", opts.MaxErrorsPerFile)
	}
}

func TestMixedParseCommandFlagsParseOptionsReflectValues(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flags := addMixedParseCommandFlags(fs)

	args := []string{
		"--repo", "/repo",
		"--java-grammar", "java17",
		"--java-parse-mode", "header-only",
		"--workers", "7",
		"--max-errors-per-file", "3",
		"--top-parse-files", "5",
		"--include-kts=false",
		"--include-build-scripts",
		"--file-timeout", "250ms",
		"--lenient",
		"--fail-on-error",
	}
	if err := fs.Parse(args); err != nil {
		t.Fatalf("fs.Parse(...) = %v", err)
	}

	opts := flags.parseOptions()
	if flags.repo() != "/repo" {
		t.Fatalf("flags.repo() = %q, want /repo", flags.repo())
	}
	if flags.grammar() != parser.JavaGrammar("java17") {
		t.Fatalf("flags.grammar() = %q, want java17", flags.grammar())
	}
	if !flags.failOnErrorEnabled() {
		t.Fatalf("flags.failOnErrorEnabled() = false, want true")
	}
	if flags.topParseFilesCount() != 5 {
		t.Fatalf("flags.topParseFilesCount() = %d, want 5", flags.topParseFilesCount())
	}
	if opts.Workers != 7 {
		t.Fatalf("opts.Workers = %d, want 7", opts.Workers)
	}
	if opts.JavaParseMode != mixedgraph.JavaParseModeHeaderOnly {
		t.Fatalf("opts.JavaParseMode = %q, want %q", opts.JavaParseMode, mixedgraph.JavaParseModeHeaderOnly)
	}
	if opts.MaxErrorsPerFile != 3 {
		t.Fatalf("opts.MaxErrorsPerFile = %d, want 3", opts.MaxErrorsPerFile)
	}
	if opts.IncludeKTS {
		t.Fatalf("opts.IncludeKTS = true, want false")
	}
	if !opts.IncludeBuildScripts {
		t.Fatalf("opts.IncludeBuildScripts = false, want true")
	}
	if opts.ParseTimeout != 250*time.Millisecond {
		t.Fatalf("opts.ParseTimeout = %s, want 250ms", opts.ParseTimeout)
	}
	if !opts.LenientSyntax {
		t.Fatalf("opts.LenientSyntax = false, want true")
	}
}

func TestAddMixedParseCommandFlagsCanCustomizeJavaParseModeDefault(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flags := addMixedParseCommandFlagsWithJavaParseModeDefault(fs, mixedgraph.JavaParseModeHeaderOnly)

	opts := flags.parseOptions()
	if opts.JavaParseMode != mixedgraph.JavaParseModeHeaderOnly {
		t.Fatalf("opts.JavaParseMode = %q, want %q", opts.JavaParseMode, mixedgraph.JavaParseModeHeaderOnly)
	}
}
