package mixedgraph

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/dh-kam/jkdeps/internal/parser"
	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

var (
	benchmarkFixtureOnce   sync.Once
	benchmarkFixtureResult RepositoryResult
	benchmarkFixtureKnown  kcg.ExternalIndex
)

func indexedEdgeCounterRecentSlotSweepValues() []int {
	return []int{0, 4, 8, defaultIndexedEdgeCounterRecentSlots(), 32, 64}
}

func BenchmarkBuildGraphPackage(b *testing.B) {
	result, external := benchmarkSyntheticFixture()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildGraph(result, external, GroupByPackage)
	}
}

func BenchmarkBuildGraphDir(b *testing.B) {
	result, external := benchmarkSyntheticFixture()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildGraph(result, external, GroupByDir)
	}
}

func BenchmarkFilterGraphMinEdgeCount8(b *testing.B) {
	result, _ := benchmarkSyntheticFixture()
	external := kcg.ExternalIndex{}
	graph := BuildGraph(result, external, GroupByPackage)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterGraph(graph, GraphFilter{MinEdgeCount: 8})
	}
}

func BenchmarkBuildAndFilterMinEdgeCount8(b *testing.B) {
	result, _ := benchmarkSyntheticFixture()
	external := kcg.ExternalIndex{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildFilteredGraph(result, external, GroupByPackage, GraphFilter{MinEdgeCount: 8})
	}
}

func BenchmarkBuildAndFilterDirMinEdgeCount8(b *testing.B) {
	result, _ := benchmarkSyntheticFixture()
	external := kcg.ExternalIndex{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildFilteredGraph(result, external, GroupByDir, GraphFilter{MinEdgeCount: 8})
	}
}

func BenchmarkBuildAndFilterRealRepoMinEdgeCount8(b *testing.B) {
	repoPath := os.Getenv("JKDEPS_BENCH_REPO")
	if repoPath == "" {
		b.Skip("set JKDEPS_BENCH_REPO to run real-repo benchmark")
	}

	result, err := ParseRepository(repoPath, ParseOptions{
		JavaGrammar:      parser.JavaGrammar20,
		Workers:          1,
		IncludeKTS:       true,
		MaxErrorsPerFile: 10,
		LenientSyntax:    false,
	})
	if err != nil {
		b.Fatalf("ParseRepository failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildFilteredGraph(result, kcg.NewExternalIndex(), GroupByPackage, GraphFilter{MinEdgeCount: 8})
	}
}

func BenchmarkIndexedEdgeCounterAddRepeatedKey(b *testing.B) {
	counter := newIndexedEdgeCounter(1)
	key := packIndexedEdgeKey(1, 2)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.add(key, 1)
	}
}

func BenchmarkIndexedEdgeCounterAddSequentialKeys(b *testing.B) {
	keys := make([]uint64, 4096)
	for i := range keys {
		keys[i] = packIndexedEdgeKey(i, i+1)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter := newIndexedEdgeCounter(len(keys))
		for _, key := range keys {
			counter.add(key, 1)
		}
	}
}

func BenchmarkIndexedEdgeCounterAddRepeatedWindow(b *testing.B) {
	keys := make([]uint64, 16)
	for i := range keys {
		keys[i] = packIndexedEdgeKey(7, i+1)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter := newIndexedEdgeCounter(len(keys))
		for repeat := 0; repeat < 256; repeat++ {
			for _, key := range keys {
				counter.add(key, 1)
			}
		}
	}
}

func BenchmarkIndexedEdgeCounterRecentSlotSweep(b *testing.B) {
	keys := make([]uint64, 16)
	for i := range keys {
		keys[i] = packIndexedEdgeKey(7, i+1)
	}
	recentSlots := indexedEdgeCounterRecentSlotSweepValues()

	b.Run("repeated_window", func(b *testing.B) {
		for _, slots := range recentSlots {
			b.Run(fmt.Sprintf("slots_%d", slots), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					counter := newIndexedEdgeCounterWithRecentSlots(len(keys), slots)
					for repeat := 0; repeat < 256; repeat++ {
						for _, key := range keys {
							counter.add(key, 1)
						}
					}
				}
			})
		}
	})

	b.Run("sequential_keys", func(b *testing.B) {
		sequential := make([]uint64, 4096)
		for i := range sequential {
			sequential[i] = packIndexedEdgeKey(i, i+1)
		}
		for _, slots := range recentSlots {
			b.Run(fmt.Sprintf("slots_%d", slots), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					counter := newIndexedEdgeCounterWithRecentSlots(len(sequential), slots)
					for _, key := range sequential {
						counter.add(key, 1)
					}
				}
			})
		}
	})
}

func BenchmarkIndexedGraphBuilderAddEdgeBackends(b *testing.B) {
	keys := make([]uint64, 16)
	for i := range keys {
		keys[i] = packIndexedEdgeKey(7, i+1)
	}

	run := func(b *testing.B, backend indexedGraphBuilderBackend) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			builder := newIndexedGraphBuilderWithBackend(64, defaultPackageGraphBuilderThreshold(), backend)
			for node := 0; node < 32; node++ {
				builder.ensureNode(fmt.Sprintf("n%d", node), NodeInternal)
			}
			for repeat := 0; repeat < 256; repeat++ {
				for _, key := range keys {
					fromIndex, toIndex := unpackIndexedEdgeKey(key)
					builder.addEdgeCount(fromIndex, toIndex, 1)
				}
			}
		}
	}

	b.Run("map", func(b *testing.B) { run(b, indexedGraphBuilderBackendMap) })
	b.Run("counter", func(b *testing.B) { run(b, indexedGraphBuilderBackendCounter) })
}

func BenchmarkIndexedGraphBuilderAddPackedEdgeCountBackends(b *testing.B) {
	keys := make([]uint64, 16)
	for i := range keys {
		keys[i] = packIndexedEdgeKey(7, i+1)
	}
	fromIndex := 7
	fromPrefix := uint64(uint32(fromIndex)) << 32

	run := func(b *testing.B, backend indexedGraphBuilderBackend) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			builder := newIndexedGraphBuilderWithBackend(64, defaultPackageGraphBuilderThreshold(), backend)
			for node := 0; node < 32; node++ {
				builder.ensureNode(fmt.Sprintf("n%d", node), NodeInternal)
			}
			for repeat := 0; repeat < 256; repeat++ {
				for _, key := range keys {
					_, toIndex := unpackIndexedEdgeKey(key)
					builder.addPackedEdgeCount(fromIndex, fromPrefix, toIndex, 1)
				}
			}
		}
	}

	b.Run("map", func(b *testing.B) { run(b, indexedGraphBuilderBackendMap) })
	b.Run("counter", func(b *testing.B) { run(b, indexedGraphBuilderBackendCounter) })
}

func BenchmarkIndexedGraphBuilderIncrementEdgeCountPackedBackends(b *testing.B) {
	keys := make([]uint64, 16)
	for i := range keys {
		keys[i] = packIndexedEdgeKey(7, i+1)
	}

	run := func(b *testing.B, backend indexedGraphBuilderBackend) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			builder := newIndexedGraphBuilderWithBackend(64, defaultPackageGraphBuilderThreshold(), backend)
			for repeat := 0; repeat < 256; repeat++ {
				for _, key := range keys {
					builder.incrementEdgeCountPacked(key, 1)
				}
			}
		}
	}

	b.Run("map", func(b *testing.B) { run(b, indexedGraphBuilderBackendMap) })
	b.Run("counter", func(b *testing.B) { run(b, indexedGraphBuilderBackendCounter) })
}

func BenchmarkIndexedGraphBuilderRecentSlotSweep(b *testing.B) {
	keys := make([]uint64, 16)
	for i := range keys {
		keys[i] = packIndexedEdgeKey(7, i+1)
	}
	recentSlots := indexedEdgeCounterRecentSlotSweepValues()

	for _, slots := range recentSlots {
		b.Run(fmt.Sprintf("slots_%d", slots), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				builder := newIndexedGraphBuilderWithBackendAndRecentSlots(64, defaultPackageGraphBuilderThreshold(), indexedGraphBuilderBackendCounter, slots)
				for node := 0; node < 32; node++ {
					builder.ensureNode(fmt.Sprintf("n%d", node), NodeInternal)
				}
				for repeat := 0; repeat < 256; repeat++ {
					for _, key := range keys {
						fromIndex, toIndex := unpackIndexedEdgeKey(key)
						builder.addEdgeCount(fromIndex, toIndex, 1)
					}
				}
			}
		})
	}
}

func BenchmarkResolveInternalDirectoryImportTarget(b *testing.B) {
	packageDirs := newPackageDirIndex(8)
	packageDirs.add("com.example.util", 7)
	imports := []string{
		"com.example.util.Helpers.format",
		"com.example.util.Helpers.parse",
		"com.example.util.Helpers.render",
		"com.example.util.Helpers.print",
	}

	b.Run("cold_cache", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache := newRecentDirectoryImportCache(8)
			for _, importPath := range imports {
				_, _ = resolveInternalDirectoryImportTarget(&cache, &packageDirs, importPath)
			}
		}
	})

	b.Run("warm_cache", func(b *testing.B) {
		cache := newRecentDirectoryImportCache(8)
		for _, importPath := range imports {
			_, _ = resolveInternalDirectoryImportTarget(&cache, &packageDirs, importPath)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, importPath := range imports {
				_, _ = resolveInternalDirectoryImportTarget(&cache, &packageDirs, importPath)
			}
		}
	})

	b.Run("exact_hit", func(b *testing.B) {
		cache := newRecentDirectoryImportCache(8)
		importPath := "com.example.util.Helpers.format"
		_, _ = resolveInternalDirectoryImportTarget(&cache, &packageDirs, importPath)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = resolveInternalDirectoryImportTarget(&cache, &packageDirs, importPath)
		}
	})

	b.Run("base_candidate_hit", func(b *testing.B) {
		cache := newRecentDirectoryImportCache(8)
		_, _ = resolveInternalDirectoryImportTarget(&cache, &packageDirs, "com.example.util.Helpers.format")

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = resolveInternalDirectoryImportTarget(&cache, &packageDirs, "com.example.util.Helpers.parse")
		}
	})

	b.Run("cached_miss", func(b *testing.B) {
		cache := newRecentDirectoryImportCache(8)
		importPath := "java.util.List"
		_, _ = resolveInternalDirectoryImportTarget(&cache, &packageDirs, importPath)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = resolveInternalDirectoryImportTarget(&cache, &packageDirs, importPath)
		}
	})
}

func BenchmarkDeriveImportBaseCandidate(b *testing.B) {
	paths := []string{
		"com.example.util.Helpers.format",
		"com.example.Service",
		"com.example.util.helpers.format",
		"java.util.*",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			_ = deriveImportBaseCandidate(path)
		}
	}
}

func BenchmarkIndexedGraphBuilderMaterializeFilteredEdges(b *testing.B) {
	builder := newIndexedGraphBuilder(64, 256)
	for i := 0; i < 32; i++ {
		builder.ensureNode(fmt.Sprintf("keep.%d", i), NodeInternal)
	}
	for i := 32; i < 64; i++ {
		builder.ensureNode(fmt.Sprintf("skip.%d", i), NodeExternal)
	}
	for from := 0; from < 32; from++ {
		for to := 0; to < 16; to++ {
			builder.addEdgeCount(from, to, (to%4)+1)
		}
		for to := 32; to < 40; to++ {
			builder.addEdgeCount(from, to, 3)
		}
	}

	filter := GraphFilter{MinEdgeCount: 2, IncludePrefix: []string{"keep."}}
	includeNode := make([]bool, len(builder.names)+1)
	newIDByOldID := make([]int, len(builder.names)+1)
	for i := 0; i < 32; i++ {
		includeNode[i+1] = true
		newIDByOldID[i+1] = i + 1
	}

	b.Run("with_prefix", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			inDegree := make([]int, 33)
			outDegree := make([]int, 33)
			_ = builder.materializeFilteredEdges(filter, includeNode, true, newIDByOldID, 512, inDegree, outDegree)
		}
	})

	b.Run("without_prefix", func(b *testing.B) {
		noPrefixFilter := GraphFilter{MinEdgeCount: 2}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			inDegree := make([]int, 33)
			outDegree := make([]int, 33)
			_ = builder.materializeFilteredEdges(noPrefixFilter, includeNode, false, newIDByOldID, 512, inDegree, outDegree)
		}
	})
}

func BenchmarkFlushEdgeCountEntries(b *testing.B) {
	entries := make([]edgeCountEntry, 0, 16)
	for i := 0; i < 16; i++ {
		entries = append(entries, edgeCountEntry{toIndex: i + 1, count: (i % 4) + 1})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := newIndexedGraphBuilder(32, 64)
		from := builder.ensureNode("from", NodeInternal)
		for j := range entries {
			builder.ensureNode(fmt.Sprintf("n%d", j), NodeInternal)
		}
		flushEdgeCountEntries(builder, from, entries)
	}
}

func BenchmarkGraphThresholdSweep(b *testing.B) {
	result, external := benchmarkSyntheticFixture()
	thresholds := []int{0, 512, 2048, 4096, 8192, 12288, 16384, 32768}

	run := func(b *testing.B, groupBy GroupBy, filtered bool) {
		for _, threshold := range thresholds {
			name := fmt.Sprintf("threshold_%d", threshold)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if filtered {
						_ = buildFilteredGraphWithThreshold(result, external, groupBy, GraphFilter{MinEdgeCount: 8}, threshold)
					} else {
						_ = buildGraphWithThreshold(result, external, groupBy, threshold)
					}
				}
			})
		}
	}

	b.Run("package", func(b *testing.B) { run(b, GroupByPackage, false) })
	b.Run("dir", func(b *testing.B) { run(b, GroupByDir, false) })
	b.Run("filtered_package", func(b *testing.B) { run(b, GroupByPackage, true) })
	b.Run("filtered_dir", func(b *testing.B) { run(b, GroupByDir, true) })
}

func buildGraphWithThreshold(result RepositoryResult, external kcg.ExternalIndex, groupBy GroupBy, threshold int) Graph {
	if !groupBy.IsValid() {
		groupBy = GroupByPackage
	}
	ctx := newGraphBuildContext(result.Files, external)
	switch groupBy {
	case GroupByDir:
		builder := newDirectoryGraphBuilderWithThreshold(ctx, len(result.Files), threshold)
		builder.indexFiles(result.Files)
		builder.addImports(result.Files)
		nodes, edges := builder.builder.materialize()
		return sortGraph(Graph{Root: result.Root, GroupBy: GroupByDir, Nodes: nodes, Edges: edges})
	default:
		builder := newPackageGraphBuilderWithThreshold(ctx, len(result.Files), threshold)
		builder.seedInternalPackages()
		builder.addImports(result.Files)
		nodes, edges := builder.builder.materialize()
		return sortGraph(Graph{Root: result.Root, GroupBy: GroupByPackage, Nodes: nodes, Edges: edges})
	}
}

func buildFilteredGraphWithThreshold(result RepositoryResult, external kcg.ExternalIndex, groupBy GroupBy, filter GraphFilter, threshold int) Graph {
	if !groupBy.IsValid() {
		groupBy = GroupByPackage
	}
	ctx := newGraphBuildContext(result.Files, external)
	switch groupBy {
	case GroupByDir:
		builder := newDirectoryGraphBuilderWithThreshold(ctx, len(result.Files), threshold)
		builder.indexFiles(result.Files)
		builder.addImports(result.Files)
		nodes, edges := builder.builder.materializeFiltered(filter)
		return sortGraph(Graph{Root: result.Root, GroupBy: GroupByDir, Nodes: nodes, Edges: edges})
	default:
		builder := newPackageGraphBuilderWithThreshold(ctx, len(result.Files), threshold)
		builder.seedInternalPackages()
		builder.addImports(result.Files)
		nodes, edges := builder.builder.materializeFiltered(filter)
		return sortGraph(Graph{Root: result.Root, GroupBy: GroupByPackage, Nodes: nodes, Edges: edges})
	}
}

func benchmarkSyntheticFixture() (RepositoryResult, kcg.ExternalIndex) {
	benchmarkFixtureOnce.Do(func() {
		benchmarkFixtureResult = makeLargeSyntheticRepositoryResult(2400, 8, 16)
		benchmarkFixtureKnown = kcg.ExternalIndex{
			Packages: map[string]struct{}{
				"java.util":          {},
				"kotlin.collections": {},
			},
			Symbols: map[string]struct{}{
				"kotlin.collections.List": {},
			},
		}
	})
	return benchmarkFixtureResult, benchmarkFixtureKnown
}

func makeLargeSyntheticRepositoryResult(numPackages, filesPerPackage, importsPerFile int) RepositoryResult {
	files := make([]FileUnit, 0, numPackages*filesPerPackage)
	for pkgIndex := 0; pkgIndex < numPackages; pkgIndex++ {
		pkgName := fmt.Sprintf("com.example.pkg%04d", pkgIndex)
		for fileIndex := 0; fileIndex < filesPerPackage; fileIndex++ {
			imports := make([]string, 0, importsPerFile)
			for importIndex := 0; importIndex < importsPerFile; importIndex++ {
				if importIndex%9 == 0 {
					imports = append(imports, "java.util.concurrent.List")
					continue
				}

				target := (pkgIndex + importIndex + 1) % numPackages
				if target == pkgIndex {
					target = (target + 1) % numPackages
				}
				imports = append(imports, fmt.Sprintf("com.example.pkg%04d.Service%02d", target, importIndex%99))
			}

			files = append(files, FileUnit{
				PackageName: pkgName,
				Relative:    fmt.Sprintf("%s/src/main/File%d.java", pkgName, fileIndex),
				Language:    LangJava,
				Imports:     imports,
			})
		}
	}

	return RepositoryResult{
		Files: files,
	}
}
