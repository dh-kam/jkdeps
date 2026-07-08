# jkdeps Public Go API

The supported import path for external Go modules is:

```go
import "github.com/dh-kam/jkdeps"
```

Do not import `internal/...` packages from another module. Go prevents that by design. Do not use packages under `cmd/...` as libraries; they are command entry points.

## Overview

The root package is a facade over the internal Java/Kotlin parser and graph builder. It exposes stable DTO-style types and hides ANTLR/generated parser details.

Main entry points:

```go
func Analyze(ctx context.Context, root string, opts Options) (Report, error)
func ParseRepository(ctx context.Context, root string, opts ParseOptions) (Repository, error)
func BuildGraph(repo Repository, opts GraphOptions) Graph
func BuildDependencyReport(repo Repository, external ExternalIndex) DependencyReport
func LoadExternalIndex(path string) (ExternalIndex, error)
func LoadExternalIndices(paths ...string) (ExternalIndex, error)
func DefaultOptions() Options
func DefaultParseOptions() ParseOptions
func DefaultGraphOptions() GraphOptions
```

## End-To-End Analysis

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

	fmt.Println(report.Repository.TotalFiles)
	fmt.Println(len(report.Graph.Nodes))
	fmt.Println(len(report.Dependencies.UnresolvedImports))
}
```

## Step-By-Step Usage

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

graph := jkdeps.BuildGraph(repo, jkdeps.GraphOptions{
	GroupBy:  jkdeps.GroupByDir,
	External: external,
	Filter: jkdeps.GraphFilter{
		MinEdgeCount: 2,
	},
})

deps := jkdeps.BuildDependencyReport(repo, external)
_ = graph
_ = deps
```

## Defaults

Zero-value options are usable.

| Option | Default |
| --- | --- |
| Java grammar | `java20` |
| Java parse mode | `header-only` |
| Kotlin scripts | regular `.kts` included, Gradle scripts excluded |
| Graph grouping | `package` |
| Parse timeout | disabled |
| Syntax mode | strict |

Use `JavaParseModeFull` to collect Java signature references and selected low-ambiguity body references. Use `KotlinScriptsAll` to include both regular `.kts` files and Gradle Kotlin build scripts.

For explicit defaults:

```go
opts := jkdeps.DefaultOptions()
opts.Parse.Workers = 8
opts.Graph.Filter = jkdeps.GraphFilter{MinEdgeCount: 2}
```

In `JavaParseModeHeaderOnly`, Java `File.Parsed` means the package/import header was extracted. It does not validate full Java syntax. Kotlin currently contributes package/import data to mixed dependency analysis; Java `full` mode additionally contributes typed references.

## Error Model

`error` means the analysis operation itself could not run: invalid options, inaccessible root, failed file walk/read, or context cancellation.

Source-level parse failures are data, not operation errors. They are reported in:

- `Repository.FailedFiles`
- `File.Parsed`
- `File.Diagnostics`

## Public Types

The root API exposes:

- `Repository`
- `File`
- `Diagnostic`
- `Reference`
- `Graph`
- `Node`
- `Edge`
- `DependencyReport`
- `FileDependency`
- `ExternalIndex`

Generated ANTLR parser types, parse trees, listener interfaces, CLI flags, profiling helpers, and internal AST interfaces are intentionally not exposed.
