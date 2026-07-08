package mixedgraph

import (
	"sort"
	"strings"
	"sync"

	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

type GraphFilter struct {
	MinEdgeCount  int
	IncludePrefix []string
	ExcludePrefix []string
}

var filterScratchPool = sync.Pool{
	New: func() any {
		return &filterScratch{}
	},
}

type filterScratch struct {
	includeNode  []bool
	usedNodeIDs  []bool
	newIDByOldID []int
	inDegree     []int
	outDegree    []int
}

func (g Graph) Filter(filter GraphFilter) Graph {
	return FilterGraph(g, filter)
}

func FilterGraph(graph Graph, filter GraphFilter) Graph {
	return filterGraph(graph, filter)
}

func BuildFilteredGraph(result RepositoryResult, externalIndex kcg.ExternalIndex, groupBy GroupBy, filter GraphFilter) Graph {
	if !groupBy.IsValid() {
		groupBy = GroupByPackage
	}
	ctx := newGraphBuildContext(result.Files, externalIndex)
	var nodes []Node
	var edges []Edge
	switch groupBy {
	case GroupByDir:
		nodes, edges = buildDirectoryGraphBuilder(result.Files, ctx).materializeFiltered(filter)
	default:
		nodes, edges = buildPackageGraphBuilder(result.Files, ctx).materializeFiltered(filter)
	}
	return sortGraph(Graph{
		Root:    result.Root,
		GroupBy: groupBy,
		Nodes:   nodes,
		Edges:   edges,
	})
}

func filterGraph(graph Graph, filter GraphFilter) Graph {
	filter = normalizeFilter(filter)
	if !filter.isActive() {
		return graph
	}

	maxNodeID := maxGraphNodeID(graph.Nodes)
	scratch := getFilterScratch(maxNodeID + 1)
	defer putFilterScratch(scratch)

	hasPrefixFilter := len(filter.IncludePrefix) > 0 || len(filter.ExcludePrefix) > 0
	includeNode := scratch.includeNode[:0]
	if hasPrefixFilter {
		includeNode = scratch.includeNode
		for _, node := range graph.Nodes {
			if shouldIncludeNode(node.Name, filter) {
				includeNode[node.ID] = true
			}
		}
	}

	usedNodeIDs := scratch.usedNodeIDs
	filteredEdgeCount := markUsedFilteredEdges(graph.Edges, includeNode, filter.MinEdgeCount, hasPrefixFilter, usedNodeIDs)

	nodes := make([]Node, 0, len(graph.Nodes))
	newIDByOldID := scratch.newIDByOldID
	for _, node := range graph.Nodes {
		if !shouldKeepFilteredNode(node.ID, includeNode, usedNodeIDs, hasPrefixFilter) {
			continue
		}
		newID := len(nodes) + 1
		newIDByOldID[node.ID] = newID
		node.ID = newID
		node.InDegree = 0
		node.OutDegree = 0
		nodes = append(nodes, node)
	}

	if len(nodes) == 0 {
		return Graph{
			Root:    graph.Root,
			GroupBy: graph.GroupBy,
		}
	}

	edges := make([]Edge, 0, filteredEdgeCount)
	inDegree := ensureScratchInts(scratch.inDegree, len(nodes)+1)
	outDegree := ensureScratchInts(scratch.outDegree, len(nodes)+1)
	scratch.inDegree = inDegree
	scratch.outDegree = outDegree
	edges = appendFilteredEdges(edges, graph.Edges, includeNode, filter.MinEdgeCount, hasPrefixFilter, newIDByOldID, inDegree, outDegree)

	for i := range nodes {
		id := nodes[i].ID
		nodes[i].InDegree = inDegree[id]
		nodes[i].OutDegree = outDegree[id]
	}

	return Graph{
		Root:    graph.Root,
		GroupBy: graph.GroupBy,
		Nodes:   nodes,
		Edges:   edges,
	}
}

func markUsedFilteredEdges(edges []Edge, includeNode []bool, minEdgeCount int, hasPrefixFilter bool, usedNodeIDs []bool) int {
	if hasPrefixFilter {
		return markUsedFilteredEdgesWithPrefix(edges, includeNode, minEdgeCount, usedNodeIDs)
	}
	return markUsedFilteredEdgesByMinCount(edges, minEdgeCount, usedNodeIDs)
}

func markUsedFilteredEdgesByMinCount(edges []Edge, minEdgeCount int, usedNodeIDs []bool) int {
	filteredEdgeCount := 0
	for _, edge := range edges {
		if edge.Count < minEdgeCount {
			continue
		}
		filteredEdgeCount++
		usedNodeIDs[edge.FromID] = true
		usedNodeIDs[edge.ToID] = true
	}
	return filteredEdgeCount
}

func markUsedFilteredEdgesWithPrefix(edges []Edge, includeNode []bool, minEdgeCount int, usedNodeIDs []bool) int {
	filteredEdgeCount := 0
	for _, edge := range edges {
		if edge.Count < minEdgeCount {
			continue
		}
		if edge.FromID >= len(includeNode) || !includeNode[edge.FromID] {
			continue
		}
		if edge.ToID >= len(includeNode) || !includeNode[edge.ToID] {
			continue
		}
		filteredEdgeCount++
		usedNodeIDs[edge.FromID] = true
		usedNodeIDs[edge.ToID] = true
	}
	return filteredEdgeCount
}

func appendFilteredEdges(edges []Edge, graphEdges []Edge, includeNode []bool, minEdgeCount int, hasPrefixFilter bool, newIDByOldID []int, inDegree, outDegree []int) []Edge {
	if hasPrefixFilter {
		return appendFilteredEdgesWithPrefix(edges, graphEdges, includeNode, minEdgeCount, newIDByOldID, inDegree, outDegree)
	}
	return appendFilteredEdgesByMinCount(edges, graphEdges, minEdgeCount, newIDByOldID, inDegree, outDegree)
}

func appendFilteredEdgesByMinCount(edges []Edge, graphEdges []Edge, minEdgeCount int, newIDByOldID []int, inDegree, outDegree []int) []Edge {
	for _, edge := range graphEdges {
		if edge.Count < minEdgeCount {
			continue
		}
		if edge.FromID >= len(newIDByOldID) || edge.ToID >= len(newIDByOldID) {
			continue
		}
		fromID := newIDByOldID[edge.FromID]
		toID := newIDByOldID[edge.ToID]
		if fromID == 0 || toID == 0 {
			continue
		}
		edges = append(edges, Edge{FromID: fromID, ToID: toID, Count: edge.Count})
		outDegree[fromID]++
		inDegree[toID]++
	}
	return edges
}

func appendFilteredEdgesWithPrefix(edges []Edge, graphEdges []Edge, includeNode []bool, minEdgeCount int, newIDByOldID []int, inDegree, outDegree []int) []Edge {
	for _, edge := range graphEdges {
		if edge.Count < minEdgeCount {
			continue
		}
		if edge.FromID >= len(includeNode) || !includeNode[edge.FromID] {
			continue
		}
		if edge.ToID >= len(includeNode) || !includeNode[edge.ToID] {
			continue
		}
		if edge.FromID >= len(newIDByOldID) || edge.ToID >= len(newIDByOldID) {
			continue
		}
		fromID := newIDByOldID[edge.FromID]
		toID := newIDByOldID[edge.ToID]
		if fromID == 0 || toID == 0 {
			continue
		}
		edges = append(edges, Edge{FromID: fromID, ToID: toID, Count: edge.Count})
		outDegree[fromID]++
		inDegree[toID]++
	}
	return edges
}

func getFilterScratch(nodeCount int) *filterScratch {
	scratch := filterScratchPool.Get().(*filterScratch)
	scratch.includeNode = ensureScratchBools(scratch.includeNode, nodeCount)
	scratch.usedNodeIDs = ensureScratchBools(scratch.usedNodeIDs, nodeCount)
	scratch.newIDByOldID = ensureScratchInts(scratch.newIDByOldID, nodeCount)
	return scratch
}

func putFilterScratch(scratch *filterScratch) {
	clear(scratch.includeNode)
	clear(scratch.usedNodeIDs)
	clear(scratch.newIDByOldID)
	clear(scratch.inDegree)
	clear(scratch.outDegree)
	filterScratchPool.Put(scratch)
}

func ensureScratchBools(buf []bool, size int) []bool {
	if cap(buf) < size {
		return make([]bool, size)
	}
	buf = buf[:size]
	clear(buf)
	return buf
}

func ensureScratchInts(buf []int, size int) []int {
	if cap(buf) < size {
		return make([]int, size)
	}
	buf = buf[:size]
	clear(buf)
	return buf
}

func maxGraphNodeID(nodes []Node) int {
	maxID := 0
	for _, node := range nodes {
		if node.ID > maxID {
			maxID = node.ID
		}
	}
	return maxID
}

func shouldKeepFilteredNode(nodeID int, includeNode, usedNodeIDs []bool, hasPrefixFilter bool) bool {
	if nodeID >= len(usedNodeIDs) {
		return false
	}
	if usedNodeIDs[nodeID] {
		return true
	}
	if !hasPrefixFilter || nodeID >= len(includeNode) {
		return false
	}
	return includeNode[nodeID]
}

func normalizeFilter(filter GraphFilter) GraphFilter {
	if filter.MinEdgeCount < 0 {
		filter.MinEdgeCount = 0
	}
	filter.IncludePrefix = normalizePrefixList(filter.IncludePrefix)
	filter.ExcludePrefix = normalizePrefixList(filter.ExcludePrefix)
	return filter
}

func normalizePrefixList(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(input))
	for _, value := range input {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func (f GraphFilter) isActive() bool {
	return f.MinEdgeCount > 0 || len(f.IncludePrefix) > 0 || len(f.ExcludePrefix) > 0
}

func shouldIncludeNode(name string, filter GraphFilter) bool {
	if len(filter.IncludePrefix) > 0 && !hasAnyPrefix(name, filter.IncludePrefix) {
		return false
	}
	if len(filter.ExcludePrefix) > 0 && hasAnyPrefix(name, filter.ExcludePrefix) {
		return false
	}
	return true
}

func hasAnyPrefix(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
