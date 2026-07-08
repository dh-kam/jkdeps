package mixedgraph

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dh-kam/jkdeps/internal/ast"
	"github.com/dh-kam/jkdeps/internal/parser"
	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

func convertParserDiagnostics(in []ast.Diagnostic) []Diagnostic {
	if len(in) == 0 {
		return nil
	}
	out := make([]Diagnostic, 0, len(in))
	for _, diag := range in {
		line := diag.Loc.Span.Start.Line
		column := diag.Loc.Span.Start.Column
		out = append(out, Diagnostic{
			Path:    diag.Loc.FilePath,
			Line:    line,
			Column:  column,
			Message: diag.Message,
		})
	}
	return out
}

type parseJob struct {
	path string
	rel  string
	lang SourceLanguage
}

type parseResult struct {
	unit FileUnit
	err  error
}

func ParseRepository(root string, opts ParseOptions) (RepositoryResult, error) {
	startedAt := time.Now()
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return RepositoryResult{}, fmt.Errorf("resolve root path: %w", err)
	}

	opts = opts.withDefaults()
	if !opts.JavaParseMode.IsValid() {
		return RepositoryResult{}, fmt.Errorf("unsupported java parse mode: %q", opts.JavaParseMode)
	}
	if opts.JavaParseMode == JavaParseModeFull && !opts.JavaGrammar.IsValid() {
		return RepositoryResult{}, fmt.Errorf("unsupported java grammar: %q", opts.JavaGrammar)
	}

	jobs, javaCount, kotlinCount, err := collectSourceFiles(rootPath, opts.IncludeKTS, opts.IncludeBuildScripts)
	if err != nil {
		return RepositoryResult{}, err
	}

	result := RepositoryResult{
		Root:        rootPath,
		TotalFiles:  len(jobs),
		JavaFiles:   javaCount,
		KotlinFiles: kotlinCount,
		Files:       make([]FileUnit, 0, len(jobs)),
	}
	if len(jobs) == 0 {
		result.Duration = time.Since(startedAt)
		return result, nil
	}

	kotlinCompiler := kcg.New(kotlinCompilerConfig(opts))

	jobChan := make(chan parseJob)
	resultChan := make(chan parseResult)

	workerCount := min(opts.Workers, len(jobs))

	var workers sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobChan {
				unit, parseErr := parseFile(job, opts, kotlinCompiler)
				resultChan <- parseResult{unit: unit, err: parseErr}
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

	var firstErr error
	for parsed := range resultChan {
		if parsed.err != nil {
			if firstErr == nil {
				firstErr = parsed.err
			}
			continue
		}
		result.Files = append(result.Files, parsed.unit)
		if parsed.unit.Parsed {
			result.ParsedFiles++
		} else {
			result.FailedFiles++
		}
	}
	if firstErr != nil {
		return RepositoryResult{}, firstErr
	}

	result.Duration = time.Since(startedAt)
	return result, nil
}

func collectSourceFiles(root string, includeKTS bool, includeBuildScripts bool) ([]parseJob, int, int, error) {
	collector := newSourceFileCollector(root, includeKTS, includeBuildScripts)
	return collector.collect()
}

// sourceFileCollector handles source file discovery and classification
type sourceFileCollector struct {
	root                string
	includeKTS          bool
	includeBuildScripts bool
	jobs                []parseJob
	javaCount           int
	kotlinCount         int
}

func newSourceFileCollector(root string, includeKTS, includeBuildScripts bool) *sourceFileCollector {
	return &sourceFileCollector{
		root:                root,
		includeKTS:          includeKTS,
		includeBuildScripts: includeBuildScripts,
		jobs:                make([]parseJob, 0, 1024),
	}
}

func (c *sourceFileCollector) collect() ([]parseJob, int, int, error) {
	err := filepath.WalkDir(c.root, c.visit)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("walk source files: %w", err)
	}
	return c.jobs, c.javaCount, c.kotlinCount, nil
}

func (c *sourceFileCollector) visit(path string, d os.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
		return c.handleDirectory(path)
	}
	return c.handleFile(path)
}

func (c *sourceFileCollector) handleDirectory(path string) error {
	name := filepath.Base(path)
	switch name {
	case ".git", "build", "out", "target", "node_modules":
		return filepath.SkipDir
	}
	return nil
}

func (c *sourceFileCollector) handleFile(path string) error {
	lang := c.detectLanguage(path)
	if lang == "" {
		return nil
	}

	rel, err := filepath.Rel(c.root, path)
	if err != nil {
		return err
	}

	c.jobs = append(c.jobs, parseJob{
		path: path,
		rel:  filepath.ToSlash(rel),
		lang: lang,
	})
	c.incrementCount(lang)
	return nil
}

func (c *sourceFileCollector) detectLanguage(path string) SourceLanguage {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".java":
		return LangJava
	case ".kt":
		return LangKotlin
	case ".kts":
		return c.detectKTSLanguage(path)
	}
	return ""
}

func (c *sourceFileCollector) detectKTSLanguage(path string) SourceLanguage {
	base := strings.ToLower(filepath.Base(path))
	isBuildScript := base == "build.gradle.kts" || base == "settings.gradle.kts"

	if isBuildScript {
		if c.includeBuildScripts {
			return LangKotlin
		}
		return ""
	}

	if c.includeKTS {
		return LangKotlin
	}
	return ""
}

func (c *sourceFileCollector) incrementCount(lang SourceLanguage) {
	switch lang {
	case LangJava:
		c.javaCount++
	case LangKotlin:
		c.kotlinCount++
	}
}

func parseFile(job parseJob, opts ParseOptions, kotlinCompiler *kcg.Compiler) (FileUnit, error) {
	source, err := os.ReadFile(job.path)
	if err != nil {
		return FileUnit{}, fmt.Errorf("read file %s: %w", job.path, err)
	}

	switch job.lang {
	case LangKotlin:
		return parseKotlinSource(job, source, kotlinCompiler), nil
	case LangJava:
		return parseJavaSource(job, source, opts), nil
	default:
		return FileUnit{}, fmt.Errorf("unsupported language: %s", job.lang)
	}
}

func parseKotlinSource(job parseJob, source []byte, compiler *kcg.Compiler) FileUnit {
	unit := compiler.ParseSource(job.path, source)
	return FileUnit{
		Path:        job.path,
		Relative:    job.rel,
		Language:    LangKotlin,
		PackageName: unit.PackageName,
		Imports:     append([]string(nil), unit.Imports...),
		Parsed:      unit.Parsed,
		Diagnostics: convertKotlinDiagnostics(unit.Diagnostics),
		Duration:    unit.Duration,
	}
}

func kotlinCompilerConfig(opts ParseOptions) kcg.Config {
	return kcg.Config{
		Workers:          1,
		MaxErrorsPerFile: opts.MaxErrorsPerFile,
		IncludeKTS:       opts.IncludeKTS,
		LenientSyntax:    opts.LenientSyntax,
		ParseTimeout:     opts.ParseTimeout,
	}
}

func parseJavaSource(job parseJob, source []byte, opts ParseOptions) FileUnit {
	startedAt := time.Now()
	text := string(source)
	pkg, imports := extractJavaHeader(text)

	if opts.JavaParseMode == JavaParseModeHeaderOnly {
		return FileUnit{
			Path:        job.path,
			Relative:    job.rel,
			Language:    LangJava,
			PackageName: pkg,
			Imports:     imports,
			Parsed:      true,
			Duration:    time.Since(startedAt),
		}
	}

	var diagnostics []Diagnostic
	var references []Reference
	if opts.ParseTimeout > 0 {
		diagnostics = parseJavaDiagnosticsWithTimeout(
			text,
			job.path,
			opts.MaxErrorsPerFile,
			opts.JavaGrammar,
			opts.ParseTimeout,
		)
	} else {
		diagnostics = parseJavaDiagnostics(text, job.path, opts.MaxErrorsPerFile, opts.JavaGrammar)
	}
	if len(diagnostics) == 0 && !opts.JavaGrammar.IsValid() {
		diagnostics = append(diagnostics, Diagnostic{
			Path:    job.path,
			Line:    0,
			Column:  0,
			Message: fmt.Sprintf("unsupported java grammar: %s", opts.JavaGrammar),
		})
	}
	if len(diagnostics) == 0 || opts.LenientSyntax {
		if sourceFile, err := parser.BuildJavaSourceFile(source, opts.JavaGrammar); err == nil {
			references = extractJavaReferences(sourceFile)
		}
	}

	return FileUnit{
		Path:        job.path,
		Relative:    job.rel,
		Language:    LangJava,
		PackageName: pkg,
		Imports:     imports,
		References:  references,
		Parsed:      len(diagnostics) == 0 || opts.LenientSyntax,
		Diagnostics: diagnostics,
		Duration:    time.Since(startedAt),
	}
}

func parseJavaDiagnostics(source string, path string, maxErrors int, grammar parser.JavaGrammar) []Diagnostic {
	if !grammar.IsValid() {
		return []Diagnostic{{
			Path:    path,
			Line:    0,
			Column:  0,
			Message: fmt.Sprintf("unsupported java grammar: %s", grammar),
		}}
	}
	result := parser.ParseWithDiagnostics(path, []byte(source), ast.LanguageJava, grammar, maxErrors)
	return convertParserDiagnostics(result.Diagnostics)
}

func parseJavaDiagnosticsWithTimeout(
	source string,
	path string,
	maxErrors int,
	grammar parser.JavaGrammar,
	timeout time.Duration,
) []Diagnostic {
	return parseJavaDiagnosticsWithTimeoutFunc(parseJavaDiagnostics, source, path, maxErrors, grammar, timeout)
}

func parseJavaDiagnosticsWithTimeoutFunc(
	parseFunc func(string, string, int, parser.JavaGrammar) []Diagnostic,
	source string,
	path string,
	maxErrors int,
	grammar parser.JavaGrammar,
	timeout time.Duration,
) []Diagnostic {
	done := make(chan []Diagnostic, 1)
	go func() {
		done <- parseFunc(source, path, maxErrors, grammar)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case diagnostics := <-done:
		return diagnostics
	case <-timer.C:
		return []Diagnostic{{
			Path:    path,
			Line:    0,
			Column:  0,
			Message: fmt.Sprintf("parse timeout after %s", timeout.Round(0)),
		}}
	}
}

func convertKotlinDiagnostics(input []kcg.Diagnostic) []Diagnostic {
	if len(input) == 0 {
		return nil
	}
	out := make([]Diagnostic, 0, len(input))
	for _, item := range input {
		out = append(out, Diagnostic{
			Path:    item.Path,
			Line:    item.Line,
			Column:  item.Column,
			Message: item.Message,
		})
	}
	return out
}
