package kotlincompilergolang

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type WebGraphNodeKind string

const (
	WebGraphNodeInternal WebGraphNodeKind = "internal"
	WebGraphNodeExternal WebGraphNodeKind = "external"
	WebGraphNodeUnknown  WebGraphNodeKind = "unknown"
)

type WebGraphNode struct {
	ID        int              `json:"id"`
	Name      string           `json:"name"`
	Kind      WebGraphNodeKind `json:"kind"`
	InDegree  int              `json:"in_degree"`
	OutDegree int              `json:"out_degree"`
}

type WebGraphEdge struct {
	FromID int `json:"from_id"`
	ToID   int `json:"to_id"`
	Count  int `json:"count"`
}

type WebGraph struct {
	Root  string         `json:"root"`
	Nodes []WebGraphNode `json:"nodes"`
	Edges []WebGraphEdge `json:"edges"`
}

func (r RepositoryResult) BuildWebGraph(external ExternalIndex) WebGraph {
	return BuildWebGraph(r, external)
}

func BuildWebGraph(result RepositoryResult, external ExternalIndex) WebGraph {
	internalPackages := map[string]struct{}{}
	for _, file := range result.Files {
		pkg := normalizeImportPath(file.PackageName)
		if pkg == "" {
			continue
		}
		internalPackages[pkg] = struct{}{}
	}

	nodeKinds := map[string]WebGraphNodeKind{}
	inDegree := map[string]int{}
	outDegree := map[string]int{}
	edgeCounts := map[string]int{}

	for pkg := range internalPackages {
		nodeKinds[pkg] = WebGraphNodeInternal
	}

	for _, file := range result.Files {
		from := normalizeImportPath(file.PackageName)
		if from == "" {
			continue
		}
		nodeKinds[from] = mergeNodeKind(nodeKinds[from], WebGraphNodeInternal)

		for _, rawImport := range file.Imports {
			importPath := normalizeImportPath(rawImport)
			if importPath == "" {
				continue
			}

			to := resolveGraphImportPackage(importPath, internalPackages, external)
			if to == "" || to == from {
				continue
			}

			targetKind := detectNodeKind(to, importPath, internalPackages, external)
			nodeKinds[to] = mergeNodeKind(nodeKinds[to], targetKind)

			key := from + "->" + to
			edgeCounts[key]++
			outDegree[from]++
			inDegree[to]++
		}
	}

	names := make([]string, 0, len(nodeKinds))
	for name := range nodeKinds {
		names = append(names, name)
	}
	sort.Strings(names)

	nodeIDs := map[string]int{}
	nodes := make([]WebGraphNode, 0, len(names))
	for i, name := range names {
		id := i + 1
		nodeIDs[name] = id
		nodes = append(nodes, WebGraphNode{
			ID:        id,
			Name:      name,
			Kind:      nodeKinds[name],
			InDegree:  inDegree[name],
			OutDegree: outDegree[name],
		})
	}

	edges := make([]WebGraphEdge, 0, len(edgeCounts))
	for key, count := range edgeCounts {
		parts := strings.Split(key, "->")
		if len(parts) != 2 {
			continue
		}
		fromID, okFrom := nodeIDs[parts[0]]
		toID, okTo := nodeIDs[parts[1]]
		if !okFrom || !okTo {
			continue
		}
		edges = append(edges, WebGraphEdge{
			FromID: fromID,
			ToID:   toID,
			Count:  count,
		})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromID != edges[j].FromID {
			return edges[i].FromID < edges[j].FromID
		}
		if edges[i].ToID != edges[j].ToID {
			return edges[i].ToID < edges[j].ToID
		}
		return edges[i].Count > edges[j].Count
	})

	return WebGraph{
		Root:  result.Root,
		Nodes: nodes,
		Edges: edges,
	}
}

func detectNodeKind(pkg, importPath string, internalPackages map[string]struct{}, external ExternalIndex) WebGraphNodeKind {
	if _, ok := internalPackages[pkg]; ok {
		return WebGraphNodeInternal
	}
	if external.HasPackage(pkg) || external.HasSymbol(pkg) || external.HasPackage(importPath) || external.HasSymbol(importPath) {
		return WebGraphNodeExternal
	}
	if isKnownExternalImport(pkg) || isKnownExternalImport(importPath) {
		return WebGraphNodeExternal
	}
	return WebGraphNodeUnknown
}

func resolveGraphImportPackage(importPath string, internalPackages map[string]struct{}, external ExternalIndex) string {
	base := inferImportPackage(importPath)
	if base == "" {
		return ""
	}
	if _, ok := internalPackages[base]; ok {
		return base
	}
	if external.HasPackage(base) {
		return base
	}
	if isKnownExternalImport(base) {
		return base
	}

	normalized := normalizeImportPath(importPath)
	if normalized == "" {
		return base
	}
	if strings.HasSuffix(normalized, ".*") {
		normalized = strings.TrimSuffix(normalized, ".*")
	}

	parts := strings.Split(normalized, ".")
	for i := len(parts) - 1; i >= 1; i-- {
		candidate := strings.Join(parts[:i], ".")
		if _, ok := internalPackages[candidate]; ok {
			return candidate
		}
		if external.HasPackage(candidate) {
			return candidate
		}
		if external.HasSymbol(candidate) {
			return candidate
		}
		if isKnownExternalImport(candidate) {
			return candidate
		}
	}

	if fallback := trimClassLikeSegment(base); fallback != "" {
		if _, ok := internalPackages[fallback]; ok {
			return fallback
		}
		if external.HasPackage(fallback) || external.HasSymbol(fallback) {
			return fallback
		}
		if isKnownExternalImport(fallback) {
			return fallback
		}
		return fallback
	}
	return base
}

func trimClassLikeSegment(path string) string {
	path = normalizeImportPath(path)
	if path == "" {
		return ""
	}
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return ""
	}
	last := parts[len(parts)-1]
	if last == "" {
		return ""
	}
	r, _ := utf8.DecodeRuneInString(last)
	if r == utf8.RuneError {
		return ""
	}
	if unicode.IsUpper(r) || strings.ContainsRune(last, '$') {
		return strings.Join(parts[:len(parts)-1], ".")
	}
	return ""
}

func mergeNodeKind(current, incoming WebGraphNodeKind) WebGraphNodeKind {
	if current == incoming {
		return current
	}
	if current == "" {
		return incoming
	}
	if current == WebGraphNodeInternal || incoming == WebGraphNodeInternal {
		return WebGraphNodeInternal
	}
	if current == WebGraphNodeExternal || incoming == WebGraphNodeExternal {
		return WebGraphNodeExternal
	}
	return WebGraphNodeUnknown
}
