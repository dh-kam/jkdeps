package kotlincompilergolang

import (
	"sort"
	"strings"
)

type DependencyNode struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type DependencyEdge struct {
	FromID int `json:"from_id"`
	ToID   int `json:"to_id"`
	Count  int `json:"count"`
}

type DependencyGraph struct {
	Nodes []DependencyNode `json:"nodes"`
	Edges []DependencyEdge `json:"edges"`
}

func (r RepositoryResult) BuildDependencyGraph() DependencyGraph {
	return BuildDependencyGraph(r)
}

func BuildDependencyGraph(result RepositoryResult) DependencyGraph {
	nodeNames := make(map[string]struct{}, len(result.Files)*2)
	edgeCounts := make(map[string]int, len(result.Files)*4)

	for _, file := range result.Files {
		from := normalizeImportPath(file.PackageName)
		if from == "" {
			continue
		}
		nodeNames[from] = struct{}{}

		for _, imp := range file.Imports {
			to := inferImportPackage(imp)
			if to == "" || to == from {
				continue
			}
			nodeNames[to] = struct{}{}
			key := from + "->" + to
			edgeCounts[key]++
		}
	}

	nodes := make([]DependencyNode, 0, len(nodeNames))
	nodeIDs := make(map[string]int, len(nodeNames))
	names := make([]string, 0, len(nodeNames))
	for name := range nodeNames {
		names = append(names, name)
	}
	sort.Strings(names)
	for idx, name := range names {
		id := idx + 1
		nodes = append(nodes, DependencyNode{ID: id, Name: name})
		nodeIDs[name] = id
	}

	edges := make([]DependencyEdge, 0, len(edgeCounts))
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
		edges = append(edges, DependencyEdge{FromID: fromID, ToID: toID, Count: count})
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

	return DependencyGraph{Nodes: nodes, Edges: edges}
}

func inferImportPackage(importPath string) string {
	path := normalizeImportPath(importPath)
	if path == "" {
		return ""
	}
	if strings.HasSuffix(path, ".*") {
		return strings.TrimSuffix(path, ".*")
	}
	if !strings.Contains(path, ".") {
		return ""
	}

	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return ""
	}
	parts = parts[:len(parts)-1]
	if len(parts) < 1 {
		return ""
	}
	return strings.Join(parts, ".")
}

func normalizeImportPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, " as "); idx >= 0 {
		value = value[:idx]
	}
	value = strings.ReplaceAll(value, "`", "")
	value = strings.Join(strings.Fields(value), "")
	value = strings.Trim(value, ".")
	return value
}
