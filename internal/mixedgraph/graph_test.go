package mixedgraph

import (
	"reflect"
	"sort"
	"testing"

	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

func TestBuildGraphByPackage(t *testing.T) {
	result := RepositoryResult{
		Root: "/repo",
		Files: []FileUnit{
			{
				Path:        "/repo/a/A.java",
				Relative:    "a/A.java",
				Language:    LangJava,
				PackageName: "com.a",
				Imports: []string{
					"com.b.Service",
					"java.util.List",
					"java.util.Collections.emptyList",
				},
			},
			{
				Path:        "/repo/b/B.kt",
				Relative:    "b/B.kt",
				Language:    LangKotlin,
				PackageName: "com.b",
				Imports: []string{
					"com.a.Model",
					"kotlin.js.Promise",
				},
			},
		},
	}

	external := kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
			"kotlin.js": {},
		},
		Symbols: map[string]struct{}{
			"kotlin.js.Promise": {},
		},
	}

	graph := BuildGraph(result, external, GroupByPackage)
	if graph.GroupBy != GroupByPackage {
		t.Fatalf("expected group_by=package, got %s", graph.GroupBy)
	}
	if len(graph.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 4 {
		t.Fatalf("expected 4 edges, got %d", len(graph.Edges))
	}

	nodeByName := map[string]Node{}
	for _, node := range graph.Nodes {
		nodeByName[node.Name] = node
	}
	if nodeByName["com.a"].Kind != NodeInternal {
		t.Fatalf("expected com.a internal")
	}
	if nodeByName["com.b"].Kind != NodeInternal {
		t.Fatalf("expected com.b internal")
	}
	if nodeByName["java.util"].Kind != NodeExternal {
		t.Fatalf("expected java.util external")
	}
	if nodeByName["kotlin.js"].Kind != NodeExternal {
		t.Fatalf("expected kotlin.js external")
	}

	nameByID := map[int]string{}
	for _, node := range graph.Nodes {
		nameByID[node.ID] = node.Name
	}
	edgeCount := 0
	for _, edge := range graph.Edges {
		if nameByID[edge.FromID] == "com.a" && nameByID[edge.ToID] == "java.util" {
			edgeCount = edge.Count
			break
		}
	}
	if edgeCount != 2 {
		t.Fatalf("expected com.a -> java.util count=2, got %d", edgeCount)
	}
}

func TestPackageGraphBuilderAddImports(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{
			{
				PackageName: "com.a",
				Imports: []string{
					"com.b.Service",
					"java.util.List",
					"java.util.Collections.emptyList",
					"com.a.Self",
				},
			},
			{
				PackageName: "com.b",
				Imports: []string{
					"com.a.Model",
				},
			},
		},
	}
	ctx := newGraphBuildContext(result.Files, kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
		},
	})
	builder := newPackageGraphBuilder(ctx, len(result.Files))
	builder.seedInternalPackages()
	builder.addImports(result.Files)

	nodes, edges := builder.builder.materialize()
	sorted := sortGraph(Graph{
		GroupBy: GroupByPackage,
		Nodes:   nodes,
		Edges:   edges,
	})
	nodes, edges = sorted.Nodes, sorted.Edges

	if len(nodes) != 3 {
		t.Fatalf("len(nodes) = %d, want 3", len(nodes))
	}
	if len(edges) != 3 {
		t.Fatalf("len(edges) = %d, want 3", len(edges))
	}

	nodeByName := map[string]Node{}
	for _, node := range nodes {
		nodeByName[node.Name] = node
	}
	if nodeByName["com.a"].Kind != NodeInternal {
		t.Fatalf("nodeByName[com.a].Kind = %q, want %q", nodeByName["com.a"].Kind, NodeInternal)
	}
	if nodeByName["com.b"].Kind != NodeInternal {
		t.Fatalf("nodeByName[com.b].Kind = %q, want %q", nodeByName["com.b"].Kind, NodeInternal)
	}
	if nodeByName["java.util"].Kind != NodeExternal {
		t.Fatalf("nodeByName[java.util].Kind = %q, want %q", nodeByName["java.util"].Kind, NodeExternal)
	}
}

func TestBuildGraphInternalUnsortedSortsToPublicGraph(t *testing.T) {
	result := RepositoryResult{
		Root: "/repo",
		Files: []FileUnit{
			{
				Path:        "/repo/a/A.java",
				Relative:    "a/A.java",
				Language:    LangJava,
				PackageName: "com.a",
				Imports: []string{
					"com.b.Service",
					"java.util.List",
					"java.util.Collections.emptyList",
				},
			},
			{
				Path:        "/repo/b/B.kt",
				Relative:    "b/B.kt",
				Language:    LangKotlin,
				PackageName: "com.b",
				Imports: []string{
					"com.a.Model",
					"kotlin.js.Promise",
				},
			},
		},
	}
	external := kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
			"kotlin.js": {},
		},
		Symbols: map[string]struct{}{
			"kotlin.js.Promise": {},
		},
	}

	got := sortGraph(BuildGraphInternalUnsorted(result, external, GroupByPackage))
	want := BuildGraph(result, external, GroupByPackage)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sorted internal graph mismatch: got=%+v want=%+v", got, want)
	}
}

func TestSortEdgesByIDsMatchesComparisonOrder(t *testing.T) {
	edges := []Edge{
		{FromID: 3, ToID: 2, Count: 1},
		{FromID: 1, ToID: 4, Count: 5},
		{FromID: 1, ToID: 2, Count: 2},
		{FromID: 2, ToID: 1, Count: 7},
		{FromID: 2, ToID: 3, Count: 9},
	}
	want := append([]Edge(nil), edges...)
	sort.Slice(want, func(i, j int) bool {
		return compareEdges(want[i], want[j]) < 0
	})

	got := append([]Edge(nil), edges...)
	sortEdgesByIDs(got)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sortEdgesByIDs mismatch: got=%v want=%v", got, want)
	}
}

func TestSortEdgesByIDsWithMaxIDMatchesComparisonOrder(t *testing.T) {
	edges := []Edge{
		{FromID: 3, ToID: 2, Count: 1},
		{FromID: 1, ToID: 4, Count: 5},
		{FromID: 1, ToID: 2, Count: 2},
		{FromID: 2, ToID: 1, Count: 7},
		{FromID: 2, ToID: 3, Count: 9},
	}
	want := append([]Edge(nil), edges...)
	sort.Slice(want, func(i, j int) bool {
		return compareEdges(want[i], want[j]) < 0
	})

	got := append([]Edge(nil), edges...)
	sortEdgesByIDsWithMaxID(got, 4)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sortEdgesByIDsWithMaxID mismatch: got=%v want=%v", got, want)
	}
}

func TestIndexedGraphBuilderMaterializeAggregatesZeroEdgeKey(t *testing.T) {
	builder := newIndexedGraphBuilder(2, 2)
	a := builder.ensureNode("a", NodeInternal)
	b := builder.ensureNode("b", NodeInternal)

	builder.addEdgeCount(a, a, 2)
	builder.addEdgeCount(a, a, 3)
	builder.addEdgeCount(a, b, 1)

	nodes, edges := builder.materialize()
	graph := sortGraph(Graph{
		GroupBy: GroupByPackage,
		Nodes:   nodes,
		Edges:   edges,
	})

	if len(graph.Edges) != 2 {
		t.Fatalf("len(graph.Edges) = %d, want 2", len(graph.Edges))
	}
	if graph.Edges[0].Count != 5 {
		t.Fatalf("graph.Edges[0].Count = %d, want 5", graph.Edges[0].Count)
	}
}

func TestIndexedGraphBuilderMaterializeNodes(t *testing.T) {
	builder := newIndexedGraphBuilder(2, 2)
	a := builder.ensureNode("a", NodeInternal)
	b := builder.ensureNode("b", NodeExternal)
	builder.addEdgeCount(a, b, 3)
	builder.addEdgeCount(b, a, 2)

	nodes := builder.materializeNodes()
	want := []Node{
		{ID: 1, Name: "a", Kind: NodeInternal, InDegree: 2, OutDegree: 3},
		{ID: 2, Name: "b", Kind: NodeExternal, InDegree: 3, OutDegree: 2},
	}
	if !reflect.DeepEqual(nodes, want) {
		t.Fatalf("materializeNodes mismatch: got=%+v want=%+v", nodes, want)
	}
}

func TestIndexedGraphBuilderMaterializeFilteredNodes(t *testing.T) {
	builder := newIndexedGraphBuilder(3, 3)
	builder.ensureNode("a", NodeInternal)
	builder.ensureNode("b", NodeExternal)
	builder.ensureNode("c", NodeUnknown)

	includeNode := make([]bool, 4)
	usedNodeIDs := make([]bool, 4)
	newIDByOldID := make([]int, 4)
	usedNodeIDs[1] = true
	usedNodeIDs[3] = true

	nodes := builder.materializeFilteredNodes(includeNode, usedNodeIDs, false, newIDByOldID)
	wantNodes := []Node{
		{ID: 1, Name: "a", Kind: NodeInternal},
		{ID: 2, Name: "c", Kind: NodeUnknown},
	}
	if !reflect.DeepEqual(nodes, wantNodes) {
		t.Fatalf("materializeFilteredNodes mismatch: got=%+v want=%+v", nodes, wantNodes)
	}

	wantIDMap := []int{0, 1, 0, 2}
	if !reflect.DeepEqual(newIDByOldID, wantIDMap) {
		t.Fatalf("newIDByOldID mismatch: got=%v want=%v", newIDByOldID, wantIDMap)
	}
}

func TestIndexedGraphBuilderMaterializeEdges(t *testing.T) {
	builder := newIndexedGraphBuilder(2, 2)
	a := builder.ensureNode("a", NodeInternal)
	b := builder.ensureNode("b", NodeExternal)
	builder.addEdgeCount(a, b, 3)
	builder.addEdgeCount(b, a, 2)

	got := sortGraph(Graph{GroupBy: GroupByPackage, Nodes: builder.materializeNodes(), Edges: builder.materializeEdges()}).Edges
	want := []Edge{
		{FromID: 1, ToID: 2, Count: 3},
		{FromID: 2, ToID: 1, Count: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("materializeEdges mismatch: got=%+v want=%+v", got, want)
	}
}

func TestIndexedGraphBuilderMaterializeFilteredEdges(t *testing.T) {
	builder := newIndexedGraphBuilder(3, 4)
	a := builder.ensureNode("a.keep", NodeInternal)
	b := builder.ensureNode("b.skip", NodeExternal)
	c := builder.ensureNode("c.keep", NodeUnknown)
	builder.addEdgeCount(a, b, 5)
	builder.addEdgeCount(a, c, 7)
	builder.addEdgeCount(c, a, 2)

	includeNode := []bool{false, true, false, true}
	newIDByOldID := []int{0, 1, 0, 2}
	inDegree := make([]int, 3)
	outDegree := make([]int, 3)

	got := sortGraph(Graph{
		GroupBy: GroupByPackage,
		Nodes: []Node{
			{ID: 1, Name: "a.keep", Kind: NodeInternal},
			{ID: 2, Name: "c.keep", Kind: NodeUnknown},
		},
		Edges: builder.materializeFilteredEdges(GraphFilter{MinEdgeCount: 2, IncludePrefix: []string{"a.", "c."}}, includeNode, true, newIDByOldID, 3, inDegree, outDegree),
	}).Edges
	want := []Edge{
		{FromID: 1, ToID: 2, Count: 7},
		{FromID: 2, ToID: 1, Count: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("materializeFilteredEdges mismatch: got=%+v want=%+v", got, want)
	}
	if !reflect.DeepEqual(inDegree, []int{0, 1, 1}) {
		t.Fatalf("inDegree mismatch: got=%v want=%v", inDegree, []int{0, 1, 1})
	}
	if !reflect.DeepEqual(outDegree, []int{0, 1, 1}) {
		t.Fatalf("outDegree mismatch: got=%v want=%v", outDegree, []int{0, 1, 1})
	}
}

func TestAppendFilteredMaterializedEdge(t *testing.T) {
	includeNode := []bool{false, true, false, true}
	newIDByOldID := []int{0, 1, 0, 2}
	inDegree := make([]int, 3)
	outDegree := make([]int, 3)
	filter := GraphFilter{MinEdgeCount: 2, IncludePrefix: []string{"a.", "c."}}

	edges := appendFilteredMaterializedEdge(nil, packIndexedEdgeKey(0, 2), 7, filter, includeNode, true, newIDByOldID, inDegree, outDegree)
	edges = appendFilteredMaterializedEdge(edges, packIndexedEdgeKey(2, 0), 2, filter, includeNode, true, newIDByOldID, inDegree, outDegree)
	edges = appendFilteredMaterializedEdge(edges, packIndexedEdgeKey(0, 1), 5, filter, includeNode, true, newIDByOldID, inDegree, outDegree)
	edges = appendFilteredMaterializedEdge(edges, packIndexedEdgeKey(0, 2), 1, filter, includeNode, true, newIDByOldID, inDegree, outDegree)

	wantEdges := []Edge{
		{FromID: 1, ToID: 2, Count: 7},
		{FromID: 2, ToID: 1, Count: 2},
	}
	if !reflect.DeepEqual(edges, wantEdges) {
		t.Fatalf("appendFilteredMaterializedEdge mismatch: got=%+v want=%+v", edges, wantEdges)
	}
	if !reflect.DeepEqual(inDegree, []int{0, 1, 1}) {
		t.Fatalf("inDegree mismatch: got=%v want=%v", inDegree, []int{0, 1, 1})
	}
	if !reflect.DeepEqual(outDegree, []int{0, 1, 1}) {
		t.Fatalf("outDegree mismatch: got=%v want=%v", outDegree, []int{0, 1, 1})
	}
}

func TestIndexedGraphBuilderIncrementEdgeCountPackedMatchesBackends(t *testing.T) {
	keyAB := packIndexedEdgeKey(0, 1)
	keyBA := packIndexedEdgeKey(1, 0)

	buildCounts := func(backend indexedGraphBuilderBackend) map[uint64]uint32 {
		builder := newIndexedGraphBuilderWithBackend(2, defaultPackageGraphBuilderThreshold(), backend)
		builder.incrementEdgeCountPacked(keyAB, 2)
		builder.incrementEdgeCountPacked(keyAB, 3)
		builder.incrementEdgeCountPacked(keyBA, 5)

		got := map[uint64]uint32{}
		builder.rangeEdges(func(key uint64, count uint32) {
			got[key] = count
		})
		return got
	}

	gotMap := buildCounts(indexedGraphBuilderBackendMap)
	gotCounter := buildCounts(indexedGraphBuilderBackendCounter)
	want := map[uint64]uint32{
		keyAB: 5,
		keyBA: 5,
	}
	if !reflect.DeepEqual(gotMap, want) {
		t.Fatalf("map backend counts mismatch: got=%v want=%v", gotMap, want)
	}
	if !reflect.DeepEqual(gotCounter, want) {
		t.Fatalf("counter backend counts mismatch: got=%v want=%v", gotCounter, want)
	}
}

func TestIndexedGraphBuilderAddNodeDegrees(t *testing.T) {
	builder := newIndexedGraphBuilder(2, 2)
	a := builder.ensureNode("a", NodeInternal)
	b := builder.ensureNode("b", NodeExternal)

	builder.addNodeDegrees(a, b, 3)
	builder.addNodeDegrees(b, a, 2)

	if got := builder.inDegree[a]; got != 2 {
		t.Fatalf("builder.inDegree[a] = %d, want 2", got)
	}
	if got := builder.outDegree[a]; got != 3 {
		t.Fatalf("builder.outDegree[a] = %d, want 3", got)
	}
	if got := builder.inDegree[b]; got != 3 {
		t.Fatalf("builder.inDegree[b] = %d, want 3", got)
	}
	if got := builder.outDegree[b]; got != 2 {
		t.Fatalf("builder.outDegree[b] = %d, want 2", got)
	}
}

func TestIndexedEdgeCounterAggregatesAndRangesAll(t *testing.T) {
	counter := newIndexedEdgeCounter(2)
	keyA := packIndexedEdgeKey(0, 0)
	keyB := packIndexedEdgeKey(1, 2)
	keyC := packIndexedEdgeKey(3, 4)

	counter.add(keyA, 2)
	counter.add(keyA, 3)
	counter.add(keyB, 5)
	counter.add(keyC, 7)
	counter.add(keyB, 11)

	if got := counter.len(); got != 3 {
		t.Fatalf("counter.len() = %d, want 3", got)
	}

	got := map[uint64]uint32{}
	counter.rangeAll(func(key uint64, count uint32) {
		got[key] = count
	})

	want := map[uint64]uint32{
		keyA: 5,
		keyB: 16,
		keyC: 7,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("counter contents mismatch: got=%v want=%v", got, want)
	}
}

func TestIndexedEdgeCounterGrowsAndPreservesCounts(t *testing.T) {
	counter := newIndexedEdgeCounter(1)
	want := map[uint64]uint32{}

	for i := 0; i < 64; i++ {
		key := packIndexedEdgeKey(i, i+1)
		count := uint32(i + 1)
		counter.add(key, count)
		want[key] = count
	}

	if got := counter.len(); got != len(want) {
		t.Fatalf("counter.len() = %d, want %d", got, len(want))
	}

	got := map[uint64]uint32{}
	counter.rangeAll(func(key uint64, count uint32) {
		got[key] = count
	})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("counter after grow mismatch: got=%v want=%v", got, want)
	}
}

func TestIndexedEdgeCounterFindIndexAndInsertAt(t *testing.T) {
	counter := newIndexedEdgeCounterWithRecentSlots(4, 0)
	keyA := packIndexedEdgeKey(1, 2)
	keyB := packIndexedEdgeKey(3, 4)

	idxA, found := counter.findIndex(keyA)
	if found {
		t.Fatalf("counter.findIndex(keyA) found = true, want false")
	}
	counter.insertAt(idxA, keyA, 5)

	idxA2, found := counter.findIndex(keyA)
	if !found {
		t.Fatalf("counter.findIndex(keyA) found = false, want true")
	}
	if idxA2 != idxA {
		t.Fatalf("counter.findIndex(keyA) idx = %d, want %d", idxA2, idxA)
	}

	idxB, found := counter.findIndex(keyB)
	if found {
		t.Fatalf("counter.findIndex(keyB) found = true, want false")
	}
	counter.insertAt(idxB, keyB, 7)

	got := map[uint64]uint32{}
	counter.rangeAll(func(key uint64, count uint32) {
		got[key] = count
	})
	want := map[uint64]uint32{
		keyA: 5,
		keyB: 7,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("counter contents mismatch: got=%v want=%v", got, want)
	}
}

func TestIndexedEdgeCounterRecentIndexHit(t *testing.T) {
	counter := newIndexedEdgeCounterWithRecentSlots(4, 8)
	key := packIndexedEdgeKey(5, 6)

	idx, found := counter.findIndex(key)
	if found {
		t.Fatalf("counter.findIndex(key) found = true, want false")
	}
	counter.insertAt(idx, key, 3)
	counter.setRecent(key, idx)

	recentIdx, ok := counter.recentIndex(key)
	if !ok {
		t.Fatalf("counter.recentIndex(key) ok = false, want true")
	}
	if recentIdx != idx {
		t.Fatalf("counter.recentIndex(key) idx = %d, want %d", recentIdx, idx)
	}
}

func TestNextIndexedEdgeCounterRecentSize(t *testing.T) {
	tests := []struct {
		in   int
		want int
	}{
		{in: 0, want: 0},
		{in: 1, want: 1},
		{in: 2, want: 2},
		{in: 3, want: 4},
		{in: 16, want: 16},
		{in: 17, want: 32},
	}

	for _, tc := range tests {
		if got := nextIndexedEdgeCounterRecentSize(tc.in); got != tc.want {
			t.Fatalf("nextIndexedEdgeCounterRecentSize(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestNewIndexedEdgeCounterWithRecentSlots(t *testing.T) {
	noRecent := newIndexedEdgeCounterWithRecentSlots(8, 0)
	if len(noRecent.recent) != 0 {
		t.Fatalf("len(noRecent.recent) = %d, want 0", len(noRecent.recent))
	}
	if noRecent.recentMask != 0 {
		t.Fatalf("noRecent.recentMask = %d, want 0", noRecent.recentMask)
	}

	rounded := newIndexedEdgeCounterWithRecentSlots(8, 17)
	if len(rounded.recent) != 32 {
		t.Fatalf("len(rounded.recent) = %d, want 32", len(rounded.recent))
	}
	if rounded.recentMask != 31 {
		t.Fatalf("rounded.recentMask = %d, want 31", rounded.recentMask)
	}

	key := packIndexedEdgeKey(7, 9)
	rounded.add(key, 2)
	rounded.add(key, 3)
	if got := rounded.len(); got != 1 {
		t.Fatalf("rounded.len() = %d, want 1", got)
	}
	got := map[uint64]uint32{}
	rounded.rangeAll(func(key uint64, count uint32) {
		got[key] = count
	})
	if got[key] != 5 {
		t.Fatalf("got[%d] = %d, want 5", key, got[key])
	}
}

func TestNewIndexedGraphBuilderUsesMapForSmallCapacityAndCounterForLargeCapacity(t *testing.T) {
	threshold := defaultIndexedGraphBuilderThreshold()
	small := newIndexedGraphBuilder(4, threshold-1)
	if !small.usesMapEdges() {
		t.Fatalf("small builder backend = %q, want %q", small.backend, indexedGraphBuilderBackendMap)
	}
	if small.edgeMap == nil {
		t.Fatalf("small builder edgeMap = nil, want initialized map")
	}

	large := newIndexedGraphBuilder(4, threshold)
	if !large.usesIndexedEdgeCounter() {
		t.Fatalf("large builder backend = %q, want %q", large.backend, indexedGraphBuilderBackendCounter)
	}
	if len(large.edgeCounts.keys) == 0 {
		t.Fatalf("large builder edge counter storage = empty, want allocated slots")
	}
}

func TestNewIndexedGraphBuilderWithBackendAndRecentSlots(t *testing.T) {
	builder := newIndexedGraphBuilderWithBackendAndRecentSlots(4, defaultPackageGraphBuilderThreshold(), indexedGraphBuilderBackendCounter, 0)
	if !builder.usesIndexedEdgeCounter() {
		t.Fatalf("builder backend = %q, want %q", builder.backend, indexedGraphBuilderBackendCounter)
	}
	if len(builder.edgeCounts.recent) != 0 {
		t.Fatalf("len(builder.edgeCounts.recent) = %d, want 0", len(builder.edgeCounts.recent))
	}

	builder = newIndexedGraphBuilderWithBackendAndRecentSlots(4, defaultPackageGraphBuilderThreshold(), indexedGraphBuilderBackendCounter, 7)
	if len(builder.edgeCounts.recent) != 8 {
		t.Fatalf("len(builder.edgeCounts.recent) = %d, want 8", len(builder.edgeCounts.recent))
	}
}

func TestDefaultIndexedGraphBuilderBackend(t *testing.T) {
	if got := defaultIndexedGraphBuilderBackend(defaultIndexedGraphBuilderThreshold() - 1); got != indexedGraphBuilderBackendMap {
		t.Fatalf("defaultIndexedGraphBuilderBackend(threshold-1) = %q, want %q", got, indexedGraphBuilderBackendMap)
	}
	if got := defaultIndexedGraphBuilderBackend(defaultIndexedGraphBuilderThreshold()); got != indexedGraphBuilderBackendCounter {
		t.Fatalf("defaultIndexedGraphBuilderBackend(threshold) = %q, want %q", got, indexedGraphBuilderBackendCounter)
	}
}

func TestDefaultGraphBuilderThresholds(t *testing.T) {
	if got := defaultIndexedGraphBuilderThreshold(); got != packageIndexedEdgeCounterMinCapacity {
		t.Fatalf("defaultIndexedGraphBuilderThreshold() = %d, want %d", got, packageIndexedEdgeCounterMinCapacity)
	}
	if got := defaultPackageGraphBuilderThreshold(); got != packageIndexedEdgeCounterMinCapacity {
		t.Fatalf("defaultPackageGraphBuilderThreshold() = %d, want %d", got, packageIndexedEdgeCounterMinCapacity)
	}
	if got := defaultDirectoryGraphBuilderThreshold(); got != directoryIndexedEdgeCounterMinCapacity {
		t.Fatalf("defaultDirectoryGraphBuilderThreshold() = %d, want %d", got, directoryIndexedEdgeCounterMinCapacity)
	}
}

func TestDefaultIndexedEdgeCounterRecentSlots(t *testing.T) {
	if got := defaultIndexedEdgeCounterRecentSlots(); got != indexedEdgeCounterRecentSlots {
		t.Fatalf("defaultIndexedEdgeCounterRecentSlots() = %d, want %d", got, indexedEdgeCounterRecentSlots)
	}
}

func TestIndexedEdgeCounterRecentSlotSweepValues(t *testing.T) {
	got := indexedEdgeCounterRecentSlotSweepValues()
	want := []int{0, 4, 8, defaultIndexedEdgeCounterRecentSlots(), 32, 64}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("indexedEdgeCounterRecentSlotSweepValues() = %v, want %v", got, want)
	}
}

func TestIndexedGraphBuilderBackendForThreshold(t *testing.T) {
	if got := indexedGraphBuilderBackendForThreshold(4095, 4096); got != indexedGraphBuilderBackendMap {
		t.Fatalf("indexedGraphBuilderBackendForThreshold(4095, 4096) = %q, want %q", got, indexedGraphBuilderBackendMap)
	}
	if got := indexedGraphBuilderBackendForThreshold(4096, 4096); got != indexedGraphBuilderBackendCounter {
		t.Fatalf("indexedGraphBuilderBackendForThreshold(4096, 4096) = %q, want %q", got, indexedGraphBuilderBackendCounter)
	}
}

func TestIndexedGraphBuilderBackendHelpers(t *testing.T) {
	if !indexedGraphBuilderBackendMap.usesMapEdges() {
		t.Fatalf("indexedGraphBuilderBackendMap.usesMapEdges() = false, want true")
	}
	if indexedGraphBuilderBackendMap.usesIndexedEdgeCounter() {
		t.Fatalf("indexedGraphBuilderBackendMap.usesIndexedEdgeCounter() = true, want false")
	}
	if indexedGraphBuilderBackendCounter.usesMapEdges() {
		t.Fatalf("indexedGraphBuilderBackendCounter.usesMapEdges() = true, want false")
	}
	if !indexedGraphBuilderBackendCounter.usesIndexedEdgeCounter() {
		t.Fatalf("indexedGraphBuilderBackendCounter.usesIndexedEdgeCounter() = false, want true")
	}
}

func TestEstimatePackageBuilderCapacities(t *testing.T) {
	nodeCapacity, edgeCapacity := estimatePackageBuilderCapacities(8, 2)
	if nodeCapacity != 16 {
		t.Fatalf("nodeCapacity = %d, want 16", nodeCapacity)
	}
	if edgeCapacity != 16 {
		t.Fatalf("edgeCapacity = %d, want 16", edgeCapacity)
	}

	nodeCapacity, edgeCapacity = estimatePackageBuilderCapacities(64, 24)
	if nodeCapacity != 24 {
		t.Fatalf("nodeCapacity = %d, want 24", nodeCapacity)
	}
	if edgeCapacity != 64 {
		t.Fatalf("edgeCapacity = %d, want 64", edgeCapacity)
	}
}

func TestNewPackageGraphBuilderConfig(t *testing.T) {
	config := newPackageGraphBuilderConfig(8, 2, 32)
	if config.nodeCapacity != 16 {
		t.Fatalf("config.nodeCapacity = %d, want 16", config.nodeCapacity)
	}
	if config.edgeCapacity != 16 {
		t.Fatalf("config.edgeCapacity = %d, want 16", config.edgeCapacity)
	}
	if !config.usesMapEdges() {
		t.Fatalf("config backend = %q, want %q", config.backend, indexedGraphBuilderBackendMap)
	}

	config = newPackageGraphBuilderConfig(64, 24, 32)
	if config.nodeCapacity != 24 {
		t.Fatalf("config.nodeCapacity = %d, want 24", config.nodeCapacity)
	}
	if config.edgeCapacity != 64 {
		t.Fatalf("config.edgeCapacity = %d, want 64", config.edgeCapacity)
	}
	if !config.usesIndexedEdgeCounter() {
		t.Fatalf("config backend = %q, want %q", config.backend, indexedGraphBuilderBackendCounter)
	}
}

func TestNewDefaultPackageGraphBuilderConfig(t *testing.T) {
	got := newDefaultPackageGraphBuilderConfig(64, 24)
	want := newPackageGraphBuilderConfig(64, 24, defaultPackageGraphBuilderThreshold())
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default package config mismatch: got=%+v want=%+v", got, want)
	}
	if got := defaultPackageGraphBuilderBackend(64, 24); got != want.backend {
		t.Fatalf("defaultPackageGraphBuilderBackend(64, 24) = %q, want %q", got, want.backend)
	}
}

func TestNewIndexedGraphBuilderConfig(t *testing.T) {
	config := newIndexedGraphBuilderConfig(16, 8, 32)
	if config.nodeCapacity != 16 {
		t.Fatalf("config.nodeCapacity = %d, want 16", config.nodeCapacity)
	}
	if config.edgeCapacity != 16 {
		t.Fatalf("config.edgeCapacity = %d, want 16", config.edgeCapacity)
	}
	if !config.usesMapEdges() {
		t.Fatalf("config backend = %q, want %q", config.backend, indexedGraphBuilderBackendMap)
	}

	config = newIndexedGraphBuilderConfig(16, 64, 32)
	if config.edgeCapacity != 64 {
		t.Fatalf("config.edgeCapacity = %d, want 64", config.edgeCapacity)
	}
	if !config.usesIndexedEdgeCounter() {
		t.Fatalf("config backend = %q, want %q", config.backend, indexedGraphBuilderBackendCounter)
	}
}

func TestNewDefaultIndexedGraphBuilderConfig(t *testing.T) {
	got := newDefaultIndexedGraphBuilderConfig(16, 64)
	want := newIndexedGraphBuilderConfig(16, 64, defaultIndexedGraphBuilderThreshold())
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default indexed config mismatch: got=%+v want=%+v", got, want)
	}
}

func TestNewIndexedGraphBuilderFromConfig(t *testing.T) {
	builder := newIndexedGraphBuilderFromConfig(indexedGraphBuilderConfig{
		nodeCapacity: 16,
		edgeCapacity: 8,
		backend:      indexedGraphBuilderBackendMap,
	})
	if !builder.usesMapEdges() {
		t.Fatalf("builder backend = %q, want %q", builder.backend, indexedGraphBuilderBackendMap)
	}
	if builder.edgeMap == nil {
		t.Fatalf("builder.edgeMap = nil, want initialized map")
	}

	builder = newIndexedGraphBuilderFromConfig(indexedGraphBuilderConfig{
		nodeCapacity: 16,
		edgeCapacity: 64,
		backend:      indexedGraphBuilderBackendCounter,
	})
	if !builder.usesIndexedEdgeCounter() {
		t.Fatalf("builder backend = %q, want %q", builder.backend, indexedGraphBuilderBackendCounter)
	}
	if len(builder.edgeCounts.keys) == 0 {
		t.Fatalf("len(builder.edgeCounts.keys) = 0, want allocated counter storage")
	}
}

func TestGraphBuilderBackends(t *testing.T) {
	if got := packageGraphBuilderBackend(8, 2, 32); got != indexedGraphBuilderBackendMap {
		t.Fatalf("packageGraphBuilderBackend(8, 2, 32) = %q, want %q", got, indexedGraphBuilderBackendMap)
	}
	if got := packageGraphBuilderBackend(64, 24, 32); got != indexedGraphBuilderBackendCounter {
		t.Fatalf("packageGraphBuilderBackend(64, 24, 32) = %q, want %q", got, indexedGraphBuilderBackendCounter)
	}
	if got := directoryGraphBuilderBackend(8, 2, 32); got != indexedGraphBuilderBackendMap {
		t.Fatalf("directoryGraphBuilderBackend(8, 2, 32) = %q, want %q", got, indexedGraphBuilderBackendMap)
	}
	if got := directoryGraphBuilderBackend(128, 24, 32); got != indexedGraphBuilderBackendCounter {
		t.Fatalf("directoryGraphBuilderBackend(128, 24, 32) = %q, want %q", got, indexedGraphBuilderBackendCounter)
	}
}

func TestNewDirectoryGraphBuilderConfig(t *testing.T) {
	config, packageCapacity := newDirectoryGraphBuilderConfig(8, 2, 32)
	if config.nodeCapacity != 16 {
		t.Fatalf("config.nodeCapacity = %d, want 16", config.nodeCapacity)
	}
	if config.edgeCapacity != 16 {
		t.Fatalf("config.edgeCapacity = %d, want 16", config.edgeCapacity)
	}
	if !config.usesMapEdges() {
		t.Fatalf("config backend = %q, want %q", config.backend, indexedGraphBuilderBackendMap)
	}
	if packageCapacity != 16 {
		t.Fatalf("packageCapacity = %d, want 16", packageCapacity)
	}

	config, packageCapacity = newDirectoryGraphBuilderConfig(128, 24, 32)
	if config.nodeCapacity != 24 {
		t.Fatalf("config.nodeCapacity = %d, want 24", config.nodeCapacity)
	}
	if config.edgeCapacity != 64 {
		t.Fatalf("config.edgeCapacity = %d, want 64", config.edgeCapacity)
	}
	if !config.usesIndexedEdgeCounter() {
		t.Fatalf("config backend = %q, want %q", config.backend, indexedGraphBuilderBackendCounter)
	}
	if packageCapacity != 24 {
		t.Fatalf("packageCapacity = %d, want 24", packageCapacity)
	}
}

func TestNewDefaultDirectoryGraphBuilderConfig(t *testing.T) {
	gotConfig, gotPackageCapacity := newDefaultDirectoryGraphBuilderConfig(128, 24)
	wantConfig, wantPackageCapacity := newDirectoryGraphBuilderConfig(128, 24, defaultDirectoryGraphBuilderThreshold())
	if !reflect.DeepEqual(gotConfig, wantConfig) {
		t.Fatalf("default directory config mismatch: got=%+v want=%+v", gotConfig, wantConfig)
	}
	if gotPackageCapacity != wantPackageCapacity {
		t.Fatalf("default directory packageCapacity = %d, want %d", gotPackageCapacity, wantPackageCapacity)
	}
	if got := defaultDirectoryGraphBuilderBackend(128, 24); got != wantConfig.backend {
		t.Fatalf("defaultDirectoryGraphBuilderBackend(128, 24) = %q, want %q", got, wantConfig.backend)
	}
}

func TestNewPackageGraphBuilderWithThresholdUsesEstimatedEdgeCapacity(t *testing.T) {
	ctx := graphBuildContext{
		internalPackages: map[string]struct{}{
			"com.a": {},
			"com.b": {},
		},
	}

	small := newPackageGraphBuilderWithThreshold(ctx, 8, 32)
	if !small.builder.usesMapEdges() {
		t.Fatalf("small.builder backend = %q, want %q", small.builder.backend, indexedGraphBuilderBackendMap)
	}

	large := newPackageGraphBuilderWithThreshold(ctx, 64, 32)
	if !large.builder.usesIndexedEdgeCounter() {
		t.Fatalf("large.builder backend = %q, want %q", large.builder.backend, indexedGraphBuilderBackendCounter)
	}
}

func TestNewDirectoryGraphBuilderWithThresholdUsesEstimatedEdgeCapacity(t *testing.T) {
	ctx := graphBuildContext{
		internalPackages: map[string]struct{}{
			"com.a": {},
			"com.b": {},
		},
	}

	small := newDirectoryGraphBuilderWithThreshold(ctx, 8, 32)
	if !small.builder.usesMapEdges() {
		t.Fatalf("small.builder backend = %q, want %q", small.builder.backend, indexedGraphBuilderBackendMap)
	}

	large := newDirectoryGraphBuilderWithThreshold(ctx, 128, 32)
	if !large.builder.usesIndexedEdgeCounter() {
		t.Fatalf("large.builder backend = %q, want %q", large.builder.backend, indexedGraphBuilderBackendCounter)
	}
}

func TestDefaultBuilderThresholdsMatchExplicitThresholds(t *testing.T) {
	ctx := graphBuildContext{
		internalPackages: map[string]struct{}{
			"com.a": {},
			"com.b": {},
		},
	}

	defaultPkg := newPackageGraphBuilder(ctx, 64)
	explicitPkg := newPackageGraphBuilderWithThreshold(ctx, 64, defaultPackageGraphBuilderThreshold())
	if defaultPkg.builder.backend != explicitPkg.builder.backend {
		t.Fatalf("package default backend = %q, explicit = %q", defaultPkg.builder.backend, explicitPkg.builder.backend)
	}

	defaultDir := newDirectoryGraphBuilder(ctx, 64)
	explicitDir := newDirectoryGraphBuilderWithThreshold(ctx, 64, defaultDirectoryGraphBuilderThreshold())
	if defaultDir.builder.backend != explicitDir.builder.backend {
		t.Fatalf("directory default backend = %q, explicit = %q", defaultDir.builder.backend, explicitDir.builder.backend)
	}
}

func TestIndexedGraphBuilderMaterializeMatchesAcrossMapAndCounter(t *testing.T) {
	build := func(edgeCapacity int) Graph {
		builder := newIndexedGraphBuilder(4, edgeCapacity)
		a := builder.ensureNode("com.a", NodeInternal)
		b := builder.ensureNode("com.b", NodeInternal)
		c := builder.ensureNode("java.util", NodeExternal)

		builder.addEdgeCount(a, b, 2)
		builder.addEdgeCount(a, c, 3)
		builder.addEdgeCount(b, a, 1)
		builder.addEdgeCount(a, a, 4)

		nodes, edges := builder.materialize()
		return sortGraph(Graph{
			GroupBy: GroupByPackage,
			Nodes:   nodes,
			Edges:   edges,
		})
	}

	gotMap := build(defaultPackageGraphBuilderThreshold() - 1)
	gotCounter := build(defaultPackageGraphBuilderThreshold())
	if !reflect.DeepEqual(gotMap, gotCounter) {
		t.Fatalf("materialize mismatch across storage backends: map=%+v counter=%+v", gotMap, gotCounter)
	}
}

func TestIndexedGraphBuilderMaterializeFilteredMatchesAcrossMapAndCounter(t *testing.T) {
	build := func(edgeCapacity int) Graph {
		builder := newIndexedGraphBuilder(6, edgeCapacity)
		a := builder.ensureNode("com.a", NodeInternal)
		b := builder.ensureNode("com.b", NodeInternal)
		c := builder.ensureNode("com.c", NodeInternal)
		j := builder.ensureNode("java.util", NodeExternal)

		builder.addEdgeCount(a, b, 5)
		builder.addEdgeCount(a, c, 1)
		builder.addEdgeCount(b, j, 7)
		builder.addEdgeCount(c, j, 2)
		builder.addEdgeCount(a, a, 3)

		nodes, edges := builder.materializeFiltered(GraphFilter{
			MinEdgeCount:  3,
			IncludePrefix: []string{"com.", "java."},
		})
		return sortGraph(Graph{
			GroupBy: GroupByPackage,
			Nodes:   nodes,
			Edges:   edges,
		})
	}

	gotMap := build(defaultPackageGraphBuilderThreshold() - 1)
	gotCounter := build(defaultPackageGraphBuilderThreshold())
	if !reflect.DeepEqual(gotMap, gotCounter) {
		t.Fatalf("materializeFiltered mismatch across storage backends: map=%+v counter=%+v", gotMap, gotCounter)
	}
}

func TestBuildGraphByDir(t *testing.T) {
	result := RepositoryResult{
		Root: "/repo",
		Files: []FileUnit{
			{
				Path:        "/repo/a/A.java",
				Relative:    "a/A.java",
				Language:    LangJava,
				PackageName: "com.a",
				Imports: []string{
					"com.b.Service",
					"java.util.List",
					"java.util.Collections.emptyList",
				},
			},
			{
				Path:        "/repo/b/B.kt",
				Relative:    "b/B.kt",
				Language:    LangKotlin,
				PackageName: "com.b",
				Imports: []string{
					"com.a.Model",
				},
			},
		},
	}
	external := kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
		},
	}

	graph := BuildGraph(result, external, GroupByDir)
	if graph.GroupBy != GroupByDir {
		t.Fatalf("expected group_by=dir, got %s", graph.GroupBy)
	}
	if len(graph.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(graph.Edges))
	}

	nodeByName := map[string]Node{}
	for _, node := range graph.Nodes {
		nodeByName[node.Name] = node
	}
	if nodeByName["a"].Kind != NodeInternal {
		t.Fatalf("expected a internal")
	}
	if nodeByName["b"].Kind != NodeInternal {
		t.Fatalf("expected b internal")
	}
	if nodeByName["java.util"].Kind != NodeExternal {
		t.Fatalf("expected java.util external")
	}
}

func TestResolveImportPackageStaticClassFallback(t *testing.T) {
	internal := map[string]struct{}{
		"com.google.common.base": {},
	}

	resolved := resolveImportPackage("com.google.common.base.Functions.identity", internal, kcg.ExternalIndex{})
	if resolved != "com.google.common.base" {
		t.Fatalf("expected fallback package com.google.common.base, got %s", resolved)
	}

	resolved = resolveImportPackage("com.example.util.helpers.format", internal, kcg.ExternalIndex{})
	if resolved != "com.example.util.helpers" {
		t.Fatalf("expected regular package inference com.example.util.helpers, got %s", resolved)
	}
}

func TestIsKnownExternalImportNormalized(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "java root", in: "java.util", want: true},
		{name: "gradle prefix", in: "org.gradle.api", want: true},
		{name: "wildcard", in: "java.util.*", want: true},
		{name: "empty wildcard", in: ".*", want: false},
		{name: "internal package", in: "com.example", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isKnownExternalImportNormalized(tc.in); got != tc.want {
				t.Fatalf("isKnownExternalImportNormalized(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestHasExternalImportEvidence(t *testing.T) {
	external := kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util":  {},
			"kotlin.js":  {},
			"org.gradle": {},
		},
		Symbols: map[string]struct{}{
			"kotlin.js.Promise": {},
		},
	}

	tests := []struct {
		name       string
		pkg        string
		importPath string
		want       bool
	}{
		{name: "known external package", pkg: "java.util", importPath: "java.util.List", want: true},
		{name: "known external prefix", pkg: "org.gradle.api", importPath: "org.gradle.api.Project", want: true},
		{name: "symbol only", pkg: "kotlin.js", importPath: "kotlin.js.Promise", want: true},
		{name: "duplicate package and import", pkg: "java.util", importPath: "java.util", want: true},
		{name: "unknown package", pkg: "com.example", importPath: "com.example.Type", want: false},
		{name: "empty import path", pkg: "com.example", importPath: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasExternalImportEvidence(tc.pkg, tc.importPath, external); got != tc.want {
				t.Fatalf("hasExternalImportEvidence(%q, %q) = %v, want %v", tc.pkg, tc.importPath, got, tc.want)
			}
		})
	}
}

func TestMergeNodeKind(t *testing.T) {
	tests := []struct {
		name     string
		current  NodeKind
		incoming NodeKind
		want     NodeKind
	}{
		{name: "same kind", current: NodeExternal, incoming: NodeExternal, want: NodeExternal},
		{name: "empty current adopts incoming", current: "", incoming: NodeUnknown, want: NodeUnknown},
		{name: "internal wins", current: NodeExternal, incoming: NodeInternal, want: NodeInternal},
		{name: "external wins over unknown", current: NodeUnknown, incoming: NodeExternal, want: NodeExternal},
		{name: "unknown remains unknown", current: NodeUnknown, incoming: NodeUnknown, want: NodeUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := mergeNodeKind(tc.current, tc.incoming); got != tc.want {
				t.Fatalf("mergeNodeKind(%q, %q) = %q, want %q", tc.current, tc.incoming, got, tc.want)
			}
		})
	}
}

func TestDetectKindFromSignals(t *testing.T) {
	tests := []struct {
		name                string
		isInternal          bool
		hasExternalEvidence bool
		want                NodeKind
	}{
		{name: "internal wins", isInternal: true, hasExternalEvidence: false, want: NodeInternal},
		{name: "internal beats external", isInternal: true, hasExternalEvidence: true, want: NodeInternal},
		{name: "external", isInternal: false, hasExternalEvidence: true, want: NodeExternal},
		{name: "unknown", isInternal: false, hasExternalEvidence: false, want: NodeUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectKind(tc.isInternal, tc.hasExternalEvidence); got != tc.want {
				t.Fatalf("detectKind(%v, %v) = %q, want %q", tc.isInternal, tc.hasExternalEvidence, got, tc.want)
			}
		})
	}
}

func TestHasIndexedExternalImportEvidence(t *testing.T) {
	external := kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
		},
		Symbols: map[string]struct{}{
			"kotlin.js.Promise": {},
		},
	}

	tests := []struct {
		name       string
		pkg        string
		importPath string
		want       bool
	}{
		{name: "package hit", pkg: "java.util", importPath: "java.util.List", want: true},
		{name: "symbol hit", pkg: "kotlin.js", importPath: "kotlin.js.Promise", want: true},
		{name: "same path no symbol lookup", pkg: "java.util", importPath: "java.util", want: true},
		{name: "same unknown path", pkg: "com.example", importPath: "com.example", want: false},
		{name: "empty import path", pkg: "com.example", importPath: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasIndexedExternalImportEvidence(tc.pkg, tc.importPath, external); got != tc.want {
				t.Fatalf("hasIndexedExternalImportEvidence(%q, %q) = %v, want %v", tc.pkg, tc.importPath, got, tc.want)
			}
		})
	}
}

func TestAddPackageDirIndexDeduplicates(t *testing.T) {
	index := newPackageDirIndex(1)

	index.add("com.example", 3)
	index.add("com.example", 3)
	index.add("com.example", 5)
	index.add("com.example", 5)

	if _, ok := index.single("com.example"); ok {
		t.Fatalf("index.single(com.example) unexpectedly reported single target")
	}
	got := index.multiple("com.example")
	if !reflect.DeepEqual(got, []int{3, 5}) {
		t.Fatalf("unexpected dir targets: got=%v want=[3 5]", got)
	}
	if _, ok := index.single("com.example"); ok {
		t.Fatalf("index.single(com.example) unexpectedly reported single target after promotion")
	}
}

func TestLocalEdgeAccumulatorAggregatesAndFlushes(t *testing.T) {
	acc := newLocalEdgeAccumulator(4)
	builder := newIndexedGraphBuilder(4, 4)

	from := builder.ensureNode("from", NodeInternal)
	a := builder.ensureNode("a", NodeInternal)
	b := builder.ensureNode("b", NodeExternal)

	acc.add(a)
	acc.add(a)
	acc.add(b)
	acc.flush(builder, from)

	nodes, edges := builder.materialize()
	graph := sortGraph(Graph{
		GroupBy: GroupByPackage,
		Nodes:   nodes,
		Edges:   edges,
	})

	if len(graph.Edges) != 2 {
		t.Fatalf("len(graph.Edges) = %d, want 2", len(graph.Edges))
	}
	if graph.Edges[0].Count != 2 {
		t.Fatalf("graph.Edges[0].Count = %d, want 2", graph.Edges[0].Count)
	}
	if got := builder.outDegree[from]; got != 3 {
		t.Fatalf("builder.outDegree[from] = %d, want 3", got)
	}
	if got := builder.inDegree[a]; got != 2 {
		t.Fatalf("builder.inDegree[a] = %d, want 2", got)
	}
	if got := builder.inDegree[b]; got != 1 {
		t.Fatalf("builder.inDegree[b] = %d, want 1", got)
	}

	acc.reset()
	if got := len(acc.entries); got != 0 {
		t.Fatalf("len(acc.entries) = %d, want 0 after reset", got)
	}
}

func TestLocalEdgeAccumulatorAddDirectoryTargetSkipsSelfAndAggregates(t *testing.T) {
	acc := newLocalEdgeAccumulator(4)
	acc.addDirectoryTarget(3, directoryImportTarget{singleDirIndex: 3, hasSingleDir: true})
	acc.addDirectoryTarget(3, directoryImportTarget{singleDirIndex: 7, hasSingleDir: true})
	acc.addDirectoryTarget(3, directoryImportTarget{dirIndices: []int{7, 9, 3, 7}})

	if got := acc.entries; !reflect.DeepEqual(got, []edgeCountEntry{
		{toIndex: 7, count: 3},
		{toIndex: 9, count: 1},
	}) {
		t.Fatalf("acc.entries = %v, want [{7 3} {9 1}]", got)
	}
}

func TestGraphNodeIndexCacheEnsureReusesIndexAndMergesKind(t *testing.T) {
	builder := newIndexedGraphBuilder(2, 2)
	cache := newGraphNodeIndexCache(2)

	first := cache.ensure(builder, "com.example", NodeUnknown)
	second := cache.ensure(builder, "com.example", NodeExternal)

	if first != second {
		t.Fatalf("cache.ensure returned different indices: first=%d second=%d", first, second)
	}
	if len(builder.names) != 1 {
		t.Fatalf("len(builder.names) = %d, want 1", len(builder.names))
	}
	if got := builder.kinds[first]; got != NodeExternal {
		t.Fatalf("builder.kinds[%d] = %q, want %q", first, got, NodeExternal)
	}
}

func TestResolveDirectoryImportTargetReusesFallbackNodeWithoutCachingInternalDirs(t *testing.T) {
	builder := newIndexedGraphBuilder(4, 4)
	nodeCache := newGraphNodeIndexCache(4)
	packageDirs := newPackageDirIndex(2)

	dirIndex := nodeCache.ensure(builder, "src/a", NodeInternal)
	packageDirs.add("com.example.internal", dirIndex)

	cache := make(map[string]directoryImportTarget, 2)

	internalTarget := resolveDirectoryImportTarget(cache, &packageDirs, &nodeCache, builder, "com.example.internal", NodeInternal)
	repeatedInternalTarget := resolveDirectoryImportTarget(cache, &packageDirs, &nodeCache, builder, "com.example.internal", NodeExternal)
	if !internalTarget.hasSingleDir || internalTarget.singleDirIndex != dirIndex {
		t.Fatalf("internalTarget single dir = (%v, %d), want (%v, %d)", internalTarget.hasSingleDir, internalTarget.singleDirIndex, true, dirIndex)
	}
	if repeatedInternalTarget.hasNode {
		t.Fatalf("repeatedInternalTarget.hasNode = true, want false")
	}
	if _, ok := cache["com.example.internal"]; ok {
		t.Fatalf("cache contains internal dir target, want fallback-only cache")
	}

	externalTarget := resolveDirectoryImportTarget(cache, &packageDirs, &nodeCache, builder, "java.util", NodeExternal)
	repeatedExternalTarget := resolveDirectoryImportTarget(cache, &packageDirs, &nodeCache, builder, "java.util", NodeUnknown)
	if !externalTarget.hasNode {
		t.Fatalf("externalTarget.hasNode = false, want true")
	}
	if externalTarget.nodeIndex != repeatedExternalTarget.nodeIndex {
		t.Fatalf("cached external node index mismatch: first=%d second=%d", externalTarget.nodeIndex, repeatedExternalTarget.nodeIndex)
	}
	if len(builder.names) != 2 {
		t.Fatalf("len(builder.names) = %d, want 2", len(builder.names))
	}
	if got := builder.kinds[externalTarget.nodeIndex]; got != NodeExternal {
		t.Fatalf("builder.kinds[%d] = %q, want %q", externalTarget.nodeIndex, got, NodeExternal)
	}
}

func TestResolveInternalDirectoryImportTarget(t *testing.T) {
	packageDirs := newPackageDirIndex(2)
	packageDirs.add("com.example.internal", 3)
	packageDirs.add("com.example.util", 7)
	importCache := newRecentDirectoryImportCache(4)

	direct, ok := resolveInternalDirectoryImportTarget(&importCache, &packageDirs, "com.example.internal.Service")
	if !ok || !direct.hasSingleDir || direct.singleDirIndex != 3 {
		t.Fatalf("direct internal target = (%+v, %v), want single dir 3", direct, ok)
	}

	ancestor, ok := resolveInternalDirectoryImportTarget(&importCache, &packageDirs, "com.example.util.Helpers.format")
	if !ok || !ancestor.hasSingleDir || ancestor.singleDirIndex != 7 {
		t.Fatalf("ancestor internal target = (%+v, %v), want single dir 7", ancestor, ok)
	}

	if _, ok := resolveInternalDirectoryImportTarget(&importCache, &packageDirs, "java.util.List"); ok {
		t.Fatalf("java.util.List unexpectedly resolved as internal")
	}
	if cached, ok := importCache.get("com.example.internal.Service"); !ok || !cached.ok {
		t.Fatalf("importCache.get(com.example.internal.Service) = (%+v, %v), want cached hit", cached, ok)
	}
}

func TestResolveInternalDirectoryImportTargetCachesBaseCandidate(t *testing.T) {
	packageDirs := newPackageDirIndex(1)
	packageDirs.add("com.example.util", 7)
	importCache := newRecentDirectoryImportCache(4)

	first, ok := resolveInternalDirectoryImportTarget(&importCache, &packageDirs, "com.example.util.Helpers.format")
	if !ok || !first.hasSingleDir || first.singleDirIndex != 7 {
		t.Fatalf("first resolve = (%+v, %v), want single dir 7", first, ok)
	}

	if cached, ok := importCache.get("com.example.util.Helpers"); !ok || !cached.ok || !cached.target.hasSingleDir || cached.target.singleDirIndex != 7 {
		t.Fatalf("importCache.get(com.example.util.Helpers) = (%+v, %v), want cached base hit", cached, ok)
	}

	delete(packageDirs.singleByPackage, "com.example.util")

	second, ok := resolveInternalDirectoryImportTarget(&importCache, &packageDirs, "com.example.util.Helpers.parse")
	if !ok || !second.hasSingleDir || second.singleDirIndex != 7 {
		t.Fatalf("second resolve = (%+v, %v), want cached single dir 7", second, ok)
	}
}

func TestResolveInternalDirectoryImportTargetCachesMissingBaseCandidate(t *testing.T) {
	packageDirs := newPackageDirIndex(1)
	importCache := newRecentDirectoryImportCache(4)

	if _, ok := resolveInternalDirectoryImportTarget(&importCache, &packageDirs, "com.example.util.Helpers.format"); ok {
		t.Fatalf("resolveInternalDirectoryImportTarget() unexpectedly resolved missing import")
	}

	if cached, ok := importCache.get("com.example.util.Helpers"); !ok || cached.ok {
		t.Fatalf("importCache.get(com.example.util.Helpers) = (%+v, %v), want cached miss", cached, ok)
	}
	if cached, ok := importCache.get("com.example.util.Helpers.format"); !ok || cached.ok {
		t.Fatalf("importCache.get(com.example.util.Helpers.format) = (%+v, %v), want cached miss", cached, ok)
	}
}

func TestParentImportCandidate(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "java", want: ""},
		{in: "java.util", want: "java"},
		{in: "com.example.util.Helpers", want: "com.example.util"},
	}

	for _, tc := range tests {
		if got := parentImportCandidate(tc.in); got != tc.want {
			t.Fatalf("parentImportCandidate(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDeriveImportBaseCandidate(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "java.util.*", want: "java.util"},
		{in: "com.example.Service", want: "com.example"},
		{in: "com.example.util.Helpers.format", want: "com.example.util.Helpers"},
		{in: "com.example.util.helpers.format", want: "com.example.util.helpers.format"},
		{in: "java", want: "java"},
		{in: "Service", want: ""},
	}

	for _, tc := range tests {
		if got := deriveImportBaseCandidate(tc.in); got != tc.want {
			t.Fatalf("deriveImportBaseCandidate(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDirectoryGraphBuilderResolveImportTarget(t *testing.T) {
	ctx := newGraphBuildContext([]FileUnit{
		{Relative: "a/A.java", PackageName: "com.example.internal"},
		{Relative: "b/B.kt", PackageName: "com.example.util"},
	}, kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
		},
	})
	builder := newDirectoryGraphBuilder(ctx, 2)
	builder.indexFiles([]FileUnit{
		{Relative: "a/A.java", PackageName: "com.example.internal"},
		{Relative: "b/B.kt", PackageName: "com.example.util"},
	})

	internal, ok := builder.resolveImportTarget("com.example.internal.Service")
	if !ok || !internal.hasSingleDir {
		t.Fatalf("internal resolveImportTarget = (%+v, %v), want single dir target", internal, ok)
	}

	external, ok := builder.resolveImportTarget("java.util.List")
	if !ok || !external.hasNode {
		t.Fatalf("external resolveImportTarget = (%+v, %v), want fallback node", external, ok)
	}
	if got := builder.builder.kinds[external.nodeIndex]; got != NodeExternal {
		t.Fatalf("builder.builder.kinds[%d] = %q, want %q", external.nodeIndex, got, NodeExternal)
	}

	if _, ok := builder.resolveImportTarget(" "); ok {
		t.Fatalf("blank import unexpectedly resolved")
	}
}

func TestNewGraphBuildContextCollectsNormalizedInternalPackages(t *testing.T) {
	ctx := newGraphBuildContext([]FileUnit{
		{PackageName: " com.example.alpha "},
		{PackageName: ".com.example.beta."},
		{PackageName: ""},
		{PackageName: " com.example.alpha "},
	}, kcg.ExternalIndex{})

	if !reflect.DeepEqual(ctx.filePackages, []string{
		"com.example.alpha",
		"com.example.beta",
		"",
		"com.example.alpha",
	}) {
		t.Fatalf("unexpected file packages: got=%v", ctx.filePackages)
	}

	if len(ctx.internalPackages) != 2 {
		t.Fatalf("len(ctx.internalPackages) = %d, want 2", len(ctx.internalPackages))
	}
	if _, ok := ctx.internalPackages["com.example.alpha"]; !ok {
		t.Fatalf("expected com.example.alpha in internal packages")
	}
	if _, ok := ctx.internalPackages["com.example.beta"]; !ok {
		t.Fatalf("expected com.example.beta in internal packages")
	}
}

func TestGraphBuildContextResolveImport(t *testing.T) {
	ctx := newGraphBuildContext([]FileUnit{
		{PackageName: "com.example.internal"},
	}, kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
		},
		Symbols: map[string]struct{}{
			"java.util.List": {},
		},
	})

	internal := ctx.resolveImport(" com.example.internal.Service ")
	if internal.Package != "com.example.internal" || internal.Kind != NodeInternal {
		t.Fatalf("internal resolve = %+v, want package=com.example.internal kind=%s", internal, NodeInternal)
	}

	external := ctx.resolveImport("java.util.List")
	if external.Package != "java.util" || external.Kind != NodeExternal {
		t.Fatalf("external resolve = %+v, want package=java.util kind=%s", external, NodeExternal)
	}

	unknown := ctx.resolveImport("com.unknown.Type")
	if unknown.Package != "com.unknown" || unknown.Kind != NodeUnknown {
		t.Fatalf("unknown resolve = %+v, want package=com.unknown kind=%s", unknown, NodeUnknown)
	}
}

func TestImportClassifierResolveNormalized(t *testing.T) {
	classifier := newImportClassifier(map[string]struct{}{
		"com.example.internal": {},
	}, kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
		},
		Symbols: map[string]struct{}{
			"kotlin.js.Promise": {},
		},
	})

	internal := classifier.resolveNormalized("com.example.internal.Service")
	if internal.Package != "com.example.internal" || internal.Kind != NodeInternal {
		t.Fatalf("internal resolveNormalized = %+v, want package=com.example.internal kind=%s", internal, NodeInternal)
	}

	external := classifier.resolveNormalized("java.util.List")
	if external.Package != "java.util" || external.Kind != NodeExternal {
		t.Fatalf("external resolveNormalized = %+v, want package=java.util kind=%s", external, NodeExternal)
	}

	fallback := classifier.resolveNormalized("com.example.UnknownType.method")
	if fallback.Package != "com.example" || fallback.Kind != NodeUnknown {
		t.Fatalf("fallback resolveNormalized = %+v, want package=com.example kind=%s", fallback, NodeUnknown)
	}
}

func TestImportClassifierCachesResolvedImports(t *testing.T) {
	classifier := newImportClassifier(map[string]struct{}{
		"com.example.internal": {},
	}, kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
		},
	})

	first := classifier.resolve(" java.util.List ")
	second := classifier.resolve("java.util.List")
	internalFirst := classifier.resolve(" com.example.internal.Service ")
	internalSecond := classifier.resolve("com.example.internal.Service")

	if first != second {
		t.Fatalf("cached resolve mismatch: first=%+v second=%+v", first, second)
	}
	if internalFirst != internalSecond {
		t.Fatalf("cached internal resolve mismatch: first=%+v second=%+v", internalFirst, internalSecond)
	}
	if len(classifier.resolved) != 2 {
		t.Fatalf("len(classifier.resolved) = %d, want 2", len(classifier.resolved))
	}
	if got := classifier.resolved["java.util.List"]; got != first {
		t.Fatalf("classifier.resolved[java.util.List] = %+v, want %+v", got, first)
	}
	if got := classifier.resolved["com.example.internal.Service"]; got != internalFirst {
		t.Fatalf("classifier.resolved[com.example.internal.Service] = %+v, want %+v", got, internalFirst)
	}
}

func TestImportClassifierCachesRawImports(t *testing.T) {
	classifier := newImportClassifier(map[string]struct{}{
		"com.example.internal": {},
	}, kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
		},
	})

	first := classifier.resolve("java.util.List")
	second := classifier.resolve("java.util.List")

	if first != second {
		t.Fatalf("raw cached resolve mismatch: first=%+v second=%+v", first, second)
	}
	if len(classifier.recentRaw) == 0 {
		t.Fatalf("len(classifier.recentRaw) = 0, want allocated cache")
	}
	slot := recentRawImportIndex("java.util.List", classifier.recentRawMask)
	if got := classifier.recentRaw[slot].key; got != "java.util.List" {
		t.Fatalf("classifier.recentRaw[%d].key = %q, want %q", slot, got, "java.util.List")
	}
	if got := classifier.recentRaw[slot].target; got != first {
		t.Fatalf("classifier.recentRaw[%d].target = %+v, want %+v", slot, got, first)
	}
}

func TestResolveImportPackageNormalizedMatchesDirectResolution(t *testing.T) {
	internal := map[string]struct{}{
		"com.google.common.base":   {},
		"com.example.util.helpers": {},
	}
	external := kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
		},
	}

	inputs := []string{
		"com.google.common.base.Functions.identity",
		"com.example.util.helpers.format",
		"java.util.List",
		"missing.pkg.Type",
		"",
	}

	for _, input := range inputs {
		got := resolveImportPackageNormalized(normalizePath(input), internal, external)
		want := resolveImportPackage(input, internal, external)
		if got != want {
			t.Fatalf("resolveImportPackageNormalized(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveImportTargetNormalizedReportsKind(t *testing.T) {
	internal := map[string]struct{}{
		"com.example.internal": {},
	}
	external := kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
			"kotlin.js": {},
		},
		Symbols: map[string]struct{}{
			"kotlin.js.Promise": {},
		},
	}

	tests := []struct {
		name       string
		importPath string
		wantPkg    string
		wantKind   NodeKind
	}{
		{name: "internal", importPath: "com.example.internal.Type", wantPkg: "com.example.internal", wantKind: NodeInternal},
		{name: "external package", importPath: "java.util.List", wantPkg: "java.util", wantKind: NodeExternal},
		{name: "external symbol", importPath: "kotlin.js.Promise", wantPkg: "kotlin.js", wantKind: NodeExternal},
		{name: "unknown", importPath: "com.example.unknown.Type", wantPkg: "com.example.unknown", wantKind: NodeUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveImportTargetNormalized(normalizePath(tc.importPath), internal, external)
			if got.Package != tc.wantPkg {
				t.Fatalf("resolveImportTargetNormalized(%q).Package = %q, want %q", tc.importPath, got.Package, tc.wantPkg)
			}
			if got.Kind != tc.wantKind {
				t.Fatalf("resolveImportTargetNormalized(%q).Kind = %q, want %q", tc.importPath, got.Kind, tc.wantKind)
			}
		})
	}
}

func TestResolveKnownImportCandidate(t *testing.T) {
	internal := map[string]struct{}{
		"com.example.internal": {},
	}
	external := kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
		},
	}

	got, ok := resolveKnownImportCandidate("com.example.internal", internal, external)
	if !ok || got.Package != "com.example.internal" || got.Kind != NodeInternal {
		t.Fatalf("internal candidate = %+v, %v", got, ok)
	}

	got, ok = resolveKnownImportCandidate("java.util", internal, external)
	if !ok || got.Package != "java.util" || got.Kind != NodeExternal {
		t.Fatalf("external candidate = %+v, %v", got, ok)
	}

	if got, ok = resolveKnownImportCandidate("com.unknown", internal, external); ok || got.Package != "" {
		t.Fatalf("unknown candidate = %+v, %v", got, ok)
	}
}

func TestResolveAncestorImportCandidate(t *testing.T) {
	internal := map[string]struct{}{
		"com.example.internal": {},
	}
	external := kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
		},
	}

	got, ok := resolveAncestorImportCandidate("com.example.internal.Service.Method", internal, external)
	if !ok || got.Package != "com.example.internal" || got.Kind != NodeInternal {
		t.Fatalf("internal ancestor = %+v, %v", got, ok)
	}

	got, ok = resolveAncestorImportCandidate("java.util.concurrent.Future", internal, external)
	if !ok || got.Package != "java.util" || got.Kind != NodeExternal {
		t.Fatalf("external ancestor = %+v, %v", got, ok)
	}

	if got, ok = resolveAncestorImportCandidate("com.unknown.Type", internal, external); ok || got.Package != "" {
		t.Fatalf("unknown ancestor = %+v, %v", got, ok)
	}
}

func TestResolveFallbackImportTarget(t *testing.T) {
	internal := map[string]struct{}{
		"com.google.common.base": {},
	}

	got, ok := resolveFallbackImportTarget("com.google.common.base.Functions", "com.google.common.base.Functions.identity", internal, kcg.ExternalIndex{})
	if !ok || got.Package != "com.google.common.base" || got.Kind != NodeInternal {
		t.Fatalf("fallback internal = %+v, %v", got, ok)
	}

	got, ok = resolveFallbackImportTarget("com.example.UnknownClass", "com.example.UnknownClass.method", nil, kcg.ExternalIndex{})
	if !ok || got.Package != "com.example" || got.Kind != NodeUnknown {
		t.Fatalf("fallback unknown = %+v, %v", got, ok)
	}
}

func TestSortGraphUsesDeterministicPolicy(t *testing.T) {
	graph := Graph{
		Nodes: []Node{
			{ID: 3, Name: "z", Kind: NodeUnknown},
			{ID: 1, Name: "a", Kind: NodeInternal},
			{ID: 2, Name: "m", Kind: NodeExternal},
		},
		Edges: []Edge{
			{FromID: 3, ToID: 1, Count: 1},
			{FromID: 2, ToID: 3, Count: 4},
			{FromID: 2, ToID: 3, Count: 2},
		},
	}

	got := sortGraph(graph)
	want := Graph{
		Nodes: []Node{
			{ID: 1, Name: "a", Kind: NodeInternal},
			{ID: 2, Name: "m", Kind: NodeExternal},
			{ID: 3, Name: "z", Kind: NodeUnknown},
		},
		Edges: []Edge{
			{FromID: 2, ToID: 3, Count: 4},
			{FromID: 2, ToID: 3, Count: 2},
			{FromID: 3, ToID: 1, Count: 1},
		},
	}

	if !reflect.DeepEqual(got.Nodes, want.Nodes) {
		t.Fatalf("sorted nodes mismatch: got=%+v want=%+v", got.Nodes, want.Nodes)
	}
	if !reflect.DeepEqual(got.Edges, want.Edges) {
		t.Fatalf("sorted edges mismatch: got=%+v want=%+v", got.Edges, want.Edges)
	}
}
