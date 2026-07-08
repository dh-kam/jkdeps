package kotlincompilergolang

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/antlr4-go/antlr/v4"
	kotlinparser "github.com/dh-kam/jkdeps/internal/parsers/kotlin"
)

var packageLinePattern = regexp.MustCompile(`^package\s+([A-Za-z_][A-Za-z0-9_\.]*)\b`)
var importLinePattern = regexp.MustCompile(`^import\s+([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*(?:\.\*)?)`)
var annotationPattern = regexp.MustCompile(`^@\w+(?:\([^)]*\))?\s*`)
var modifierPattern = regexp.MustCompile(`^(public|private|protected|internal|expect|actual|open|final|abstract|sealed|data|value|inline|tailrec|operator|infix|external|suspend|const|lateinit|override|vararg|noinline|crossinline|reified|enum|annotation|inner)\b\s*`)
var classLikePattern = regexp.MustCompile(`^(class|interface|object|typealias)\s+([A-Za-z_][A-Za-z0-9_]*)`)
var propertyPattern = regexp.MustCompile(`^(val|var)\s+([A-Za-z_][A-Za-z0-9_]*)`)
var functionPattern = regexp.MustCompile(`^fun\b(?:\s*<[^>\n]+>)?\s*(?:[A-Za-z_][A-Za-z0-9_<>?,.\s]*\.)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
var platformModifierNormalizationPattern = regexp.MustCompile(`\b(?:expect|actual)\s+((?:public|private|protected|internal|enum|sealed|annotation|data|inner|tailrec|operator|inline|infix|external|suspend|override|abstract|final|open|const|lateinit|vararg|noinline|crossinline|reified|companion|class|interface|fun|object|val|var|typealias|constructor)\b)`)
var valueClassNormalizationPattern = regexp.MustCompile(`\bvalue\s+(class\b)`)
var funInterfaceNormalizationPattern = regexp.MustCompile(`\bfun\s+(interface\b)`)
var contextReceiverNormalizationPattern = regexp.MustCompile(`(?m)^[ \t]*context\s*\([^)\n]*\)\s*`)
var contextFunctionTypeNormalizationPattern = regexp.MustCompile(`\bcontext\s*\([^)\n]*\)\s*(\([^)\n]*\)\s*->)`)
var sealedInterfaceNormalizationPattern = regexp.MustCompile(`\bsealed\s+(interface\b)`)
var lambdaLabelNormalizationPattern = regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*@\{`)
var extensionFunctionTypeCastNormalizationPattern = regexp.MustCompile(`\s+as\s+\([^\n]*\.\(\)\s*->\s*[A-Za-z0-9_?.<>,\[\] ]+\)`)
var arrayExtensionFunctionTypeCastNormalizationPattern = regexp.MustCompile(`\s+as\s+Array<out\s+[^\n]*\.\(\)\s*->\s*[^>\n]+>`)
var functionTypeCastNormalizationPattern = regexp.MustCompile(`\s+as\s+(?:suspend\s+)?\([^)\n]*\)\s*->\s*[A-Za-z0-9_?.<>,\[\] ]+`)
var suspendReceiverFunctionTypeNormalizationPattern = regexp.MustCompile(`\bsuspend\s+[A-Za-z_][A-Za-z0-9_.<>,?]*\.\(\)\s*->`)
var receiverFunctionTypeNormalizationPattern = regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_.<>,?]*\.\(\)\s*->`)
var definitelyNonNullTypeNormalizationPattern = regexp.MustCompile(`\s*&\s*Any\b`)
var nullableTypeCastNormalizationPattern = regexp.MustCompile(`\bas\s+([A-Za-z_][A-Za-z0-9_<>,\.]*)\?`)
var extensionStarReceiverNormalizationPattern = regexp.MustCompile(`(?m)(\bfun[^\n]*?)<\*>\.`)
var extensionGenericReceiverNormalizationPattern = regexp.MustCompile(`(?m)(\bfun\b(?:\s*<[^>\n]+>)?\s+[A-Za-z_][A-Za-z0-9_?.]*)<[^>\n]+>\.`)
var genericSupertypeNormalizationPattern = regexp.MustCompile(`(\)\s*:\s*[A-Za-z_][A-Za-z0-9_.]*)<[^<>{}\n]+>|(\b(?:class|object)\s+[^\n:{]+:\s*[A-Za-z_][A-Za-z0-9_.]*)<[^<>{}\n]+>|(,\s*[A-Za-z_][A-Za-z0-9_.]*)<[^<>{}\n]+>`)
var objectExpressionSupertypeNormalizationPattern = regexp.MustCompile(`object\s*:\s*[A-Za-z_][A-Za-z0-9_<>,?. ]*\s*\{`)
var stringTemplateExpressionNormalizationPattern = regexp.MustCompile(`\$\{[^{}\n]*\}`)
var anonymousFunctionAssignmentPattern = regexp.MustCompile(`=\s*fun\s*\(([^)\n]*)\)\s*(?::\s*[^{}\n]+)?\s*\{`)
var emptyDelegationConstructorCallPattern = regexp.MustCompile(`:\s*[A-Za-z_][A-Za-z0-9_<>,?. ]*\(\)`)
var unsignedIntegerLiteralPattern = regexp.MustCompile(`\b([0-9]+)[uU]\b`)
var rangeUntilOperatorNormalizationPattern = regexp.MustCompile(`\.\.<`)
var standaloneUnderscoreNormalizationPattern = regexp.MustCompile(`(^|[^A-Za-z0-9_$])_([^A-Za-z0-9_$]|$)`)
var localFunctionDeclarationPattern = regexp.MustCompile(`(?m)^[ \t]*(?:suspend\s+)?fun\b`)
var runTestComplexBodyPattern = regexp.MustCompile(`(?m)"[^"\n]*\\n"\s*\+`)

type Compiler struct {
	config Config
}

type parseSourceOutcome struct {
	tree        antlr.ParseTree
	diagnostics []Diagnostic
}

func New(config Config) *Compiler {
	return &Compiler{config: config.withDefaults()}
}

func (c *Compiler) ParseFile(path string) (FileUnit, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return FileUnit{}, fmt.Errorf("read file: %w", err)
	}
	unit := c.ParseSource(path, source)
	return unit, nil
}

func (c *Compiler) ParseSource(path string, source []byte) FileUnit {
	startedAt := time.Now()
	analysis := c.analyzeSource(path, source)
	return buildFileUnit(path, analysis, time.Since(startedAt))
}

func buildFileUnit(path string, analysis parseSourceAnalysis, duration time.Duration) FileUnit {
	return FileUnit{
		Path:         path,
		PackageName:  analysis.packageName,
		Imports:      analysis.imports,
		Declarations: analysis.declarations,
		Parsed:       analysis.parsed,
		Diagnostics:  analysis.diagnostics,
		Duration:     duration,
	}
}

func hasLikelySyntaxErrorInSource(source []byte) bool {
	text := string(source)
	if strings.TrimSpace(text) == "" {
		return false
	}
	if hasUnmatchedDelimiters(text) {
		return true
	}
	trimmed := strings.TrimSpace(text)
	if strings.HasSuffix(trimmed, "(") || strings.HasSuffix(trimmed, ".") {
		return true
	}
	return false
}

func hasUnmatchedDelimiters(text string) bool {
	inLineComment := false
	inBlockComment := false
	inChar := false
	inString := false
	inRawString := false
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	lastNonWs := byte(0)

	for i := 0; i < len(text); i++ {
		ch := text[i]
		lastNonWs = ch
		if inLineComment {
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && i+1 < len(text) && text[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if inRawString {
			if ch == '"' && i+2 < len(text) && text[i+1] == '"' && text[i+2] == '"' {
				inRawString = false
				i += 2
			}
			continue
		}
		if inChar {
			if ch == '\\' {
				i++
				continue
			}
			if ch == '\'' {
				inChar = false
			}
			continue
		}
		if inString {
			if ch == '\\' {
				i++
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '/' && i+1 < len(text) && text[i+1] == '/' {
			inLineComment = true
			i++
			continue
		}
		if ch == '/' && i+1 < len(text) && text[i+1] == '*' {
			inBlockComment = true
			i++
			continue
		}
		if ch == '"' {
			if i+2 < len(text) && text[i+1] == '"' && text[i+2] == '"' {
				inRawString = true
				i += 2
				continue
			}
			inString = true
			continue
		}
		if ch == '\'' {
			inChar = true
			continue
		}

		switch ch {
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		}
	}
	return parenDepth > 0 || bracketDepth > 0 || braceDepth > 0 || lastNonWs == '{' || lastNonWs == '(' || lastNonWs == '['
}

func (c *Compiler) parseWithANTLR(path string, source []byte) parseSourceOutcome {
	antlrSource := source
	if shouldNormalizeKotlinSourceForANTLR(source) {
		antlrSource = normalizeKotlinSourceForANTLR(source)
	}

	return c.parseWithConfiguredTimeout(path, func() parseSourceOutcome {
		return c.parseBestRuleVariant(path, antlrSource)
	})
}

func (c *Compiler) parseBestRuleVariant(path string, source []byte) parseSourceOutcome {
	fileOutcome := c.parseWithRuleVariants(path, source, false)
	if !isKotlinScriptPath(path) || len(fileOutcome.diagnostics) == 0 {
		return fileOutcome
	}
	scriptOutcome := c.parseWithRuleVariants(path, source, true)
	return choosePreferredParseOutcome(fileOutcome, scriptOutcome)
}

func choosePreferredParseOutcome(primary parseSourceOutcome, alternate parseSourceOutcome) parseSourceOutcome {
	// Prefer the parse with fewer diagnostics to avoid regressing on grammar edge-cases.
	if len(alternate.diagnostics) < len(primary.diagnostics) {
		return alternate
	}
	return primary
}

func (c *Compiler) parseWithRuleVariants(path string, source []byte, scriptMode bool) parseSourceOutcome {
	best := c.parseWithRule(path, source, scriptMode)
	tryCandidate := func(candidateSource []byte) bool {
		if len(best.diagnostics) == 0 {
			return true
		}
		if bytes.Equal(candidateSource, source) {
			return false
		}
		candidate := c.parseWithRule(path, candidateSource, scriptMode)
		if len(candidate.diagnostics) < len(best.diagnostics) {
			best = candidate
		}
		return len(best.diagnostics) == 0
	}

	for _, candidateSource := range buildRuleVariantCandidates(source) {
		if tryCandidate(candidateSource) {
			return best
		}
	}

	return best
}

func (c *Compiler) parseWithRule(path string, source []byte, scriptMode bool) parseSourceOutcome {
	listener := newSyntaxErrorListener(path, c.config.MaxErrorsPerFile)
	var tree antlr.ParseTree

	parseErr := safeParse(func() {
		input := antlr.NewInputStream(string(source))
		lexer := kotlinparser.NewKotlinLexer(input)
		lexer.RemoveErrorListeners()
		lexer.AddErrorListener(listener)

		tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		parser := kotlinparser.NewKotlinParser(tokens)
		parser.RemoveErrorListeners()
		parser.AddErrorListener(listener)
		parser.BuildParseTrees = true
		if scriptMode {
			tree = parser.Script()
		} else {
			tree = parser.KotlinFile()
		}
	})

	if parseErr != nil {
		listener.addPanic(parseErr)
	}
	return parseSourceOutcome{
		tree:        tree,
		diagnostics: listener.Diagnostics(),
	}
}

func isKotlinScriptPath(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".kts")
}

func normalizeKnownTrailingLambdaCalls(source []byte) []byte {
	if len(source) == 0 {
		return source
	}

	text := string(source)
	if !strings.Contains(text, "{") {
		return source
	}

	out := make([]byte, 0, len(text)+16)
	changed := false
	braceTransformStack := make([]bool, 0, 32)

	for i := 0; i < len(text); {
		ch := text[i]
		next := byte(0)
		if i+1 < len(text) {
			next = text[i+1]
		}

		// Keep comments/literals untouched while keeping brace stack in sync.
		if ch == '/' && next == '/' {
			start := i
			i += 2
			for i < len(text) && text[i] != '\n' {
				i++
			}
			out = append(out, text[start:i]...)
			continue
		}
		if ch == '/' && next == '*' {
			start := i
			i += 2
			for i+1 < len(text) {
				if text[i] == '*' && text[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			if i > len(text) {
				i = len(text)
			}
			out = append(out, text[start:i]...)
			continue
		}
		if ch == '"' {
			if i+2 < len(text) && text[i+1] == '"' && text[i+2] == '"' {
				start := i
				i += 3
				for i+2 < len(text) {
					if text[i] == '"' && text[i+1] == '"' && text[i+2] == '"' {
						i += 3
						break
					}
					i++
				}
				if i > len(text) {
					i = len(text)
				}
				out = append(out, text[start:i]...)
				continue
			}
			start := i
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '"' {
					i++
					break
				}
				i++
			}
			if i > len(text) {
				i = len(text)
			}
			out = append(out, text[start:i]...)
			continue
		}
		if ch == '\'' {
			start := i
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '\'' {
					i++
					break
				}
				i++
			}
			if i > len(text) {
				i = len(text)
			}
			out = append(out, text[start:i]...)
			continue
		}

		if ch == '{' {
			switch {
			case hasDotCallBeforeBrace(text, i, "loop"), hasDotCallBeforeBrace(text, i, "let"):
				trimTrailingInlineSpaces(&out)
				out = append(out, '(', '{')
				braceTransformStack = append(braceTransformStack, true)
				changed = true
				i++
				continue
			case hasFoldCallBeforeBrace(text, i):
				trimTrailingInlineSpaces(&out)
				if len(out) > 0 && out[len(out)-1] == ')' {
					out = out[:len(out)-1]
					out = append(out, ',', ' ', '{')
					braceTransformStack = append(braceTransformStack, true)
					changed = true
					i++
					continue
				}
			}

			out = append(out, '{')
			braceTransformStack = append(braceTransformStack, false)
			i++
			continue
		}

		if ch == '}' {
			out = append(out, '}')
			needsCloseParen := false
			if n := len(braceTransformStack); n > 0 {
				needsCloseParen = braceTransformStack[n-1]
				braceTransformStack = braceTransformStack[:n-1]
			}
			if needsCloseParen {
				out = append(out, ')')
			}
			i++
			continue
		}

		out = append(out, ch)
		i++
	}

	if !changed {
		return source
	}
	return out
}

func normalizeObjectLiteralExpressions(source []byte) []byte {
	if len(source) == 0 || !bytes.Contains(source, []byte("object")) {
		return source
	}

	text := string(source)
	out := make([]byte, 0, len(text))
	changed := false

	for i := 0; i < len(text); {
		ch := text[i]
		next := byte(0)
		if i+1 < len(text) {
			next = text[i+1]
		}

		if ch == '/' && next == '/' {
			start := i
			i += 2
			for i < len(text) && text[i] != '\n' {
				i++
			}
			out = append(out, text[start:i]...)
			continue
		}
		if ch == '/' && next == '*' {
			start := i
			i += 2
			for i+1 < len(text) {
				if text[i] == '*' && text[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			if i > len(text) {
				i = len(text)
			}
			out = append(out, text[start:i]...)
			continue
		}
		if ch == '"' {
			start := i
			if i+2 < len(text) && text[i+1] == '"' && text[i+2] == '"' {
				i += 3
				for i+2 < len(text) {
					if text[i] == '"' && text[i+1] == '"' && text[i+2] == '"' {
						i += 3
						break
					}
					i++
				}
			} else {
				i++
				for i < len(text) {
					if text[i] == '\\' {
						i += 2
						continue
					}
					if text[i] == '"' {
						i++
						break
					}
					i++
				}
			}
			if i > len(text) {
				i = len(text)
			}
			out = append(out, text[start:i]...)
			continue
		}
		if ch == '\'' {
			start := i
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '\'' {
					i++
					break
				}
				i++
			}
			if i > len(text) {
				i = len(text)
			}
			out = append(out, text[start:i]...)
			continue
		}

		if i+6 <= len(text) && text[i:i+6] == "object" {
			prevBoundary := i == 0 || !isIdentifierByte(text[i-1])
			nextBoundary := i+6 == len(text) || !isIdentifierByte(text[i+6])
			if prevBoundary && nextBoundary {
				prevWord := previousIdentifierBefore(text, i)
				if prevWord != "companion" && !isDeclarationKeyword(prevWord) {
					j := i + 6
					for j < len(text) && isInlineOrNewlineWhitespace(text[j]) {
						j++
					}
					if j < len(text) && isIdentifierByte(text[j]) {
						out = append(out, text[i])
						i++
						continue
					}
					bracePos := j
					for bracePos < len(text) && text[bracePos] != '{' && text[bracePos] != '\n' && text[bracePos] != ';' {
						bracePos++
					}
					if bracePos < len(text) && text[bracePos] == '{' {
						end := findMatchingCloseBraceSimple(text, bracePos)
						if end > bracePos {
							out = append(out, "null"...)
							i = end + 1
							changed = true
							continue
						}
					}
				}
			}
		}

		out = append(out, ch)
		i++
	}

	if !changed {
		return source
	}
	return out
}

func hasDotCallBeforeBrace(text string, bracePos int, callName string) bool {
	j := bracePos - 1
	for j >= 0 && isInlineWhitespace(text[j]) {
		j--
	}
	if j < 0 || !isIdentifierByte(text[j]) {
		return false
	}

	end := j
	for j >= 0 && isIdentifierByte(text[j]) {
		j--
	}
	start := j + 1
	if text[start:end+1] != callName {
		return false
	}

	for j >= 0 && isInlineWhitespace(text[j]) {
		j--
	}
	return j >= 0 && text[j] == '.'
}

func hasFoldCallBeforeBrace(text string, bracePos int) bool {
	j := bracePos - 1
	for j >= 0 && isInlineWhitespace(text[j]) {
		j--
	}
	if j < 0 || text[j] != ')' {
		return false
	}

	open := findMatchingOpenParenSimple(text, j)
	if open < 0 {
		return false
	}

	k := open - 1
	for k >= 0 && isInlineWhitespace(text[k]) {
		k--
	}
	if k < 0 || !isIdentifierByte(text[k]) {
		return false
	}
	end := k
	for k >= 0 && isIdentifierByte(text[k]) {
		k--
	}
	start := k + 1
	if text[start:end+1] != "fold" {
		return false
	}

	for k >= 0 && isInlineWhitespace(text[k]) {
		k--
	}
	return k >= 0 && text[k] == '.'
}

func normalizeGenericTrailingLambdaCalls(source []byte) []byte {
	return normalizeGenericTrailingLambdaCallsWithOptions(source, genericTrailingOptions{
		rewriteRunTestBare:       true,
		rewriteRunTestSimpleOnly: false,
	})
}

func normalizeGenericTrailingLambdaCallsSimpleRunTest(source []byte) []byte {
	return normalizeGenericTrailingLambdaCallsWithOptions(source, genericTrailingOptions{
		rewriteRunTestBare:       true,
		rewriteRunTestSimpleOnly: true,
	})
}

func normalizeGenericTrailingLambdaCallsWithoutRunTest(source []byte) []byte {
	return normalizeGenericTrailingLambdaCallsWithOptions(source, genericTrailingOptions{
		rewriteRunTestBare:       false,
		rewriteRunTestSimpleOnly: false,
	})
}

type genericTrailingOptions struct {
	rewriteRunTestBare       bool
	rewriteRunTestSimpleOnly bool
}

func normalizeGenericTrailingLambdaCallsWithOptions(source []byte, opts genericTrailingOptions) []byte {
	if len(source) == 0 {
		return source
	}

	text := string(source)
	if !strings.Contains(text, "{") {
		return source
	}

	out := make([]byte, 0, len(text)+16)
	changed := false
	braceTransformStack := make([]bool, 0, 32)

	for i := 0; i < len(text); {
		ch := text[i]
		next := byte(0)
		if i+1 < len(text) {
			next = text[i+1]
		}

		// Keep comments/literals untouched while keeping brace stack in sync.
		if ch == '/' && next == '/' {
			start := i
			i += 2
			for i < len(text) && text[i] != '\n' {
				i++
			}
			out = append(out, text[start:i]...)
			continue
		}
		if ch == '/' && next == '*' {
			start := i
			i += 2
			for i+1 < len(text) {
				if text[i] == '*' && text[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			if i > len(text) {
				i = len(text)
			}
			out = append(out, text[start:i]...)
			continue
		}
		if ch == '"' {
			if i+2 < len(text) && text[i+1] == '"' && text[i+2] == '"' {
				start := i
				i += 3
				for i+2 < len(text) {
					if text[i] == '"' && text[i+1] == '"' && text[i+2] == '"' {
						i += 3
						break
					}
					i++
				}
				if i > len(text) {
					i = len(text)
				}
				out = append(out, text[start:i]...)
				continue
			}
			start := i
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '"' {
					i++
					break
				}
				i++
			}
			if i > len(text) {
				i = len(text)
			}
			out = append(out, text[start:i]...)
			continue
		}
		if ch == '\'' {
			start := i
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '\'' {
					i++
					break
				}
				i++
			}
			if i > len(text) {
				i = len(text)
			}
			out = append(out, text[start:i]...)
			continue
		}

		if ch == '{' {
			if argsEmpty, ok := hasGenericCallableParenBeforeBrace(text, i); ok {
				trimTrailingInlineSpaces(&out)
				if len(out) > 0 && out[len(out)-1] == ')' {
					out = out[:len(out)-1]
					if !argsEmpty {
						out = append(out, ',', ' ')
					}
					out = append(out, '{')
					braceTransformStack = append(braceTransformStack, true)
					changed = true
					i++
					continue
				}
			}

			if callee, ok := knownBareTrailingLambdaCallee(text, i); ok {
				if callee == "runTest" && !opts.rewriteRunTestBare {
					out = append(out, '{')
					braceTransformStack = append(braceTransformStack, false)
					i++
					continue
				}
				if shouldRewriteBareTrailingLambdaCall(text, i, callee, opts) {
					trimTrailingInlineSpaces(&out)
					out = append(out, '(', '{')
					braceTransformStack = append(braceTransformStack, true)
					changed = true
					i++
					continue
				}
			}

			out = append(out, '{')
			braceTransformStack = append(braceTransformStack, false)
			i++
			continue
		}

		if ch == '}' {
			out = append(out, '}')
			needsCloseParen := false
			if n := len(braceTransformStack); n > 0 {
				needsCloseParen = braceTransformStack[n-1]
				braceTransformStack = braceTransformStack[:n-1]
			}
			if needsCloseParen {
				out = append(out, ')')
			}
			i++
			continue
		}

		out = append(out, ch)
		i++
	}

	if !changed {
		return source
	}
	return out
}

func normalizeDelegationConstructorsAndUnsignedLiterals(source []byte) []byte {
	if len(source) == 0 {
		return source
	}

	text := string(source)
	normalized := emptyDelegationConstructorCallPattern.ReplaceAllStringFunc(text, func(match string) string {
		trimmed := strings.TrimSuffix(match, "()")
		last := trailingIdentifier(trimmed)
		if last == "this" || last == "super" {
			return match
		}
		return trimmed
	})
	normalized = unsignedIntegerLiteralPattern.ReplaceAllString(normalized, "$1")

	if normalized == text {
		return source
	}
	return []byte(normalized)
}

func hasGenericCallableParenBeforeBrace(text string, bracePos int) (bool, bool) {
	j := bracePos - 1
	for j >= 0 && isInlineWhitespace(text[j]) {
		j--
	}
	if j < 0 || text[j] != ')' {
		return false, false
	}

	open := findMatchingOpenParenSimple(text, j)
	if open < 0 {
		return false, false
	}

	k := open - 1
	for k >= 0 && isInlineOrNewlineWhitespace(text[k]) {
		k--
	}
	if k < 0 || !isIdentifierByte(text[k]) {
		return false, false
	}

	end := k
	for k >= 0 && isIdentifierByte(text[k]) {
		k--
	}
	start := k + 1
	prevNonSpaceChar := byte(0)
	for p := start - 1; p >= 0; p-- {
		if isInlineOrNewlineWhitespace(text[p]) {
			continue
		}
		prevNonSpaceChar = text[p]
		break
	}
	// Do not rewrite class/object delegation headers like `class X : Base() { ... }`.
	if prevNonSpaceChar == ':' {
		return false, false
	}
	callee := text[start : end+1]
	if isBlockedGenericParenTrailingLambdaCallee(callee) {
		return false, false
	}
	if isControlKeyword(callee) {
		return false, false
	}
	prevWord := previousIdentifierBefore(text, start)
	if isDeclarationKeyword(prevWord) {
		return false, false
	}

	argsEmpty := true
	for idx := open + 1; idx < j; idx++ {
		if !isInlineOrNewlineWhitespace(text[idx]) {
			argsEmpty = false
			break
		}
	}
	return argsEmpty, true
}

func hasKnownBareTrailingLambdaCall(text string, bracePos int) bool {
	_, ok := knownBareTrailingLambdaCallee(text, bracePos)
	return ok
}

func knownBareTrailingLambdaCallee(text string, bracePos int) (string, bool) {
	j := bracePos - 1
	for j >= 0 && isInlineWhitespace(text[j]) {
		j--
	}
	if j < 0 || !isIdentifierByte(text[j]) {
		return "", false
	}

	end := j
	for j >= 0 && isIdentifierByte(text[j]) {
		j--
	}
	start := j + 1
	name := text[start : end+1]
	if !isKnownBareTrailingLambdaCallee(name) {
		return "", false
	}
	prevWord := previousIdentifierBefore(text, start)
	if isDeclarationKeyword(prevWord) {
		return "", false
	}
	return name, true
}

func shouldRewriteBareTrailingLambdaCall(text string, bracePos int, callee string, opts genericTrailingOptions) bool {
	switch callee {
	case "runTest":
		end := findMatchingCloseBraceSimple(text, bracePos)
		if end <= bracePos {
			return true
		}
		return shouldRewriteRunTestLambdaBody(text[bracePos+1:end], opts)
	default:
		return true
	}
}

func shouldRewriteRunTestLambdaBody(body string, opts genericTrailingOptions) bool {
	if localFunctionDeclarationPattern.MatchString(body) {
		return false
	}
	if opts.rewriteRunTestSimpleOnly && runTestComplexBodyPattern.MatchString(body) {
		return false
	}
	return true
}

func previousIdentifierBefore(text string, index int) string {
	j := index - 1
	for j >= 0 && isInlineOrNewlineWhitespace(text[j]) {
		j--
	}
	if j < 0 || !isIdentifierByte(text[j]) {
		return ""
	}
	end := j
	for j >= 0 && isIdentifierByte(text[j]) {
		j--
	}
	start := j + 1
	return text[start : end+1]
}

func isControlKeyword(name string) bool {
	switch name {
	case "if", "for", "while", "when", "catch":
		return true
	default:
		return false
	}
}

func isDeclarationKeyword(name string) bool {
	switch name {
	case "fun", "class", "interface", "object", "typealias", "val", "var", "constructor":
		return true
	default:
		return false
	}
}

func isBlockedGenericParenTrailingLambdaCallee(name string) bool {
	switch name {
	case "runBlocking", "launch", "async":
		return true
	default:
		return false
	}
}

func isKnownBareTrailingLambdaCallee(name string) bool {
	switch name {
	case "runTest", "launch", "thread", "autoreleasepool", "async", "SerializersModule":
		return true
	default:
		return false
	}
}

func trailingIdentifier(text string) string {
	if text == "" {
		return ""
	}
	i := len(text) - 1
	for i >= 0 && !isIdentifierByte(text[i]) {
		i--
	}
	if i < 0 {
		return ""
	}
	end := i
	for i >= 0 && isIdentifierByte(text[i]) {
		i--
	}
	return text[i+1 : end+1]
}

func findMatchingOpenParenSimple(text string, closePos int) int {
	depth := 0
	for i := closePos; i >= 0; i-- {
		switch text[i] {
		case ')':
			depth++
		case '(':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findMatchingCloseBraceSimple(text string, openPos int) int {
	if openPos < 0 || openPos >= len(text) || text[openPos] != '{' {
		return -1
	}

	depth := 0
	for i := openPos; i < len(text); {
		ch := text[i]
		next := byte(0)
		if i+1 < len(text) {
			next = text[i+1]
		}

		if ch == '/' && next == '/' {
			i += 2
			for i < len(text) && text[i] != '\n' {
				i++
			}
			continue
		}
		if ch == '/' && next == '*' {
			i += 2
			for i+1 < len(text) {
				if text[i] == '*' && text[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}
		if ch == '"' {
			if i+2 < len(text) && text[i+1] == '"' && text[i+2] == '"' {
				i += 3
				for i+2 < len(text) {
					if text[i] == '"' && text[i+1] == '"' && text[i+2] == '"' {
						i += 3
						break
					}
					i++
				}
				continue
			}
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '"' {
					i++
					break
				}
				i++
			}
			continue
		}
		if ch == '\'' {
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '\'' {
					i++
					break
				}
				i++
			}
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

func trimTrailingInlineSpaces(buf *[]byte) {
	for len(*buf) > 0 {
		last := (*buf)[len(*buf)-1]
		if !isInlineWhitespace(last) {
			return
		}
		*buf = (*buf)[:len(*buf)-1]
	}
}

func isInlineWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r'
}

func isInlineOrNewlineWhitespace(ch byte) bool {
	return isInlineWhitespace(ch) || ch == '\n'
}

func isIdentifierByte(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}

func isAnnotationNameStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		ch == '_'
}

func skipStringTemplateExpression(text string, i int) int {
	depth := 1
	for i < len(text) && depth > 0 {
		switch text[i] {
		case '{':
			depth++
			i++
		case '}':
			depth--
			i++
		case '"':
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '"' {
					i++
					break
				}
				i++
			}
		case '\'':
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '\'' {
					i++
					break
				}
				i++
			}
		default:
			i++
		}
	}
	return i
}

func stripAnnotationArguments(text string) string {
	if text == "" || !strings.Contains(text, "@") {
		return text
	}

	var out strings.Builder
	out.Grow(len(text))

	for i := 0; i < len(text); {
		ch := text[i]
		if ch != '@' || !isLikelyAnnotationStart(text, i) {
			out.WriteByte(ch)
			i++
			continue
		}

		start := i
		i++ // consume '@'
		for i < len(text) && isAnnotationNameChar(text[i]) {
			i++
		}
		for i < len(text) && (text[i] == ' ' || text[i] == '\t' || text[i] == '\r') {
			i++
		}

		if i < len(text) && text[i] == '(' {
			out.WriteString(text[start:i])
			out.WriteString("()")
			i = skipParenthesized(text, i)
			continue
		}

		out.WriteString(text[start:i])
	}

	return out.String()
}

func isLikelyAnnotationStart(text string, at int) bool {
	if at <= 0 || at >= len(text) {
		return at == 0
	}

	prev := text[at-1]
	switch prev {
	case ' ', '\t', '\r', '\n', '(', ',', ';', '{', '}', '<':
		return true
	}
	return false
}

func normalizeParenthesizedLambdaBodies(source []byte) []byte {
	if len(source) == 0 {
		return source
	}
	text := string(source)
	if !strings.Contains(text, "{") || !strings.Contains(text, "\n") {
		return source
	}

	out := make([]byte, 0, len(text)+16)
	changed := false
	parenDepth := 0

	for i := 0; i < len(text); {
		ch := text[i]
		next := byte(0)
		if i+1 < len(text) {
			next = text[i+1]
		}

		// Keep comments/literals untouched.
		if ch == '/' && next == '/' {
			start := i
			i += 2
			for i < len(text) && text[i] != '\n' {
				i++
			}
			out = append(out, text[start:i]...)
			continue
		}
		if ch == '/' && next == '*' {
			start := i
			i += 2
			for i+1 < len(text) {
				if text[i] == '*' && text[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			if i > len(text) {
				i = len(text)
			}
			out = append(out, text[start:i]...)
			continue
		}
		if ch == '"' {
			// Triple quote.
			if i+2 < len(text) && text[i+1] == '"' && text[i+2] == '"' {
				start := i
				i += 3
				for i+2 < len(text) {
					if text[i] == '"' && text[i+1] == '"' && text[i+2] == '"' {
						i += 3
						break
					}
					i++
				}
				if i > len(text) {
					i = len(text)
				}
				out = append(out, text[start:i]...)
				continue
			}
			start := i
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '"' {
					i++
					break
				}
				i++
			}
			if i > len(text) {
				i = len(text)
			}
			out = append(out, text[start:i]...)
			continue
		}
		if ch == '\'' {
			start := i
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '\'' {
					i++
					break
				}
				i++
			}
			if i > len(text) {
				i = len(text)
			}
			out = append(out, text[start:i]...)
			continue
		}

		if ch == '(' {
			parenDepth++
			out = append(out, ch)
			i++
			continue
		}
		if ch == ')' {
			if parenDepth > 0 {
				parenDepth--
			}
			out = append(out, ch)
			i++
			continue
		}

		if ch == '{' && parenDepth > 0 && isLikelyLambdaStart(text, i) {
			end := findMatchingBrace(text, i)
			if end <= i {
				out = append(out, ch)
				i++
				continue
			}
			block := text[i : end+1]
			rewritten := rewriteLambdaBlockNewlines(block)
			if rewritten != block {
				changed = true
			}
			out = append(out, rewritten...)
			i = end + 1
			continue
		}

		out = append(out, ch)
		i++
	}

	if !changed {
		return source
	}
	return out
}

func isLikelyLambdaStart(text string, bracePos int) bool {
	j := bracePos - 1
	for j >= 0 {
		ch := text[j]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			j--
			continue
		}
		return ch == '(' || ch == ',' || ch == '='
	}
	return false
}

func findMatchingBrace(text string, start int) int {
	depth := 0
	for i := start; i < len(text); i++ {
		ch := text[i]
		next := byte(0)
		if i+1 < len(text) {
			next = text[i+1]
		}
		if ch == '/' && next == '/' {
			i += 2
			for i < len(text) && text[i] != '\n' {
				i++
			}
			continue
		}
		if ch == '/' && next == '*' {
			i += 2
			for i+1 < len(text) {
				if text[i] == '*' && text[i+1] == '/' {
					i++
					break
				}
				i++
			}
			continue
		}
		if ch == '"' {
			if i+2 < len(text) && text[i+1] == '"' && text[i+2] == '"' {
				i += 3
				for i+2 < len(text) {
					if text[i] == '"' && text[i+1] == '"' && text[i+2] == '"' {
						i += 2
						break
					}
					i++
				}
				continue
			}
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if i < len(text) && text[i] == '"' {
					break
				}
				i++
			}
			continue
		}
		if ch == '\'' {
			i++
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if i < len(text) && text[i] == '\'' {
					break
				}
				i++
			}
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func rewriteLambdaBlockNewlines(block string) string {
	// block is assumed to include leading '{' and trailing matching '}'.
	if len(block) < 2 {
		return block
	}
	bodyStart := 1
	depthParen := 0
	depthSquare := 0
	depthBrace := 1
	arrowPos := -1

	for i := 1; i < len(block)-1; i++ {
		ch := block[i]
		next := byte(0)
		if i+1 < len(block) {
			next = block[i+1]
		}

		if ch == '/' && next == '/' {
			i += 2
			for i < len(block)-1 && block[i] != '\n' {
				i++
			}
			continue
		}
		if ch == '/' && next == '*' {
			i += 2
			for i+1 < len(block)-1 {
				if block[i] == '*' && block[i+1] == '/' {
					i++
					break
				}
				i++
			}
			continue
		}
		if ch == '"' {
			if i+2 < len(block)-1 && block[i+1] == '"' && block[i+2] == '"' {
				i += 3
				for i+2 < len(block)-1 {
					if block[i] == '"' && block[i+1] == '"' && block[i+2] == '"' {
						i += 2
						break
					}
					i++
				}
				continue
			}
			i++
			for i < len(block)-1 {
				if block[i] == '\\' {
					i += 2
					continue
				}
				if i < len(block)-1 && block[i] == '"' {
					break
				}
				i++
			}
			continue
		}
		if ch == '\'' {
			i++
			for i < len(block)-1 {
				if block[i] == '\\' {
					i += 2
					continue
				}
				if i < len(block)-1 && block[i] == '\'' {
					break
				}
				i++
			}
			continue
		}

		switch ch {
		case '(':
			depthParen++
		case ')':
			if depthParen > 0 {
				depthParen--
			}
		case '[':
			depthSquare++
		case ']':
			if depthSquare > 0 {
				depthSquare--
			}
		case '{':
			depthBrace++
		case '}':
			depthBrace--
		}

		if arrowPos < 0 && depthParen == 0 && depthSquare == 0 && depthBrace == 1 && ch == '-' && next == '>' {
			arrowPos = i
			bodyStart = i + 2
			break
		}
	}

	rewritten := make([]byte, 0, len(block)+8)
	changed := false
	inLineComment := false
	inBlockComment := false
	inSingleQuote := false
	inDoubleQuote := false
	inTripleQuote := false

	for i := 0; i < len(block); i++ {
		ch := block[i]
		next := byte(0)
		if i+1 < len(block) {
			next = block[i+1]
		}

		if inLineComment {
			rewritten = append(rewritten, ch)
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			rewritten = append(rewritten, ch)
			if ch == '*' && next == '/' {
				rewritten = append(rewritten, next)
				i++
				inBlockComment = false
			}
			continue
		}
		if inSingleQuote {
			rewritten = append(rewritten, ch)
			if ch == '\\' && i+1 < len(block) {
				rewritten = append(rewritten, next)
				i++
				continue
			}
			if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}
		if inDoubleQuote {
			rewritten = append(rewritten, ch)
			if inTripleQuote {
				if ch == '"' && i+2 < len(block) && block[i+1] == '"' && block[i+2] == '"' {
					rewritten = append(rewritten, block[i+1], block[i+2])
					i += 2
					inDoubleQuote = false
					inTripleQuote = false
				}
				continue
			}
			if ch == '\\' && i+1 < len(block) {
				rewritten = append(rewritten, next)
				i++
				continue
			}
			if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if ch == '/' && next == '/' {
			rewritten = append(rewritten, ch, next)
			i++
			inLineComment = true
			continue
		}
		if ch == '/' && next == '*' {
			rewritten = append(rewritten, ch, next)
			i++
			inBlockComment = true
			continue
		}
		if ch == '\'' {
			rewritten = append(rewritten, ch)
			inSingleQuote = true
			continue
		}
		if ch == '"' {
			rewritten = append(rewritten, ch)
			if i+2 < len(block) && block[i+1] == '"' && block[i+2] == '"' {
				rewritten = append(rewritten, block[i+1], block[i+2])
				i += 2
				inDoubleQuote = true
				inTripleQuote = true
				continue
			}
			inDoubleQuote = true
			inTripleQuote = false
			continue
		}

		if i > bodyStart && ch == '\n' {
			rewritten = append(rewritten, ';')
			changed = true
			continue
		}
		if i > bodyStart && ch == '\r' {
			changed = true
			continue
		}

		rewritten = append(rewritten, ch)
	}
	if !changed {
		return block
	}
	return string(rewritten)
}

func isAnnotationNameChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' || ch == '.' || ch == ':' || ch == '`'
}

func skipParenthesized(text string, start int) int {
	if start >= len(text) || text[start] != '(' {
		return start
	}

	depth := 0
	inSingleQuote := false
	inDoubleQuote := false
	inLineComment := false
	inBlockComment := false

	i := start
	for i < len(text) {
		ch := text[i]
		next := byte(0)
		if i+1 < len(text) {
			next = text[i+1]
		}

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
			}
			i++
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i += 2
				continue
			}
			i++
			continue
		}
		if inSingleQuote {
			if ch == '\\' && i+1 < len(text) {
				i += 2
				continue
			}
			if ch == '\'' {
				inSingleQuote = false
			}
			i++
			continue
		}
		if inDoubleQuote {
			if ch == '\\' && i+1 < len(text) {
				i += 2
				continue
			}
			if ch == '"' {
				inDoubleQuote = false
			}
			i++
			continue
		}

		if ch == '/' && next == '/' {
			inLineComment = true
			i += 2
			continue
		}
		if ch == '/' && next == '*' {
			inBlockComment = true
			i += 2
			continue
		}
		if ch == '\'' {
			inSingleQuote = true
			i++
			continue
		}
		if ch == '"' {
			inDoubleQuote = true
			i++
			continue
		}

		if ch == '(' {
			depth++
			i++
			continue
		}
		if ch == ')' {
			depth--
			i++
			if depth == 0 {
				return i
			}
			continue
		}
		i++
	}

	return len(text)
}

func (c *Compiler) ParseRepository(root string) (RepositoryResult, error) {
	startedAt := time.Now()
	rootPath, err := resolveRepositoryRoot(root)
	if err != nil {
		return RepositoryResult{}, err
	}
	if c.config.ParseBackend == ParseBackendEmbeddable {
		return c.parseRepositoryWithEmbeddable(rootPath)
	}

	files, err := c.collectRepositoryFiles(rootPath)
	if err != nil {
		return RepositoryResult{}, err
	}
	return c.buildRepositoryResult(rootPath, files, startedAt), nil
}

func newRepositoryResult(root string, totalFiles int) RepositoryResult {
	return RepositoryResult{
		Root:       root,
		TotalFiles: totalFiles,
		Files:      make([]FileUnit, 0, totalFiles),
	}
}

func (c *Compiler) collectRepositoryUnits(files []string) []FileUnit {
	jobs := make(chan string)
	outputs := make(chan FileUnit)

	var workers sync.WaitGroup
	for i := 0; i < c.repositoryWorkerCount(len(files)); i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for path := range jobs {
				outputs <- c.parseRepositoryUnit(path)
			}
		}()
	}

	go func() {
		for _, path := range files {
			jobs <- path
		}
		close(jobs)
		workers.Wait()
		close(outputs)
	}()

	units := make([]FileUnit, 0, len(files))
	for unit := range outputs {
		units = append(units, unit)
	}
	return units
}

func (c *Compiler) repositoryWorkerCount(fileCount int) int {
	if fileCount <= 0 {
		return 0
	}
	if c.config.Workers > fileCount {
		return fileCount
	}
	return c.config.Workers
}

func (c *Compiler) parseRepositoryUnit(path string) FileUnit {
	unit, readErr := c.ParseFile(path)
	if readErr == nil {
		return unit
	}
	return FileUnit{
		Path:   path,
		Parsed: false,
		Diagnostics: []Diagnostic{{
			Path:     path,
			Line:     0,
			Column:   0,
			Message:  readErr.Error(),
			Severity: SeverityError,
		}},
	}
}

func appendRepositoryUnit(result *RepositoryResult, unit FileUnit) {
	result.Files = append(result.Files, unit)
	if unit.Parsed {
		result.ParsedFiles++
		return
	}
	result.FailedFiles++
}

func sortRepositoryUnits(units []FileUnit) {
	sort.Slice(units, func(i, j int) bool {
		return units[i].Path < units[j].Path
	})
}

func collectKotlinFiles(root string, includeKTS bool, includeBuildScripts bool) ([]string, error) {
	files := make([]string, 0, 1024)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "build" || name == "out" || name == "target" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".kt" {
			files = append(files, path)
			return nil
		}
		if ext == ".kts" {
			lowerBase := strings.ToLower(filepath.Base(path))
			isBuildScript := lowerBase == "build.gradle.kts" || lowerBase == "settings.gradle.kts"
			if isBuildScript {
				if includeBuildScripts {
					files = append(files, path)
				}
				return nil
			}
			if includeKTS {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk kotlin files: %w", err)
	}
	return files, nil
}

func extractHeader(source []byte) (string, []string) {
	lines := strings.Split(string(source), "\n")
	pkg := ""
	set := make(map[string]struct{}, 8)
	inBlockComment := false

	for _, line := range lines {
		cleanLine, blockCommentNow := removeComments(line, inBlockComment)
		inBlockComment = blockCommentNow

		trimmed := strings.TrimSpace(cleanLine)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "#!") || strings.HasPrefix(trimmed, "@file:") {
			continue
		}

		if matches := packageLinePattern.FindStringSubmatch(trimmed); len(matches) == 2 {
			if pkg == "" {
				pkg = matches[1]
			}
			continue
		}

		if matches := importLinePattern.FindStringSubmatch(trimmed); len(matches) == 2 {
			set[matches[1]] = struct{}{}
			continue
		}

		if strings.HasPrefix(trimmed, "@") {
			continue
		}

		break
	}

	if len(set) == 0 {
		return pkg, nil
	}

	imports := make([]string, 0, len(set))
	for imp := range set {
		imports = append(imports, imp)
	}
	sort.Strings(imports)
	return pkg, imports
}

func extractTopLevelDeclarations(source []byte) []TopLevelDeclaration {
	lines := strings.Split(string(source), "\n")
	declarations := make([]TopLevelDeclaration, 0, 8)
	braceDepth := 0
	inBlockComment := false

	for idx, line := range lines {
		cleanLine, blockCommentNow := removeComments(line, inBlockComment)
		inBlockComment = blockCommentNow

		if braceDepth == 0 {
			decl, ok := parseTopLevelDeclaration(strings.TrimSpace(cleanLine), idx+1)
			if ok {
				declarations = append(declarations, decl)
			}
		}

		braceDepth += countBraceDelta(cleanLine)
		if braceDepth < 0 {
			braceDepth = 0
		}
	}

	return declarations
}

func mergeUniqueStrings(left, right []string) []string {
	if len(left) == 0 && len(right) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(left)+len(right))
	for _, value := range left {
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	for _, value := range right {
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func mergeTopLevelDeclarations(left, right []TopLevelDeclaration) []TopLevelDeclaration {
	if len(left) == 0 && len(right) == 0 {
		return nil
	}
	if len(left) == 0 {
		out := make([]TopLevelDeclaration, len(right))
		copy(out, right)
		return out
	}
	if len(right) == 0 {
		return left
	}

	seen := make(map[string]struct{}, len(left)+len(right))
	for _, decl := range left {
		seen[string(decl.Kind)+"|"+decl.Name] = struct{}{}
	}
	out := make([]TopLevelDeclaration, 0, len(left)+len(right))
	out = append(out, left...)
	for _, decl := range right {
		key := string(decl.Kind) + "|" + decl.Name
		if _, exists := seen[key]; exists {
			continue
		}
		out = append(out, decl)
		seen[key] = struct{}{}
	}
	return out
}

func parseTopLevelDeclaration(line string, lineNo int) (TopLevelDeclaration, bool) {
	if line == "" {
		return TopLevelDeclaration{}, false
	}

	working := line
	for {
		next := annotationPattern.ReplaceAllString(working, "")
		if next == working {
			break
		}
		working = strings.TrimSpace(next)
	}

	modifiers := make([]string, 0, 4)
	for {
		matches := modifierPattern.FindStringSubmatch(working)
		if len(matches) != 2 {
			break
		}
		modifiers = append(modifiers, matches[1])
		working = strings.TrimSpace(strings.TrimPrefix(working, matches[0]))
	}

	if matches := classLikePattern.FindStringSubmatch(working); len(matches) == 3 {
		return TopLevelDeclaration{
			Kind:      mapClassLikeKind(matches[1]),
			Name:      matches[2],
			Line:      lineNo,
			Modifiers: modifiers,
		}, true
	}
	if matches := functionPattern.FindStringSubmatch(working); len(matches) == 2 {
		return TopLevelDeclaration{
			Kind:      DeclFunction,
			Name:      matches[1],
			Line:      lineNo,
			Modifiers: modifiers,
		}, true
	}
	if matches := propertyPattern.FindStringSubmatch(working); len(matches) == 3 {
		return TopLevelDeclaration{
			Kind:      DeclProperty,
			Name:      matches[2],
			Line:      lineNo,
			Modifiers: modifiers,
		}, true
	}
	return TopLevelDeclaration{}, false
}

func mapClassLikeKind(value string) DeclarationKind {
	switch value {
	case "class":
		return DeclClass
	case "interface":
		return DeclInterface
	case "object":
		return DeclObject
	case "typealias":
		return DeclTypeAlias
	default:
		return DeclClass
	}
}

func removeComments(line string, inBlock bool) (string, bool) {
	var b strings.Builder
	i := 0
	for i < len(line) {
		if inBlock {
			end := strings.Index(line[i:], "*/")
			if end < 0 {
				return b.String(), true
			}
			i += end + 2
			inBlock = false
			continue
		}

		if strings.HasPrefix(line[i:], "//") {
			break
		}
		if strings.HasPrefix(line[i:], "/*") {
			inBlock = true
			i += 2
			continue
		}

		b.WriteByte(line[i])
		i++
	}
	return b.String(), inBlock
}

func countBraceDelta(line string) int {
	delta := 0
	for _, ch := range line {
		switch ch {
		case '{':
			delta++
		case '}':
			delta--
		}
	}
	return delta
}

func safeParse(fn func()) (panicErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			panicErr = fmt.Errorf("parser panic: %v", recovered)
		}
	}()
	fn()
	return nil
}

type syntaxErrorListener struct {
	*antlr.DefaultErrorListener
	path        string
	maxErrors   int
	diagnostics []Diagnostic
}

func newSyntaxErrorListener(path string, maxErrors int) *syntaxErrorListener {
	if maxErrors <= 0 {
		maxErrors = 1
	}
	return &syntaxErrorListener{
		DefaultErrorListener: &antlr.DefaultErrorListener{},
		path:                 path,
		maxErrors:            maxErrors,
		diagnostics:          make([]Diagnostic, 0, maxErrors),
	}
}

func (l *syntaxErrorListener) SyntaxError(
	_ antlr.Recognizer,
	_ interface{},
	line int,
	column int,
	msg string,
	_ antlr.RecognitionException,
) {
	if len(l.diagnostics) >= l.maxErrors {
		return
	}
	l.diagnostics = append(l.diagnostics, Diagnostic{
		Path:     l.path,
		Line:     line,
		Column:   column,
		Message:  msg,
		Severity: SeverityError,
	})
}

func (l *syntaxErrorListener) addPanic(parseErr error) {
	if len(l.diagnostics) >= l.maxErrors {
		return
	}
	l.diagnostics = append(l.diagnostics, Diagnostic{
		Path:     l.path,
		Line:     0,
		Column:   0,
		Message:  parseErr.Error(),
		Severity: SeverityError,
	})
}

func (l *syntaxErrorListener) Diagnostics() []Diagnostic {
	if len(l.diagnostics) == 0 {
		return nil
	}
	out := make([]Diagnostic, len(l.diagnostics))
	copy(out, l.diagnostics)
	return out
}
