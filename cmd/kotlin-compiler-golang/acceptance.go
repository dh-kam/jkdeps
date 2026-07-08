package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/dh-kam/jkdeps/internal/cliutil"
	"github.com/dh-kam/jkdeps/internal/flagutil"
	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

func runAcceptance(args []string) int {
	fs := cliutil.NewFlagSet("acceptance", args)

	repoPath := fs.String("repo", ".", "Repository root or directory to parse")
	workers := fs.Int("workers", 4, "Number of worker goroutines")
	maxErrors := fs.Int("max-errors-per-file", 10, "Maximum diagnostics per file")
	includeKTS := fs.Bool("include-kts", true, "Include .kts files")
	includeBuildScripts := fs.Bool("include-build-scripts", false, "Include build scripts (*.gradle.kts, settings.gradle.kts)")
	lenient := fs.Bool("lenient", false, "Treat syntax errors as non-fatal for file parse status")
	fileTimeout := fs.Duration("file-timeout", 0, "Per-file parse timeout (for example 2s, 500ms; 0 disables timeout)")
	parserBackend := fs.String("parser-backend", string(kcg.ParseBackendANTLR), "Parser backend: antlr | embeddable")
	printJSON := fs.Bool("json", false, "Print acceptance report as JSON")
	outPath := fs.String("out", "", "Write acceptance report JSON to this file")
	maxFailedFiles := fs.Int("max-failed-files", -1, "Fail when parsed failed files exceed this value (disabled when < 0)")
	maxUnresolvedImports := fs.Int("max-unresolved-imports", -1, "Fail when unresolved imports exceed this value (disabled when < 0)")
	maxFilesWithDiagnostics := fs.Int("max-files-with-diagnostics", -1, "Fail when files with diagnostics exceed this value (disabled when < 0)")
	maxTotalDiagnostics := fs.Int("max-total-diagnostics", -1, "Fail when total diagnostics exceed this value (disabled when < 0)")
	minResolvedRate := fs.Float64("min-resolved-rate", -1, "Fail when resolved import rate (%) is below this value (disabled when < 0)")
	maxParseFailureRate := fs.Float64("max-parse-failure-rate", -1, "Fail when parse failure rate (%) exceeds this value (disabled when < 0)")
	var inventoryPaths stringListFlag
	fs.Var(&inventoryPaths, "inventory", "Path to inventory JSON for external resolution (repeatable or comma-separated)")

	if ok, code := cliutil.ParseFlagSet(fs, args); !ok {
		return code
	}
	inventoryPaths = stringListFlag(flagutil.UniqueStrings(inventoryPaths))
	backend, err := parseParserBackend(*parserBackend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --parser-backend: %v\n", err)
		return 2
	}

	compiler := kcg.New(kcg.Config{
		Workers:             *workers,
		MaxErrorsPerFile:    *maxErrors,
		IncludeKTS:          *includeKTS,
		IncludeBuildScripts: *includeBuildScripts,
		LenientSyntax:       *lenient,
		ParseTimeout:        *fileTimeout,
		ParseBackend:        backend,
	})
	result, err := compiler.ParseRepository(*repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "acceptance failed: %v\n", err)
		return 1
	}

	externalIndex := kcg.NewExternalIndex()
	if len(inventoryPaths) > 0 {
		index, err := kcg.LoadExternalIndices(inventoryPaths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load inventory: %v\n", err)
			return 1
		}
		externalIndex = index
	}

	table := result.BuildSymbolTable()
	resolve := kcg.ResolveImportsWithExternal(result, table, externalIndex)
	graph := result.BuildWebGraph(externalIndex)
	report := kcg.BuildAcceptanceReport(result, table, resolve, graph)

	if *outPath != "" {
		if err := writeAcceptanceReport(*outPath, report); err != nil {
			fmt.Fprintf(os.Stderr, "write acceptance report: %v\n", err)
			return 1
		}
	}

	if *printJSON {
		if err := cliutil.WritePrettyJSON(os.Stdout, report); err != nil {
			fmt.Fprintf(os.Stderr, "encode acceptance report: %v\n", err)
			return 1
		}
	} else {
		printAcceptanceReport(report, inventoryPaths, *outPath)
	}

	if err := validateAcceptanceThresholds(
		report,
		*maxFailedFiles,
		*maxUnresolvedImports,
		*maxFilesWithDiagnostics,
		*maxTotalDiagnostics,
		*minResolvedRate,
		*maxParseFailureRate,
	); err != nil {
		fmt.Fprintf(os.Stderr, "acceptance gate failed: %v\n", err)
		return 1
	}
	return 0
}

func writeAcceptanceReport(path string, report kcg.AcceptanceReport) error {
	return cliutil.WritePrettyJSONFile(path, report)
}

func printAcceptanceReport(report kcg.AcceptanceReport, inventoryPaths []string, outPath string) {
	cliutil.WriteSummaryLine(os.Stdout, "Root", "%s", report.Root)
	cliutil.WriteSummaryLine(os.Stdout, "Parse", "total=%d parsed=%d failed=%d diag_files=%d diagnostics=%d success=%.2f%%",
		report.Parse.TotalFiles, report.Parse.ParsedFiles, report.Parse.FailedFiles, report.Parse.FilesWithDiagnostics, report.Parse.TotalDiagnostics, report.Parse.SuccessRate)
	cliutil.WriteSummaryLine(os.Stdout, "Resolve", "total=%d resolved=%d unresolved=%d rate=%.2f%%",
		report.Resolve.TotalImports, report.Resolve.ResolvedImports, report.Resolve.UnresolvedImports, report.Resolve.ResolvedRate)
	cliutil.WriteSummaryLine(os.Stdout, "Symbols", "total=%d packages=%d files=%d classes=%d interfaces=%d objects=%d functions=%d properties=%d typealiases=%d",
		report.Symbols.Total, report.Symbols.Packages, report.Symbols.Files, report.Symbols.Classes,
		report.Symbols.Interfaces, report.Symbols.Objects, report.Symbols.Functions, report.Symbols.Properties, report.Symbols.TypeAliases)
	cliutil.WriteSummaryLine(os.Stdout, "Graph", "nodes=%d edges=%d internal=%d external=%d unknown=%d",
		report.Graph.Nodes, report.Graph.Edges, report.Graph.InternalNodes, report.Graph.ExternalNodes, report.Graph.UnknownNodes)
	cliutil.WriteSummaryLine(os.Stdout, "Duration", "%s", report.Parse.ParseDuration.Round(0))
	if len(inventoryPaths) > 0 {
		cliutil.WriteSummaryLine(os.Stdout, "Inventory", "files=%d", len(inventoryPaths))
	}
	if outPath != "" {
		cliutil.WriteSummaryLine(os.Stdout, "Report", "%s", outPath)
	}

	if len(report.Resolve.UnresolvedByReason) == 0 {
		return
	}

	type reasonCount struct {
		reason string
		count  int
	}
	reasons := make([]reasonCount, 0, len(report.Resolve.UnresolvedByReason))
	for reason, count := range report.Resolve.UnresolvedByReason {
		reasons = append(reasons, reasonCount{reason: reason, count: count})
	}
	sort.Slice(reasons, func(i, j int) bool {
		if reasons[i].count != reasons[j].count {
			return reasons[i].count > reasons[j].count
		}
		return reasons[i].reason < reasons[j].reason
	})
	cliutil.WriteSectionHeader(os.Stdout, "Unresolved Reasons")
	for _, item := range reasons {
		fmt.Printf("- %s (%d)\n", item.reason, item.count)
	}
}

func validateAcceptanceThresholds(
	report kcg.AcceptanceReport,
	maxFailedFiles,
	maxUnresolvedImports,
	maxFilesWithDiagnostics,
	maxTotalDiagnostics int,
	minResolvedRate,
	maxParseFailureRate float64,
) error {
	if maxFailedFiles >= 0 && report.Parse.FailedFiles > maxFailedFiles {
		return fmt.Errorf("failed_files=%d > max_failed_files=%d", report.Parse.FailedFiles, maxFailedFiles)
	}
	if maxUnresolvedImports >= 0 && report.Resolve.UnresolvedImports > maxUnresolvedImports {
		return fmt.Errorf("unresolved_imports=%d > max_unresolved_imports=%d", report.Resolve.UnresolvedImports, maxUnresolvedImports)
	}
	if maxFilesWithDiagnostics >= 0 && report.Parse.FilesWithDiagnostics > maxFilesWithDiagnostics {
		return fmt.Errorf("files_with_diagnostics=%d > max_files_with_diagnostics=%d", report.Parse.FilesWithDiagnostics, maxFilesWithDiagnostics)
	}
	if maxTotalDiagnostics >= 0 && report.Parse.TotalDiagnostics > maxTotalDiagnostics {
		return fmt.Errorf("total_diagnostics=%d > max_total_diagnostics=%d", report.Parse.TotalDiagnostics, maxTotalDiagnostics)
	}
	if minResolvedRate >= 0 && report.Resolve.ResolvedRate < minResolvedRate {
		return fmt.Errorf("resolved_rate=%.2f%% < min_resolved_rate=%.2f%%", report.Resolve.ResolvedRate, minResolvedRate)
	}
	if maxParseFailureRate >= 0 && report.Parse.FailureRate > maxParseFailureRate {
		return fmt.Errorf("parse_failure_rate=%.2f%% > max_parse_failure_rate=%.2f%%", report.Parse.FailureRate, maxParseFailureRate)
	}
	return nil
}
