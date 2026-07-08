package kotlincompilergolang

import "testing"

func TestBuildWebGraph(t *testing.T) {
	result := RepositoryResult{
		Root: "/repo",
		Files: []FileUnit{
			{
				Path:        "/repo/A.kt",
				PackageName: "com.example.a",
				Imports: []string{
					"com.example.b.Service",
					"com.example.b.Inner.Factory.VALUE",
					"kotlin.js.Promise",
					"kotlin.time.Duration.Companion.milliseconds",
					"missing.pkg.Type",
				},
			},
			{
				Path:        "/repo/B.kt",
				PackageName: "com.example.b",
				Imports: []string{
					"com.example.a.Model",
				},
			},
		},
	}

	external := ExternalIndex{
		Packages: map[string]struct{}{
			"kotlin.js": {},
		},
		Symbols: map[string]struct{}{
			"kotlin.js.Promise":    {},
			"kotlin.time.Duration": {},
		},
	}

	graph := BuildWebGraph(result, external)
	if graph.Root != "/repo" {
		t.Fatalf("unexpected root: %s", graph.Root)
	}
	if len(graph.Nodes) != 5 {
		t.Fatalf("expected 5 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 5 {
		t.Fatalf("expected 5 edges, got %d", len(graph.Edges))
	}

	nodeByName := map[string]WebGraphNode{}
	for _, node := range graph.Nodes {
		nodeByName[node.Name] = node
	}

	if nodeByName["com.example.a"].Kind != WebGraphNodeInternal {
		t.Fatalf("expected com.example.a to be internal, got %s", nodeByName["com.example.a"].Kind)
	}
	if nodeByName["kotlin.js"].Kind != WebGraphNodeExternal {
		t.Fatalf("expected kotlin.js to be external, got %s", nodeByName["kotlin.js"].Kind)
	}
	if nodeByName["kotlin.time.Duration"].Kind != WebGraphNodeExternal {
		t.Fatalf("expected kotlin.time.Duration to be external, got %s", nodeByName["kotlin.time.Duration"].Kind)
	}
	if nodeByName["missing.pkg"].Kind != WebGraphNodeUnknown {
		t.Fatalf("expected missing.pkg to be unknown, got %s", nodeByName["missing.pkg"].Kind)
	}

	nameByID := map[int]string{}
	for _, node := range graph.Nodes {
		nameByID[node.ID] = node.Name
	}
	hasEdge := func(from, to string, count int) bool {
		for _, edge := range graph.Edges {
			if nameByID[edge.FromID] == from && nameByID[edge.ToID] == to && edge.Count == count {
				return true
			}
		}
		return false
	}

	if !hasEdge("com.example.a", "com.example.b", 2) {
		t.Fatalf("missing edge com.example.a -> com.example.b (count=2)")
	}
	if !hasEdge("com.example.a", "kotlin.time.Duration", 1) {
		t.Fatalf("missing edge com.example.a -> kotlin.time.Duration")
	}
	if !hasEdge("com.example.a", "kotlin.js", 1) {
		t.Fatalf("missing edge com.example.a -> kotlin.js")
	}
	if !hasEdge("com.example.a", "missing.pkg", 1) {
		t.Fatalf("missing edge com.example.a -> missing.pkg")
	}
	if !hasEdge("com.example.b", "com.example.a", 1) {
		t.Fatalf("missing edge com.example.b -> com.example.a")
	}
}

func TestBuildWebGraphKnownExternalPackages(t *testing.T) {
	result := RepositoryResult{
		Root: "/tmp/repo",
		Files: []FileUnit{
			{
				Path:        "/tmp/repo/A.kt",
				PackageName: "com.example",
				Imports: []string{
					"java.util.concurrent.atomic.AtomicInteger",
					"android.annotation.*",
					"org.junit.Test",
					"platform.CoreFoundation.*",
				},
				Parsed: true,
			},
		},
		TotalFiles: 1,
	}

	graph := BuildWebGraph(result, ExternalIndex{})
	nodeByName := map[string]WebGraphNode{}
	for _, node := range graph.Nodes {
		nodeByName[node.Name] = node
	}

	if nodeByName["java.util.concurrent.atomic"].Kind != WebGraphNodeExternal {
		t.Fatalf("expected java.util.concurrent.atomic to be external, got %+v", nodeByName["java.util.concurrent.atomic"])
	}
	if nodeByName["android.annotation"].Kind != WebGraphNodeExternal {
		t.Fatalf("expected android.annotation to be external, got %+v", nodeByName["android.annotation"])
	}
	if nodeByName["org.junit"].Kind != WebGraphNodeExternal {
		t.Fatalf("expected org.junit to be external, got %+v", nodeByName["org.junit"])
	}
	if nodeByName["platform.CoreFoundation"].Kind != WebGraphNodeExternal {
		t.Fatalf("expected platform.CoreFoundation to be external, got %+v", nodeByName["platform.CoreFoundation"])
	}
}
