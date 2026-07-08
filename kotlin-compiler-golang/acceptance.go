package kotlincompilergolang

import "time"

type AcceptanceParseMetrics struct {
	TotalFiles           int           `json:"total_files"`
	ParsedFiles          int           `json:"parsed_files"`
	FailedFiles          int           `json:"failed_files"`
	FilesWithDiagnostics int           `json:"files_with_diagnostics"`
	TotalDiagnostics     int           `json:"total_diagnostics"`
	ErrorDiagnostics     int           `json:"error_diagnostics"`
	SuccessRate          float64       `json:"success_rate"`
	FailureRate          float64       `json:"failure_rate"`
	DiagnosticsFileRate  float64       `json:"diagnostics_file_rate"`
	ParseDuration        time.Duration `json:"parse_duration"`
}

type AcceptanceSymbolMetrics struct {
	Total       int `json:"total"`
	Packages    int `json:"packages"`
	Files       int `json:"files"`
	Classes     int `json:"classes"`
	Interfaces  int `json:"interfaces"`
	Objects     int `json:"objects"`
	Functions   int `json:"functions"`
	Properties  int `json:"properties"`
	TypeAliases int `json:"type_aliases"`
}

type AcceptanceResolveMetrics struct {
	TotalImports       int            `json:"total_imports"`
	ResolvedImports    int            `json:"resolved_imports"`
	UnresolvedImports  int            `json:"unresolved_imports"`
	ResolvedRate       float64        `json:"resolved_rate"`
	UnresolvedByReason map[string]int `json:"unresolved_by_reason,omitempty"`
}

type AcceptanceGraphMetrics struct {
	Nodes         int `json:"nodes"`
	Edges         int `json:"edges"`
	InternalNodes int `json:"internal_nodes"`
	ExternalNodes int `json:"external_nodes"`
	UnknownNodes  int `json:"unknown_nodes"`
}

type AcceptanceReport struct {
	Root    string                   `json:"root"`
	Parse   AcceptanceParseMetrics   `json:"parse"`
	Symbols AcceptanceSymbolMetrics  `json:"symbols"`
	Resolve AcceptanceResolveMetrics `json:"resolve"`
	Graph   AcceptanceGraphMetrics   `json:"graph"`
}

func BuildAcceptanceReport(result RepositoryResult, table SymbolTable, resolve ResolutionReport, graph WebGraph) AcceptanceReport {
	parseSuccessRate := 100.0
	if result.TotalFiles > 0 {
		parseSuccessRate = (float64(result.ParsedFiles) / float64(result.TotalFiles)) * 100
	}
	parseFailureRate := 100.0 - parseSuccessRate
	filesWithDiagnostics := 0
	totalDiagnostics := 0
	errorDiagnostics := 0
	for _, file := range result.Files {
		diagCount := len(file.Diagnostics)
		if diagCount == 0 {
			continue
		}
		filesWithDiagnostics++
		totalDiagnostics += diagCount
		for _, diag := range file.Diagnostics {
			if diag.Severity == SeverityError {
				errorDiagnostics++
			}
		}
	}
	diagnosticsFileRate := 0.0
	if result.TotalFiles > 0 {
		diagnosticsFileRate = (float64(filesWithDiagnostics) / float64(result.TotalFiles)) * 100
	}

	resolvedRate := 100.0
	if resolve.TotalImports > 0 {
		resolvedRate = (float64(resolve.ResolvedImports) / float64(resolve.TotalImports)) * 100
	}

	counts := table.CountByKind()
	unresolvedByReason := map[string]int{}
	for _, item := range resolve.Items {
		if item.Resolved {
			continue
		}
		reason := item.Reason
		if reason == "" {
			reason = "unknown"
		}
		unresolvedByReason[reason]++
	}
	if len(unresolvedByReason) == 0 {
		unresolvedByReason = nil
	}

	graphMetrics := AcceptanceGraphMetrics{
		Nodes: len(graph.Nodes),
		Edges: len(graph.Edges),
	}
	for _, node := range graph.Nodes {
		switch node.Kind {
		case WebGraphNodeInternal:
			graphMetrics.InternalNodes++
		case WebGraphNodeExternal:
			graphMetrics.ExternalNodes++
		default:
			graphMetrics.UnknownNodes++
		}
	}

	return AcceptanceReport{
		Root: result.Root,
		Parse: AcceptanceParseMetrics{
			TotalFiles:           result.TotalFiles,
			ParsedFiles:          result.ParsedFiles,
			FailedFiles:          result.FailedFiles,
			FilesWithDiagnostics: filesWithDiagnostics,
			TotalDiagnostics:     totalDiagnostics,
			ErrorDiagnostics:     errorDiagnostics,
			SuccessRate:          parseSuccessRate,
			FailureRate:          parseFailureRate,
			DiagnosticsFileRate:  diagnosticsFileRate,
			ParseDuration:        result.Duration,
		},
		Symbols: AcceptanceSymbolMetrics{
			Total:       len(table.Symbols),
			Packages:    counts[SymbolPackage],
			Files:       counts[SymbolFile],
			Classes:     counts[SymbolClass],
			Interfaces:  counts[SymbolInterface],
			Objects:     counts[SymbolObject],
			Functions:   counts[SymbolFunction],
			Properties:  counts[SymbolProperty],
			TypeAliases: counts[SymbolTypeAlias],
		},
		Resolve: AcceptanceResolveMetrics{
			TotalImports:       resolve.TotalImports,
			ResolvedImports:    resolve.ResolvedImports,
			UnresolvedImports:  resolve.UnresolvedImports,
			ResolvedRate:       resolvedRate,
			UnresolvedByReason: unresolvedByReason,
		},
		Graph: graphMetrics,
	}
}
