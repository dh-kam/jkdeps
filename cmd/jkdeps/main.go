package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"

	"github.com/dh-kam/jkdeps/internal/cliutil"
	"github.com/dh-kam/jkdeps/internal/parser"
)

var topLevelCommands = []string{
	"smoke-parse",
	"roundtrip-check",
	"graph",
	"deps",
}

var topLevelCommandSet = cliutil.BuildCommandSet(topLevelCommands)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return 2
	}

	switch args[0] {
	case "smoke-parse":
		return runSmokeParse(args[1:])
	case "roundtrip-check":
		return runRoundTripCheck(args[1:])
	case "graph":
		return runGraph(args[1:])
	case "deps":
		return runDeps(args[1:])
	case "help", "-h", "--help":
		if len(args) > 1 && isTopLevelCommand(args[1]) {
			return run(append([]string{args[1], "--help"}, args[2:]...))
		}
		printUsage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printUsage(os.Stderr)
		return 2
	}
}

func isTopLevelCommand(candidate string) bool {
	_, ok := topLevelCommandSet[candidate]
	return ok
}

func runSmokeParse(args []string) int {
	fs := cliutil.NewFlagSet("smoke-parse", args)

	repoPath := fs.String("repo", ".", "Path to repository root")
	javaGrammar := fs.String("java-grammar", string(parser.JavaGrammarDefault), "Java grammar: java|java7|java8|java9|java11|java17|java20|java21|java25")
	workers := fs.Int("workers", runtime.NumCPU(), "Number of parser workers")
	maxErrors := fs.Int("max-errors", 20, "Max file errors to print")
	includeKTS := fs.Bool("include-kts", true, "Parse .kts files as Kotlin")
	fileTimeout := fs.Duration("file-timeout", 0, "Per-file parse timeout (for example 2s, 500ms; 0 disables timeout)")
	gcPercent := fs.Int("gc-percent", 300, "Temporary GOGC percent during parsing (0 keeps current runtime setting)")
	failOnError := fs.Bool("fail-on-error", true, "Exit with code 1 when parse failures exist")

	if ok, code := cliutil.ParseFlagSet(fs, args); !ok {
		return code
	}

	if err := startCLIProfiling(); err != nil {
		fmt.Fprintf(os.Stderr, "profile init failed: %v\n", err)
		return 1
	}
	defer stopCLIProfiling()

	summary, err := parser.ParseRepository(*repoPath, parser.ParseOptions{
		JavaGrammar:   parser.JavaGrammar(*javaGrammar),
		Workers:       *workers,
		MaxErrorFiles: *maxErrors,
		IncludeKTS:    *includeKTS,
		ParseTimeout:  *fileTimeout,
		GCPercent:     *gcPercent,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "smoke-parse failed: %v\n", err)
		return 1
	}

	printSummary(summary)
	return failOnErrorExitCode(*failOnError, summary.FailedFiles)
}

func printSummary(summary parser.Summary) {
	successRate := 100.0
	if summary.TotalFiles > 0 {
		successRate = (float64(summary.ParsedFiles) / float64(summary.TotalFiles)) * 100
	}

	cliutil.WriteSummaryLine(os.Stdout, "Root", "%s", summary.Root)
	cliutil.WriteSummaryLine(os.Stdout, "Java Grammar", "%s", summary.JavaGrammar)
	cliutil.WriteSummaryLine(os.Stdout, "Files", "total=%d java=%d kotlin=%d", summary.TotalFiles, summary.JavaFiles, summary.KotlinFiles)
	cliutil.WriteSummaryLine(os.Stdout, "Parse", "parsed=%d failed=%d success=%.2f%%", summary.ParsedFiles, summary.FailedFiles, successRate)
	cliutil.WriteSummaryLine(os.Stdout, "Duration", "%s", summary.Duration.Round(0))

	if len(summary.Failures) > 0 {
		sort.Slice(summary.Failures, func(i, j int) bool {
			return summary.Failures[i].Path < summary.Failures[j].Path
		})
		cliutil.WriteSectionHeader(os.Stdout, "Errors")
		for _, failure := range summary.Failures {
			fmt.Printf("- %s: %s\n", failure.Path, failure.Error)
		}
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Java-Kotlin Dependencies Analyzer (jkdeps)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  jkdeps smoke-parse [flags]")
	fmt.Fprintln(w, "  jkdeps roundtrip-check [flags]")
	fmt.Fprintln(w, "  jkdeps graph [flags]")
	fmt.Fprintln(w, "  jkdeps deps [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  smoke-parse   Parse all .java/.kt files in a repository and print a summary")
	fmt.Fprintln(w, "  roundtrip-check  Rebuild package/import headers and compare normalized source")
	fmt.Fprintln(w, "  graph         Build mixed Java/Kotlin dependency graph and emit JSON/HTML viewer")
	fmt.Fprintln(w, "  deps          Analyze file-level Java/Kotlin dependencies and unresolved imports")
}
