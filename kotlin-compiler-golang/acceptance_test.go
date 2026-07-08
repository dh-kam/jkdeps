package kotlincompilergolang

import "testing"

func TestBuildAcceptanceReport(t *testing.T) {
	result := RepositoryResult{
		Root:        "/repo",
		TotalFiles:  4,
		ParsedFiles: 3,
		FailedFiles: 1,
		Files: []FileUnit{
			{
				Path:   "/repo/a.kt",
				Parsed: true,
				Diagnostics: []Diagnostic{
					{Severity: SeverityError, Message: "error 1"},
				},
			},
			{
				Path:   "/repo/b.kt",
				Parsed: true,
				Diagnostics: []Diagnostic{
					{Severity: SeverityError, Message: "error 2"},
					{Severity: SeverityError, Message: "error 3"},
				},
			},
			{
				Path:   "/repo/c.kt",
				Parsed: true,
			},
			{
				Path:   "/repo/d.kt",
				Parsed: false,
			},
		},
	}
	table := SymbolTable{
		Symbols: []Symbol{
			{ID: 1, Kind: SymbolPackage},
			{ID: 2, Kind: SymbolFile},
			{ID: 3, Kind: SymbolClass},
			{ID: 4, Kind: SymbolObject},
			{ID: 5, Kind: SymbolFunction},
			{ID: 6, Kind: SymbolProperty},
			{ID: 7, Kind: SymbolTypeAlias},
		},
	}
	resolve := ResolutionReport{
		TotalImports:      10,
		ResolvedImports:   8,
		UnresolvedImports: 2,
		Items: []ImportResolution{
			{Resolved: true},
			{Resolved: false, Reason: "wildcard package not found"},
			{Resolved: false, Reason: "symbol/package not found"},
		},
	}
	graph := WebGraph{
		Nodes: []WebGraphNode{
			{ID: 1, Kind: WebGraphNodeInternal},
			{ID: 2, Kind: WebGraphNodeExternal},
			{ID: 3, Kind: WebGraphNodeUnknown},
		},
		Edges: []WebGraphEdge{
			{FromID: 1, ToID: 2, Count: 2},
			{FromID: 2, ToID: 3, Count: 1},
		},
	}

	report := BuildAcceptanceReport(result, table, resolve, graph)
	if report.Root != "/repo" {
		t.Fatalf("unexpected root: %s", report.Root)
	}
	if report.Parse.TotalFiles != 4 || report.Parse.ParsedFiles != 3 || report.Parse.FailedFiles != 1 {
		t.Fatalf("unexpected parse metrics: %+v", report.Parse)
	}
	if report.Parse.FilesWithDiagnostics != 2 || report.Parse.TotalDiagnostics != 3 || report.Parse.ErrorDiagnostics != 3 {
		t.Fatalf("unexpected parse diagnostics metrics: %+v", report.Parse)
	}
	if report.Parse.SuccessRate != 75 || report.Parse.FailureRate != 25 {
		t.Fatalf("unexpected parse rates: success=%.2f failure=%.2f", report.Parse.SuccessRate, report.Parse.FailureRate)
	}
	if report.Parse.DiagnosticsFileRate != 50 {
		t.Fatalf("unexpected diagnostics file rate: %.2f", report.Parse.DiagnosticsFileRate)
	}
	if report.Symbols.Total != 7 || report.Symbols.Packages != 1 || report.Symbols.Classes != 1 {
		t.Fatalf("unexpected symbol metrics: %+v", report.Symbols)
	}
	if report.Resolve.TotalImports != 10 || report.Resolve.ResolvedImports != 8 || report.Resolve.UnresolvedImports != 2 {
		t.Fatalf("unexpected resolve metrics: %+v", report.Resolve)
	}
	if report.Resolve.ResolvedRate != 80 {
		t.Fatalf("unexpected resolved rate: %.2f", report.Resolve.ResolvedRate)
	}
	if report.Resolve.UnresolvedByReason["wildcard package not found"] != 1 || report.Resolve.UnresolvedByReason["symbol/package not found"] != 1 {
		t.Fatalf("unexpected unresolved reasons: %+v", report.Resolve.UnresolvedByReason)
	}
	if report.Graph.Nodes != 3 || report.Graph.Edges != 2 {
		t.Fatalf("unexpected graph metrics: %+v", report.Graph)
	}
	if report.Graph.InternalNodes != 1 || report.Graph.ExternalNodes != 1 || report.Graph.UnknownNodes != 1 {
		t.Fatalf("unexpected graph node-kind counts: %+v", report.Graph)
	}
}

func TestBuildAcceptanceReportHandlesEmptyInputs(t *testing.T) {
	result := RepositoryResult{
		Root:        "/repo",
		TotalFiles:  0,
		ParsedFiles: 0,
		FailedFiles: 0,
		Files:       nil,
	}
	table := SymbolTable{
		Symbols: nil,
	}
	resolve := ResolutionReport{
		TotalImports:      0,
		ResolvedImports:   0,
		UnresolvedImports: 0,
		Items:             nil,
	}
	graph := WebGraph{
		Nodes: nil,
		Edges: nil,
	}

	report := BuildAcceptanceReport(result, table, resolve, graph)
	if report.Parse.SuccessRate != 100 || report.Parse.FailureRate != 100-100 {
		t.Fatalf("unexpected parse rates: success=%.2f failure=%.2f", report.Parse.SuccessRate, report.Parse.FailureRate)
	}
	if report.Parse.DiagnosticsFileRate != 0 {
		t.Fatalf("unexpected diagnostics file rate: %.2f", report.Parse.DiagnosticsFileRate)
	}
	if report.Resolve.ResolvedRate != 100 {
		t.Fatalf("unexpected resolved rate: %.2f", report.Resolve.ResolvedRate)
	}
	if report.Resolve.UnresolvedByReason != nil {
		t.Fatalf("expected nil unresolved-by-reason, got: %+v", report.Resolve.UnresolvedByReason)
	}
	if report.Graph.Nodes != 0 || report.Graph.Edges != 0 || report.Graph.InternalNodes != 0 || report.Graph.ExternalNodes != 0 || report.Graph.UnknownNodes != 0 {
		t.Fatalf("unexpected graph metrics: %+v", report.Graph)
	}
}

func TestBuildAcceptanceReportMapsEmptyReasonToUnknown(t *testing.T) {
	result := RepositoryResult{
		Root: "/repo",
	}
	table := SymbolTable{
		Symbols: []Symbol{},
	}
	resolve := ResolutionReport{
		TotalImports:      2,
		ResolvedImports:   0,
		UnresolvedImports: 2,
		Items: []ImportResolution{
			{Resolved: false, Reason: ""},
			{Resolved: false, Reason: ""},
		},
	}
	graph := WebGraph{}

	report := BuildAcceptanceReport(result, table, resolve, graph)
	if report.Resolve.UnresolvedByReason["unknown"] != 2 {
		t.Fatalf("unexpected unresolved reason aggregation: %+v", report.Resolve.UnresolvedByReason)
	}
}

func TestBuildAcceptanceReportCountsUnknownNodeKinds(t *testing.T) {
	graph := WebGraph{
		Nodes: []WebGraphNode{
			{ID: 1, Kind: WebGraphNodeInternal},
			{ID: 2, Kind: WebGraphNodeKind("custom")},
			{ID: 3, Kind: WebGraphNodeExternal},
		},
		Edges: []WebGraphEdge{
			{FromID: 1, ToID: 2},
			{FromID: 2, ToID: 3},
		},
	}

	report := BuildAcceptanceReport(RepositoryResult{}, SymbolTable{}, ResolutionReport{}, graph)
	if report.Graph.Nodes != 3 || report.Graph.Edges != 2 {
		t.Fatalf("unexpected graph counts: nodes=%d edges=%d", report.Graph.Nodes, report.Graph.Edges)
	}
	if report.Graph.InternalNodes != 1 || report.Graph.ExternalNodes != 1 || report.Graph.UnknownNodes != 1 {
		t.Fatalf("unexpected graph node kind counts: %+v", report.Graph)
	}
}

func TestBuildAcceptanceReportSeparatesErrorAndWarningDiagnostics(t *testing.T) {
	result := RepositoryResult{
		TotalFiles:  2,
		ParsedFiles: 2,
		FailedFiles: 0,
		Files: []FileUnit{
			{
				Path:        "a.kt",
				Parsed:      true,
				Diagnostics: []Diagnostic{{Severity: SeverityError, Message: "error"}},
			},
			{
				Path:        "b.kt",
				Parsed:      true,
				Diagnostics: []Diagnostic{{Severity: DiagnosticSeverity("warning"), Message: "warning"}},
			},
		},
	}

	report := BuildAcceptanceReport(result, SymbolTable{}, ResolutionReport{}, WebGraph{})
	if report.Parse.TotalDiagnostics != 2 || report.Parse.ErrorDiagnostics != 1 {
		t.Fatalf("unexpected diagnostic metrics: total=%d errors=%d", report.Parse.TotalDiagnostics, report.Parse.ErrorDiagnostics)
	}
	if report.Parse.FilesWithDiagnostics != 2 {
		t.Fatalf("unexpected files-with-diagnostics metric: %d", report.Parse.FilesWithDiagnostics)
	}
	if report.Parse.DiagnosticsFileRate != 100 {
		t.Fatalf("unexpected diagnostics file rate: %.2f", report.Parse.DiagnosticsFileRate)
	}
}
