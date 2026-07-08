# jkdeps

[한국어 README](README.ko.md)

`jkdeps` analyzes Java/Kotlin repositories from Go. It parses source files, extracts packages/imports/type references, builds dependency graphs, and exposes a small public Go API that other modules can import:

```go
import "github.com/dh-kam/jkdeps"
```

The repository also contains command-line tools for smoke parsing, dependency graph generation, Kotlin import resolution, inventory generation, and Kotlin compiler porting experiments.

## What It Provides

- A public root package, `github.com/dh-kam/jkdeps`, for mixed Java/Kotlin repository analysis.
- Java parsing with selectable grammars: `java`, `java7`, `java8`, `java9`, `java11`, `java17`, `java20`, `java21`, and `java25`.
- Fast Java header-only analysis for package/import dependency scans.
- Full Java parse mode for signatures and low-ambiguity body references.
- Kotlin parsing for `.kt` and optional `.kts`/Gradle script files.
- Package-level or directory-level dependency graphs.
- File-level dependency reports with unresolved import/reference tracking.
- External inventory loading for stdlib/runtime/package resolution.
- CLI tools under `cmd/jkdeps`, `cmd/kotlin-compiler-golang`, and `cmd/ktcg-inventory`.

## Status

`jkdeps` is pre-v1. The root package is the intended stable API surface. Packages under `internal/...` and commands under `cmd/...` are not public library APIs.

The lower-level package `github.com/dh-kam/jkdeps/kotlin-compiler-golang` remains available for Kotlin-specific experiments, but new consumers should prefer the root `jkdeps` package unless they specifically need that experimental surface.

This repository currently carries a local ANTLR Go runtime fork under `third_party/antlr4-go`, wired through `go.mod`. Clone-based builds are the safest way to consume the CLI until those runtime fixes are upstreamed or published as a separate module.

## Quick Start

```bash
git clone git@github.com:dh-kam/jkdeps.git
cd jkdeps
go test ./...
```

Run a mixed Java/Kotlin dependency scan:

```bash
go run ./cmd/jkdeps deps \
  --repo /path/to/repository \
  --java-grammar java20 \
  --workers 8 \
  --group-by package \
  --json
```

Build a browser-viewable graph:

```bash
go run ./cmd/jkdeps graph \
  --repo /path/to/repository \
  --java-grammar java20 \
  --file-timeout 2s \
  --group-by package \
  --inventory ./kotlin-compiler-golang/inventory/runtime-index.json \
  --lenient \
  --out ./jkdeps-mixed-graph
```

Run a parser smoke test:

```bash
go run ./cmd/jkdeps smoke-parse \
  --repo /path/to/repository \
  --java-grammar java20 \
  --workers 8 \
  --max-errors 20 \
  --file-timeout 2s
```

## Go API

Use the root package from another Go module:

```bash
go get github.com/dh-kam/jkdeps
```

Analyze a repository end to end:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dh-kam/jkdeps"
)

func main() {
	report, err := jkdeps.Analyze(context.Background(), "/path/to/repository", jkdeps.Options{
		Parse: jkdeps.ParseOptions{
			Workers:       8,
			JavaGrammar:   jkdeps.JavaGrammar20,
			JavaParseMode: jkdeps.JavaParseModeHeaderOnly,
			KotlinScripts: jkdeps.KotlinScriptsRegular,
			ParseTimeout:  2 * time.Second,
		},
		Graph: jkdeps.GraphOptions{
			GroupBy: jkdeps.GroupByPackage,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("parsed %d/%d files\n", report.Repository.ParsedFiles, report.Repository.TotalFiles)
	fmt.Printf("graph: %d nodes, %d edges\n", len(report.Graph.Nodes), len(report.Graph.Edges))
	fmt.Printf("unresolved imports: %d\n", len(report.Dependencies.UnresolvedImports))
}
```

Parse first, then build the artifacts you need:

```go
repo, err := jkdeps.ParseRepository(ctx, "/path/to/repository", jkdeps.ParseOptions{
	Workers:       8,
	JavaParseMode: jkdeps.JavaParseModeFull,
	KotlinScripts: jkdeps.KotlinScriptsAll,
})
if err != nil {
	return err
}

external, err := jkdeps.LoadExternalIndex("./kotlin-compiler-golang/inventory/runtime-index.json")
if err != nil {
	return err
}

graph := repo.BuildGraph(jkdeps.GraphOptions{
	GroupBy:  jkdeps.GroupByDir,
	External: external,
	Filter: jkdeps.GraphFilter{
		MinEdgeCount:  2,
		IncludePrefix: []string{"com.example."},
	},
})
deps := repo.BuildDependencyReport(external)
_ = graph
_ = deps
```

### Zero-Value Defaults

`jkdeps.Options{}` is usable:

- Java grammar: `java20`
- Java parse mode: `header-only`
- Kotlin scripts: regular `.kts` included, Gradle scripts excluded
- Graph grouping: `package`
- Workers: implementation default
- Max errors per file: implementation default
- Parse timeout: disabled
- Syntax mode: strict

For explicit defaults, use:

```go
opts := jkdeps.DefaultOptions()
opts.Parse.Workers = 8
opts.Graph.Filter = jkdeps.GraphFilter{MinEdgeCount: 2}
```

Use `JavaParseModeFull` when you need Java signature references from `extends`, `implements`, fields, method returns/parameters, constructor parameters, `throws`, and selected body references such as constructor calls, class literals, casts, `instanceof`, local variable types, catch types, and method references.

In `JavaParseModeHeaderOnly`, a Java file marked as parsed means the package/import header was extracted; it does not mean the whole Java source passed syntax validation. Kotlin currently contributes package/import information to mixed dependency analysis, while Java `full` mode also contributes typed references.

### Public Data Model

The root package returns plain DTO-style structs:

- `Repository`: aggregate parse counts, duration, and sorted files.
- `File`: path, relative path, language, package, imports, references, parse status, diagnostics, and duration.
- `Graph`: nodes and weighted edges grouped by package or directory.
- `DependencyReport`: file-level dependency rows plus unresolved imports/references.
- `ExternalIndex`: package/symbol hints loaded from inventory JSON or created in code.

Errors are reserved for operational failures such as invalid options, inaccessible roots, failed walks/reads, and context cancellation. Source parse failures are reported in `File.Diagnostics` and counted in `Repository.FailedFiles`.

## CLI Commands

### `cmd/jkdeps`

Mixed Java/Kotlin repository tool:

```bash
go run ./cmd/jkdeps smoke-parse --repo /path/to/repository
go run ./cmd/jkdeps deps --repo /path/to/repository --json
go run ./cmd/jkdeps graph --repo /path/to/repository --out ./jkdeps-mixed-graph
go run ./cmd/jkdeps roundtrip-check --repo /path/to/repository --rewrite-mode lossless
```

`deps` defaults Java parsing to `header-only` for fast import-based dependency extraction. Use `--java-parse-mode full` when typed Java references are needed.

### `cmd/kotlin-compiler-golang`

Kotlin-focused parser/resolver/compiler-porting workspace:

```bash
go run ./cmd/kotlin-compiler-golang parse --repo /path/to/repo --json
go run ./cmd/kotlin-compiler-golang symbols --repo /path/to/repo --json
go run ./cmd/kotlin-compiler-golang deps --repo /path/to/repo --json
go run ./cmd/kotlin-compiler-golang resolve --repo /path/to/repo --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --json
go run ./cmd/kotlin-compiler-golang graph --repo /path/to/repo --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --lenient --out ./jkdeps-graph
go run ./cmd/kotlin-compiler-golang acceptance --repo /path/to/repo --inventory ./kotlin-compiler-golang/inventory/runtime-index.json --max-failed-files 0 --max-unresolved-imports 0
```

### `cmd/ktcg-inventory`

Inventory builder used by the Kotlin runtime/stdlib resolution flow. The repository also provides shell wrappers:

```bash
./scripts/build_kcg_inventory.sh
./scripts/build_kcg_stdlib_js_inventory.sh
./scripts/build_kcg_browser_inventory.sh
./scripts/build_kcg_atomicfu_inventory.sh
./scripts/build_kcg_runtime_inventory.sh
```

## Parser Generation

ANTLR grammars are stored under `grammars/` and generated Go parsers are committed under `internal/parsers/`.

Regenerate parsers:

```bash
./scripts/generate_parsers.sh
```

The script downloads `tools/antlr-4.13.2-complete.jar` when needed. Downloaded jars, generated `.tokens`/`.interp` side files, and local build artifacts are intentionally ignored by Git.

## Repository Layout

```text
.
├── api.go                         # public root Go API
├── cmd/jkdeps                     # mixed Java/Kotlin CLI
├── cmd/kotlin-compiler-golang     # Kotlin-focused CLI
├── cmd/ktcg-inventory             # inventory CLI
├── docs                           # design, validation, and porting notes
├── grammars                       # Java/Kotlin ANTLR grammars
├── internal/ast                   # internal AST contracts
├── internal/mixedgraph            # internal parser/graph implementation
├── internal/parser                # internal ANTLR parser adapter
├── internal/parsers               # generated ANTLR Go parsers
├── kotlin-compiler-golang         # experimental Kotlin package
├── scripts                        # generation, inventory, validation helpers
├── testdata                       # representative Java/Kotlin samples
└── third_party/antlr4-go          # local ANTLR Go runtime fork
```

## Validation

Common checks:

```bash
go test ./...
make kcg-acceptance
make jkdeps-graph
make oss-matrix
```

Large sample runs use `/tmp/jkdeps-samples` by default. Fetch or refresh pinned samples with:

```bash
./scripts/fetch_samples.sh
```

## License

MIT. See [LICENSE](LICENSE).
