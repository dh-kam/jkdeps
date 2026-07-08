package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/antlr4-go/antlr/v4"
	ast "github.com/dh-kam/jkdeps/internal/ast"
	java20parser "github.com/dh-kam/jkdeps/internal/parsers/java20"
	java8parser "github.com/dh-kam/jkdeps/internal/parsers/java8"
	java9parser "github.com/dh-kam/jkdeps/internal/parsers/java9"
	javaorigparser "github.com/dh-kam/jkdeps/internal/parsers/javaorig"
	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

// Kotlin source normalization patterns for modern syntax support

// Platform-specific modifiers (expect/actual) - normalize to standard declaration
var kotlinPlatformModifierPattern = regexp.MustCompile(`\b(?:expect|actual)\s+((?:public|private|protected|internal|enum|sealed|annotation|data|inner|tailrec|operator|inline|infix|external|suspend|override|abstract|final|open|const|lateinit|vararg|noinline|crossinline|reified|class|interface|fun|object|val|var|typealias|constructor)\b)`)

// Value class (inline value class) - Kotlin 1.5+
var kotlinValueClassPattern = regexp.MustCompile(`\bvalue\s+(class\b)`)

// Fun interface (functional interface) - Kotlin 1.4+
var kotlinFunInterfacePattern = regexp.MustCompile(`\bfun\s+(interface\b)`)

// Context receivers - Kotlin 1.6.20+
var kotlinContextReceiverPattern = regexp.MustCompile(`(?m)^[ \t]*context\s*\([^)\n]*\)\s*`)

// Sealed interface - Kotlin 1.5+
var kotlinSealedInterfacePattern = regexp.MustCompile(`\bsealed\s+(interface\b)`)

// Labeled lambda expressions - Kotlin 1.3+
// Handles both "label@{" and "label@ {" syntax
var kotlinLambdaLabelPattern = regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*@\s*\{`)

// Function type cast with suspend - Kotlin 1.6+
var kotlinFunctionTypeCastPattern = regexp.MustCompile(`\s+as\s+(?:suspend\s+)?\([^)\n]*\)\s*->\s*[A-Za-z0-9_?.<>,\[\] ]+`)

// Nullable type cast in when expressions
var kotlinNullableTypeCastPattern = regexp.MustCompile(`\bas\s+([A-Za-z_][A-Za-z0-9_<>,\.]*)\?`)

// Extension function with star projection - Kotlin 1.5+
var kotlinExtensionStarReceiverPattern = regexp.MustCompile(`(?m)(\bfun[^\n]*?)<\*>\.`)

// Extension function with generic receiver type arguments
var kotlinExtensionGenericReceiverPattern = regexp.MustCompile(`(?m)(\bfun\b(?:\s*<[^>\n]+>)?\s+[A-Za-z_][A-Za-z0-9_?.]*)<[^>\n]+>\.`)

// Anonymous function assignment syntax - Kotlin 1.3+
var kotlinAnonymousFunctionPattern = regexp.MustCompile(`=\s*fun\s*\(([^)\n]*)\)\s*(?::\s*[^{}\n]+)?\s*\{`)

// String template expressions - normalize complex templates
var kotlinStringTemplatePattern = regexp.MustCompile(`\$\{[^{}\n]*\}`)

const fileErrorMessageLimit = 3

type JavaGrammar string

const (
	JavaGrammarDefault JavaGrammar = "java20"
	JavaGrammarOrig    JavaGrammar = "java"
	JavaGrammar7       JavaGrammar = "java7"
	JavaGrammar8       JavaGrammar = "java8"
	JavaGrammar9       JavaGrammar = "java9"
	JavaGrammar11      JavaGrammar = "java11"
	JavaGrammar20      JavaGrammar = "java20"
	JavaGrammar17      JavaGrammar = "java17"
	JavaGrammar21      JavaGrammar = "java21"
	JavaGrammar25      JavaGrammar = "java25"
)

func (g JavaGrammar) IsValid() bool {
	switch g {
	case JavaGrammarOrig, JavaGrammar7, JavaGrammar8, JavaGrammar9, JavaGrammar11, JavaGrammar17, JavaGrammar20, JavaGrammar21, JavaGrammar25:
		return true
	default:
		return false
	}
}

type ParseOptions struct {
	JavaGrammar   JavaGrammar
	Workers       int
	MaxErrorFiles int
	IncludeKTS    bool
	ParseTimeout  time.Duration
	GCPercent     int
}

type FileFailure struct {
	Path  string
	Error string
}

type Summary struct {
	Root        string
	JavaGrammar JavaGrammar
	Duration    time.Duration
	TotalFiles  int
	JavaFiles   int
	KotlinFiles int
	ParsedFiles int
	FailedFiles int
	Failures    []FileFailure
}

// Language type aliases for convenience
type sourceLanguage = ast.SourceLanguage

const (
	langJava   sourceLanguage = ast.LanguageJava
	langKotlin sourceLanguage = ast.LanguageKotlin
)

type parseJob struct {
	path string
	lang sourceLanguage
}

type parseResult struct {
	lang sourceLanguage
	path string
	err  error
}

func ParseRepository(root string, opts ParseOptions) (Summary, error) {
	startedAt := time.Now()
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return Summary{}, fmt.Errorf("resolve root path: %w", err)
	}

	if opts.JavaGrammar == "" {
		opts.JavaGrammar = JavaGrammarDefault
	}
	if !opts.JavaGrammar.IsValid() {
		return Summary{}, fmt.Errorf("unsupported java grammar: %q", opts.JavaGrammar)
	}
	if opts.Workers <= 0 {
		opts.Workers = runtime.NumCPU()
	}
	if opts.MaxErrorFiles <= 0 {
		opts.MaxErrorFiles = 20
	}
	if opts.GCPercent > 0 {
		prev := debug.SetGCPercent(opts.GCPercent)
		defer debug.SetGCPercent(prev)
	}

	jobs, javaCount, kotlinCount, err := collectSourceFiles(rootPath, opts.IncludeKTS)
	if err != nil {
		return Summary{}, err
	}

	summary := Summary{
		Root:        rootPath,
		JavaGrammar: opts.JavaGrammar,
		TotalFiles:  len(jobs),
		JavaFiles:   javaCount,
		KotlinFiles: kotlinCount,
	}

	if len(jobs) == 0 {
		summary.Duration = time.Since(startedAt)
		return summary, nil
	}

	jobChan := make(chan parseJob)
	resultChan := make(chan parseResult)

	workerCount := min(opts.Workers, len(jobs))

	var workers sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobChan {
				resultChan <- parseResult{
					lang: job.lang,
					path: job.path,
					err:  parseFile(job.path, job.lang, opts.JavaGrammar, opts.ParseTimeout),
				}
			}
		}()
	}

	go func() {
		for _, job := range jobs {
			jobChan <- job
		}
		close(jobChan)
		workers.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		if result.err != nil {
			summary.FailedFiles++
			if len(summary.Failures) < opts.MaxErrorFiles {
				summary.Failures = append(summary.Failures, FileFailure{
					Path:  result.path,
					Error: result.err.Error(),
				})
			}
			continue
		}
		summary.ParsedFiles++
	}

	summary.Duration = time.Since(startedAt)
	return summary, nil
}

func collectSourceFiles(root string, includeKTS bool) ([]parseJob, int, int, error) {
	jobs := make([]parseJob, 0, 1024)
	javaCount := 0
	kotlinCount := 0

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "build" || name == "out" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".java":
			jobs = append(jobs, parseJob{path: path, lang: langJava})
			javaCount++
		case ".kt":
			jobs = append(jobs, parseJob{path: path, lang: langKotlin})
			kotlinCount++
		case ".kts":
			if includeKTS {
				jobs = append(jobs, parseJob{path: path, lang: langKotlin})
				kotlinCount++
			}
		}
		return nil
	})
	if err != nil {
		return nil, 0, 0, fmt.Errorf("walk source files: %w", err)
	}
	return jobs, javaCount, kotlinCount, nil
}

func parseFile(path string, lang sourceLanguage, javaGrammar JavaGrammar, parseTimeout time.Duration) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	return parseWithTimeout(parseTimeout, func() error {
		switch lang {
		case langJava:
			return parseJava(source, javaGrammar)
		case langKotlin:
			return parseKotlin(source)
		default:
			return fmt.Errorf("unsupported language: %s", lang)
		}
	})
}

func parseWithTimeout(timeout time.Duration, parseFn func() error) error {
	if timeout <= 0 {
		return parseFn()
	}

	done := make(chan error, 1)
	go func() {
		done <- parseFn()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-done:
		return err
	case <-timer.C:
		return fmt.Errorf("parse timeout after %s", timeout.Round(0))
	}
}

func parseJava(source []byte, grammar JavaGrammar) error {
	source = normalizeJavaSourceForANTLR(source)
	switch grammar {
	case JavaGrammarOrig:
		return parseJavaOrigLL(source)
	case JavaGrammar7, JavaGrammar8:
		return parseJava8LL(source)
	case JavaGrammar9:
		return parseJava9LL(source)
	case JavaGrammar11, JavaGrammar17, JavaGrammar20, JavaGrammar21, JavaGrammar25:
		return parseJava20LL(source)
	default:
		return fmt.Errorf("unsupported java grammar: %q", grammar)
	}
}

func parseJavaOrigLL(source []byte) error {
	return safeParse(func() error {
		return parseJavaWithFallback(
			func() (error, bool) {
				return parseJavaOrigOnce(source, antlr.PredictionModeSLL)
			},
			func() error {
				err, _ := parseJavaOrigOnce(source, antlr.PredictionModeLL)
				return err
			},
		)
	})
}

func parseJava8LL(source []byte) error {
	return safeParse(func() error {
		return parseJavaWithFallback(
			func() (error, bool) {
				return parseJava8Once(source, antlr.PredictionModeSLL)
			},
			func() error {
				err, _ := parseJava8Once(source, antlr.PredictionModeLL)
				return err
			},
		)
	})
}

func parseJava9LL(source []byte) error {
	return safeParse(func() error {
		return parseJavaWithFallback(
			func() (error, bool) {
				return parseJava9Once(source, antlr.PredictionModeSLL)
			},
			func() error {
				err, _ := parseJava9Once(source, antlr.PredictionModeLL)
				return err
			},
		)
	})
}

func parseJava20LL(source []byte) error {
	return safeParse(func() error {
		return parseJavaWithFallback(
			func() (error, bool) {
				return parseJava20Once(source, antlr.PredictionModeSLL)
			},
			func() error {
				err, _ := parseJava20Once(source, antlr.PredictionModeLL)
				return err
			},
		)
	})
}

func parseJavaWithFallback(runSLL func() (error, bool), runLL func() error) error {
	err, shouldFallback := runSLL()
	if !shouldFallback {
		return err
	}
	return runLL()
}

func parseJavaOrigOnce(source []byte, predictionMode int) (error, bool) {
	listener := newSyntaxErrorListener(fileErrorMessageLimit)
	input := antlr.NewInputStream(string(source))
	lexer := javaorigparser.NewJavaLexer(input)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(listener)

	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := javaorigparser.NewJavaParser(tokens)
	parser.BuildParseTrees = false
	configureJavaParser(parser, listener, predictionMode)

	parseCanceled := false
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				if isParseCancellation(recovered) {
					parseCanceled = true
					return
				}
				panic(recovered)
			}
		}()
		parser.CompilationUnit()
	}()
	if parseCanceled {
		return nil, true
	}
	return finalizeJavaParseResult(listener, parser, predictionMode)
}

func parseJava8Once(source []byte, predictionMode int) (error, bool) {
	listener := newSyntaxErrorListener(fileErrorMessageLimit)
	input := antlr.NewInputStream(string(source))
	lexer := java8parser.NewJava8Lexer(input)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(listener)

	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := java8parser.NewJava8Parser(tokens)
	parser.BuildParseTrees = false
	configureJavaParser(parser, listener, predictionMode)

	parseCanceled := false
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				if isParseCancellation(recovered) {
					parseCanceled = true
					return
				}
				panic(recovered)
			}
		}()
		parser.CompilationUnit()
	}()
	if parseCanceled {
		return nil, true
	}
	return finalizeJavaParseResult(listener, parser, predictionMode)
}

func parseJava9Once(source []byte, predictionMode int) (error, bool) {
	listener := newSyntaxErrorListener(fileErrorMessageLimit)
	input := antlr.NewInputStream(string(source))
	lexer := java9parser.NewJava9Lexer(input)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(listener)

	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := java9parser.NewJava9Parser(tokens)
	parser.BuildParseTrees = false
	configureJavaParser(parser, listener, predictionMode)

	parseCanceled := false
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				if isParseCancellation(recovered) {
					parseCanceled = true
					return
				}
				panic(recovered)
			}
		}()
		parser.CompilationUnit()
	}()
	if parseCanceled {
		return nil, true
	}
	return finalizeJavaParseResult(listener, parser, predictionMode)
}

func parseJava20Once(source []byte, predictionMode int) (error, bool) {
	listener := newSyntaxErrorListener(fileErrorMessageLimit)
	input := antlr.NewInputStream(string(source))
	lexer := java20parser.NewJava20Lexer(input)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(listener)

	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := java20parser.NewJava20Parser(tokens)
	parser.BuildParseTrees = false
	configureJavaParser(parser, listener, predictionMode)

	parseCanceled := false
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				if isParseCancellation(recovered) {
					parseCanceled = true
					return
				}
				panic(recovered)
			}
		}()
		parser.CompilationUnit()
	}()
	if parseCanceled {
		return nil, true
	}
	return finalizeJavaParseResult(listener, parser, predictionMode)
}

func finalizeJavaParseResult(listener *syntaxErrorListener, parser antlr.Parser, predictionMode int) (error, bool) {
	if parseErr := listener.Err(); parseErr != nil {
		if predictionMode == antlr.PredictionModeSLL {
			return nil, true
		}
		return parseErr, false
	}
	if parser.HasError() {
		if predictionMode == antlr.PredictionModeSLL {
			return nil, true
		}
		return parserRecognitionError(parser.GetError()), false
	}
	return nil, false
}

func configureJavaParser(parser antlr.Parser, listener *syntaxErrorListener, predictionMode int) {
	parser.RemoveErrorListeners()
	parser.AddErrorListener(listener)
	parser.GetInterpreter().SetPredictionMode(predictionMode)
	if predictionMode == antlr.PredictionModeSLL {
		parser.SetErrorHandler(newPanicBailErrorStrategy())
	}
}

func isParseCancellation(recovered any) bool {
	switch recovered.(type) {
	case *antlr.ParseCancellationException, antlr.ParseCancellationException:
		return true
	default:
		return false
	}
}

func parserRecognitionError(recognitionErr antlr.RecognitionException) error {
	if recognitionErr == nil {
		return errors.New("antlr parse error")
	}
	switch recognitionErr.(type) {
	case *antlr.ParseCancellationException, antlr.ParseCancellationException:
		return errors.New("antlr parse canceled")
	default:
		return fmt.Errorf("antlr parse error: %T", recognitionErr)
	}
}

func parseKotlin(source []byte) error {
	compiler := kcg.New(kcg.Config{
		Workers:          1,
		MaxErrorsPerFile: fileErrorMessageLimit,
		IncludeKTS:       true,
	})
	unit := compiler.ParseSource("<memory>.kt", source)
	if unit.Parsed {
		return nil
	}
	if len(unit.Diagnostics) == 0 {
		return errors.New("kotlin parse failed")
	}
	messages := make([]string, 0, len(unit.Diagnostics))
	for _, diagnostic := range unit.Diagnostics {
		messages = append(messages, fmt.Sprintf("%d:%d %s", diagnostic.Line, diagnostic.Column, diagnostic.Message))
	}
	return errors.New(strings.Join(messages, " | "))
}

// normalizeKotlinSource applies preprocessing to handle modern Kotlin syntax
// that the ANTLR grammar doesn't natively support (expect/actual, value class, etc.)
func normalizeKotlinSource(source []byte) []byte {
	if len(source) == 0 {
		return source
	}

	// Check if normalization is needed
	if !needsKotlinNormalization(source) {
		return source
	}

	text := string(source)
	normalized := kotlinPlatformModifierPattern.ReplaceAllString(text, "$1")
	normalized = kotlinValueClassPattern.ReplaceAllString(normalized, "$1")
	normalized = kotlinFunInterfacePattern.ReplaceAllString(normalized, "$1")
	normalized = kotlinContextReceiverPattern.ReplaceAllString(normalized, "")
	normalized = kotlinSealedInterfacePattern.ReplaceAllString(normalized, "$1")
	normalized = kotlinLambdaLabelPattern.ReplaceAllString(normalized, "{")
	normalized = kotlinFunctionTypeCastPattern.ReplaceAllString(normalized, "")
	normalized = kotlinNullableTypeCastPattern.ReplaceAllString(normalized, "as $1")
	normalized = kotlinExtensionStarReceiverPattern.ReplaceAllString(normalized, "$1.")
	normalized = kotlinExtensionGenericReceiverPattern.ReplaceAllString(normalized, "$1.")
	normalized = kotlinAnonymousFunctionPattern.ReplaceAllString(normalized, "= { $1 ->")
	normalized = kotlinStringTemplatePattern.ReplaceAllString(normalized, "0")

	if normalized == text {
		return source
	}
	return []byte(normalized)
}

// needsKotlinNormalization checks if the source contains modern Kotlin constructs
func needsKotlinNormalization(source []byte) bool {
	if len(source) == 0 {
		return false
	}
	text := string(source)

	// Use the actual regex patterns for accurate detection
	return kotlinPlatformModifierPattern.MatchString(text) ||
		kotlinValueClassPattern.MatchString(text) ||
		kotlinFunInterfacePattern.MatchString(text) ||
		kotlinContextReceiverPattern.MatchString(text) ||
		kotlinSealedInterfacePattern.MatchString(text) ||
		kotlinLambdaLabelPattern.MatchString(text) ||
		kotlinFunctionTypeCastPattern.MatchString(text) ||
		kotlinNullableTypeCastPattern.MatchString(text) ||
		kotlinExtensionStarReceiverPattern.MatchString(text) ||
		kotlinExtensionGenericReceiverPattern.MatchString(text) ||
		kotlinAnonymousFunctionPattern.MatchString(text) ||
		kotlinStringTemplatePattern.MatchString(text)
}

func safeParse(parseFn func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("parser panic: %v", recovered)
		}
	}()
	return parseFn()
}

type syntaxErrorListener struct {
	*antlr.DefaultErrorListener
	errors []string
	limit  int
}

func newSyntaxErrorListener(limit int) *syntaxErrorListener {
	if limit <= 0 {
		limit = 1
	}
	return &syntaxErrorListener{
		DefaultErrorListener: &antlr.DefaultErrorListener{},
		errors:               make([]string, 0, limit),
		limit:                limit,
	}
}

func (l *syntaxErrorListener) SyntaxError(
	_ antlr.Recognizer,
	_ any,
	line int,
	column int,
	msg string,
	_ antlr.RecognitionException,
) {
	if len(l.errors) >= l.limit {
		return
	}
	l.errors = append(l.errors, fmt.Sprintf("%d:%d %s", line, column, msg))
}

func (l *syntaxErrorListener) Err() error {
	if len(l.errors) == 0 {
		return nil
	}
	return errors.New(strings.Join(l.errors, " | "))
}
