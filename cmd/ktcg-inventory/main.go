package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/dh-kam/jkdeps/internal/cliutil"
	"github.com/dh-kam/jkdeps/internal/flagutil"
)

type stringListFlag = flagutil.StringListFlag

type packageStat struct {
	Package string `json:"package"`
	Count   int    `json:"count"`
}

type inventory struct {
	JarPath         string        `json:"jar_path,omitempty"`
	SourceJars      []string      `json:"source_jars,omitempty"`
	GeneratedAt     time.Time     `json:"generated_at"`
	ClassFiles      int           `json:"class_files"`
	TopLevelClasses int           `json:"top_level_classes"`
	BuiltinsFiles   int           `json:"builtins_files"`
	MetadataFiles   int           `json:"metadata_files,omitempty"`
	Packages        []packageStat `json:"packages"`
	Symbols         []string      `json:"symbols,omitempty"`
}

type runConfig struct {
	jarPaths       *stringListFlag
	outPath        *string
	includeSymbols *bool
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return 2
	}
	if isHelpAlias(args[0]) {
		printUsage(os.Stdout)
		return 0
	}

	fs, cfg := newRunFlagSet(args)

	if ok, code := cliutil.ParseFlagSet(fs, args); !ok {
		return code
	}
	jarPaths := uniquePaths(*cfg.jarPaths)
	if len(jarPaths) == 0 {
		fmt.Fprintln(os.Stderr, "--jar is required at least once")
		return 2
	}

	inv, err := buildInventory(jarPaths, *cfg.includeSymbols)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build inventory: %v\n", err)
		return 1
	}

	if *cfg.outPath == "" {
		if err := cliutil.WritePrettyJSON(os.Stdout, inv); err != nil {
			fmt.Fprintf(os.Stderr, "marshal inventory: %v\n", err)
			return 1
		}
		return 0
	}

	if err := cliutil.WritePrettyJSONFile(*cfg.outPath, inv); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		return 1
	}
	fmt.Printf("wrote %s\n", *cfg.outPath)
	return 0
}

func newRunFlagSet(args []string) (*flag.FlagSet, runConfig) {
	fs := cliutil.NewFlagSet("ktcg-inventory", args)

	var jarPaths stringListFlag
	fs.Var(&jarPaths, "jar", "Path to input jar (repeatable or comma-separated)")
	outPath := fs.String("out", "", "Output JSON path (default stdout)")
	includeSymbols := fs.Bool("symbols", true, "Include external symbol FQNs extracted from class entries")

	return fs, runConfig{
		jarPaths:       &jarPaths,
		outPath:        outPath,
		includeSymbols: includeSymbols,
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "ktcg-inventory")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ktcg-inventory --jar <path> [--jar <path> ...] [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fs := flag.NewFlagSet("ktcg-inventory", flag.ContinueOnError)
	fs.SetOutput(w)
	var jarPaths stringListFlag
	fs.Var(&jarPaths, "jar", "Path to input jar (repeatable or comma-separated)")
	fs.String("out", "", "Output JSON path (default stdout)")
	fs.Bool("symbols", true, "Include external symbol FQNs extracted from class entries")
	fs.PrintDefaults()
}

func isHelpAlias(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}

func buildInventory(jarPaths []string, includeSymbols bool) (inventory, error) {
	pkgCounts := map[string]int{}
	symbolSet := map[string]struct{}{}
	classFiles := 0
	topLevel := 0
	builtinsFiles := 0
	metadataFiles := 0

	for _, jarPath := range jarPaths {
		r, err := zip.OpenReader(jarPath)
		if err != nil {
			return inventory{}, err
		}

		for _, file := range r.File {
			name := file.Name
			if strings.HasSuffix(name, "/") {
				continue
			}

			kind, entryPath := classifyEntry(name)
			if kind == entrySkip {
				continue
			}

			switch kind {
			case entryClass:
				classFiles++
				if !strings.Contains(path.Base(name), "$") {
					topLevel++
				}
				if includeSymbols {
					symbol := classPathToSymbol(entryPath)
					if symbol != "" {
						symbolSet[symbol] = struct{}{}
					}
				}
			case entryBuiltins:
				builtinsFiles++
			case entryMetadata:
				metadataFiles++
			}

			pkg := packageFromEntry(kind, entryPath)
			if pkg == "" {
				continue
			}
			pkgCounts[pkg]++
		}

		if err := r.Close(); err != nil {
			return inventory{}, err
		}
	}

	packages := make([]packageStat, 0, len(pkgCounts))
	for pkg, count := range pkgCounts {
		packages = append(packages, packageStat{Package: pkg, Count: count})
	}
	sort.Slice(packages, func(i, j int) bool {
		if packages[i].Count == packages[j].Count {
			return packages[i].Package < packages[j].Package
		}
		return packages[i].Count > packages[j].Count
	})

	symbols := make([]string, 0, len(symbolSet))
	for symbol := range symbolSet {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)

	out := inventory{
		SourceJars:      append([]string(nil), jarPaths...),
		GeneratedAt:     time.Now().UTC(),
		ClassFiles:      classFiles,
		TopLevelClasses: topLevel,
		BuiltinsFiles:   builtinsFiles,
		MetadataFiles:   metadataFiles,
		Packages:        packages,
	}
	if len(jarPaths) == 1 {
		out.JarPath = jarPaths[0]
	}
	if includeSymbols {
		out.Symbols = symbols
	}
	return out, nil
}

type entryKind int

const (
	entrySkip entryKind = iota
	entryClass
	entryBuiltins
	entryMetadata
)

func classifyEntry(name string) (entryKind, string) {
	switch {
	case strings.HasSuffix(name, ".class"):
		return entryClass, strings.TrimSuffix(name, ".class")
	case strings.HasSuffix(name, ".kotlin_builtins"):
		return entryBuiltins, strings.TrimSuffix(name, ".kotlin_builtins")
	case strings.HasSuffix(name, ".kotlin_metadata"):
		return entryMetadata, strings.TrimSuffix(name, ".kotlin_metadata")
	case strings.HasSuffix(name, ".kjsm"):
		return entryMetadata, strings.TrimSuffix(name, ".kjsm")
	case strings.HasSuffix(name, ".knm"):
		return entryMetadata, strings.TrimSuffix(name, ".knm")
	default:
		return entrySkip, ""
	}
}

func classPathToSymbol(entryPath string) string {
	if entryPath == "" {
		return ""
	}
	fqn := strings.ReplaceAll(entryPath, "/", ".")
	fqn = strings.ReplaceAll(fqn, "$", ".")
	if fqn == "" {
		return ""
	}
	if fqn == "package-info" || fqn == "module-info" ||
		strings.HasSuffix(fqn, ".package-info") || strings.HasSuffix(fqn, ".module-info") {
		return ""
	}
	return fqn
}

func packageFromEntry(kind entryKind, entryPath string) string {
	if entryPath == "" {
		return ""
	}
	if kind == entryMetadata {
		if pkg := packageFromKlibLinkdata(entryPath); pkg != "" {
			return pkg
		}
	}

	pkg := strings.TrimSuffix(path.Dir(entryPath), "/")
	if pkg == "." || pkg == "" {
		return ""
	}
	return strings.ReplaceAll(pkg, "/", ".")
}

func packageFromKlibLinkdata(entryPath string) string {
	const marker = "/linkdata/package_"
	idx := strings.Index(entryPath, marker)
	if idx == -1 {
		const altMarker = "linkdata/package_"
		idx = strings.Index(entryPath, altMarker)
		if idx == -1 {
			return ""
		}
		entryPath = entryPath[idx+len(altMarker):]
	} else {
		entryPath = entryPath[idx+len(marker):]
	}
	if entryPath == "" {
		return ""
	}
	if slash := strings.Index(entryPath, "/"); slash >= 0 {
		entryPath = entryPath[:slash]
	}
	entryPath = strings.TrimSpace(entryPath)
	if entryPath == "" {
		return ""
	}
	return entryPath
}

func uniquePaths(paths []string) []string {
	return flagutil.UniqueStrings(paths)
}
