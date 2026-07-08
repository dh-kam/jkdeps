package main

import (
	"fmt"
	"os"

	"github.com/dh-kam/jkdeps/internal/cliutil"
	"github.com/dh-kam/jkdeps/internal/mixedgraph"
	"github.com/dh-kam/jkdeps/internal/parser"
)

type dependencyOutput struct {
	Root                 string                       `json:"root"`
	JavaGrammar          parser.JavaGrammar           `json:"java_grammar"`
	JavaParseMode        mixedgraph.JavaParseMode     `json:"java_parse_mode"`
	GroupBy              mixedgraph.GroupBy           `json:"group_by"`
	TotalFiles           int                          `json:"total_files"`
	JavaFiles            int                          `json:"java_files"`
	KotlinFiles          int                          `json:"kotlin_files"`
	ParsedFiles          int                          `json:"parsed_files"`
	FailedFiles          int                          `json:"failed_files"`
	DurationSeconds      float64                      `json:"duration_seconds"`
	Graph                mixedgraph.Graph             `json:"graph"`
	FileDependencies     []mixedgraph.FileDependency  `json:"file_dependencies"`
	UnresolvedReferences []mixedgraph.FileDependency  `json:"unresolved_references,omitempty"`
	UnresolvedImports    []mixedgraph.FileDependency  `json:"unresolved_imports"`
	SlowParseFiles       []mixedgraph.ParseFileTiming `json:"slow_parse_files,omitempty"`
	DependencyCount      int                          `json:"dependency_count"`
	UnresolvedCount      int                          `json:"unresolved_count"`
	InventoryPackages    int                          `json:"inventory_packages"`
	InventorySymbols     int                          `json:"inventory_symbols"`
	InventoryFiles       int                          `json:"inventory_files"`
}

func runDeps(args []string) int {
	fs := cliutil.NewFlagSet("deps", args)

	parseFlags := addMixedParseCommandFlagsWithJavaParseModeDefault(fs, mixedgraph.JavaParseModeHeaderOnly)
	printJSON := fs.Bool("json", false, "Print dependency analysis as JSON")
	groupBy := fs.String("group-by", string(mixedgraph.GroupByPackage), "Graph grouping: package|dir")
	outPath := fs.String("out", "", "Optional output JSON path (overrides stdout when set)")
	var inventoryPaths stringListFlag
	fs.Var(&inventoryPaths, "inventory", "Path to external inventory JSON (repeatable or comma-separated)")

	if ok, code := cliutil.ParseFlagSet(fs, args); !ok {
		return code
	}

	group, err := parseGroupByFlag(*groupBy)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	result, err := parseFlags.parseRepository()
	if err != nil {
		fmt.Fprintf(os.Stderr, "deps failed: %v\n", err)
		return 1
	}

	inventoryPaths, external, err := loadExternalIndexFlags(inventoryPaths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load inventory: %v\n", err)
		return 1
	}

	graph := result.BuildGraph(external, group)
	dependencyReport := mixedgraph.BuildFileDependencyReport(result, external)

	output := dependencyOutput{
		Root:                 result.Root,
		JavaGrammar:          parseFlags.grammar(),
		JavaParseMode:        parseFlags.parseOptions().JavaParseMode,
		GroupBy:              group,
		TotalFiles:           result.TotalFiles,
		JavaFiles:            result.JavaFiles,
		KotlinFiles:          result.KotlinFiles,
		ParsedFiles:          result.ParsedFiles,
		FailedFiles:          result.FailedFiles,
		DurationSeconds:      result.Duration.Seconds(),
		Graph:                graph,
		FileDependencies:     dependencyReport.Dependencies,
		UnresolvedReferences: dependencyReport.UnresolvedReferences,
		UnresolvedImports:    dependencyReport.UnresolvedImports,
		SlowParseFiles:       result.SlowestFiles(parseFlags.topParseFilesCount()),
		DependencyCount:      dependencyReport.TotalDependencies,
		UnresolvedCount:      len(dependencyReport.UnresolvedReferences),
		InventoryFiles:       len(inventoryPaths),
		InventoryPackages:    len(external.Packages),
		InventorySymbols:     len(external.Symbols),
	}
	exitCode := failOnErrorExitCode(parseFlags.failOnErrorEnabled(), result.FailedFiles)

	// Handle JSON output
	if *printJSON || *outPath != "" {
		// Always print JSON to stdout if requested
		if *printJSON {
			if err := cliutil.WritePrettyJSON(os.Stdout, output); err != nil {
				fmt.Fprintf(os.Stderr, "encode dependencies: %v\n", err)
				return 1
			}
		}
		// Write to file if outPath is specified
		if *outPath != "" {
			if err := cliutil.WritePrettyJSONFile(*outPath, output); err != nil {
				fmt.Fprintf(os.Stderr, "write dependencies: %v\n", err)
				return 1
			}
		}
		return exitCode
	}

	writeMixedRepositorySummary(os.Stdout, result, parseFlags.grammar(), true)
	writeSlowParseFilesSummary(os.Stdout, result, parseFlags.topParseFilesCount())
	cliutil.WriteSummaryLine(os.Stdout, "Dependencies", "total=%d internal=%d external=%d unresolved=%d", dependencyReport.TotalDependencies, dependencyReport.InternalDependencies, dependencyReport.ExternalDependencies, len(dependencyReport.UnresolvedReferences))
	cliutil.WriteSummaryLine(os.Stdout, "Graph", "nodes=%d edges=%d", len(graph.Nodes), len(graph.Edges))
	cliutil.WriteSummaryLine(os.Stdout, "Output", "%s", func() string {
		if *outPath == "" {
			return "stdout"
		}
		return *outPath
	}())
	if len(dependencyReport.UnresolvedReferences) == 0 {
		return 0
	}

	limit := min(20, len(dependencyReport.UnresolvedReferences))
	cliutil.WriteSectionHeader(os.Stdout, fmt.Sprintf("Unresolved References (first %d)", limit))
	for _, dep := range dependencyReport.UnresolvedReferences[:limit] {
		fmt.Printf("- %s -> %s (%q)\n", dep.FilePath, dep.ToPackage, dep.ImportPath)
	}
	return exitCode
}
