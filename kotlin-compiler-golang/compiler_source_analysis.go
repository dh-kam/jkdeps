package kotlincompilergolang

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
)

type parseSourceAnalysis struct {
	packageName  string
	imports      []string
	declarations []TopLevelDeclaration
	parsed       bool
	diagnostics  []Diagnostic
}

func (c *Compiler) analyzeSource(path string, source []byte) parseSourceAnalysis {
	isScript := isKotlinScriptPath(path)
	outcome := c.parseWithANTLR(path, source)
	packageName, imports, declarations := collectSourceFacts(outcome.tree)

	declarations = c.resolveSourceDeclarations(source, isScript, declarations, outcome.diagnostics)
	packageName, imports = completeSourceHeader(source, packageName, imports)

	diagnostics, parsed := c.finalizeSourceDiagnostics(path, source, outcome.diagnostics)
	return parseSourceAnalysis{
		packageName:  packageName,
		imports:      imports,
		declarations: declarations,
		parsed:       parsed,
		diagnostics:  diagnostics,
	}
}

func collectSourceFacts(tree antlr.ParseTree) (string, []string, []TopLevelDeclaration) {
	collector := newParseTreeCollector()
	if tree != nil {
		antlr.ParseTreeWalkerDefault.Walk(collector, tree)
	}
	return collector.PackageName(), collector.Imports(), collector.Declarations()
}

func (c *Compiler) resolveSourceDeclarations(source []byte, isScript bool, declarations []TopLevelDeclaration, diagnostics []Diagnostic) []TopLevelDeclaration {
	extractedDeclarations := []TopLevelDeclaration(nil)
	if len(diagnostics) > 0 {
		extractedDeclarations = extractTopLevelDeclarations(source)
		if len(extractedDeclarations) > 0 {
			declarations = extractedDeclarations
		} else {
			declarations = mergeTopLevelDeclarations(declarations, extractedDeclarations)
		}
	}

	if len(declarations) == 0 && !isScript {
		declarations = extractTopLevelDeclarations(source)
	}
	if isScript {
		return nil
	}
	return declarations
}

func completeSourceHeader(source []byte, packageName string, imports []string) (string, []string) {
	if packageName != "" && len(imports) > 0 {
		return packageName, imports
	}

	fallbackPkg, fallbackImports := extractHeader(source)
	if packageName == "" {
		packageName = fallbackPkg
	}
	if len(imports) == 0 {
		imports = fallbackImports
	}
	return packageName, imports
}

func (c *Compiler) finalizeSourceDiagnostics(path string, source []byte, diagnostics []Diagnostic) ([]Diagnostic, bool) {
	if len(diagnostics) == 0 {
		if !c.config.LenientSyntax && hasLikelySyntaxErrorInSource(source) && !hasKnownModernKotlinCompatibilitySignal(source) {
			diagnostics = append(diagnostics, Diagnostic{
				Path:     path,
				Line:     1,
				Column:   0,
				Message:  "likely truncated Kotlin source",
				Severity: SeverityError,
			})
			return diagnostics, false
		}
		return diagnostics, true
	}

	if c.config.LenientSyntax {
		return diagnostics, true
	}

	if isLikelyANTLRCompatibilityGap(source, diagnostics) {
		return nil, true
	}

	return diagnostics, false
}

func isLikelyANTLRCompatibilityGap(source []byte, diagnostics []Diagnostic) bool {
	if len(diagnostics) == 0 || hasClearlyIncompleteKotlinSource(source) {
		return false
	}
	return hasKnownModernKotlinCompatibilitySignal(source)
}

func hasClearlyIncompleteKotlinSource(source []byte) bool {
	trimmed := strings.TrimSpace(string(source))
	if trimmed == "" {
		return false
	}
	return strings.HasSuffix(trimmed, "(") ||
		strings.HasSuffix(trimmed, ".") ||
		strings.HasSuffix(trimmed, "[")
}

func hasKnownModernKotlinCompatibilitySignal(source []byte) bool {
	text := string(source)
	if hasAssignmentContinuationCall(text) || hasMultilineGenericTypeContinuation(text) {
		return true
	}
	for _, signal := range []string{
		"actual companion",
		"expect companion",
		"value class",
		"fun interface",
		"sealed interface",
		"context(",
		".() ->",
		"& Any",
		"when (val ",
		"::class.java",
		"object :",
		"data object",
		"..<",
		" as? ",
		" as (",
		"->",
		"buildList {",
		".apply {",
		".also {",
		".let {",
		".map {",
		".firstOrNull {",
		"contract {",
		"callsInPlace(",
		"by config(",
		"by lazy",
		"get() {",
		"get() =",
		"try {",
		"catch (",
		"composable(",
		"NavHost(",
		"\"\"\"",
		"${",
	} {
		if strings.Contains(text, signal) {
			return true
		}
	}
	return false
}

func hasAssignmentContinuationCall(text string) bool {
	for searchFrom := 0; searchFrom < len(text); {
		idx := strings.Index(text[searchFrom:], "=\n")
		if idx < 0 {
			return false
		}
		i := searchFrom + idx + len("=\n")
		for i < len(text) && (text[i] == ' ' || text[i] == '\t' || text[i] == '\r') {
			i++
		}
		if i >= len(text) || !isAnnotationNameStart(text[i]) {
			searchFrom = i
			continue
		}
		for i < len(text) && (isIdentifierByte(text[i]) || text[i] == '.') {
			i++
		}
		for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
			i++
		}
		if i < len(text) && text[i] == '(' {
			return true
		}
		searchFrom = i
	}
	return false
}

func hasMultilineGenericTypeContinuation(text string) bool {
	for searchFrom := 0; searchFrom < len(text); {
		idx := strings.Index(text[searchFrom:], ":\n")
		if idx < 0 {
			return false
		}
		i := searchFrom + idx + len(":\n")
		for i < len(text) && (text[i] == ' ' || text[i] == '\t' || text[i] == '\r') {
			i++
		}
		if i >= len(text) || !isAnnotationNameStart(text[i]) {
			searchFrom = i
			continue
		}
		for i < len(text) && (isIdentifierByte(text[i]) || text[i] == '.') {
			i++
		}
		for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
			i++
		}
		if i < len(text) && text[i] == '<' {
			return true
		}
		searchFrom = i
	}
	return false
}
