package parser

import (
	"time"

	ast "github.com/dh-kam/jkdeps/internal/ast"
)

// Language type alias
type language = ast.SourceLanguage

// ParseWithDiagnostics parses source and returns detailed result with diagnostics
// This is used by mixedgraph package for dependency analysis
func ParseWithDiagnostics(path string, source []byte, lang language, grammar JavaGrammar, maxErrors int) ParseDiagnosticsResult {
	startedAt := time.Now()

	// Normalize Kotlin source if needed
	if lang == langKotlin {
		source = normalizeKotlinSource(source)
	}

	listener := newSyntaxErrorListener(maxErrors)
	var parseErr error

	switch lang {
	case langJava:
		parseErr = parseJavaWithListener(source, grammar, listener)
	case langKotlin:
		parseErr = parseKotlinWithListener(source, listener)
	default:
		parseErr = parseJavaWithListener(source, grammar, listener) // fallback
	}

	// Extract package and imports
	var pkgName string
	var imports []string
	if lang == langJava {
		pkgName, imports = extractJavaHeader(string(source))
	}

	// Build diagnostics
	diagnostics := make([]ast.Diagnostic, 0)
	if listener.Err() != nil {
		for _, msg := range listener.errors {
			diagnostics = append(diagnostics, ast.Diagnostic{
				Loc:      ast.Location{FilePath: path},
				Severity: ast.SeverityError,
				Message:  msg,
			})
		}
	}
	if parseErr != nil {
		diagnostics = append(diagnostics, ast.Diagnostic{
			Loc:      ast.Location{FilePath: path},
			Severity: ast.SeverityError,
			Message:  parseErr.Error(),
		})
	}

	return ParseDiagnosticsResult{
		Path:        path,
		Language:    lang,
		PackageName: pkgName,
		Imports:     imports,
		Parsed:      len(diagnostics) == 0,
		Diagnostics: diagnostics,
		Duration:    time.Since(startedAt),
	}
}

// ParseDiagnosticsResult represents the result of parsing with diagnostics
type ParseDiagnosticsResult struct {
	Path        string
	Language    ast.SourceLanguage
	PackageName string
	Imports     []string
	Parsed      bool
	Diagnostics []ast.Diagnostic
	Duration    time.Duration
}

// parseJavaWithListener parses Java source with the given listener
func parseJavaWithListener(source []byte, grammar JavaGrammar, listener *syntaxErrorListener) error {
	source = normalizeJavaSourceForANTLR(source)
	switch grammar {
	case JavaGrammarOrig:
		return parseJavaOrigLL(source) // Uses default error handling
	case JavaGrammar7, JavaGrammar8:
		return parseJava8LL(source)
	case JavaGrammar9:
		return parseJava9LL(source)
	case JavaGrammar11, JavaGrammar17, JavaGrammar20, JavaGrammar21, JavaGrammar25:
		return parseJava20LL(source)
	default:
		return parseJava20LL(source)
	}
}

// parseKotlinWithListener parses Kotlin source with the given listener
func parseKotlinWithListener(source []byte, listener *syntaxErrorListener) error {
	return parseKotlin(source)
}
