package mixedgraph

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

var knownExternalRoots = [...]string{"java", "javax", "jdk", "sun", "android"}

var knownExternalPrefixes = [...]string{
	"org.codehaus.mojo.animal_sniffer",
	"org.junit",
	"junit.framework",
	"org.jetbrains.kotlinx.lincheck",
	"org.openjdk.jmh",
	"org.openjdk.jol",
	"platform",
	"kotlinx.cinterop",
	"kotlinx.knit",
	"kotlinx.benchmark",
	"org.gradle",
	"ru.vyarus.gradle.plugin.animalsniffer",
}

type NodeKind string

const (
	NodeInternal NodeKind = "internal"
	NodeExternal NodeKind = "external"
	NodeUnknown  NodeKind = "unknown"
)

type Node struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Kind      NodeKind `json:"kind"`
	InDegree  int      `json:"in_degree"`
	OutDegree int      `json:"out_degree"`
}

type Edge struct {
	FromID int `json:"from_id"`
	ToID   int `json:"to_id"`
	Count  int `json:"count"`
}

type Graph struct {
	Root    string  `json:"root"`
	GroupBy GroupBy `json:"group_by"`
	Nodes   []Node  `json:"nodes"`
	Edges   []Edge  `json:"edges"`
}

type resolvedImportTarget struct {
	Package string
	Kind    NodeKind
}

type graphSortPolicy struct {
	checkSortedNodes bool
	checkSortedEdges bool
	useCountingEdges bool
}

type graphBuildContext struct {
	filePackages     []string
	internalPackages map[string]struct{}
	resolver         importClassifier
}

type packageDirIndex struct {
	singleByPackage map[string]int
	multiByPackage  map[string][]int
}

type importClassifier struct {
	internalPackages map[string]struct{}
	external         kcg.ExternalIndex
	resolved         map[string]resolvedImportTarget
	recentRaw        []recentResolvedImport
	recentRawMask    uint32
}

type recentResolvedImport struct {
	key    string
	target resolvedImportTarget
}

type graphNodeIndexCache struct {
	lastName  string
	lastIndex int
	hasLast   bool
}

type indexedGraphBuilderBackend string

const (
	indexedGraphBuilderBackendMap     indexedGraphBuilderBackend = "map"
	indexedGraphBuilderBackendCounter indexedGraphBuilderBackend = "counter"
)

type directoryImportTarget struct {
	singleDirIndex int
	hasSingleDir   bool
	dirIndices     []int
	nodeIndex      int
	hasNode        bool
}

type resolvedDirectoryImport struct {
	target directoryImportTarget
	ok     bool
}

type recentDirectoryImportCache struct {
	entries []recentDirectoryImportEntry
	mask    uint32
}

type recentDirectoryImportEntry struct {
	key   string
	value resolvedDirectoryImport
}

const (
	maxCachedResolvedImports       = 256
	importClassifierRecentRawSlots = 32
	// Package and directory builders keep separate defaults so benchmark-driven
	// tuning can diverge later without reshaping the builder config flow. The
	// current synthetic sweep keeps both at 32768: package graphs are faster
	// there, and directory graphs have not shown a stable win at 16384.
	packageIndexedEdgeCounterMinCapacity   = 32768
	directoryIndexedEdgeCounterMinCapacity = 32768
	indexedEdgeCounterRecentSlots          = 16
)

var deterministicGraphSortPolicy graphSortPolicy

var deterministicDirGraphSortPolicy = graphSortPolicy{
	checkSortedNodes: true,
	checkSortedEdges: true,
	useCountingEdges: true,
}

var sortScratchPool = sync.Pool{
	New: func() any {
		return &sortScratch{}
	},
}

type sortScratch struct {
	oldToNew []int
	counts   []int
	edgeBuf  []Edge
}

type edgeCountEntry struct {
	toIndex int
	count   int
}

type localEdgeAccumulator struct {
	entries   []edgeCountEntry
	lastPos   int
	lastToIdx int
	hasLast   bool
}

type packageGraphBuilder struct {
	ctx        graphBuildContext
	builder    *indexedGraphBuilder
	nodeCache  graphNodeIndexCache
	localEdges localEdgeAccumulator
}

type directoryGraphBuilder struct {
	ctx                 graphBuildContext
	builder             *indexedGraphBuilder
	nodeCache           graphNodeIndexCache
	packageDirs         packageDirIndex
	importTargets       map[string]directoryImportTarget
	internalImportCache recentDirectoryImportCache
	fromDirIndices      []int
	localEdges          localEdgeAccumulator
}

type indexedGraphBuilder struct {
	nameToIndex map[string]int
	names       []string
	kinds       []NodeKind
	inDegree    []uint32
	outDegree   []uint32
	edgeCounts  indexedEdgeCounter
	edgeMap     map[uint64]uint32
	backend     indexedGraphBuilderBackend
}

type indexedEdgeCounter struct {
	keys       []uint64
	counts     []uint32
	mask       uint64
	size       int
	hasZero    bool
	zeroCount  uint32
	recent     []indexedEdgeCounterRecent
	recentMask uint64
}

type indexedGraphBuilderConfig struct {
	nodeCapacity int
	edgeCapacity int
	backend      indexedGraphBuilderBackend
}

type indexedEdgeCounterRecent struct {
	key uint64
	idx uint32
}

func (r RepositoryResult) BuildGraph(external kcg.ExternalIndex, groupBy GroupBy) Graph {
	return BuildGraph(r, external, groupBy)
}

func BuildGraph(result RepositoryResult, external kcg.ExternalIndex, groupBy GroupBy) Graph {
	return sortGraph(BuildGraphInternalUnsorted(result, external, groupBy))
}

func BuildGraphInternalUnsorted(result RepositoryResult, external kcg.ExternalIndex, groupBy GroupBy) Graph {
	if !groupBy.IsValid() {
		groupBy = GroupByPackage
	}
	ctx := newGraphBuildContext(result.Files, external)
	switch groupBy {
	case GroupByDir:
		return buildDirectoryGraph(result, ctx)
	default:
		return buildPackageGraph(result, ctx)
	}
}

func buildPackageGraph(result RepositoryResult, ctx graphBuildContext) Graph {
	builder := buildPackageGraphBuilder(result.Files, ctx)
	nodes, edges := builder.materialize()
	return Graph{
		Root:    result.Root,
		GroupBy: GroupByPackage,
		Nodes:   nodes,
		Edges:   edges,
	}
}

func buildPackageGraphBuilder(files []FileUnit, ctx graphBuildContext) *indexedGraphBuilder {
	pkgBuilder := newPackageGraphBuilder(ctx, len(files))
	pkgBuilder.seedInternalPackages()
	pkgBuilder.addImports(files)
	pkgBuilder.addReferences(files)
	return pkgBuilder.builder
}

func newPackageGraphBuilder(ctx graphBuildContext, fileCount int) packageGraphBuilder {
	config := newDefaultPackageGraphBuilderConfig(fileCount, len(ctx.internalPackages))
	return packageGraphBuilder{
		ctx:        ctx,
		builder:    newIndexedGraphBuilderFromConfig(config),
		nodeCache:  newGraphNodeIndexCache(config.nodeCapacity),
		localEdges: newLocalEdgeAccumulator(16),
	}
}

func newPackageGraphBuilderWithThreshold(ctx graphBuildContext, fileCount int, threshold int) packageGraphBuilder {
	config := newPackageGraphBuilderConfig(fileCount, len(ctx.internalPackages), threshold)
	return packageGraphBuilder{
		ctx:        ctx,
		builder:    newIndexedGraphBuilderFromConfig(config),
		nodeCache:  newGraphNodeIndexCache(config.nodeCapacity),
		localEdges: newLocalEdgeAccumulator(16),
	}
}

func (b *packageGraphBuilder) seedInternalPackages() {
	for pkg := range b.ctx.internalPackages {
		b.nodeCache.ensure(b.builder, pkg, NodeInternal)
	}
}

func (b *packageGraphBuilder) addImports(files []FileUnit) {
	for i, file := range files {
		b.addFileImports(file, b.ctx.filePackages[i])
	}
}

func (b *packageGraphBuilder) addReferences(files []FileUnit) {
	for i, file := range files {
		b.addFileReferences(file, b.ctx.filePackages[i])
	}
}

func (b *packageGraphBuilder) addFileImports(file FileUnit, from string) {
	if from == "" {
		return
	}
	fromIndex := b.nodeCache.ensure(b.builder, from, NodeInternal)
	b.localEdges.reset()
	for _, rawImport := range file.Imports {
		target := b.ctx.resolveImport(rawImport)
		if target.Package == "" || target.Package == from {
			continue
		}
		toIndex := b.nodeCache.ensure(b.builder, target.Package, target.Kind)
		b.localEdges.add(toIndex)
	}
	b.localEdges.flush(b.builder, fromIndex)
}

func (b *packageGraphBuilder) addFileReferences(file FileUnit, from string) {
	if from == "" || len(file.References) == 0 {
		return
	}
	fromIndex := b.nodeCache.ensure(b.builder, from, NodeInternal)
	b.localEdges.reset()
	for _, ref := range file.References {
		target := b.ctx.resolveReference(file, ref.Path)
		if target.Package == "" || target.Package == from {
			continue
		}
		toIndex := b.nodeCache.ensure(b.builder, target.Package, target.Kind)
		b.localEdges.add(toIndex)
	}
	b.localEdges.flush(b.builder, fromIndex)
}

func buildDirectoryGraph(result RepositoryResult, ctx graphBuildContext) Graph {
	builder := buildDirectoryGraphBuilder(result.Files, ctx)
	nodes, edges := builder.materialize()
	return Graph{
		Root:    result.Root,
		GroupBy: GroupByDir,
		Nodes:   nodes,
		Edges:   edges,
	}
}

func buildDirectoryGraphBuilder(files []FileUnit, ctx graphBuildContext) *indexedGraphBuilder {
	dirBuilder := newDirectoryGraphBuilder(ctx, len(files))
	dirBuilder.indexFiles(files)
	dirBuilder.addImports(files)
	dirBuilder.addReferences(files)
	return dirBuilder.builder
}

func newDirectoryGraphBuilder(ctx graphBuildContext, fileCount int) directoryGraphBuilder {
	config, packageCapacity := newDefaultDirectoryGraphBuilderConfig(fileCount, len(ctx.internalPackages))
	return directoryGraphBuilder{
		ctx:                 ctx,
		builder:             newIndexedGraphBuilderFromConfig(config),
		nodeCache:           newGraphNodeIndexCache(fileCount),
		packageDirs:         newPackageDirIndex(packageCapacity),
		importTargets:       make(map[string]directoryImportTarget, minInt(len(ctx.internalPackages), 256)),
		internalImportCache: newRecentDirectoryImportCache(32),
		fromDirIndices:      make([]int, fileCount),
		localEdges:          newLocalEdgeAccumulator(32),
	}
}

func newDirectoryGraphBuilderWithThreshold(ctx graphBuildContext, fileCount int, threshold int) directoryGraphBuilder {
	config, packageCapacity := newDirectoryGraphBuilderConfig(fileCount, len(ctx.internalPackages), threshold)
	return directoryGraphBuilder{
		ctx:                 ctx,
		builder:             newIndexedGraphBuilderFromConfig(config),
		nodeCache:           newGraphNodeIndexCache(fileCount),
		packageDirs:         newPackageDirIndex(packageCapacity),
		importTargets:       make(map[string]directoryImportTarget, minInt(len(ctx.internalPackages), 256)),
		internalImportCache: newRecentDirectoryImportCache(32),
		fromDirIndices:      make([]int, fileCount),
		localEdges:          newLocalEdgeAccumulator(32),
	}
}

func (b *directoryGraphBuilder) indexFiles(files []FileUnit) {
	for i, file := range files {
		fromDir := normalizeDir(file.Relative)
		b.fromDirIndices[i] = b.nodeCache.ensure(b.builder, fromDir, NodeInternal)
		pkg := b.ctx.filePackages[i]
		if pkg != "" {
			b.packageDirs.add(pkg, b.fromDirIndices[i])
		}
	}
}

func (b *directoryGraphBuilder) addImports(files []FileUnit) {
	for i, file := range files {
		b.addFileImports(file, b.fromDirIndices[i])
	}
}

func (b *directoryGraphBuilder) addReferences(files []FileUnit) {
	for i, file := range files {
		b.addFileReferences(file, b.fromDirIndices[i])
	}
}

func (b *directoryGraphBuilder) addFileImports(file FileUnit, fromDirIndex int) {
	b.localEdges.reset()
	for _, rawImport := range file.Imports {
		target, ok := b.resolveImportTarget(rawImport)
		if !ok {
			continue
		}
		if target.hasSingleDir || len(target.dirIndices) > 0 {
			b.localEdges.addDirectoryTarget(fromDirIndex, target)
			continue
		}
		if target.hasNode && target.nodeIndex != fromDirIndex {
			b.localEdges.add(target.nodeIndex)
		}
	}
	b.localEdges.flush(b.builder, fromDirIndex)
}

func (b *directoryGraphBuilder) addFileReferences(file FileUnit, fromDirIndex int) {
	if len(file.References) == 0 {
		return
	}
	b.localEdges.reset()
	for _, ref := range file.References {
		target, ok := b.resolveReferenceTarget(file, ref.Path)
		if !ok {
			continue
		}
		if target.hasSingleDir || len(target.dirIndices) > 0 {
			b.localEdges.addDirectoryTarget(fromDirIndex, target)
			continue
		}
		if target.hasNode && target.nodeIndex != fromDirIndex {
			b.localEdges.add(target.nodeIndex)
		}
	}
	b.localEdges.flush(b.builder, fromDirIndex)
}

func (b *directoryGraphBuilder) resolveImportTarget(rawImport string) (directoryImportTarget, bool) {
	normalizedImport := normalizePath(rawImport)
	if normalizedImport == "" {
		return directoryImportTarget{}, false
	}
	if target, ok := resolveInternalDirectoryImportTarget(&b.internalImportCache, &b.packageDirs, normalizedImport); ok {
		return target, true
	}
	resolved := b.ctx.resolveImportNormalized(normalizedImport)
	if resolved.Package == "" {
		return directoryImportTarget{}, false
	}
	return resolveDirectoryImportTarget(b.importTargets, &b.packageDirs, &b.nodeCache, b.builder, resolved.Package, resolved.Kind), true
}

func (b *directoryGraphBuilder) resolveReferenceTarget(file FileUnit, rawPath string) (directoryImportTarget, bool) {
	resolved := b.ctx.resolveReference(file, rawPath)
	if resolved.Package == "" {
		return directoryImportTarget{}, false
	}
	return resolveDirectoryImportTarget(b.importTargets, &b.packageDirs, &b.nodeCache, b.builder, resolved.Package, resolved.Kind), true
}

func newGraphBuildContext(files []FileUnit, external kcg.ExternalIndex) graphBuildContext {
	packageCapacity := estimateInternalPackageCapacity(len(files))
	ctx := graphBuildContext{
		filePackages:     make([]string, len(files)),
		internalPackages: make(map[string]struct{}, packageCapacity),
	}
	for i, file := range files {
		pkg := normalizePath(file.PackageName)
		ctx.filePackages[i] = pkg
		if pkg != "" {
			ctx.internalPackages[pkg] = struct{}{}
		}
	}
	ctx.resolver = newImportClassifier(ctx.internalPackages, external)
	return ctx
}

func (c graphBuildContext) resolveImport(importPath string) resolvedImportTarget {
	return c.resolver.resolve(importPath)
}

func (c graphBuildContext) resolveImportNormalized(importPath string) resolvedImportTarget {
	return c.resolver.resolveNormalized(importPath)
}

func newGraphNodeIndexCache(capacity int) graphNodeIndexCache {
	_ = capacity
	return graphNodeIndexCache{}
}

func (c graphNodeIndexCache) get(name string) (int, bool) {
	if c.hasLast && c.lastName == name {
		return c.lastIndex, true
	}
	return 0, false
}

func (c *graphNodeIndexCache) ensure(builder *indexedGraphBuilder, name string, kind NodeKind) int {
	if idx, ok := c.get(name); ok {
		builder.mergeNodeKind(idx, kind)
		return idx
	}
	idx := builder.ensureNode(name, kind)
	c.lastName = name
	c.lastIndex = idx
	c.hasLast = true
	return idx
}

func newPackageDirIndex(capacity int) packageDirIndex {
	return packageDirIndex{
		singleByPackage: make(map[string]int, capacity),
	}
}

func (i *packageDirIndex) add(pkg string, dirIndex int) {
	if dirIndices, ok := i.multiByPackage[pkg]; ok {
		if n := len(dirIndices); n > 0 && dirIndices[n-1] == dirIndex {
			return
		}
		for _, existing := range dirIndices {
			if existing == dirIndex {
				return
			}
		}
		i.multiByPackage[pkg] = append(dirIndices, dirIndex)
		return
	}
	existing, ok := i.singleByPackage[pkg]
	if !ok {
		i.singleByPackage[pkg] = dirIndex
		return
	}
	if existing == dirIndex {
		return
	}
	delete(i.singleByPackage, pkg)
	if i.multiByPackage == nil {
		i.multiByPackage = make(map[string][]int, minInt(len(i.singleByPackage)+1, 64))
	}
	i.multiByPackage[pkg] = []int{existing, dirIndex}
}

func (i *packageDirIndex) single(pkg string) (int, bool) {
	dirIndex, ok := i.singleByPackage[pkg]
	return dirIndex, ok
}

func (i *packageDirIndex) multiple(pkg string) []int { return i.multiByPackage[pkg] }

func (i *packageDirIndex) directoryTarget(pkg string) (directoryImportTarget, bool) {
	if dirIndex, ok := i.single(pkg); ok {
		return directoryImportTarget{singleDirIndex: dirIndex, hasSingleDir: true}, true
	}
	if dirIndices := i.multiple(pkg); len(dirIndices) > 0 {
		return directoryImportTarget{dirIndices: dirIndices}, true
	}
	return directoryImportTarget{}, false
}

func addDirectoryImportEdgeCounts(entries []edgeCountEntry, fromDirIndex int, target directoryImportTarget) []edgeCountEntry {
	if target.hasSingleDir {
		if target.singleDirIndex != fromDirIndex {
			return addEdgeCountEntry(entries, target.singleDirIndex)
		}
		return entries
	}
	for _, toDirIndex := range target.dirIndices {
		if toDirIndex != fromDirIndex {
			entries = addEdgeCountEntry(entries, toDirIndex)
		}
	}
	return entries
}

func newLocalEdgeAccumulator(capacity int) localEdgeAccumulator {
	return localEdgeAccumulator{
		entries: make([]edgeCountEntry, 0, capacity),
	}
}

func (a *localEdgeAccumulator) reset() {
	a.entries = a.entries[:0]
	a.lastPos = 0
	a.lastToIdx = 0
	a.hasLast = false
}

func (a *localEdgeAccumulator) add(toIndex int) {
	if a.hasLast && a.lastToIdx == toIndex && a.lastPos < len(a.entries) && a.entries[a.lastPos].toIndex == toIndex {
		a.entries[a.lastPos].count++
		return
	}
	for i := range a.entries {
		if a.entries[i].toIndex == toIndex {
			a.entries[i].count++
			a.lastPos = i
			a.lastToIdx = toIndex
			a.hasLast = true
			return
		}
	}
	a.entries = append(a.entries, edgeCountEntry{toIndex: toIndex, count: 1})
	a.lastPos = len(a.entries) - 1
	a.lastToIdx = toIndex
	a.hasLast = true
}

func (a *localEdgeAccumulator) addDirectoryTarget(fromDirIndex int, target directoryImportTarget) {
	if target.hasSingleDir {
		if target.singleDirIndex != fromDirIndex {
			a.add(target.singleDirIndex)
		}
		return
	}
	for _, toDirIndex := range target.dirIndices {
		if toDirIndex != fromDirIndex {
			a.add(toDirIndex)
		}
	}
}

func (a *localEdgeAccumulator) flush(builder *indexedGraphBuilder, fromIndex int) {
	flushEdgeCountEntries(builder, fromIndex, a.entries)
}

func addEdgeCountEntry(entries []edgeCountEntry, toIndex int) []edgeCountEntry {
	for i := range entries {
		if entries[i].toIndex == toIndex {
			entries[i].count++
			return entries
		}
	}
	return append(entries, edgeCountEntry{toIndex: toIndex, count: 1})
}

func flushEdgeCountEntries(builder *indexedGraphBuilder, fromIndex int, entries []edgeCountEntry) {
	fromPrefix := uint64(uint32(fromIndex)) << 32
	totalOut := uint32(0)
	for _, entry := range entries {
		builder.incrementEdgeCountPacked(fromPrefix|uint64(uint32(entry.toIndex)), entry.count)
		builder.inDegree[entry.toIndex] += uint32(entry.count)
		totalOut += uint32(entry.count)
	}
	if totalOut != 0 {
		builder.outDegree[fromIndex] += totalOut
	}
}

func newRecentDirectoryImportCache(capacity int) recentDirectoryImportCache {
	if capacity < 1 {
		capacity = 1
	}
	size := 1
	for size < capacity {
		size <<= 1
	}
	return recentDirectoryImportCache{
		entries: make([]recentDirectoryImportEntry, size),
		mask:    uint32(size - 1),
	}
}

func (c *recentDirectoryImportCache) get(key string) (resolvedDirectoryImport, bool) {
	if len(c.entries) == 0 {
		return resolvedDirectoryImport{}, false
	}
	entry := c.entries[directoryImportCacheIndex(key, c.mask)]
	if entry.key == key {
		return entry.value, true
	}
	return resolvedDirectoryImport{}, false
}

func (c *recentDirectoryImportCache) set(key string, value resolvedDirectoryImport) {
	if len(c.entries) == 0 {
		return
	}
	c.entries[directoryImportCacheIndex(key, c.mask)] = recentDirectoryImportEntry{key: key, value: value}
}

func directoryImportCacheIndex(key string, mask uint32) uint32 {
	hash := uint32(len(key))
	if len(key) > 0 {
		hash = hash*131 + uint32(key[0])
		hash = hash*131 + uint32(key[len(key)-1])
		hash = hash*131 + uint32(key[len(key)/2])
	}
	return hash & mask
}

func resolveInternalDirectoryImportTarget(importCache *recentDirectoryImportCache, packageDirs *packageDirIndex, importPath string) (directoryImportTarget, bool) {
	if resolved, ok := importCache.get(importPath); ok {
		return resolved.target, resolved.ok
	}
	baseCandidate := deriveImportBaseCandidate(importPath)
	candidate := baseCandidate
	if baseCandidate != "" && baseCandidate != importPath {
		if resolved, ok := importCache.get(baseCandidate); ok {
			importCache.set(importPath, resolved)
			return resolved.target, resolved.ok
		}
	}
	for candidate != "" {
		if target, ok := resolvePackageDirectoryTarget(packageDirs, candidate); ok {
			resolved := resolvedDirectoryImport{target: target, ok: true}
			cacheResolvedDirectoryImport(importCache, importPath, baseCandidate, candidate, resolved)
			return target, true
		}
		candidate = parentImportCandidate(candidate)
	}
	cacheMissingDirectoryImport(importCache, importPath, baseCandidate)
	return directoryImportTarget{}, false
}

func cacheResolvedDirectoryImport(importCache *recentDirectoryImportCache, importPath, baseCandidate, candidate string, resolved resolvedDirectoryImport) {
	if baseCandidate != "" {
		importCache.set(baseCandidate, resolved)
	}
	if candidate != "" {
		importCache.set(candidate, resolved)
	}
	importCache.set(importPath, resolved)
}

func cacheMissingDirectoryImport(importCache *recentDirectoryImportCache, importPath, baseCandidate string) {
	if baseCandidate != "" {
		importCache.set(baseCandidate, resolvedDirectoryImport{})
	}
	importCache.set(importPath, resolvedDirectoryImport{})
}

func parentImportCandidate(candidate string) string {
	dot := strings.LastIndexByte(candidate, '.')
	if dot < 0 {
		return ""
	}
	return candidate[:dot]
}

func resolvePackageDirectoryTarget(packageDirs *packageDirIndex, pkg string) (directoryImportTarget, bool) {
	return packageDirs.directoryTarget(pkg)
}

func resolveDirectoryImportTarget(cache map[string]directoryImportTarget, packageDirs *packageDirIndex, nodeCache *graphNodeIndexCache, builder *indexedGraphBuilder, pkg string, kind NodeKind) directoryImportTarget {
	if target, ok := cache[pkg]; ok {
		if target.hasNode {
			builder.mergeNodeKind(target.nodeIndex, kind)
		}
		return target
	}
	if target, ok := resolvePackageDirectoryTarget(packageDirs, pkg); ok {
		return target
	}
	target := directoryImportTarget{
		nodeIndex: nodeCache.ensure(builder, pkg, kind),
		hasNode:   true,
	}
	cache[pkg] = target
	return target
}

func newIndexedGraphBuilder(nodeCapacity, edgeCapacity int) *indexedGraphBuilder {
	config := newDefaultIndexedGraphBuilderConfig(nodeCapacity, edgeCapacity)
	return newIndexedGraphBuilderFromConfig(config)
}

func newIndexedGraphBuilderWithBackend(nodeCapacity, edgeCapacity int, backend indexedGraphBuilderBackend) *indexedGraphBuilder {
	return newIndexedGraphBuilderWithBackendAndRecentSlots(nodeCapacity, edgeCapacity, backend, defaultIndexedEdgeCounterRecentSlots())
}

func newIndexedGraphBuilderFromConfig(config indexedGraphBuilderConfig) *indexedGraphBuilder {
	return newIndexedGraphBuilderWithBackend(config.nodeCapacity, config.edgeCapacity, config.backend)
}

func newIndexedGraphBuilderWithBackendAndRecentSlots(nodeCapacity, edgeCapacity int, backend indexedGraphBuilderBackend, recentSlots int) *indexedGraphBuilder {
	builder := &indexedGraphBuilder{
		nameToIndex: make(map[string]int, nodeCapacity),
		names:       make([]string, 0, nodeCapacity),
		kinds:       make([]NodeKind, 0, nodeCapacity),
		inDegree:    make([]uint32, 0, nodeCapacity),
		outDegree:   make([]uint32, 0, nodeCapacity),
		backend:     backend,
	}
	if backend == indexedGraphBuilderBackendCounter {
		builder.edgeCounts = newIndexedEdgeCounterWithRecentSlots(edgeCapacity, recentSlots)
	} else {
		builder.edgeMap = make(map[uint64]uint32, edgeCapacity)
	}
	return builder
}

func newIndexedGraphBuilderConfig(nodeCapacity, edgeCapacity int, threshold int) indexedGraphBuilderConfig {
	if edgeCapacity < nodeCapacity {
		edgeCapacity = nodeCapacity
	}
	return indexedGraphBuilderConfig{
		nodeCapacity: nodeCapacity,
		edgeCapacity: edgeCapacity,
		backend:      indexedGraphBuilderBackendForThreshold(edgeCapacity, threshold),
	}
}

func newDefaultIndexedGraphBuilderConfig(nodeCapacity, edgeCapacity int) indexedGraphBuilderConfig {
	return newIndexedGraphBuilderConfig(nodeCapacity, edgeCapacity, defaultIndexedGraphBuilderThreshold())
}

func defaultIndexedGraphBuilderThreshold() int {
	return defaultPackageGraphBuilderThreshold()
}

func defaultIndexedGraphBuilderBackend(edgeCapacity int) indexedGraphBuilderBackend {
	return newDefaultIndexedGraphBuilderConfig(0, edgeCapacity).backend
}

// Threshold-based selection remains intentionally simple: current benchmarks
// show the counter backend primarily wins on larger package-style workloads,
// while smaller or directory-heavy shapes do not justify more complex rules.
func indexedGraphBuilderBackendForThreshold(edgeCapacity int, threshold int) indexedGraphBuilderBackend {
	if edgeCapacity >= threshold {
		return indexedGraphBuilderBackendCounter
	}
	return indexedGraphBuilderBackendMap
}

func (b indexedGraphBuilderBackend) usesIndexedEdgeCounter() bool {
	return b == indexedGraphBuilderBackendCounter
}

func (b indexedGraphBuilderBackend) usesMapEdges() bool {
	return b == indexedGraphBuilderBackendMap
}

func (b *indexedGraphBuilder) usesIndexedEdgeCounter() bool {
	return b.backend.usesIndexedEdgeCounter()
}

func (c indexedGraphBuilderConfig) usesIndexedEdgeCounter() bool {
	return c.backend.usesIndexedEdgeCounter()
}

func (c indexedGraphBuilderConfig) usesMapEdges() bool {
	return c.backend.usesMapEdges()
}

func (b *indexedGraphBuilder) usesMapEdges() bool {
	return b.backend.usesMapEdges()
}

func (b *indexedGraphBuilder) ensureNode(name string, kind NodeKind) int {
	if idx, ok := b.nameToIndex[name]; ok {
		b.mergeNodeKind(idx, kind)
		return idx
	}
	idx := len(b.names)
	b.nameToIndex[name] = idx
	b.names = append(b.names, name)
	b.kinds = append(b.kinds, kind)
	b.inDegree = append(b.inDegree, 0)
	b.outDegree = append(b.outDegree, 0)
	return idx
}

func (b *indexedGraphBuilder) mergeNodeKind(index int, kind NodeKind) {
	b.kinds[index] = mergeNodeKind(b.kinds[index], kind)
}

func (b *indexedGraphBuilder) addEdge(fromIndex, toIndex int) {
	b.addEdgeCount(fromIndex, toIndex, 1)
}

func (b *indexedGraphBuilder) addEdgeCount(fromIndex, toIndex, count int) {
	b.addPackedEdgeCount(fromIndex, uint64(uint32(fromIndex))<<32, toIndex, count)
}

func (b *indexedGraphBuilder) addPackedEdgeCount(fromIndex int, fromPrefix uint64, toIndex, count int) {
	key := fromPrefix | uint64(uint32(toIndex))
	b.incrementEdgeCountPacked(key, count)
	b.addNodeDegrees(fromIndex, toIndex, count)
}

func (b *indexedGraphBuilder) incrementEdgeCountPacked(key uint64, count int) {
	if b.usesIndexedEdgeCounter() {
		b.edgeCounts.add(key, uint32(count))
	} else {
		b.edgeMap[key] += uint32(count)
	}
}

func (b *indexedGraphBuilder) addNodeDegrees(fromIndex, toIndex, count int) {
	b.outDegree[fromIndex] += uint32(count)
	b.inDegree[toIndex] += uint32(count)
}

func (b *indexedGraphBuilder) edgeLen() int {
	if b.usesIndexedEdgeCounter() {
		return b.edgeCounts.len()
	}
	return len(b.edgeMap)
}

func (b *indexedGraphBuilder) rangeEdges(fn func(uint64, uint32)) {
	if b.usesIndexedEdgeCounter() {
		b.edgeCounts.rangeAll(fn)
		return
	}
	for key, count := range b.edgeMap {
		fn(key, count)
	}
}

func (b *indexedGraphBuilder) materialize() ([]Node, []Edge) {
	nodes := b.materializeNodes()
	edges := b.materializeEdges()
	return nodes, edges
}

func (b *indexedGraphBuilder) materializeNodes() []Node {
	nodes := make([]Node, len(b.names))
	for i, name := range b.names {
		nodes[i] = Node{
			ID:        i + 1,
			Name:      name,
			Kind:      b.kinds[i],
			InDegree:  int(b.inDegree[i]),
			OutDegree: int(b.outDegree[i]),
		}
	}
	return nodes
}

func (b *indexedGraphBuilder) materializeFilteredNodes(includeNode, usedNodeIDs []bool, hasPrefixFilter bool, newIDByOldID []int) []Node {
	keptNodeCount := 0
	for i := range b.names {
		if shouldKeepFilteredNode(i+1, includeNode, usedNodeIDs, hasPrefixFilter) {
			keptNodeCount++
		}
	}

	nodes := make([]Node, 0, keptNodeCount)
	for i, name := range b.names {
		oldID := i + 1
		if !shouldKeepFilteredNode(oldID, includeNode, usedNodeIDs, hasPrefixFilter) {
			continue
		}
		newID := len(nodes) + 1
		newIDByOldID[oldID] = newID
		nodes = append(nodes, Node{ID: newID, Name: name, Kind: b.kinds[i]})
	}
	return nodes
}

func assignMaterializedNodeDegrees(nodes []Node, inDegree, outDegree []int) {
	for i := range nodes {
		id := nodes[i].ID
		nodes[i].InDegree = inDegree[id]
		nodes[i].OutDegree = outDegree[id]
	}
}

func (b *indexedGraphBuilder) materializeEdges() []Edge {
	edges := make([]Edge, 0, b.edgeLen())
	b.rangeEdges(func(key uint64, count uint32) {
		fromIndex, toIndex := unpackIndexedEdgeKey(key)
		edges = append(edges, Edge{FromID: fromIndex + 1, ToID: toIndex + 1, Count: int(count)})
	})
	return edges
}

func (b *indexedGraphBuilder) materializeFilteredEdges(filter GraphFilter, includeNode []bool, hasPrefixFilter bool, newIDByOldID []int, edgeCapacity int, inDegree, outDegree []int) []Edge {
	edges := make([]Edge, 0, edgeCapacity)
	b.rangeEdges(func(key uint64, count uint32) {
		edges = appendFilteredMaterializedEdge(edges, key, count, filter, includeNode, hasPrefixFilter, newIDByOldID, inDegree, outDegree)
	})
	return edges
}

func appendFilteredMaterializedEdge(edges []Edge, key uint64, count uint32, filter GraphFilter, includeNode []bool, hasPrefixFilter bool, newIDByOldID []int, inDegree, outDegree []int) []Edge {
	if int(count) < filter.MinEdgeCount {
		return edges
	}
	fromIndex, toIndex := unpackIndexedEdgeKey(key)
	if hasPrefixFilter {
		if fromIndex+1 >= len(includeNode) || !includeNode[fromIndex+1] {
			return edges
		}
		if toIndex+1 >= len(includeNode) || !includeNode[toIndex+1] {
			return edges
		}
	}
	fromID := newIDByOldID[fromIndex+1]
	toID := newIDByOldID[toIndex+1]
	if fromID == 0 || toID == 0 {
		return edges
	}
	edges = append(edges, Edge{FromID: fromID, ToID: toID, Count: int(count)})
	outDegree[fromID]++
	inDegree[toID]++
	return edges
}

func (b *indexedGraphBuilder) materializeFiltered(filter GraphFilter) ([]Node, []Edge) {
	filter = normalizeFilter(filter)
	if !filter.isActive() {
		return b.materialize()
	}

	maxNodeID := len(b.names)
	scratch := getFilterScratch(maxNodeID + 1)
	defer putFilterScratch(scratch)

	hasPrefixFilter := len(filter.IncludePrefix) > 0 || len(filter.ExcludePrefix) > 0
	includeNode := scratch.includeNode[:0]
	if hasPrefixFilter {
		includeNode = scratch.includeNode
		for i, name := range b.names {
			if shouldIncludeNode(name, filter) {
				includeNode[i+1] = true
			}
		}
	}

	usedNodeIDs := scratch.usedNodeIDs
	filteredEdgeCount := 0
	b.rangeEdges(func(key uint64, count uint32) {
		if int(count) < filter.MinEdgeCount {
			return
		}
		fromIndex, toIndex := unpackIndexedEdgeKey(key)
		fromID := fromIndex + 1
		toID := toIndex + 1
		if hasPrefixFilter {
			if fromID >= len(includeNode) || !includeNode[fromID] {
				return
			}
			if toID >= len(includeNode) || !includeNode[toID] {
				return
			}
		}
		filteredEdgeCount++
		usedNodeIDs[fromID] = true
		usedNodeIDs[toID] = true
	})

	newIDByOldID := scratch.newIDByOldID
	nodes := b.materializeFilteredNodes(includeNode, usedNodeIDs, hasPrefixFilter, newIDByOldID)
	if len(nodes) == 0 {
		return nil, nil
	}

	edges := make([]Edge, 0, filteredEdgeCount)
	inDegree := ensureScratchInts(scratch.inDegree, len(nodes)+1)
	outDegree := ensureScratchInts(scratch.outDegree, len(nodes)+1)
	scratch.inDegree = inDegree
	scratch.outDegree = outDegree
	edges = b.materializeFilteredEdges(filter, includeNode, hasPrefixFilter, newIDByOldID, filteredEdgeCount, inDegree, outDegree)
	assignMaterializedNodeDegrees(nodes, inDegree, outDegree)
	return nodes, edges
}

func packIndexedEdgeKey(fromIndex, toIndex int) uint64 {
	return uint64(uint32(fromIndex))<<32 | uint64(uint32(toIndex))
}

func unpackIndexedEdgeKey(key uint64) (int, int) {
	return int(uint32(key >> 32)), int(uint32(key))
}

func newIndexedEdgeCounter(capacity int) indexedEdgeCounter {
	return newIndexedEdgeCounterWithRecentSlots(capacity, defaultIndexedEdgeCounterRecentSlots())
}

func defaultIndexedEdgeCounterRecentSlots() int {
	return indexedEdgeCounterRecentSlots
}

func newIndexedEdgeCounterWithRecentSlots(capacity int, recentSlots int) indexedEdgeCounter {
	slots := nextIndexedEdgeCounterSize(capacity)
	counter := indexedEdgeCounter{
		keys:   make([]uint64, slots),
		counts: make([]uint32, slots),
		mask:   uint64(slots - 1),
	}
	if recentSlots > 0 {
		size := nextIndexedEdgeCounterRecentSize(recentSlots)
		counter.recent = make([]indexedEdgeCounterRecent, size)
		counter.recentMask = uint64(size - 1)
	}
	return counter
}

func nextIndexedEdgeCounterSize(capacity int) int {
	size := 16
	target := maxInt(1, capacity*4)
	for size < target {
		size <<= 1
	}
	return size
}

func nextIndexedEdgeCounterRecentSize(size int) int {
	if size <= 0 {
		return 0
	}
	if size == 1 {
		return 1
	}
	pow2 := 1
	for pow2 < size {
		pow2 <<= 1
	}
	return pow2
}

func (c *indexedEdgeCounter) len() int {
	if c.hasZero {
		return c.size + 1
	}
	return c.size
}

func (c *indexedEdgeCounter) add(key uint64, count uint32) {
	if key == 0 {
		c.zeroCount += count
		c.hasZero = true
		return
	}
	if idx, ok := c.recentIndex(key); ok {
		c.counts[idx] += count
		return
	}
	if c.needsGrow() {
		c.grow()
	}
	idx, found := c.findIndex(key)
	if found {
		c.counts[idx] += count
		c.setRecent(key, idx)
		return
	}
	c.insertAt(idx, key, count)
}

func (c *indexedEdgeCounter) recentIndex(key uint64) (int, bool) {
	if len(c.recent) == 0 {
		return 0, false
	}
	recent := c.recent[c.recentSlot(key)]
	if recent.key == key {
		idx := int(recent.idx)
		if idx < len(c.keys) && c.keys[idx] == key {
			return idx, true
		}
	}
	return 0, false
}

func (c *indexedEdgeCounter) setRecent(key uint64, idx int) {
	if len(c.recent) == 0 {
		return
	}
	c.recent[c.recentSlot(key)] = indexedEdgeCounterRecent{key: key, idx: uint32(idx)}
}

func (c *indexedEdgeCounter) recentSlot(key uint64) int {
	return int(key>>32^key) & int(c.recentMask)
}

func (c *indexedEdgeCounter) findIndex(key uint64) (int, bool) {
	mask := int(c.mask)
	idx := int(indexedEdgeCounterHash(key) & c.mask)
	stored := c.keys[idx]
	if stored == 0 {
		return idx, false
	}
	if stored == key {
		return idx, true
	}

	idx = (idx + 1) & mask
	stored = c.keys[idx]
	if stored == 0 {
		return idx, false
	}
	if stored == key {
		return idx, true
	}

	idx = (idx + 1) & mask
	stored = c.keys[idx]
	if stored == 0 {
		return idx, false
	}
	if stored == key {
		return idx, true
	}

	for {
		idx = (idx + 1) & mask
		stored = c.keys[idx]
		if stored == 0 {
			return idx, false
		}
		if stored == key {
			return idx, true
		}
	}
}

func (c *indexedEdgeCounter) insertAt(idx int, key uint64, count uint32) {
	c.keys[idx] = key
	c.counts[idx] = count
	c.size++
}

func (c *indexedEdgeCounter) needsGrow() bool {
	return (c.size+1)*10 >= len(c.keys)*7
}

func (c *indexedEdgeCounter) grow() {
	oldKeys := c.keys
	oldCounts := c.counts
	newSize := len(oldKeys) << 1
	if newSize == 0 {
		newSize = 16
	}
	c.keys = make([]uint64, newSize)
	c.counts = make([]uint32, newSize)
	c.mask = uint64(newSize - 1)
	c.size = 0
	if len(c.recent) > 0 {
		clear(c.recent)
	}
	for i, key := range oldKeys {
		if key != 0 {
			c.add(key, oldCounts[i])
		}
	}
}

func (c *indexedEdgeCounter) rangeAll(fn func(uint64, uint32)) {
	if c.hasZero {
		fn(0, c.zeroCount)
	}
	for i, key := range c.keys {
		if key != 0 {
			fn(key, c.counts[i])
		}
	}
}

func indexedEdgeCounterHash(key uint64) uint64 {
	return key * 11400714819323198485
}

func sortGraph(graph Graph) Graph {
	if len(graph.Nodes) == 0 {
		return graph
	}
	if graph.GroupBy == GroupByDir {
		return deterministicDirGraphSortPolicy.sort(graph)
	}
	return deterministicGraphSortPolicy.sort(graph)
}

func (p graphSortPolicy) sort(graph Graph) Graph {
	scratch := getSortScratch(len(graph.Nodes) + 1)
	defer putSortScratch(scratch)

	p.sortNodes(graph.Nodes)
	oldToNew := scratch.oldToNew
	for i := range graph.Nodes {
		oldID := graph.Nodes[i].ID
		newID := i + 1
		oldToNew[oldID] = newID
		graph.Nodes[i].ID = newID
	}
	for i := range graph.Edges {
		graph.Edges[i].FromID = oldToNew[graph.Edges[i].FromID]
		graph.Edges[i].ToID = oldToNew[graph.Edges[i].ToID]
	}
	p.sortEdges(graph.Edges, len(graph.Nodes))
	return graph
}

func (p graphSortPolicy) sortNodes(nodes []Node) {
	if len(nodes) < 2 {
		return
	}
	if p.checkSortedNodes && areNodesSortedByName(nodes) {
		return
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})
}

func (p graphSortPolicy) sortEdges(edges []Edge, maxID int) {
	if len(edges) < 2 {
		return
	}
	if p.useCountingEdges {
		sortEdgesByIDsWithMaxID(edges, maxID)
		return
	}
	if p.checkSortedEdges && areEdgesSorted(edges) {
		return
	}
	sort.Slice(edges, func(i, j int) bool {
		return compareEdges(edges[i], edges[j]) < 0
	})
}

func sortEdgesByIDs(edges []Edge) {
	maxID := 0
	for _, edge := range edges {
		if edge.FromID > maxID {
			maxID = edge.FromID
		}
		if edge.ToID > maxID {
			maxID = edge.ToID
		}
	}
	sortEdgesByIDsWithMaxID(edges, maxID)
}

func sortEdgesByIDsWithMaxID(edges []Edge, maxID int) {
	scratch := getSortScratch(maxID + len(edges) + 1)
	defer putSortScratch(scratch)
	counts := ensureSortScratchInts(scratch.counts, maxID+1)
	scratch.counts = counts
	buf := ensureSortScratchEdges(scratch.edgeBuf, len(edges))
	scratch.edgeBuf = buf
	countingSortEdgesByField(edges, buf, counts, func(edge Edge) int { return edge.ToID })
	countingSortEdgesByField(buf, edges, counts, func(edge Edge) int { return edge.FromID })
}

func countingSortEdgesByField(src, dst []Edge, counts []int, key func(Edge) int) {
	clear(counts)
	for _, edge := range src {
		counts[key(edge)]++
	}
	for i := 1; i < len(counts); i++ {
		counts[i] += counts[i-1]
	}
	for i := len(src) - 1; i >= 0; i-- {
		edge := src[i]
		index := key(edge)
		counts[index]--
		dst[counts[index]] = edge
	}
}

func areNodesSortedByName(nodes []Node) bool {
	for i := 1; i < len(nodes); i++ {
		if nodes[i-1].Name > nodes[i].Name {
			return false
		}
	}
	return true
}

func areEdgesSorted(edges []Edge) bool {
	for i := 1; i < len(edges); i++ {
		if compareEdges(edges[i-1], edges[i]) > 0 {
			return false
		}
	}
	return true
}

func compareEdges(a, b Edge) int {
	if a.FromID != b.FromID {
		if a.FromID < b.FromID {
			return -1
		}
		return 1
	}
	if a.ToID != b.ToID {
		if a.ToID < b.ToID {
			return -1
		}
		return 1
	}
	if a.Count > b.Count {
		return -1
	}
	if a.Count < b.Count {
		return 1
	}
	return 0
}

func getSortScratch(size int) *sortScratch {
	scratch := sortScratchPool.Get().(*sortScratch)
	scratch.oldToNew = ensureSortScratchInts(scratch.oldToNew, size)
	return scratch
}

func putSortScratch(scratch *sortScratch) {
	clear(scratch.oldToNew)
	sortScratchPool.Put(scratch)
}

func ensureSortScratchInts(buf []int, size int) []int {
	if cap(buf) < size {
		return make([]int, size)
	}
	buf = buf[:size]
	clear(buf)
	return buf
}

func ensureSortScratchEdges(buf []Edge, size int) []Edge {
	if cap(buf) < size {
		return make([]Edge, size)
	}
	return buf[:size]
}

func estimateInternalPackageCapacity(fileCount int) int {
	if fileCount < 16 {
		return fileCount
	}
	return maxInt(16, fileCount/8)
}

func estimatePackageBuilderCapacities(fileCount int, packageCount int) (int, int) {
	nodeCapacity := maxInt(16, packageCount)
	edgeCapacity := maxInt(nodeCapacity, fileCount)
	return nodeCapacity, edgeCapacity
}

func newPackageGraphBuilderConfig(fileCount int, packageCount int, threshold int) indexedGraphBuilderConfig {
	nodeCapacity, edgeCapacity := estimatePackageBuilderCapacities(fileCount, packageCount)
	return newIndexedGraphBuilderConfig(nodeCapacity, edgeCapacity, threshold)
}

func newDefaultPackageGraphBuilderConfig(fileCount int, packageCount int) indexedGraphBuilderConfig {
	return newPackageGraphBuilderConfig(fileCount, packageCount, defaultPackageGraphBuilderThreshold())
}

func defaultPackageGraphBuilderThreshold() int {
	return packageIndexedEdgeCounterMinCapacity
}

// Package graphs keep the larger default threshold because the latest sweep
// still shows the counter backend winning there despite higher alloc counts.
func packageGraphBuilderBackend(fileCount int, packageCount int, threshold int) indexedGraphBuilderBackend {
	return newPackageGraphBuilderConfig(fileCount, packageCount, threshold).backend
}

func defaultPackageGraphBuilderBackend(fileCount int, packageCount int) indexedGraphBuilderBackend {
	return newDefaultPackageGraphBuilderConfig(fileCount, packageCount).backend
}

func estimateDirectoryBuilderCapacities(fileCount int, packageCount int) (int, int, int) {
	nodeCapacity := maxInt(16, packageCount)
	edgeCapacity := maxInt(nodeCapacity, fileCount/2)
	packageCapacity := maxInt(16, packageCount)
	return nodeCapacity, edgeCapacity, packageCapacity
}

func newDirectoryGraphBuilderConfig(fileCount int, packageCount int, threshold int) (indexedGraphBuilderConfig, int) {
	nodeCapacity, edgeCapacity, packageCapacity := estimateDirectoryBuilderCapacities(fileCount, packageCount)
	return newIndexedGraphBuilderConfig(nodeCapacity, edgeCapacity, threshold), packageCapacity
}

func newDefaultDirectoryGraphBuilderConfig(fileCount int, packageCount int) (indexedGraphBuilderConfig, int) {
	return newDirectoryGraphBuilderConfig(fileCount, packageCount, defaultDirectoryGraphBuilderThreshold())
}

func defaultDirectoryGraphBuilderThreshold() int {
	return directoryIndexedEdgeCounterMinCapacity
}

// Directory graphs currently share the same default threshold, but only
// because 16384 has not shown a stable CPU win over 32768 in the sweep.
func directoryGraphBuilderBackend(fileCount int, packageCount int, threshold int) indexedGraphBuilderBackend {
	config, _ := newDirectoryGraphBuilderConfig(fileCount, packageCount, threshold)
	return config.backend
}

func defaultDirectoryGraphBuilderBackend(fileCount int, packageCount int) indexedGraphBuilderBackend {
	config, _ := newDefaultDirectoryGraphBuilderConfig(fileCount, packageCount)
	return config.backend
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func detectKind(isInternal bool, hasExternalEvidence bool) NodeKind {
	if isInternal {
		return NodeInternal
	}
	if hasExternalEvidence {
		return NodeExternal
	}
	return NodeUnknown
}

func mergeNodeKind(current, incoming NodeKind) NodeKind {
	if current == incoming {
		return current
	}
	if current == "" {
		return incoming
	}
	if current == NodeInternal || incoming == NodeInternal {
		return NodeInternal
	}
	if current == NodeExternal || incoming == NodeExternal {
		return NodeExternal
	}
	return NodeUnknown
}

func normalizeDir(path string) string {
	if path == "" {
		return ""
	}
	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == "." {
		return ""
	}
	return strings.Trim(dir, "/")
}

func resolveImportPackage(importPath string, internalPackages map[string]struct{}, external kcg.ExternalIndex) string {
	return resolveImportPackageNormalized(normalizePath(importPath), internalPackages, external)
}

func inferImportPackage(importPath string) string {
	return inferImportPackageNormalized(normalizePath(importPath))
}

func resolveImportPackageNormalized(path string, internalPackages map[string]struct{}, external kcg.ExternalIndex) string {
	return resolveImportTargetNormalized(path, internalPackages, external).Package
}

func resolveImportTarget(importPath string, internalPackages map[string]struct{}, external kcg.ExternalIndex) resolvedImportTarget {
	return newImportClassifier(internalPackages, external).resolve(importPath)
}

func resolveImportTargetNormalized(path string, internalPackages map[string]struct{}, external kcg.ExternalIndex) resolvedImportTarget {
	return newImportClassifier(internalPackages, external).resolveNormalized(path)
}

func newImportClassifier(internalPackages map[string]struct{}, external kcg.ExternalIndex) importClassifier {
	classifier := importClassifier{
		internalPackages: internalPackages,
		external:         external,
		resolved:         make(map[string]resolvedImportTarget, minInt(maxCachedResolvedImports, 64)),
	}
	if importClassifierRecentRawSlots > 0 {
		classifier.recentRaw = make([]recentResolvedImport, importClassifierRecentRawSlots)
		classifier.recentRawMask = uint32(importClassifierRecentRawSlots - 1)
	}
	return classifier
}

func (c importClassifier) resolve(importPath string) resolvedImportTarget {
	if target, ok := c.getRecentRaw(importPath); ok {
		return target
	}
	target := c.resolveNormalized(normalizePath(importPath))
	c.setRecentRaw(importPath, target)
	return target
}

func (c importClassifier) getRecentRaw(importPath string) (resolvedImportTarget, bool) {
	if len(c.recentRaw) == 0 || importPath == "" {
		return resolvedImportTarget{}, false
	}
	entry := c.recentRaw[recentRawImportIndex(importPath, c.recentRawMask)]
	if entry.key == importPath {
		return entry.target, true
	}
	return resolvedImportTarget{}, false
}

func (c importClassifier) setRecentRaw(importPath string, target resolvedImportTarget) {
	if len(c.recentRaw) == 0 || importPath == "" {
		return
	}
	c.recentRaw[recentRawImportIndex(importPath, c.recentRawMask)] = recentResolvedImport{
		key:    importPath,
		target: target,
	}
}

func recentRawImportIndex(key string, mask uint32) uint32 {
	hash := uint32(len(key))
	if len(key) > 0 {
		hash = hash*131 + uint32(key[0])
		hash = hash*131 + uint32(key[len(key)-1])
		hash = hash*131 + uint32(key[len(key)/2])
	}
	return hash & mask
}

func (c importClassifier) resolveNormalized(path string) resolvedImportTarget {
	if path == "" {
		return resolvedImportTarget{}
	}
	if len(c.resolved) > 0 {
		if target, ok := c.resolved[path]; ok {
			return target
		}
	}
	target := c.resolveNormalizedUncached(path)
	c.cacheResolved(path, target)
	return target
}

func (c importClassifier) resolveNormalizedUncached(path string) resolvedImportTarget {
	base := deriveImportBaseCandidate(path)
	if base == "" {
		return resolvedImportTarget{}
	}
	if target, ok := c.resolveKnownCandidate(base); ok {
		return target
	}
	if target, ok := c.resolveAncestorCandidate(path); ok {
		return target
	}
	if target, ok := c.resolveFallbackTarget(base, path); ok {
		return target
	}
	return c.classifyTarget(base, path)
}

func (c importClassifier) cacheResolved(path string, target resolvedImportTarget) {
	if !shouldCacheResolvedImport(target) {
		return
	}
	if len(c.resolved) >= maxCachedResolvedImports {
		return
	}
	c.resolved[path] = target
}

func shouldCacheResolvedImport(target resolvedImportTarget) bool {
	return target.Package != ""
}

func resolveKnownImportCandidate(candidate string, internalPackages map[string]struct{}, external kcg.ExternalIndex) (resolvedImportTarget, bool) {
	return newImportClassifier(internalPackages, external).resolveKnownCandidate(candidate)
}

func (c importClassifier) resolveKnownCandidate(candidate string) (resolvedImportTarget, bool) {
	if candidate == "" {
		return resolvedImportTarget{}, false
	}
	if c.hasInternalPackage(candidate) {
		return resolvedImportTarget{Package: candidate, Kind: NodeInternal}, true
	}
	if hasExternalPackage(candidate, c.external) || isKnownExternalPackageCandidate(candidate) {
		return resolvedImportTarget{Package: candidate, Kind: NodeExternal}, true
	}
	return resolvedImportTarget{}, false
}

func resolveAncestorImportCandidate(path string, internalPackages map[string]struct{}, external kcg.ExternalIndex) (resolvedImportTarget, bool) {
	return newImportClassifier(internalPackages, external).resolveAncestorCandidate(path)
}

func (c importClassifier) resolveAncestorCandidate(path string) (resolvedImportTarget, bool) {
	for candidate := parentImportCandidate(trimImportWildcard(path)); candidate != ""; candidate = parentImportCandidate(candidate) {
		if target, ok := c.resolveKnownCandidate(candidate); ok {
			return target, true
		}
	}
	return resolvedImportTarget{}, false
}

func resolveFallbackImportTarget(base, importPath string, internalPackages map[string]struct{}, external kcg.ExternalIndex) (resolvedImportTarget, bool) {
	return newImportClassifier(internalPackages, external).resolveFallbackTarget(base, importPath)
}

func (c importClassifier) resolveFallbackTarget(base, importPath string) (resolvedImportTarget, bool) {
	fallback := fallbackTrimmedImportPackage(base, importPath)
	if fallback == "" {
		return resolvedImportTarget{}, false
	}
	return c.classifyTarget(fallback, importPath), true
}

func (c importClassifier) classifyTarget(pkg, importPath string) resolvedImportTarget {
	return resolvedImportTarget{
		Package: pkg,
		Kind:    detectKind(c.hasInternalPackage(pkg), hasExternalImportEvidence(pkg, importPath, c.external)),
	}
}

func (c importClassifier) hasInternalPackage(pkg string) bool {
	_, ok := c.internalPackages[pkg]
	return ok
}

func deriveImportBaseCandidate(path string) string {
	path = trimImportWildcard(path)
	if path == "" {
		return ""
	}
	last := lastImportSegment(path)
	if looksLikeTypeSegment(last) {
		return parentImportCandidate(path)
	}
	parent := parentImportCandidate(path)
	if parent != "" && looksLikeTypeSegment(lastImportSegment(parent)) {
		return parent
	}
	return path
}

func trimImportWildcard(path string) string {
	if strings.HasSuffix(path, ".*") {
		return strings.TrimSuffix(path, ".*")
	}
	return path
}

func lastImportSegment(path string) string {
	if dot := strings.LastIndexByte(path, '.'); dot >= 0 {
		return path[dot+1:]
	}
	return path
}

func looksLikeTypeSegment(segment string) bool {
	r, _ := utf8.DecodeRuneInString(segment)
	return unicode.IsUpper(r)
}

func trimClassLikeSegment(path string) string {
	parent := parentImportCandidate(path)
	if parent == "" {
		return path
	}
	if looksLikeTypeSegment(lastImportSegment(path)) {
		return parent
	}
	return path
}

func fallbackTrimmedImportPackage(base, importPath string) string {
	if base == "" {
		return ""
	}
	trimmed := trimClassLikeSegment(base)
	if trimmed != "" && trimmed != base {
		return trimmed
	}
	if base == importPath {
		if parent := parentImportCandidate(base); parent != "" {
			return parent
		}
	}
	return base
}

func inferImportPackageNormalized(path string) string {
	if strings.HasSuffix(path, ".*") {
		return strings.TrimSuffix(path, ".*")
	}
	path = trimImportWildcard(path)
	if path == "" {
		return ""
	}
	base := deriveImportBaseCandidate(path)
	if base == "" {
		return ""
	}
	return fallbackTrimmedImportPackage(base, path)
}

func hasExternalImportEvidence(pkg string, importPath string, external kcg.ExternalIndex) bool {
	return hasKnownExternalImportEvidence(pkg, importPath) || hasIndexedExternalImportEvidence(pkg, importPath, external)
}

func hasKnownExternalImportEvidence(pkg string, importPath string) bool {
	return isKnownExternalImportNormalized(pkg) || isKnownExternalImportNormalized(importPath)
}

func hasIndexedExternalImportEvidence(pkg string, importPath string, external kcg.ExternalIndex) bool {
	if pkg == "" {
		return false
	}
	if hasExternalPackage(pkg, external) {
		return true
	}
	if importPath == "" || importPath == pkg {
		return false
	}
	_, ok := external.Symbols[importPath]
	return ok
}

func hasExternalPackage(pkg string, external kcg.ExternalIndex) bool {
	_, ok := external.Packages[pkg]
	return ok
}

func isKnownExternalImport(importPath string) bool {
	return isKnownExternalImportNormalized(normalizePath(importPath))
}

func isKnownExternalImportNormalized(path string) bool {
	if path == "" || path == ".*" {
		return false
	}
	path = trimImportWildcard(path)
	for _, root := range knownExternalRoots {
		if hasExactOrPrefix(path, root) {
			return true
		}
	}
	for _, prefix := range knownExternalPrefixes {
		if hasExactOrPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func hasExactOrPrefix(path string, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+".")
}

func isKnownExternalPackageCandidate(path string) bool {
	for _, root := range knownExternalRoots {
		if path == root {
			return true
		}
	}
	for _, prefix := range knownExternalPrefixes {
		if path == prefix {
			return true
		}
	}
	return false
}
