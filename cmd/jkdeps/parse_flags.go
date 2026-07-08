package main

import (
	"flag"
	"runtime"
	"time"

	"github.com/dh-kam/jkdeps/internal/mixedgraph"
	"github.com/dh-kam/jkdeps/internal/parser"
)

type mixedParseCommandFlags struct {
	repoPath            *string
	javaGrammar         *string
	javaParseMode       *string
	workers             *int
	maxErrors           *int
	topParseFiles       *int
	includeKTS          *bool
	includeBuildScripts *bool
	fileTimeout         *time.Duration
	lenient             *bool
	failOnError         *bool
}

func addMixedParseCommandFlags(fs *flag.FlagSet) mixedParseCommandFlags {
	return addMixedParseCommandFlagsWithJavaParseModeDefault(fs, mixedgraph.JavaParseModeFull)
}

func addMixedParseCommandFlagsWithJavaParseModeDefault(fs *flag.FlagSet, defaultMode mixedgraph.JavaParseMode) mixedParseCommandFlags {
	if !defaultMode.IsValid() {
		defaultMode = mixedgraph.JavaParseModeFull
	}
	return mixedParseCommandFlags{
		repoPath:            fs.String("repo", ".", "Path to repository root"),
		javaGrammar:         fs.String("java-grammar", string(parser.JavaGrammarDefault), "Java grammar: java|java7|java8|java9|java11|java17|java20|java21|java25"),
		javaParseMode:       fs.String("java-parse-mode", string(defaultMode), "Java parse mode: full|header-only"),
		workers:             fs.Int("workers", runtime.NumCPU(), "Number of parser workers"),
		maxErrors:           fs.Int("max-errors-per-file", 10, "Maximum diagnostics per file"),
		topParseFiles:       fs.Int("top-parse-files", 0, "Show the N slowest parsed files in summary output (and deps JSON)"),
		includeKTS:          fs.Bool("include-kts", true, "Parse .kts files as Kotlin"),
		includeBuildScripts: fs.Bool("include-build-scripts", false, "Include build scripts (*.gradle.kts, settings.gradle.kts)"),
		fileTimeout:         fs.Duration("file-timeout", 0, "Per-file parse timeout (for example 2s, 500ms; 0 disables timeout)"),
		lenient:             fs.Bool("lenient", false, "Treat syntax errors as non-fatal for file parse status"),
		failOnError:         fs.Bool("fail-on-error", false, "Exit with code 1 when parse failures exist"),
	}
}

func (f mixedParseCommandFlags) parseOptions() mixedgraph.ParseOptions {
	return mixedgraph.ParseOptions{
		JavaGrammar:         parser.JavaGrammar(*f.javaGrammar),
		JavaParseMode:       mixedgraph.JavaParseMode(*f.javaParseMode),
		Workers:             *f.workers,
		IncludeKTS:          *f.includeKTS,
		IncludeBuildScripts: *f.includeBuildScripts,
		MaxErrorsPerFile:    *f.maxErrors,
		LenientSyntax:       *f.lenient,
		ParseTimeout:        *f.fileTimeout,
	}
}

func (f mixedParseCommandFlags) parseRepository() (mixedgraph.RepositoryResult, error) {
	return parseMixedRepository(f)
}

func (f mixedParseCommandFlags) grammar() parser.JavaGrammar {
	return parser.JavaGrammar(*f.javaGrammar)
}

func (f mixedParseCommandFlags) repo() string {
	return *f.repoPath
}

func (f mixedParseCommandFlags) failOnErrorEnabled() bool {
	return *f.failOnError
}

func (f mixedParseCommandFlags) topParseFilesCount() int {
	return *f.topParseFiles
}
