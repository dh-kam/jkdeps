package kotlincompilergolang

import "testing"

func TestResolveImports(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{
			{
				Path:        "/repo/A.kt",
				PackageName: "com.example.a",
				Imports: []string{
					"com.example.b.User",
					"com.example.b.util.*",
					"missing.pkg.Type",
				},
			},
		},
	}

	table := SymbolTable{Symbols: []Symbol{
		{ID: 1, Name: "com.example.a", FQN: "com.example.a", Kind: SymbolPackage},
		{ID: 2, Name: "com.example.b", FQN: "com.example.b", Kind: SymbolPackage},
		{ID: 3, Name: "com.example.b.util", FQN: "com.example.b.util", Kind: SymbolPackage},
		{ID: 4, Name: "User", FQN: "com.example.b.User", Kind: SymbolClass},
	}}

	report := ResolveImports(result, table)
	if report.TotalImports != 3 {
		t.Fatalf("expected 3 imports, got %d", report.TotalImports)
	}
	if report.ResolvedImports != 2 {
		t.Fatalf("expected 2 resolved imports, got %d", report.ResolvedImports)
	}
	if report.UnresolvedImports != 1 {
		t.Fatalf("expected 1 unresolved import, got %d", report.UnresolvedImports)
	}

	resolved := 0
	unresolved := 0
	for _, item := range report.Items {
		if item.Resolved {
			resolved++
		}
		if !item.Resolved {
			unresolved++
		}
	}
	if resolved != 2 || unresolved != 1 {
		t.Fatalf("unexpected resolution breakdown: resolved=%d unresolved=%d", resolved, unresolved)
	}
}

func TestResolveImportsWithExternalPackages(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{
			{
				Path:        "/repo/A.kt",
				PackageName: "com.example.a",
				Imports: []string{
					"kotlin.coroutines.*",
					"kotlin.js.Promise",
					"org.w3c.dom.Window",
				},
			},
		},
	}

	table := SymbolTable{Symbols: []Symbol{
		{ID: 1, Name: "com.example.a", FQN: "com.example.a", Kind: SymbolPackage},
	}}
	external := map[string]struct{}{
		"kotlin.coroutines": {},
		"kotlin.js":         {},
	}

	report := ResolveImportsWithPackages(result, table, external)
	if report.TotalImports != 3 {
		t.Fatalf("expected 3 imports, got %d", report.TotalImports)
	}
	if report.ResolvedImports != 2 {
		t.Fatalf("expected 2 resolved imports, got %d", report.ResolvedImports)
	}
	if report.UnresolvedImports != 1 {
		t.Fatalf("expected 1 unresolved import, got %d", report.UnresolvedImports)
	}

	externalResolved := 0
	for _, item := range report.Items {
		if item.Resolved && item.External {
			externalResolved++
		}
	}
	if externalResolved != 2 {
		t.Fatalf("expected 2 external resolutions, got %d", externalResolved)
	}
}

func TestResolveImportsWithExternalSymbols(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{
			{
				Path:        "/repo/A.kt",
				PackageName: "com.example.a",
				Imports: []string{
					"kotlin.js.Promise",
					"kotlin.coroutines.*",
				},
			},
		},
	}

	table := SymbolTable{Symbols: []Symbol{
		{ID: 1, Name: "com.example.a", FQN: "com.example.a", Kind: SymbolPackage},
	}}
	external := ExternalIndex{
		Packages: map[string]struct{}{
			"kotlin.coroutines": {},
		},
		Symbols: map[string]struct{}{
			"kotlin.js.Promise": {},
		},
	}

	report := ResolveImportsWithExternal(result, table, external)
	if report.TotalImports != 2 {
		t.Fatalf("expected 2 imports, got %d", report.TotalImports)
	}
	if report.ResolvedImports != 2 {
		t.Fatalf("expected 2 resolved imports, got %d", report.ResolvedImports)
	}
	if report.UnresolvedImports != 0 {
		t.Fatalf("expected 0 unresolved imports, got %d", report.UnresolvedImports)
	}
}

func TestResolveImportsWildcardSymbolAndNestedMember(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{
			{
				Path:        "/repo/A.kt",
				PackageName: "kotlinx.coroutines",
				Imports: []string{
					"kotlinx.coroutines.CoroutineStart.*",
					"kotlinx.coroutines.channels.Channel.Factory.CONFLATED",
				},
			},
		},
	}

	table := SymbolTable{Symbols: []Symbol{
		{ID: 1, Name: "kotlinx.coroutines", FQN: "kotlinx.coroutines", Kind: SymbolPackage},
		{ID: 2, Name: "CoroutineStart", FQN: "kotlinx.coroutines.CoroutineStart", Kind: SymbolClass},
		{ID: 3, Name: "kotlinx.coroutines.channels", FQN: "kotlinx.coroutines.channels", Kind: SymbolPackage},
		{ID: 4, Name: "Channel", FQN: "kotlinx.coroutines.channels.Channel", Kind: SymbolInterface},
	}}

	report := ResolveImports(result, table)
	if report.TotalImports != 2 {
		t.Fatalf("expected 2 imports, got %d", report.TotalImports)
	}
	if report.ResolvedImports != 2 || report.UnresolvedImports != 0 {
		t.Fatalf("unexpected resolve counts: resolved=%d unresolved=%d", report.ResolvedImports, report.UnresolvedImports)
	}
}

func TestResolveImportsWithExternalPrefixFallback(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{
			{
				Path:        "/repo/A.kt",
				PackageName: "com.example",
				Imports: []string{
					"kotlin.time.Duration.Companion.milliseconds",
				},
			},
		},
	}

	table := SymbolTable{Symbols: []Symbol{
		{ID: 1, Name: "com.example", FQN: "com.example", Kind: SymbolPackage},
	}}
	external := ExternalIndex{
		Symbols: map[string]struct{}{
			"kotlin.time.Duration": {},
		},
	}

	report := ResolveImportsWithExternal(result, table, external)
	if report.TotalImports != 1 {
		t.Fatalf("expected 1 import, got %d", report.TotalImports)
	}
	if report.ResolvedImports != 1 || report.UnresolvedImports != 0 {
		t.Fatalf("unexpected resolve counts: resolved=%d unresolved=%d", report.ResolvedImports, report.UnresolvedImports)
	}
	if !report.Items[0].External {
		t.Fatalf("expected external resolution")
	}
}

func TestResolveImportsWithKnownExternalPackages(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{
			{
				Path:        "/tmp/Sample.kt",
				PackageName: "com.example",
				Imports: []string{
					"java.util.concurrent.*",
					"java.lang.ThreadLocal",
					"android.annotation.*",
					"org.codehaus.mojo.animal_sniffer.*",
					"org.junit.Test",
					"org.openjdk.jmh.annotations.*",
					"platform.posix.*",
				},
				Parsed: true,
			},
		},
		TotalFiles: 1,
	}
	table := SymbolTable{
		Symbols: []Symbol{
			{ID: 1, Kind: SymbolPackage, Name: "com.example", FQN: "com.example"},
		},
	}

	report := ResolveImports(result, table)
	if report.ResolvedImports != 7 || report.UnresolvedImports != 0 {
		t.Fatalf("unexpected resolve counts: resolved=%d unresolved=%d", report.ResolvedImports, report.UnresolvedImports)
	}
	for _, item := range report.Items {
		if !item.Resolved || !item.External {
			t.Fatalf("expected known external import to resolve as external: %+v", item)
		}
	}
}

func TestResolveImportsRecordsUnresolvedReasons(t *testing.T) {
	result := RepositoryResult{
		Files: []FileUnit{
			{
				Path:        "/repo/A.kt",
				PackageName: "com.example",
				Imports: []string{
					"zz.zzz.UnknownType",
					"zz.zzz.*",
					"",
					"bad .",
				},
			},
		},
	}
	table := SymbolTable{Symbols: []Symbol{
		{ID: 1, Name: "com.example", FQN: "com.example", Kind: SymbolPackage},
	}}

	report := ResolveImports(result, table)
	if report.TotalImports != 4 {
		t.Fatalf("expected 4 imports, got %d", report.TotalImports)
	}
	if report.ResolvedImports != 0 {
		t.Fatalf("expected 0 resolved imports, got %d", report.ResolvedImports)
	}
	if report.UnresolvedImports != 3 {
		t.Fatalf("expected 3 unresolved imports, got %d", report.UnresolvedImports)
	}
	if len(report.Items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(report.Items))
	}

	reasons := map[string]int{}
	for _, item := range report.Items {
		reasons[item.Reason]++
	}
	if reasons["empty import"] != 1 {
		t.Fatalf("expected 1 empty import reason, got %d", reasons["empty import"])
	}
	if reasons["wildcard package not found"] != 1 {
		t.Fatalf("expected 1 wildcard package not found reason, got %d", reasons["wildcard package not found"])
	}
	if reasons["symbol/package not found"] != 2 {
		t.Fatalf("expected 2 symbol/package not found reasons, got %d", reasons["symbol/package not found"])
	}
}

func TestWildcardPackage(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "wildcard import",
			input:    "kotlinx.coroutines.*",
			expected: "kotlinx.coroutines",
		},
		{
			name:     "single segment wildcard",
			input:    "kotlin.*",
			expected: "kotlin",
		},
		{
			name:     "no wildcard",
			input:    "kotlinx.coroutines",
			expected: "",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := wildcardPackage(tc.input)
			if got != tc.expected {
				t.Fatalf("wildcardPackage(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestResolveImportPrefix(t *testing.T) {
	symbolByFQN := map[string]Symbol{
		"com.example.User":    {ID: 10},
		"com.example.Top":     {ID: 11},
		"com.deep.Target":     {ID: 12},
		"com.deep.nested":     {ID: 13},
		"com.external.Symbol": {ID: 14},
	}
	packageByName := map[string]Symbol{
		"com.example":     {ID: 1},
		"com.deep.nested": {ID: 2},
	}

	t.Run("resolves exact symbol", func(t *testing.T) {
		external := NewExternalIndex()
		resolved, externalResolved, symbolID := resolveImportPrefix("com.example.User", packageByName, symbolByFQN, external)
		if !resolved || externalResolved || symbolID != 10 {
			t.Fatalf("resolveImportPrefix() = (%v,%v,%d), want (true,false,10)", resolved, externalResolved, symbolID)
		}
	})

	t.Run("falls back to package", func(t *testing.T) {
		external := NewExternalIndex()
		resolved, externalResolved, symbolID := resolveImportPrefix("com.example.Missing", packageByName, symbolByFQN, external)
		if !resolved || externalResolved || symbolID != 1 {
			t.Fatalf("resolveImportPrefix() = (%v,%v,%d), want (true,false,1)", resolved, externalResolved, symbolID)
		}
	})

	t.Run("resolves nearest prefix symbol", func(t *testing.T) {
		external := NewExternalIndex()
		resolved, externalResolved, symbolID := resolveImportPrefix("com.example.Top.Inner.Name", packageByName, symbolByFQN, external)
		if !resolved || externalResolved || symbolID != 11 {
			t.Fatalf("resolveImportPrefix() = (%v,%v,%d), want (true,false,11)", resolved, externalResolved, symbolID)
		}
	})

	t.Run("resolves external package fallback", func(t *testing.T) {
		external := NewExternalIndex()
		external.Packages["com.ext"] = struct{}{}
		resolved, externalResolved, symbolID := resolveImportPrefix("com.ext.Module.Type", packageByName, symbolByFQN, external)
		if !resolved || !externalResolved || symbolID != 0 {
			t.Fatalf("resolveImportPrefix() = (%v,%v,%d), want (true,true,0)", resolved, externalResolved, symbolID)
		}
	})

	t.Run("normalizes and resolves spaced wildcard-like dotted import", func(t *testing.T) {
		external := NewExternalIndex()
		resolved, externalResolved, symbolID := resolveImportPrefix("  com . example . User  ", packageByName, symbolByFQN, external)
		if !resolved || externalResolved || symbolID != 10 {
			t.Fatalf("resolveImportPrefix() = (%v,%v,%d), want (true,false,10)", resolved, externalResolved, symbolID)
		}
	})

	t.Run("returns false on empty input", func(t *testing.T) {
		external := NewExternalIndex()
		resolved, externalResolved, symbolID := resolveImportPrefix("   ", packageByName, symbolByFQN, external)
		if resolved || externalResolved || symbolID != 0 {
			t.Fatalf("resolveImportPrefix() = (%v,%v,%d), want (false,false,0)", resolved, externalResolved, symbolID)
		}
	})

	t.Run("resolves none", func(t *testing.T) {
		external := NewExternalIndex()
		resolved, externalResolved, symbolID := resolveImportPrefix("com.unknown.Module", packageByName, symbolByFQN, external)
		if resolved || externalResolved || symbolID != 0 {
			t.Fatalf("resolveImportPrefix() = (%v,%v,%d), want (false,false,0)", resolved, externalResolved, symbolID)
		}
	})
}
