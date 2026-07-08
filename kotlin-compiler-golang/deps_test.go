package kotlincompilergolang

import "testing"

func TestBuildDependencyGraph(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{
			{
				Path:        "/repo/A.kt",
				PackageName: "com.example.a",
				Imports: []string{
					"com.example.b.Service",
					"com.example.b.util.*",
					"kotlin.collections.List",
				},
			},
			{
				Path:        "/repo/B.kt",
				PackageName: "com.example.b",
				Imports: []string{
					"com.example.a.Model",
					"com.example.b.Service",
				},
			},
		},
	}

	graph := BuildDependencyGraph(result)
	if len(graph.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 4 {
		t.Fatalf("expected 4 edges, got %d", len(graph.Edges))
	}

	names := map[int]string{}
	for _, node := range graph.Nodes {
		names[node.ID] = node.Name
	}

	hasEdge := func(from, to string, count int) bool {
		for _, edge := range graph.Edges {
			if names[edge.FromID] == from && names[edge.ToID] == to && edge.Count == count {
				return true
			}
		}
		return false
	}

	if !hasEdge("com.example.a", "com.example.b", 1) {
		t.Fatalf("missing edge com.example.a -> com.example.b")
	}
	if !hasEdge("com.example.a", "com.example.b.util", 1) {
		t.Fatalf("missing edge com.example.a -> com.example.b.util")
	}
	if !hasEdge("com.example.a", "kotlin.collections", 1) {
		t.Fatalf("missing edge com.example.a -> kotlin.collections")
	}
	if !hasEdge("com.example.b", "com.example.a", 1) {
		t.Fatalf("missing edge com.example.b -> com.example.a")
	}
}

func TestBuildDependencyGraphEmptyInput(t *testing.T) {
	graph := BuildDependencyGraph(RepositoryResult{})
	if len(graph.Nodes) != 0 {
		t.Fatalf("expected 0 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Fatalf("expected 0 edges, got %d", len(graph.Edges))
	}
}

func TestBuildDependencyGraphEmptyFilesNoPanic(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{},
	}
	graph := BuildDependencyGraph(result)
	if len(graph.Nodes) != 0 || len(graph.Edges) != 0 {
		t.Fatalf("expected empty graph, got nodes=%d edges=%d", len(graph.Nodes), len(graph.Edges))
	}
}

func TestBuildDependencyGraphSkipsInvalidOrSelfImportsAndAccumulatesCounts(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{
			{
				Path:        "/repo/A.kt",
				PackageName: "com.example.a",
				Imports: []string{
					"com.example.a.Local", // filtered as self edge
					"org.example.Service",
					"org.example.Service",
					"util",                           // ignored (single segment)
					"SingleSegmentOnly",              // ignored (single segment)
					"kotlin.collections.List",        // keeps to kotlin.collections
					"kotlin.collections.*",           // keeps to kotlin.collections
					"   kotlin.collections . List  ", // normalized/trimmed
				},
			},
		},
	}

	graph := BuildDependencyGraph(result)
	if len(graph.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 2 {
		t.Fatalf("expected 2 edge types, got %d", len(graph.Edges))
	}

	names := map[int]string{}
	for _, node := range graph.Nodes {
		names[node.ID] = node.Name
	}

	findCount := func(from, to string) int {
		for _, edge := range graph.Edges {
			if names[edge.FromID] == from && names[edge.ToID] == to {
				return edge.Count
			}
		}
		return 0
	}

	if got := findCount("com.example.a", "org.example"); got != 2 {
		t.Fatalf("org.example edge count got=%d want=2", got)
	}
	if got := findCount("com.example.a", "kotlin.collections"); got != 3 {
		t.Fatalf("kotlin.collections edge count got=%d want=3", got)
	}
	if got := findCount("com.example.a", "util"); got != 0 {
		t.Fatalf("util edge should not exist; got=%d", got)
	}
}

func TestInferImportPackage(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "class import keeps package",
			input:    "com.example.sample.Model",
			expected: "com.example.sample",
		},
		{
			name:     "wildcard import keeps package",
			input:    "com.example.sample.*",
			expected: "com.example.sample",
		},
		{
			name:     "single segment package has no container",
			input:    "NoDotImport",
			expected: "",
		},
		{
			name:     "two-part import resolves to root package",
			input:    "kotlin.Unit",
			expected: "kotlin",
		},
		{
			name:     "empty string is empty",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := inferImportPackage(tc.input)
			if got != tc.expected {
				t.Fatalf("inferImportPackage(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestNormalizeImportPath(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trims spaces",
			input:    "  com.example.Model  ",
			expected: "com.example.Model",
		},
		{
			name:     "removes alias",
			input:    "com.example.Model as M",
			expected: "com.example.Model",
		},
		{
			name:     "removes backticks",
			input:    "`com`.`example`.`Model`",
			expected: "com.example.Model",
		},
		{
			name:     "collapses spaces inside fragments",
			input:    "com . example . Model",
			expected: "com.example.Model",
		},
		{
			name:     "trims dot boundaries",
			input:    "..kotlin.list..",
			expected: "kotlin.list",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeImportPath(tc.input)
			if got != tc.expected {
				t.Fatalf("normalizeImportPath(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}
