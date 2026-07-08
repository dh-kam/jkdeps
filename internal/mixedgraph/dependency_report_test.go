package mixedgraph

import (
	"reflect"
	"testing"

	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

func TestBuildFileDependencyReportCountsAndUnresolved(t *testing.T) {
	result := RepositoryResult{
		Root: "/repo",
		Files: []FileUnit{
			{
				Path:        "/repo/src/main/java/com/example/App.java",
				PackageName: "com.example",
				Imports: []string{
					"com.example.internal.Helper",
					"java.util.List",
					"com.unknown.Missing",
					"com.external.Symbol",
					"kotlin.collections.List",
				},
			},
			{
				Path:        "/repo/src/main/kotlin/com/example/internal/Util.kt",
				PackageName: "com.example.internal",
				Imports: []string{
					"com.example.App",
					"com.unknown.Bad",
					"kotlin.io.Path",
				},
			},
		},
	}

	external := kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util":          {},
			"kotlin.collections": {},
			"kotlin.io":          {},
		},
		Symbols: map[string]struct{}{
			"com.external.Symbol": {},
		},
	}

	report := BuildFileDependencyReport(result, external)
	if report.Root != "/repo" {
		t.Fatalf("unexpected root: %q", report.Root)
	}
	if report.KnownPackages != 2 {
		t.Fatalf("expected 2 known packages, got %d", report.KnownPackages)
	}
	if report.TotalDependencies != 8 {
		t.Fatalf("expected 8 total dependencies, got %d", report.TotalDependencies)
	}
	if report.InternalDependencies != 2 {
		t.Fatalf("expected 2 internal dependencies, got %d", report.InternalDependencies)
	}
	if report.ExternalDependencies != 4 {
		t.Fatalf("expected 4 external dependencies, got %d", report.ExternalDependencies)
	}
	if report.UnknownDependencies != 2 {
		t.Fatalf("expected 2 unknown dependencies, got %d", report.UnknownDependencies)
	}
	if len(report.UnresolvedImports) != 2 {
		t.Fatalf("expected 2 unresolved imports, got %d", len(report.UnresolvedImports))
	}

	byFile := map[string]int{}
	for _, dep := range report.Dependencies {
		if dep.Kind == NodeUnknown {
			byFile[dep.FilePath]++
		}
	}
	if byFile["/repo/src/main/java/com/example/App.java"] != 1 {
		t.Fatalf("expected 1 unresolved dependency from App.java, got %d", byFile["/repo/src/main/java/com/example/App.java"])
	}
	if byFile["/repo/src/main/kotlin/com/example/internal/Util.kt"] != 1 {
		t.Fatalf("expected 1 unresolved dependency from Util.kt, got %d", byFile["/repo/src/main/kotlin/com/example/internal/Util.kt"])
	}

	first := report.UnresolvedImports[0]
	last := report.UnresolvedImports[1]
	if first.FilePath > last.FilePath {
		t.Fatalf("expected unresolved imports to be sorted by file path")
	}
	if first.Kind != NodeUnknown || last.Kind != NodeUnknown {
		t.Fatalf("expected unresolved imports to be NodeUnknown")
	}
	if first.ToPackage != "com.unknown" || last.ToPackage != "com.unknown" {
		t.Fatalf("expected unresolved imports to target package com.unknown, got %q, %q", first.ToPackage, last.ToPackage)
	}
}

func TestDependencyReportAccumulatorCountsKinds(t *testing.T) {
	acc := newDependencyReportAccumulator("/repo", 2)
	acc.add(FileDependency{Kind: NodeInternal})
	acc.add(FileDependency{Kind: NodeExternal})
	acc.add(FileDependency{Kind: NodeUnknown, ImportPath: "com.unknown.Type"})

	report := acc.report
	if report.Root != "/repo" {
		t.Fatalf("report.Root = %q, want /repo", report.Root)
	}
	if report.KnownPackages != 2 {
		t.Fatalf("report.KnownPackages = %d, want 2", report.KnownPackages)
	}
	if report.TotalDependencies != 3 {
		t.Fatalf("report.TotalDependencies = %d, want 3", report.TotalDependencies)
	}
	if report.InternalDependencies != 1 {
		t.Fatalf("report.InternalDependencies = %d, want 1", report.InternalDependencies)
	}
	if report.ExternalDependencies != 1 {
		t.Fatalf("report.ExternalDependencies = %d, want 1", report.ExternalDependencies)
	}
	if report.UnknownDependencies != 1 {
		t.Fatalf("report.UnknownDependencies = %d, want 1", report.UnknownDependencies)
	}
	if len(report.UnresolvedImports) != 1 {
		t.Fatalf("len(report.UnresolvedImports) = %d, want 1", len(report.UnresolvedImports))
	}
}

func TestDependencyReportSortPolicySortsDeterministically(t *testing.T) {
	report := DependencyReport{
		Dependencies: []FileDependency{
			{FilePath: "/repo/b.kt", ToPackage: "z.pkg", ImportPath: "z.pkg.Type"},
			{FilePath: "/repo/a.java", ToPackage: "b.pkg", ImportPath: "b.pkg.Type"},
			{FilePath: "/repo/a.java", ToPackage: "a.pkg", ImportPath: "a.pkg.Type"},
		},
		UnresolvedImports: []FileDependency{
			{FilePath: "/repo/b.kt", ImportPath: "z.pkg.Type"},
			{FilePath: "/repo/a.java", ImportPath: "b.pkg.Type"},
			{FilePath: "/repo/a.java", ImportPath: "a.pkg.Type"},
		},
	}

	got := deterministicDependencyReportSortPolicy.sort(report)
	wantDeps := []FileDependency{
		{FilePath: "/repo/a.java", ToPackage: "a.pkg", ImportPath: "a.pkg.Type"},
		{FilePath: "/repo/a.java", ToPackage: "b.pkg", ImportPath: "b.pkg.Type"},
		{FilePath: "/repo/b.kt", ToPackage: "z.pkg", ImportPath: "z.pkg.Type"},
	}
	wantUnresolved := []FileDependency{
		{FilePath: "/repo/a.java", ImportPath: "a.pkg.Type"},
		{FilePath: "/repo/a.java", ImportPath: "b.pkg.Type"},
		{FilePath: "/repo/b.kt", ImportPath: "z.pkg.Type"},
	}

	if !reflect.DeepEqual(got.Dependencies, wantDeps) {
		t.Fatalf("sorted dependencies mismatch: got=%+v want=%+v", got.Dependencies, wantDeps)
	}
	if !reflect.DeepEqual(got.UnresolvedImports, wantUnresolved) {
		t.Fatalf("sorted unresolved mismatch: got=%+v want=%+v", got.UnresolvedImports, wantUnresolved)
	}
}

func TestDependencyReportBuilderBuildDependency(t *testing.T) {
	builder := newDependencyReportBuilder(newGraphBuildContext([]FileUnit{
		{PackageName: "com.example"},
		{PackageName: "com.example.internal"},
	}, kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
		},
	}), "/repo")

	internal, ok := builder.buildDependency("/repo/App.java", "com.example", "com.example.internal.Helper", ReferenceKindImport, builder.ctx.resolveImport("com.example.internal.Helper"))
	if !ok || internal.Kind != NodeInternal || internal.ToPackage != "com.example.internal" {
		t.Fatalf("internal buildDependency = (%+v, %v), want internal com.example.internal", internal, ok)
	}

	external, ok := builder.buildDependency("/repo/App.java", "com.example", "java.util.List", ReferenceKindImport, builder.ctx.resolveImport("java.util.List"))
	if !ok || external.Kind != NodeExternal || external.ToPackage != "java.util" {
		t.Fatalf("external buildDependency = (%+v, %v), want external java.util", external, ok)
	}

	if _, ok := builder.buildDependency("/repo/App.java", "com.example", "com.example.Type", ReferenceKindImport, builder.ctx.resolveImport("com.example.Type")); ok {
		t.Fatalf("self import unexpectedly resolved")
	}
	if _, ok := builder.buildDependency("/repo/App.java", "com.example", " ", ReferenceKindImport, builder.ctx.resolveImport(" ")); ok {
		t.Fatalf("blank import unexpectedly resolved")
	}
}

func TestDependencyReportBuilderAddFileImportsSkipsEmptyFromPackage(t *testing.T) {
	builder := newDependencyReportBuilder(newGraphBuildContext([]FileUnit{
		{PackageName: "com.example"},
	}, kcg.ExternalIndex{}), "/repo")

	builder.addFileImports(FileUnit{
		Path: "/repo/App.java",
		Imports: []string{
			"com.example.Other",
			"java.util.List",
		},
	}, "")

	if got := builder.acc.report.TotalDependencies; got != 0 {
		t.Fatalf("builder.acc.report.TotalDependencies = %d, want 0", got)
	}
}

func TestBuildFileDependencyReportIncludesTypedReferences(t *testing.T) {
	result := RepositoryResult{
		Root: "/repo",
		Files: []FileUnit{
			{
				Path:        "/repo/src/main/java/com/example/Sample.java",
				PackageName: "com.example",
				References: []Reference{
					{Path: "com.example.base.BaseType", Kind: ReferenceKindExtends},
					{Path: "java.util.List", Kind: ReferenceKindFieldType},
					{Path: "java.io.IOException", Kind: ReferenceKindThrows},
				},
			},
			{
				Path:        "/repo/src/main/java/com/example/base/BaseType.java",
				PackageName: "com.example.base",
			},
		},
	}

	external := kcg.ExternalIndex{
		Packages: map[string]struct{}{
			"java.util": {},
			"java.io":   {},
		},
	}

	report := BuildFileDependencyReport(result, external)
	if report.TotalDependencies != 3 {
		t.Fatalf("report.TotalDependencies = %d, want 3", report.TotalDependencies)
	}

	kinds := map[ReferenceKind]int{}
	for _, dep := range report.Dependencies {
		kinds[dep.ReferenceKind]++
	}
	if kinds[ReferenceKindExtends] != 1 {
		t.Fatalf("extends dependency count = %d, want 1", kinds[ReferenceKindExtends])
	}
	if kinds[ReferenceKindFieldType] != 1 {
		t.Fatalf("field_type dependency count = %d, want 1", kinds[ReferenceKindFieldType])
	}
	if kinds[ReferenceKindThrows] != 1 {
		t.Fatalf("throws dependency count = %d, want 1", kinds[ReferenceKindThrows])
	}
}
