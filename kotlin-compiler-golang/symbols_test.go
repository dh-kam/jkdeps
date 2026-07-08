package kotlincompilergolang

import "testing"

func TestBuildSymbolTable(t *testing.T) {
	result := RepositoryResult{
		Root: "/repo",
		Files: []FileUnit{
			{
				Path:        "/repo/src/A.kt",
				PackageName: "com.example",
				Declarations: []TopLevelDeclaration{
					{Kind: DeclClass, Name: "User", Line: 5},
					{Kind: DeclFunction, Name: "buildUser", Line: 12},
				},
			},
			{
				Path:        "/repo/src/B.kt",
				PackageName: "com.example",
				Declarations: []TopLevelDeclaration{
					{Kind: DeclProperty, Name: "version", Line: 3},
				},
			},
		},
	}

	table := BuildSymbolTable(result)
	if len(table.Symbols) != 6 {
		t.Fatalf("expected 6 symbols, got %d", len(table.Symbols))
	}

	counts := table.CountByKind()
	if counts[SymbolPackage] != 1 {
		t.Fatalf("expected 1 package symbol, got %d", counts[SymbolPackage])
	}
	if counts[SymbolFile] != 2 {
		t.Fatalf("expected 2 file symbols, got %d", counts[SymbolFile])
	}
	if counts[SymbolClass] != 1 || counts[SymbolFunction] != 1 || counts[SymbolProperty] != 1 {
		t.Fatalf("unexpected declaration symbol counts: %+v", counts)
	}

	found := table.FindByPrefix("com.example.User")
	if len(found) != 1 {
		t.Fatalf("expected 1 symbol with prefix com.example.User, got %d", len(found))
	}
	if found[0].Kind != SymbolClass {
		t.Fatalf("expected class symbol, got %s", found[0].Kind)
	}
}

func TestBuildSymbolTableNestedParenting(t *testing.T) {
	result := RepositoryResult{
		Root: "/repo",
		Files: []FileUnit{
			{
				Path:        "/repo/src/Nested.kt",
				PackageName: "com.example",
				Declarations: []TopLevelDeclaration{
					{Kind: DeclClass, Name: "Outer", Line: 1},
					{Kind: DeclClass, Name: "Outer.Inner", Line: 2},
					{Kind: DeclObject, Name: "Outer.Inner.Leaf", Line: 3},
				},
			},
		},
	}

	table := BuildSymbolTable(result)
	byFQN := map[string]Symbol{}
	for _, symbol := range table.Symbols {
		byFQN[symbol.FQN] = symbol
	}

	outer, ok := byFQN["com.example.Outer"]
	if !ok {
		t.Fatalf("missing symbol com.example.Outer")
	}
	inner, ok := byFQN["com.example.Outer.Inner"]
	if !ok {
		t.Fatalf("missing symbol com.example.Outer.Inner")
	}
	leaf, ok := byFQN["com.example.Outer.Inner.Leaf"]
	if !ok {
		t.Fatalf("missing symbol com.example.Outer.Inner.Leaf")
	}
	fileSymbol, ok := byFQN["com.example:Nested.kt"]
	if !ok {
		t.Fatalf("missing file symbol com.example:Nested.kt")
	}

	if outer.ParentID != fileSymbol.ID {
		t.Fatalf("expected Outer parent=file symbol id=%d, got %d", fileSymbol.ID, outer.ParentID)
	}
	if inner.ParentID != outer.ID {
		t.Fatalf("expected Inner parent=Outer id=%d, got %d", outer.ID, inner.ParentID)
	}
	if leaf.ParentID != inner.ID {
		t.Fatalf("expected Leaf parent=Inner id=%d, got %d", inner.ID, leaf.ParentID)
	}
}
