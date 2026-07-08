package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dh-kam/jkdeps/internal/cliutil"
	"github.com/dh-kam/jkdeps/internal/flagutil"
	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

var topLevelCommands = []string{
	"parse",
	"symbols",
	"deps",
	"resolve",
	"graph",
	"compile",
	"acceptance",
}

var topLevelCommandSet = cliutil.BuildCommandSet(topLevelCommands)

type stringListFlag = flagutil.StringListFlag

type stringListNoSplitFlag = flagutil.StringListNoSplitFlag

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return 2
	}

	args = normalizeHelpInvocationArgs(args)

	switch args[0] {
	case "parse":
		return runParse(args[1:])
	case "symbols":
		return runSymbols(args[1:])
	case "deps":
		return runDeps(args[1:])
	case "resolve":
		return runResolve(args[1:])
	case "graph":
		return runGraph(args[1:])
	case "compile":
		return runCompile(args[1:])
	case "acceptance":
		return runAcceptance(args[1:])
	case "help", "-h", "--help":
		if len(args) > 1 {
			if isTopLevelCommand(args[1]) {
				subcommand := append([]string{args[1]}, "--help")
				if len(args) > 2 {
					subcommand = append(subcommand, args[2:]...)
				}
				return run(subcommand)
			}
			return run([]string{args[1]})
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

func isHelpAlias(candidate string) bool {
	return candidate == "help" || candidate == "-h" || candidate == "--help"
}

func normalizeHelpInvocationArgs(args []string) []string {
	if len(args) < 2 {
		return args
	}

	if !isHelpAlias(args[0]) {
		return args
	}

	if !isHelpAlias(args[1]) {
		return args
	}

	if len(args) >= 3 && isTopLevelCommand(args[2]) {
		return append([]string{args[2], "--help"}, args[3:]...)
	}

	return args[1:2]
}

func runParse(args []string) int {
	fs := cliutil.NewFlagSet("parse", args)

	repoPath := fs.String("repo", ".", "Repository root or directory to parse")
	workers := fs.Int("workers", 4, "Number of worker goroutines")
	maxErrors := fs.Int("max-errors-per-file", 10, "Maximum diagnostics per file")
	includeKTS := fs.Bool("include-kts", true, "Include .kts files")
	includeBuildScripts := fs.Bool("include-build-scripts", false, "Include build scripts (*.gradle.kts, settings.gradle.kts)")
	lenient := fs.Bool("lenient", false, "Treat syntax errors as non-fatal for file parse status")
	fileTimeout := fs.Duration("file-timeout", 0, "Per-file parse timeout (for example 2s, 500ms; 0 disables timeout)")
	parserBackend := fs.String("parser-backend", string(kcg.ParseBackendANTLR), "Parser backend: antlr | embeddable")
	printJSON := fs.Bool("json", false, "Print full result as JSON")
	failOnError := fs.Bool("fail-on-error", true, "Exit with code 1 when failures exist")

	if ok, code := cliutil.ParseFlagSet(fs, args); !ok {
		return code
	}
	backend, err := parseParserBackend(*parserBackend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --parser-backend: %v\n", err)
		return 2
	}

	compiler := createCompiler(*workers, *maxErrors, *includeKTS, *includeBuildScripts, *lenient, *fileTimeout, backend)

	result, err := compiler.ParseRepository(*repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse failed: %v\n", err)
		return 1
	}

	if *printJSON {
		if err := cliutil.WritePrettyJSON(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "encode result: %v\n", err)
			return 1
		}
	} else {
		printSummary(result)
	}

	if *failOnError && result.FailedFiles > 0 {
		return 1
	}
	return 0
}

func runSymbols(args []string) int {
	fs := cliutil.NewFlagSet("symbols", args)

	repoPath := fs.String("repo", ".", "Repository root or directory to parse")
	workers := fs.Int("workers", 4, "Number of worker goroutines")
	maxErrors := fs.Int("max-errors-per-file", 10, "Maximum diagnostics per file")
	includeKTS := fs.Bool("include-kts", true, "Include .kts files")
	includeBuildScripts := fs.Bool("include-build-scripts", false, "Include build scripts (*.gradle.kts, settings.gradle.kts)")
	lenient := fs.Bool("lenient", false, "Treat syntax errors as non-fatal for file parse status")
	fileTimeout := fs.Duration("file-timeout", 0, "Per-file parse timeout (for example 2s, 500ms; 0 disables timeout)")
	parserBackend := fs.String("parser-backend", string(kcg.ParseBackendANTLR), "Parser backend: antlr | embeddable")
	printJSON := fs.Bool("json", false, "Print full symbol table as JSON")

	if ok, code := cliutil.ParseFlagSet(fs, args); !ok {
		return code
	}
	backend, err := parseParserBackend(*parserBackend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --parser-backend: %v\n", err)
		return 2
	}

	compiler := createCompiler(*workers, *maxErrors, *includeKTS, *includeBuildScripts, *lenient, *fileTimeout, backend)
	result, err := compiler.ParseRepository(*repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "symbols failed: %v\n", err)
		return 1
	}

	table := result.BuildSymbolTable()
	if *printJSON {
		if err := cliutil.WritePrettyJSON(os.Stdout, table); err != nil {
			fmt.Fprintf(os.Stderr, "encode symbols: %v\n", err)
			return 1
		}
		return 0
	}

	counts := table.CountByKind()
	cliutil.WriteSummaryLine(os.Stdout, "Root", "%s", result.Root)
	cliutil.WriteSummaryLine(os.Stdout, "Symbols", "total=%d", len(table.Symbols))
	cliutil.WriteSummaryLine(os.Stdout, "Kinds", "package=%d file=%d class=%d interface=%d object=%d function=%d property=%d typealias=%d",
		counts[kcg.SymbolPackage],
		counts[kcg.SymbolFile],
		counts[kcg.SymbolClass],
		counts[kcg.SymbolInterface],
		counts[kcg.SymbolObject],
		counts[kcg.SymbolFunction],
		counts[kcg.SymbolProperty],
		counts[kcg.SymbolTypeAlias],
	)
	cliutil.WriteSummaryLine(os.Stdout, "Parse", "parsed=%d failed=%d", result.ParsedFiles, result.FailedFiles)
	cliutil.WriteSummaryLine(os.Stdout, "Duration", "%s", result.Duration.Round(0))
	return 0
}

func runDeps(args []string) int {
	fs := cliutil.NewFlagSet("deps", args)

	repoPath := fs.String("repo", ".", "Repository root or directory to parse")
	workers := fs.Int("workers", 4, "Number of worker goroutines")
	maxErrors := fs.Int("max-errors-per-file", 10, "Maximum diagnostics per file")
	includeKTS := fs.Bool("include-kts", true, "Include .kts files")
	includeBuildScripts := fs.Bool("include-build-scripts", false, "Include build scripts (*.gradle.kts, settings.gradle.kts)")
	lenient := fs.Bool("lenient", false, "Treat syntax errors as non-fatal for file parse status")
	fileTimeout := fs.Duration("file-timeout", 0, "Per-file parse timeout (for example 2s, 500ms; 0 disables timeout)")
	parserBackend := fs.String("parser-backend", string(kcg.ParseBackendANTLR), "Parser backend: antlr | embeddable")
	printJSON := fs.Bool("json", false, "Print dependency graph as JSON")

	if ok, code := cliutil.ParseFlagSet(fs, args); !ok {
		return code
	}
	backend, err := parseParserBackend(*parserBackend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --parser-backend: %v\n", err)
		return 2
	}

	compiler := createCompiler(*workers, *maxErrors, *includeKTS, *includeBuildScripts, *lenient, *fileTimeout, backend)
	result, err := compiler.ParseRepository(*repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "deps failed: %v\n", err)
		return 1
	}

	graph := result.BuildDependencyGraph()
	if *printJSON {
		if err := cliutil.WritePrettyJSON(os.Stdout, graph); err != nil {
			fmt.Fprintf(os.Stderr, "encode deps graph: %v\n", err)
			return 1
		}
		return 0
	}

	cliutil.WriteSummaryLine(os.Stdout, "Root", "%s", result.Root)
	cliutil.WriteSummaryLine(os.Stdout, "Packages", "%d", len(graph.Nodes))
	cliutil.WriteSummaryLine(os.Stdout, "Edges", "%d", len(graph.Edges))
	cliutil.WriteSummaryLine(os.Stdout, "Parse", "parsed=%d failed=%d", result.ParsedFiles, result.FailedFiles)
	cliutil.WriteSummaryLine(os.Stdout, "Duration", "%s", result.Duration.Round(0))

	if len(graph.Edges) == 0 {
		return 0
	}
	nodeNameByID := make(map[int]string, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodeNameByID[node.ID] = node.Name
	}

	edges := make([]kcg.DependencyEdge, len(graph.Edges))
	copy(edges, graph.Edges)
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Count != edges[j].Count {
			return edges[i].Count > edges[j].Count
		}
		if edges[i].FromID != edges[j].FromID {
			return edges[i].FromID < edges[j].FromID
		}
		return edges[i].ToID < edges[j].ToID
	})

	limit := min(10, len(edges))
	cliutil.WriteSectionHeader(os.Stdout, "Top Edges")
	for i := range limit {
		edge := edges[i]
		fmt.Printf("- %s -> %s (%d)\n", nodeNameByID[edge.FromID], nodeNameByID[edge.ToID], edge.Count)
	}
	return 0
}

func runResolve(args []string) int {
	fs := cliutil.NewFlagSet("resolve", args)

	repoPath := fs.String("repo", ".", "Repository root or directory to parse")
	workers := fs.Int("workers", 4, "Number of worker goroutines")
	maxErrors := fs.Int("max-errors-per-file", 10, "Maximum diagnostics per file")
	includeKTS := fs.Bool("include-kts", true, "Include .kts files")
	includeBuildScripts := fs.Bool("include-build-scripts", false, "Include build scripts (*.gradle.kts, settings.gradle.kts)")
	lenient := fs.Bool("lenient", false, "Treat syntax errors as non-fatal for file parse status")
	fileTimeout := fs.Duration("file-timeout", 0, "Per-file parse timeout (for example 2s, 500ms; 0 disables timeout)")
	parserBackend := fs.String("parser-backend", string(kcg.ParseBackendANTLR), "Parser backend: antlr | embeddable")
	printJSON := fs.Bool("json", false, "Print import resolution report as JSON")
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

	compiler := createCompiler(*workers, *maxErrors, *includeKTS, *includeBuildScripts, *lenient, *fileTimeout, backend)
	result, err := compiler.ParseRepository(*repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve failed: %v\n", err)
		return 1
	}

	externalIndex, err := loadExternalIndices(inventoryPaths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	report := kcg.ResolveImportsWithExternal(result, result.BuildSymbolTable(), externalIndex)
	if *printJSON {
		if err := cliutil.WritePrettyJSON(os.Stdout, report); err != nil {
			fmt.Fprintf(os.Stderr, "encode resolution report: %v\n", err)
			return 1
		}
		return 0
	}

	resolutionRate := 100.0
	if report.TotalImports > 0 {
		resolutionRate = (float64(report.ResolvedImports) / float64(report.TotalImports)) * 100
	}
	cliutil.WriteSummaryLine(os.Stdout, "Root", "%s", result.Root)
	cliutil.WriteSummaryLine(os.Stdout, "Imports", "total=%d resolved=%d unresolved=%d", report.TotalImports, report.ResolvedImports, report.UnresolvedImports)
	if len(externalIndex.Packages) > 0 || len(externalIndex.Symbols) > 0 {
		cliutil.WriteSummaryLine(os.Stdout, "Inventory", "files=%d packages=%d symbols=%d", len(inventoryPaths), len(externalIndex.Packages), len(externalIndex.Symbols))
	}
	cliutil.WriteSummaryLine(os.Stdout, "Resolved", "%.2f%%", resolutionRate)
	cliutil.WriteSummaryLine(os.Stdout, "Parse", "parsed=%d failed=%d", result.ParsedFiles, result.FailedFiles)
	cliutil.WriteSummaryLine(os.Stdout, "Duration", "%s", result.Duration.Round(0))

	if report.UnresolvedImports == 0 {
		return 0
	}

	unresolved := make([]kcg.ImportResolution, 0, report.UnresolvedImports)
	for _, item := range report.Items {
		if !item.Resolved {
			unresolved = append(unresolved, item)
		}
	}
	sort.Slice(unresolved, func(i, j int) bool {
		if unresolved[i].FilePath != unresolved[j].FilePath {
			return unresolved[i].FilePath < unresolved[j].FilePath
		}
		if unresolved[i].Reason != unresolved[j].Reason {
			return unresolved[i].Reason < unresolved[j].Reason
		}
		return unresolved[i].ImportPath < unresolved[j].ImportPath
	})

	limit := min(10, len(unresolved))
	cliutil.WriteSectionHeader(os.Stdout, "Unresolved Imports")
	for i := range limit {
		item := unresolved[i]
		fmt.Printf("- %s :: %s (%s)\n", item.FilePath, item.ImportPath, item.Reason)
	}
	return 0
}

func runGraph(args []string) int {
	fs := cliutil.NewFlagSet("graph", args)

	repoPath := fs.String("repo", ".", "Repository root or directory to parse")
	workers := fs.Int("workers", 4, "Number of worker goroutines")
	maxErrors := fs.Int("max-errors-per-file", 10, "Maximum diagnostics per file")
	includeKTS := fs.Bool("include-kts", true, "Include .kts files")
	includeBuildScripts := fs.Bool("include-build-scripts", false, "Include build scripts (*.gradle.kts, settings.gradle.kts)")
	lenient := fs.Bool("lenient", false, "Treat syntax errors as non-fatal for file parse status")
	fileTimeout := fs.Duration("file-timeout", 0, "Per-file parse timeout (for example 2s, 500ms; 0 disables timeout)")
	parserBackend := fs.String("parser-backend", string(kcg.ParseBackendANTLR), "Parser backend: antlr | embeddable")
	printJSON := fs.Bool("json", false, "Print web graph as JSON")
	outPath := fs.String("out", "jkdeps-graph", "Output path base (or explicit .html/.json path)")
	var inventoryPaths stringListFlag
	fs.Var(&inventoryPaths, "inventory", "Path to inventory JSON for external resolution (repeatable or comma-separated)")

	if ok, code := cliutil.ParseFlagSet(fs, args); !ok {
		return code
	}
	backend, err := parseParserBackend(*parserBackend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --parser-backend: %v\n", err)
		return 2
	}
	inventoryPaths = stringListFlag(flagutil.UniqueStrings(inventoryPaths))

	compiler := createCompiler(*workers, *maxErrors, *includeKTS, *includeBuildScripts, *lenient, *fileTimeout, backend)
	result, err := compiler.ParseRepository(*repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "graph failed: %v\n", err)
		return 1
	}

	externalIndex, err := loadExternalIndices(inventoryPaths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	graph := result.BuildWebGraph(externalIndex)
	if *printJSON {
		if err := cliutil.WritePrettyJSON(os.Stdout, graph); err != nil {
			fmt.Fprintf(os.Stderr, "encode web graph: %v\n", err)
			return 1
		}
		return 0
	}

	htmlPath, jsonPath := cliutil.GraphOutputPaths(*outPath, "jkdeps-graph")
	if err := writeGraphArtifacts(htmlPath, jsonPath, graph); err != nil {
		fmt.Fprintf(os.Stderr, "write graph artifacts: %v\n", err)
		return 1
	}

	cliutil.WriteSummaryLine(os.Stdout, "Root", "%s", graph.Root)
	cliutil.WriteSummaryLine(os.Stdout, "Nodes", "%d", len(graph.Nodes))
	cliutil.WriteSummaryLine(os.Stdout, "Edges", "%d", len(graph.Edges))
	if len(inventoryPaths) > 0 {
		cliutil.WriteSummaryLine(os.Stdout, "Inventory", "files=%d packages=%d symbols=%d", len(inventoryPaths), len(externalIndex.Packages), len(externalIndex.Symbols))
	}
	cliutil.WriteSummaryLine(os.Stdout, "Parse", "parsed=%d failed=%d", result.ParsedFiles, result.FailedFiles)
	cliutil.WriteSummaryLine(os.Stdout, "Duration", "%s", result.Duration.Round(0))
	cliutil.WriteSummaryLine(os.Stdout, "Output", "%s", jsonPath)
	cliutil.WriteSummaryLine(os.Stdout, "Viewer", "%s", htmlPath)
	return 0
}

func runCompile(args []string) int {
	fs := cliutil.NewFlagSet("compile", args)

	repoPath := fs.String("repo", ".", "Repository root to compile")
	outPath := fs.String("out", "./out", "Compiler output path")
	var sourcePaths stringListNoSplitFlag
	var excludePaths stringListFlag
	kotlinc := fs.String("kotlinc", "kotlinc", "Kotlin compiler executable (default: kotlinc)")
	includeKTS := fs.Bool("include-kts", true, "Include .kts files")
	includeBuildScripts := fs.Bool("include-build-scripts", false, "Include build scripts (*.gradle.kts, settings.gradle.kts)")
	includeRuntime := fs.Bool("include-runtime", false, "Forward -include-runtime to kotlinc")
	jvmTarget := fs.String("jvm-target", "", "Forward -jvm-target")
	failOnError := fs.Bool("fail-on-error", true, "Exit with code 1 when compilation fails")
	dryRun := fs.Bool("dry-run", false, "Print the kotlinc command and return without executing")
	listSources := fs.Bool("list-sources", false, "List selected Kotlin sources and return without compilation")
	var argFiles stringListNoSplitFlag
	var classpath stringListFlag
	var plugins stringListFlag
	var compilerArgs stringListNoSplitFlag
	fs.Var(&classpath, "classpath", "Classpath entry (repeatable or comma-separated, resolved against --repo)")
	fs.Var(&sourcePaths, "source", "Kotlin source file or directory (repeatable, resolved against --repo)")
	fs.Var(&excludePaths, "exclude", "Path prefix to exclude from source collection (repeatable or comma-separated, resolved against --repo)")
	fs.Var(&plugins, "plugin", "Path to a compiler plugin jar, can be repeated (resolved against --repo)")
	fs.Var(&compilerArgs, "arg", "Extra arg to pass to kotlinc")
	fs.Var(&argFiles, "arg-file", "Path to file containing one kotlinc argument per line (repeated, resolved against --repo)")

	if ok, code := cliutil.ParseFlagSet(fs, args); !ok {
		return code
	}

	repoRoot := strings.TrimSpace(*repoPath)
	if repoRoot == "" {
		repoRoot = "."
	}
	repoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve repo path %q: %v\n", *repoPath, err)
		return 1
	}

	argFiles, err = resolvePathsForCompile(argFiles, repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve arg-file paths: %v\n", err)
		return 1
	}
	extraArgs, err := collectExtraCompilerArgs(argFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read arg-file: %v\n", err)
		return 1
	}
	compilerArgs = append(compilerArgs, extraArgs...)
	passthroughArgs := fs.Args()

	sourcePaths, err = resolvePathsForCompile(sourcePaths, repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve source paths: %v\n", err)
		return 1
	}
	excludePaths, err = resolvePathsForCompile(excludePaths, repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve exclude paths: %v\n", err)
		return 1
	}
	classpath, err = resolvePathsForCompile(classpath, repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve classpath paths: %v\n", err)
		return 1
	}
	plugins, err = resolvePathsForCompile(plugins, repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve plugin paths: %v\n", err)
		return 1
	}

	sourcePaths = flagutil.UniqueStrings(sourcePaths)
	classpath = flagutil.UniqueStrings(classpath)
	excludePaths = normalizeExcludePaths(excludePaths)
	plugins = flagutil.UniqueStrings(plugins)
	var files []string
	if len(sourcePaths) > 0 {
		files, err = collectKotlinFilesForCompilePaths(sourcePaths, excludePaths, *includeKTS, *includeBuildScripts)
	} else {
		files, err = collectKotlinFilesForCompile(repoRoot, excludePaths, *includeKTS, *includeBuildScripts)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect kotlin files: %v\n", err)
		return 1
	}
	if len(files) == 0 {
		sourceDesc := repoRoot
		if len(sourcePaths) > 0 {
			sourceDesc = strings.Join(sourcePaths, ",")
		}
		fmt.Fprintf(os.Stderr, "no Kotlin sources found in %s\n", sourceDesc)
		return 1
	}
	if *listSources {
		fmt.Printf("sources=%d\n", len(files))
		for _, source := range files {
			fmt.Println(source)
		}
		return 0
	}

	if outDir := filepath.Dir(*outPath); outDir != "." && outDir != "" {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "create output directory %q: %v\n", outDir, err)
			return 1
		}
	}

	argsOut := make([]string, 0, 16+len(files)+len(compilerArgs)+len(plugins)*2)
	if *jvmTarget != "" {
		argsOut = append(argsOut, "-jvm-target", *jvmTarget)
	}
	if *includeRuntime {
		argsOut = append(argsOut, "-include-runtime")
	}
	if len(classpath) > 0 {
		argsOut = append(argsOut, "-cp", strings.Join(classpath, string(filepath.ListSeparator)))
	}
	for _, pluginPath := range plugins {
		pluginPath = strings.TrimSpace(pluginPath)
		if pluginPath == "" {
			continue
		}
		argsOut = append(argsOut, "-Xplugin="+pluginPath)
	}
	argsOut = append(argsOut, "-d", *outPath)
	argsOut = append(argsOut, files...)
	argsOut = append(argsOut, compilerArgs...)
	if len(passthroughArgs) > 0 {
		argsOut = append(argsOut, passthroughArgs...)
	}

	if *dryRun {
		fmt.Printf("kotlin-compiler-golang compile -> %s\n", *kotlinc)
		fmt.Printf("%s\n", quoteCommandArgs(*kotlinc, argsOut))
		return 0
	}

	cmd := exec.Command(*kotlinc, argsOut...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if !*failOnError {
			fmt.Fprintf(os.Stderr, "compile command failed: %v\n", err)
			return 0
		}
		fmt.Fprintf(os.Stderr, "compile command failed: %v\n", err)
		return 1
	}

	fmt.Printf("Kotlin compile completed: %d files -> %s (compiler=%s)\n", len(files), *outPath, *kotlinc)
	return 0
}

func resolvePathsForCompile(paths []string, repoRoot string) ([]string, error) {
	out := make([]string, 0, len(paths))
	for _, pathValue := range paths {
		resolved, err := resolvePathAgainstRepo(pathValue, repoRoot)
		if err != nil {
			return nil, err
		}
		resolved = strings.TrimSpace(resolved)
		if resolved == "" {
			continue
		}
		out = append(out, resolved)
	}
	return out, nil
}

func resolvePathAgainstRepo(pathValue string, repoRoot string) (string, error) {
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" {
		return "", nil
	}
	if filepath.IsAbs(pathValue) {
		return filepath.Clean(pathValue), nil
	}
	if repoRoot == "" {
		return filepath.Abs(pathValue)
	}
	return filepath.Clean(filepath.Join(repoRoot, pathValue)), nil
}

func collectKotlinFilesForCompile(root string, excludePaths []string, includeKTS bool, includeBuildScripts bool) ([]string, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	files, err := collectKotlinFilesForCompilePaths([]string{root}, excludePaths, includeKTS, includeBuildScripts)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func collectKotlinFilesForCompilePaths(paths []string, excludePaths []string, includeKTS bool, includeBuildScripts bool) ([]string, error) {
	files := make([]string, 0, 1024)
	seen := map[string]struct{}{}
	excludeMap := make(map[string]struct{}, len(excludePaths))
	for _, raw := range excludePaths {
		cleaned := strings.TrimSpace(raw)
		if cleaned == "" {
			continue
		}
		excludeMap[filepath.Clean(cleaned)] = struct{}{}
	}

	for _, root := range paths {
		if strings.TrimSpace(root) == "" {
			continue
		}
		root = filepath.Clean(root)
		info, err := os.Stat(root)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			if isKotlinSourcePath(root, includeKTS, includeBuildScripts) {
				if !filepath.IsAbs(root) {
					root, err = filepath.Abs(root)
					if err != nil {
						return nil, err
					}
				}
				if isExcludedPath(root, excludeMap) {
					continue
				}
				seen[root] = struct{}{}
			}
			continue
		}

		err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				name := d.Name()
				if isExcludedPath(path, excludeMap) {
					return filepath.SkipDir
				}
				if isBuildArtifactDir(name) {
					return filepath.SkipDir
				}
				return nil
			}
			if !isKotlinSourcePath(path, includeKTS, includeBuildScripts) {
				return nil
			}
			absPath := path
			if !filepath.IsAbs(absPath) {
				absPath, err = filepath.Abs(absPath)
				if err != nil {
					return err
				}
			}
			if isExcludedPath(absPath, excludeMap) {
				return nil
			}
			seen[absPath] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	for path := range seen {
		files = append(files, path)
	}
	sort.Strings(files)
	return files, nil
}

func isBuildArtifactDir(name string) bool {
	switch name {
	case ".git", "build", "out", "target", "node_modules":
		return true
	default:
		return false
	}
}

func normalizeExcludePaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, path := range flagutil.UniqueStrings(paths) {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		path = filepath.Clean(path)
		if !filepath.IsAbs(path) {
			if absPath, err := filepath.Abs(path); err == nil {
				path = absPath
			}
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}

func isExcludedPath(path string, excludeMap map[string]struct{}) bool {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		if absPath, err := filepath.Abs(path); err == nil {
			path = absPath
		}
	}
	for prefix := range excludeMap {
		if prefix == "" {
			continue
		}
		if path == prefix {
			return true
		}
		if strings.HasPrefix(path+string(filepath.Separator), prefix+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func isKotlinSourcePath(path string, includeKTS bool, includeBuildScripts bool) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".kt" {
		return true
	}
	if ext != ".kts" {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	isBuildScript := base == "build.gradle.kts" || base == "settings.gradle.kts"
	if isBuildScript {
		return includeBuildScripts
	}
	return includeKTS
}

func collectExtraCompilerArgs(pathList []string) ([]string, error) {
	args := make([]string, 0, 16)
	for _, path := range pathList {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open %q: %w", path, err)
		}

		reader := bufio.NewReader(f)
		for {
			line, readErr := reader.ReadString('\n')
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "\ufeff")
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				if readErr != nil {
					if errors.Is(readErr, io.EOF) {
						break
					}
					_ = f.Close()
					return nil, fmt.Errorf("read %q: %w", path, readErr)
				}
				continue
			}
			tokens, err := splitArgLine(line)
			if err != nil {
				_ = f.Close()
				return nil, fmt.Errorf("parse %q: %w", path, err)
			}
			for _, token := range tokens {
				args = append(args, token)
			}
			if readErr != nil {
				if errors.Is(readErr, io.EOF) {
					break
				}
				_ = f.Close()
				return nil, fmt.Errorf("read %q: %w", path, readErr)
			}
		}
		if err := f.Close(); err != nil {
			return nil, fmt.Errorf("close %q: %w", path, err)
		}
	}
	return args, nil
}

func splitArgLine(line string) ([]string, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	isSpace := func(c byte) bool {
		return c == ' ' || c == '\t' || c == '\r' || c == '\n'
	}

	var (
		tokens  = make([]string, 0, 4)
		current strings.Builder
		inToken bool
		inQuote byte
		escaped bool
	)

	flush := func() {
		if !inToken {
			return
		}
		tokens = append(tokens, current.String())
		current.Reset()
		inToken = false
	}

	addChar := func(c byte) {
		current.WriteByte(c)
		inToken = true
	}

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if escaped {
			addChar(ch)
			escaped = false
			continue
		}

		if inQuote == 0 {
			switch ch {
			case '\\':
				escaped = true
				inToken = true
			case '\'', '"':
				inQuote = ch
				inToken = true
			case '#':
				if !inToken {
					return tokens, nil
				}
				addChar(ch)
			case ' ', '\t', '\r', '\n':
				flush()
				for i+1 < len(line) && isSpace(line[i+1]) {
					i++
				}
				if i+1 < len(line) && line[i+1] == '#' {
					return tokens, nil
				}
			default:
				addChar(ch)
			}
			continue
		}

		if ch == inQuote {
			inQuote = 0
			continue
		}

		if inQuote == '"' && ch == '\\' {
			if i+1 >= len(line) {
				addChar('\\')
				continue
			}
			next := line[i+1]
			switch next {
			case '\\', '"':
				addChar(next)
				i++
				continue
			default:
				addChar(ch)
			}
			continue
		}

		addChar(ch)
	}

	if escaped {
		addChar('\\')
	}
	if inQuote != 0 {
		return nil, fmt.Errorf("unterminated quote in %q", line)
	}
	flush()
	return tokens, nil
}

func quoteCommandArgs(executable string, args []string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, strconv.Quote(executable))
	for _, arg := range args {
		parts = append(parts, strconv.Quote(arg))
	}
	return strings.Join(parts, " ")
}

func parseParserBackend(value string) (kcg.ParseBackend, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(kcg.ParseBackendANTLR), string(kcg.ParseBackendEmbeddable):
		return kcg.ParseBackendFromString(strings.ToLower(strings.TrimSpace(value))), nil
	default:
		return "", fmt.Errorf("invalid parser backend %q (expected antlr or embeddable)", value)
	}
}

// createCompiler constructs a new Compiler with the given configuration.
// This centralizes the repeated compiler initialization pattern used across command handlers.
func createCompiler(workers, maxErrors int, includeKTS, includeBuildScripts, lenient bool,
	fileTimeout time.Duration, backend kcg.ParseBackend) *kcg.Compiler {
	return kcg.New(kcg.Config{
		Workers:             workers,
		MaxErrorsPerFile:    maxErrors,
		IncludeKTS:          includeKTS,
		IncludeBuildScripts: includeBuildScripts,
		LenientSyntax:       lenient,
		ParseTimeout:        fileTimeout,
		ParseBackend:        backend,
	})
}

// loadExternalIndices loads external symbol indices from the given inventory paths.
// Returns a new ExternalIndex if paths are empty, or loads indices from the provided files.
func loadExternalIndices(inventoryPaths []string) (kcg.ExternalIndex, error) {
	index := kcg.NewExternalIndex()
	if len(inventoryPaths) > 0 {
		loaded, err := kcg.LoadExternalIndices(inventoryPaths)
		if err != nil {
			return kcg.ExternalIndex{}, fmt.Errorf("load inventory: %w", err)
		}
		return loaded, nil
	}
	return index, nil
}

func printSummary(result kcg.RepositoryResult) {
	successRate := 100.0
	if result.TotalFiles > 0 {
		successRate = (float64(result.ParsedFiles) / float64(result.TotalFiles)) * 100
	}

	cliutil.WriteSummaryLine(os.Stdout, "Root", "%s", result.Root)
	cliutil.WriteSummaryLine(os.Stdout, "Files", "total=%d", result.TotalFiles)
	cliutil.WriteSummaryLine(os.Stdout, "Parse", "parsed=%d failed=%d success=%.2f%%", result.ParsedFiles, result.FailedFiles, successRate)
	cliutil.WriteSummaryLine(os.Stdout, "Duration", "%s", result.Duration.Round(0))

	if result.FailedFiles == 0 {
		return
	}

	failed := make([]kcg.FileUnit, 0, result.FailedFiles)
	for _, unit := range result.Files {
		if !unit.Parsed {
			failed = append(failed, unit)
		}
	}
	sort.Slice(failed, func(i, j int) bool {
		return failed[i].Path < failed[j].Path
	})

	cliutil.WriteSectionHeader(os.Stdout, "Failures")
	for _, unit := range failed {
		fmt.Printf("- %s\n", unit.Path)
		for _, diag := range unit.Diagnostics {
			fmt.Printf("  - %d:%d %s\n", diag.Line, diag.Column, diag.Message)
		}
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "kotlin-compiler-golang - pure Go Kotlin parser porting workspace")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	for _, command := range topLevelCommands {
		if command == "compile" {
			fmt.Fprintln(w, "  kotlin-compiler-golang compile [flags] [-- <extra kotlinc args>]")
			fmt.Fprintln(w, "    (compile paths: --source/--exclude/--plugin/--arg-file/--classpath are resolved against --repo)")
			continue
		}
		fmt.Fprintf(w, "  kotlin-compiler-golang %s [flags]\n", command)
	}
}

func writeGraphArtifacts(htmlPath, jsonPath string, graph kcg.WebGraph) error {
	if err := os.MkdirAll(filepath.Dir(htmlPath), 0o755); err != nil {
		return err
	}
	if err := cliutil.WritePrettyJSONFile(jsonPath, graph); err != nil {
		return err
	}

	html := buildGraphHTML(filepath.Base(jsonPath))
	if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
		return err
	}
	return nil
}

func buildGraphHTML(dataFile string) string {
	quotedDataFile := strconv.Quote(dataFile)
	return strings.ReplaceAll(graphHTMLTemplate, "__DATA_FILE__", quotedDataFile)
}

const graphHTMLTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>jkdeps Graph Viewer</title>
  <script src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
  <style>
    :root {
      --bg: #f4f6f8;
      --card: #ffffff;
      --text: #132238;
      --subtle: #5e6b7a;
      --internal: #0b6e4f;
      --external: #2364aa;
      --unknown: #b14a3a;
    }
    html, body {
      margin: 0;
      padding: 0;
      height: 100%;
      background: radial-gradient(circle at 10% 10%, #ffffff 0%, var(--bg) 45%);
      color: var(--text);
      font-family: "Avenir Next", "Segoe UI", Arial, sans-serif;
    }
    .layout {
      height: 100%;
      display: grid;
      grid-template-rows: auto 1fr;
    }
    .header {
      padding: 16px 20px;
      border-bottom: 1px solid #dbe3ea;
      background: linear-gradient(95deg, #ffffff 0%, #f1f5f9 100%);
    }
    .title {
      margin: 0;
      font-size: 20px;
      line-height: 1.2;
      letter-spacing: 0.02em;
    }
    .meta {
      margin-top: 6px;
      color: var(--subtle);
      font-size: 13px;
    }
    #graph {
      width: 100%;
      height: 100%;
      background: var(--card);
    }
    .legend {
      margin-top: 8px;
      font-size: 12px;
      color: var(--subtle);
      display: flex;
      gap: 14px;
      flex-wrap: wrap;
    }
    .chip::before {
      content: "";
      display: inline-block;
      width: 9px;
      height: 9px;
      border-radius: 50%;
      margin-right: 6px;
      vertical-align: middle;
    }
    .chip.internal::before { background: var(--internal); }
    .chip.external::before { background: var(--external); }
    .chip.unknown::before { background: var(--unknown); }
  </style>
</head>
<body>
  <div class="layout">
    <div class="header">
      <h1 class="title">jkdeps Dependency Graph</h1>
      <div id="meta" class="meta">Loading...</div>
      <div class="legend">
        <span class="chip internal">internal package</span>
        <span class="chip external">external package</span>
        <span class="chip unknown">unresolved package</span>
      </div>
    </div>
    <div id="graph"></div>
  </div>
  <script>
    const dataFile = __DATA_FILE__;

    function colorByKind(kind) {
      if (kind === "internal") return "#0b6e4f";
      if (kind === "external") return "#2364aa";
      return "#b14a3a";
    }

    fetch(dataFile)
      .then((res) => {
        if (!res.ok) throw new Error("failed to load graph JSON");
        return res.json();
      })
      .then((graph) => {
        document.getElementById("meta").textContent =
          "Root: " + graph.root + " | Nodes: " + graph.nodes.length + " | Edges: " + graph.edges.length;

        const nodes = graph.nodes.map((n) => ({
          id: n.id,
          label: n.name,
          color: colorByKind(n.kind),
          title: n.name + "\nkind=" + n.kind + "\nin=" + n.in_degree + " out=" + n.out_degree,
          shape: "dot",
          size: 8 + Math.min(26, n.in_degree + n.out_degree)
        }));

        const edges = graph.edges.map((e) => ({
          from: e.from_id,
          to: e.to_id,
          value: e.count,
          width: 1 + Math.log2(e.count + 1),
          title: "count=" + e.count,
          arrows: "to",
          color: { color: "#8aa3b7", opacity: 0.65 }
        }));

        const container = document.getElementById("graph");
        const network = new vis.Network(
          container,
          { nodes: new vis.DataSet(nodes), edges: new vis.DataSet(edges) },
          {
            interaction: { hover: true, navigationButtons: true, zoomView: true },
            physics: {
              stabilization: false,
              barnesHut: {
                gravitationalConstant: -2200,
                springLength: 180,
                springConstant: 0.04
              }
            },
            nodes: {
              borderWidth: 0,
              font: { color: "#132238", size: 13, face: "Avenir Next" }
            },
            edges: {
              smooth: { enabled: true, type: "dynamic" },
              arrows: { to: { enabled: true, scaleFactor: 0.6 } }
            }
          }
        );

        window.network = network;
      })
      .catch((err) => {
        document.getElementById("meta").textContent = "Failed to load graph: " + err.message;
      });
  </script>
</body>
</html>
`
