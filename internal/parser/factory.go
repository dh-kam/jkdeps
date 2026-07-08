package parser

import (
	"fmt"

	ast "github.com/dh-kam/jkdeps/internal/ast"
)

// ParserFactory creates parsers for different languages
type ParserFactory struct {
	defaultJavaGrammar JavaGrammar
}

// NewParserFactory creates a new parser factory
func NewParserFactory(defaultJavaGrammar JavaGrammar) *ParserFactory {
	if defaultJavaGrammar == "" {
		defaultJavaGrammar = JavaGrammarDefault
	}
	return &ParserFactory{
		defaultJavaGrammar: defaultJavaGrammar,
	}
}

// CreateParser creates a parser for the given language
func (f *ParserFactory) CreateParser(lang ast.SourceLanguage) (ast.Parser, error) {
	switch lang {
	case ast.LanguageJava:
		return NewANTLRParser(f.defaultJavaGrammar, ast.LanguageJava), nil
	case ast.LanguageKotlin:
		return NewANTLRParser(JavaGrammarDefault, ast.LanguageKotlin), nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}
}

// SupportedLanguages returns the list of supported languages
func (f *ParserFactory) SupportedLanguages() []ast.SourceLanguage {
	return []ast.SourceLanguage{ast.LanguageJava, ast.LanguageKotlin}
}

// Ensure ParserFactory implements ast.ParserFactory
var _ ast.ParserFactory = (*ParserFactory)(nil)
