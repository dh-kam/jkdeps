package mixedgraph

import (
	"reflect"
	"testing"

	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

func TestFilterGraphMinEdgeCount(t *testing.T) {
	graph := Graph{
		Root:    "/repo",
		GroupBy: GroupByPackage,
		Nodes: []Node{
			{ID: 1, Name: "a", Kind: NodeInternal},
			{ID: 2, Name: "b", Kind: NodeInternal},
			{ID: 3, Name: "c", Kind: NodeExternal},
		},
		Edges: []Edge{
			{FromID: 1, ToID: 2, Count: 2},
			{FromID: 1, ToID: 3, Count: 1},
		},
	}

	filtered := FilterGraph(graph, GraphFilter{MinEdgeCount: 2})
	if len(filtered.Nodes) != 2 {
		t.Fatalf("expected 2 nodes after filtering, got %d", len(filtered.Nodes))
	}
	if len(filtered.Edges) != 1 {
		t.Fatalf("expected 1 edge after filtering, got %d", len(filtered.Edges))
	}
	if filtered.Edges[0].Count != 2 {
		t.Fatalf("expected remaining edge count=2, got %d", filtered.Edges[0].Count)
	}
}

func TestFilterGraphIncludeExcludePrefix(t *testing.T) {
	graph := Graph{
		Root:    "/repo",
		GroupBy: GroupByPackage,
		Nodes: []Node{
			{ID: 1, Name: "com.a", Kind: NodeInternal},
			{ID: 2, Name: "com.b", Kind: NodeInternal},
			{ID: 3, Name: "java.util", Kind: NodeExternal},
		},
		Edges: []Edge{
			{FromID: 1, ToID: 2, Count: 3},
			{FromID: 1, ToID: 3, Count: 4},
		},
	}

	filtered := FilterGraph(graph, GraphFilter{
		IncludePrefix: []string{"com."},
		ExcludePrefix: []string{"com.b"},
	})
	if len(filtered.Nodes) != 1 {
		t.Fatalf("expected 1 node after include/exclude filtering, got %d", len(filtered.Nodes))
	}
	if filtered.Nodes[0].Name != "com.a" {
		t.Fatalf("expected remaining node com.a, got %s", filtered.Nodes[0].Name)
	}
	if len(filtered.Edges) != 0 {
		t.Fatalf("expected no edges after include/exclude filtering, got %d", len(filtered.Edges))
	}
}

func TestFilterGraphIncludeSubgraph(t *testing.T) {
	graph := Graph{
		Root:    "/repo",
		GroupBy: GroupByPackage,
		Nodes: []Node{
			{ID: 1, Name: "com.a", Kind: NodeInternal},
			{ID: 2, Name: "com.b", Kind: NodeInternal},
			{ID: 3, Name: "java.util", Kind: NodeExternal},
		},
		Edges: []Edge{
			{FromID: 1, ToID: 2, Count: 3},
			{FromID: 1, ToID: 3, Count: 4},
		},
	}

	filtered := FilterGraph(graph, GraphFilter{
		IncludePrefix: []string{"com."},
	})
	if len(filtered.Nodes) != 2 {
		t.Fatalf("expected 2 nodes after include filtering, got %d", len(filtered.Nodes))
	}
	if len(filtered.Edges) != 1 {
		t.Fatalf("expected 1 edge after include filtering, got %d", len(filtered.Edges))
	}
}

func TestFilterGraphMinEdgeAndPrefixKeepsSeedNodes(t *testing.T) {
	graph := Graph{
		Root:    "/repo",
		GroupBy: GroupByPackage,
		Nodes: []Node{
			{ID: 1, Name: "com.app", Kind: NodeInternal},
			{ID: 2, Name: "com.lib", Kind: NodeInternal},
			{ID: 3, Name: "java.util", Kind: NodeExternal},
		},
		Edges: []Edge{
			{FromID: 1, ToID: 2, Count: 2},
			{FromID: 2, ToID: 3, Count: 7},
		},
	}

	filtered := FilterGraph(graph, GraphFilter{
		MinEdgeCount:  8,
		IncludePrefix: []string{"com."},
	})

	if len(filtered.Nodes) != 2 {
		t.Fatalf("expected 2 seed nodes to remain, got %d", len(filtered.Nodes))
	}
	if len(filtered.Edges) != 0 {
		t.Fatalf("expected no edges after min-edge filtering, got %d", len(filtered.Edges))
	}

	seen := map[string]struct{}{}
	for _, node := range filtered.Nodes {
		seen[node.Name] = struct{}{}
	}
	if _, ok := seen["com.app"]; !ok {
		t.Fatalf("expected com.app to remain")
	}
	if _, ok := seen["com.lib"]; !ok {
		t.Fatalf("expected com.lib to remain")
	}
}

func TestNormalizeFilterTrimsAndSortsPrefixesAndClampsMinEdge(t *testing.T) {
	got := normalizeFilter(GraphFilter{
		MinEdgeCount:  -7,
		IncludePrefix: []string{"  com.", "", "com.", "org."},
		ExcludePrefix: []string{"tmp.", " ", "tmp.", "build."},
	})

	if got.MinEdgeCount != 0 {
		t.Fatalf("unexpected min edge count: got=%d want=0", got.MinEdgeCount)
	}
	wantInclude := []string{"com.", "org."}
	wantExclude := []string{"build.", "tmp."}
	if !reflect.DeepEqual(got.IncludePrefix, wantInclude) {
		t.Fatalf("include prefixes mismatch: got=%v want=%v", got.IncludePrefix, wantInclude)
	}
	if !reflect.DeepEqual(got.ExcludePrefix, wantExclude) {
		t.Fatalf("exclude prefixes mismatch: got=%v want=%v", got.ExcludePrefix, wantExclude)
	}
}

func TestFilterGraphPreservesEdgeOrdering(t *testing.T) {
	graph := Graph{
		Root:    "/repo",
		GroupBy: GroupByPackage,
		Nodes: []Node{
			{ID: 1, Name: "com.a", Kind: NodeInternal},
			{ID: 2, Name: "com.b", Kind: NodeInternal},
			{ID: 3, Name: "com.c", Kind: NodeInternal},
			{ID: 4, Name: "com.d", Kind: NodeInternal},
		},
		Edges: []Edge{
			{FromID: 1, ToID: 2, Count: 5},
			{FromID: 1, ToID: 4, Count: 4},
			{FromID: 2, ToID: 4, Count: 3},
			{FromID: 3, ToID: 4, Count: 2},
		},
	}

	filtered := FilterGraph(graph, GraphFilter{
		IncludePrefix: []string{"com."},
		ExcludePrefix: []string{"com.c"},
		MinEdgeCount:  3,
	})

	want := []Edge{
		{FromID: 1, ToID: 2, Count: 5},
		{FromID: 1, ToID: 3, Count: 4},
		{FromID: 2, ToID: 3, Count: 3},
	}
	if !reflect.DeepEqual(filtered.Edges, want) {
		t.Fatalf("filtered edges mismatch: got=%v want=%v", filtered.Edges, want)
	}
}

func TestBuildFilteredGraphMatchesBuildThenFilter(t *testing.T) {
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
	filter := GraphFilter{
		MinEdgeCount:  2,
		IncludePrefix: []string{"com.", "java."},
	}

	got := BuildFilteredGraph(result, external, GroupByPackage, filter)
	want := FilterGraph(BuildGraph(result, external, GroupByPackage), filter)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildFilteredGraph mismatch: got=%+v want=%+v", got, want)
	}
}

func TestBuildFilteredGraphMatchesBuildThenFilterByDir(t *testing.T) {
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
	filter := GraphFilter{
		MinEdgeCount:  1,
		IncludePrefix: []string{"a", "b", "java."},
	}

	got := BuildFilteredGraph(result, external, GroupByDir, filter)
	want := FilterGraph(BuildGraph(result, external, GroupByDir), filter)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildFilteredGraph dir mismatch: got=%+v want=%+v", got, want)
	}
}

func TestGraphMethodFilter(t *testing.T) {
	graph := Graph{
		Root:    "/repo",
		GroupBy: GroupByPackage,
		Nodes: []Node{
			{ID: 1, Name: "a", Kind: NodeInternal},
			{ID: 2, Name: "b", Kind: NodeInternal},
		},
		Edges: []Edge{
			{FromID: 1, ToID: 2, Count: 1},
		},
	}

	filter := GraphFilter{MinEdgeCount: 2}
	filtered := graph.Filter(filter)

	// Should behave the same as FilterGraph
	expected := FilterGraph(graph, filter)
	if len(filtered.Nodes) != len(expected.Nodes) {
		t.Fatalf("graph.Filter() nodes = %d, want %d", len(filtered.Nodes), len(expected.Nodes))
	}
	if len(filtered.Edges) != len(expected.Edges) {
		t.Fatalf("graph.Filter() edges = %d, want %d", len(filtered.Edges), len(expected.Edges))
	}
}
