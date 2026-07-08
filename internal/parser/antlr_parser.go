package parser

import (
	"context"
	"fmt"
	"io"
	"time"

	ast "github.com/dh-kam/jkdeps/internal/ast"
)

// ANTLRParser implements ast.Parser using ANTLR4 grammars
type ANTLRParser struct {
	grammar    JavaGrammar
	language   ast.SourceLanguage
	fileReader ast.SourceReader
}

// NewANTLRParser creates a new ANTLR-based parser with OS file reader
func NewANTLRParser(grammar JavaGrammar, language ast.SourceLanguage) *ANTLRParser {
	return &ANTLRParser{
		grammar:    grammar,
		language:   language,
		fileReader: NewOSFileReader(),
	}
}

// NewANTLRParserWithReader creates a new ANTLR-based parser with custom file reader
func NewANTLRParserWithReader(grammar JavaGrammar, language ast.SourceLanguage, reader ast.SourceReader) *ANTLRParser {
	return &ANTLRParser{
		grammar:    grammar,
		language:   language,
		fileReader: reader,
	}
}

// ParseFile parses a source file and returns the result
func (p *ANTLRParser) ParseFile(path string, opts ast.ParseOptions) (ast.ParseResult, error) {
	source, err := p.fileReader.Read(path)
	if err != nil {
		return ast.ParseResult{}, fmt.Errorf("read file: %w", err)
	}

	return p.ParseSource(source, opts)
}

// ParseSource parses source code from a byte slice
func (p *ANTLRParser) ParseSource(source []byte, opts ast.ParseOptions) (ast.ParseResult, error) {
	if opts.Language == "" {
		opts.Language = p.language
	}

	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second // default timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	resultChan := make(chan ast.ParseResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := p.parseInternal(source, opts)
		if err != nil {
			select {
			case errChan <- err:
			case <-ctx.Done():
			}
			return
		}
		select {
		case resultChan <- result:
		case <-ctx.Done():
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return ast.ParseResult{}, err
	case <-ctx.Done():
		return ast.ParseResult{
			Success: false,
			Diagnostics: []ast.Diagnostic{
				{
					Severity: ast.SeverityError,
					Message:  fmt.Sprintf("parse timeout after %s", opts.Timeout),
				},
			},
		}, nil
	}
}

// ParseReader parses source code from an io.Reader
func (p *ANTLRParser) ParseReader(r io.Reader, opts ast.ParseOptions) (ast.ParseResult, error) {
	source, err := io.ReadAll(r)
	if err != nil {
		return ast.ParseResult{}, fmt.Errorf("read source: %w", err)
	}
	return p.ParseSource(source, opts)
}

// Language returns the language this parser handles
func (p *ANTLRParser) Language() ast.SourceLanguage {
	return p.language
}

// parseInternal performs the actual parsing with timing and result building
func (p *ANTLRParser) parseInternal(source []byte, opts ast.ParseOptions) (ast.ParseResult, error) {
	started := time.Now()

	// Execute parsing
	parseErr := p.executeParse(source, opts)
	duration := time.Since(started)

	// Build diagnostics from parse error
	diagnostics := p.buildDiagnostics(parseErr, opts.Lenient)
	success := parseErr == nil || opts.Lenient

	// Build result
	result := ast.ParseResult{
		Success:     success,
		Diagnostics: diagnostics,
		Duration:    duration,
	}

	// Build AST if requested
	if opts.BuildAST && success {
		if sourceFile, err := p.buildAST(source, opts); err == nil {
			result.SourceFile = sourceFile
		}
	}

	return result, nil
}

// executeParse performs the core parsing operation
func (p *ANTLRParser) executeParse(source []byte, opts ast.ParseOptions) error {
	switch opts.Language {
	case ast.LanguageJava:
		return parseJava(source, p.grammar)
	case ast.LanguageKotlin:
		return parseKotlin(source)
	default:
		return fmt.Errorf("unsupported language: %s", opts.Language)
	}
}

// buildDiagnostics creates diagnostics from parse error
func (p *ANTLRParser) buildDiagnostics(parseErr error, lenient bool) []ast.Diagnostic {
	if parseErr == nil {
		return nil
	}
	if !lenient {
		return nil
	}
	return []ast.Diagnostic{
		{
			Severity: ast.SeverityError,
			Message:  parseErr.Error(),
		},
	}
}

// buildAST constructs the AST from parsed source using strategy pattern
func (p *ANTLRParser) buildAST(source []byte, opts ast.ParseOptions) (*ast.SourceFile, error) {
	strategy := p.getASTBuilderStrategy(opts.Language)
	return strategy.Build(source, opts.Language)
}

// getASTBuilderStrategy returns the appropriate AST builder strategy
func (p *ANTLRParser) getASTBuilderStrategy(lang ast.SourceLanguage) ASTBuilderStrategy {
	switch lang {
	case ast.LanguageJava:
		return &javaASTBuilder{grammar: p.grammar}
	case ast.LanguageKotlin:
		return &kotlinASTBuilder{}
	default:
		return &defaultASTBuilder{}
	}
}

// ASTBuilderStrategy defines the interface for language-specific AST building
type ASTBuilderStrategy interface {
	Build(source []byte, lang ast.SourceLanguage) (*ast.SourceFile, error)
}

// javaASTBuilder builds AST for Java source files
type javaASTBuilder struct {
	grammar JavaGrammar
}

// kotlinASTBuilder builds AST for Kotlin source files
type kotlinASTBuilder struct{}

func (b *kotlinASTBuilder) Build(source []byte, lang ast.SourceLanguage) (*ast.SourceFile, error) {
	pkgName, imports := extractKotlinHeader(string(source))
	return &ast.SourceFile{
		Language: lang,
		Package:  ast.PackageDeclaration{Name: pkgName},
		Imports:  convertImports(imports),
		Parsed:   true,
	}, nil
}

// defaultASTBuilder provides a minimal AST builder for unsupported languages
type defaultASTBuilder struct{}

func (b *defaultASTBuilder) Build(source []byte, lang ast.SourceLanguage) (*ast.SourceFile, error) {
	return &ast.SourceFile{
		Language: lang,
		Parsed:   true,
	}, nil
}

// convertImports converts string import paths to ast.Import
func convertImports(importPaths []string) []ast.Import {
	if len(importPaths) == 0 {
		return nil
	}

	imports := make([]ast.Import, 0, len(importPaths))
	for _, path := range importPaths {
		imports = append(imports, ast.Import{
			Path: path,
		})
	}
	return imports
}

// Ensure ANTLRParser implements ast.Parser
var _ ast.Parser = (*ANTLRParser)(nil)
