package ast

import (
	"io"
	"time"
)

// ParseResult represents the result of parsing a source file
type ParseResult struct {
	SourceFile  *SourceFile   `json:"source_file,omitempty"`
	Diagnostics []Diagnostic  `json:"diagnostics,omitempty"`
	Success     bool          `json:"success"`
	Duration    time.Duration `json:"duration"`
}

// ParseOptions represents options for parsing
type ParseOptions struct {
	// Language specifies the source language (empty for auto-detection)
	Language SourceLanguage `json:"language,omitempty"`
	// BuildAST enables full AST construction (vs. just package/imports extraction)
	BuildAST bool `json:"build_ast,omitempty"`
	// IncludeComments enables comment parsing
	IncludeComments bool `json:"include_comments,omitempty"`
	// Lenient enables lenient parsing (collects errors instead of failing)
	Lenient bool `json:"lenient,omitempty"`
	// MaxErrors limits the number of errors collected (0 = unlimited)
	MaxErrors int `json:"max_errors,omitempty"`
	// Timeout specifies the parse timeout (0 = no timeout)
	Timeout time.Duration `json:"timeout,omitempty"`
}

// Parser is the interface for parsing source code
//
// This interface follows the Interface Segregation Principle by providing
// a focused interface for parsing operations. Implementations can be for
// specific languages (Java, Kotlin) or specific parser technologies
// (ANTLR, handwritten, etc.).
type Parser interface {
	// ParseFile parses a source file and returns the result
	ParseFile(path string, opts ParseOptions) (ParseResult, error)

	// ParseSource parses source code from a byte slice
	ParseSource(source []byte, opts ParseOptions) (ParseResult, error)

	// ParseReader parses source code from an io.Reader
	ParseReader(r io.Reader, opts ParseOptions) (ParseResult, error)

	// Language returns the language this parser handles
	Language() SourceLanguage
}

// Lexer is the interface for lexical analysis (tokenization)
//
// Separating lexing from parsing follows the Single Responsibility Principle
// and allows for optimized tokenization strategies per language.
type Lexer interface {
	// Tokenize returns the tokens for the given source
	Tokenize(source []byte) ([]Token, []Diagnostic, error)

	// Language returns the language this lexer handles
	Language() SourceLanguage
}

// Token represents a single token from the lexer
type Token struct {
	Type     TokenType `json:"type"`
	Value    string    `json:"value,omitempty"`
	Location Location  `json:"location,omitempty"`
}

// TokenType represents the type of token
type TokenType string

const (
	TokenEOF        TokenType = "EOF"
	TokenIdentifier TokenType = "identifier"
	TokenKeyword    TokenType = "keyword"
	TokenLiteral    TokenType = "literal"
	TokenOperator   TokenType = "operator"
	TokenSeparator  TokenType = "separator"
	TokenComment    TokenType = "comment"
	TokenWhitespace TokenType = "whitespace"
	TokenUnknown    TokenType = "unknown"
)

// ParserFactory is a factory for creating parsers
//
// This enables dependency injection and allows parser implementations
// to be swapped without changing client code.
type ParserFactory interface {
	// CreateParser creates a new parser for the given language
	CreateParser(lang SourceLanguage) (Parser, error)

	// SupportedLanguages returns the list of supported languages
	SupportedLanguages() []SourceLanguage
}

// ASTBuilder is an interface for building AST nodes from parse trees
//
// This allows different AST construction strategies and enables
// testing of AST construction separately from parsing.
type ASTBuilder interface {
	// BuildFile builds a SourceFile from a parse tree
	BuildFile(tree ParseTree) (*SourceFile, error)

	// BuildDeclaration builds a Declaration from a parse tree node
	BuildDeclaration(node ParseTreeNode) (Declaration, error)

	// BuildTypeReference builds a TypeReference from a parse tree node
	BuildTypeReference(node ParseTreeNode) (*TypeReference, error)
}

// ParseTree represents the raw parse tree from a parser
type ParseTree interface {
	// Root returns the root node of the parse tree
	Root() ParseTreeNode

	// Tokens returns the tokens from the parse (if available)
	Tokens() []Token

	// Diagnostics returns parsing diagnostics
	Diagnostics() []Diagnostic
}

// ParseTreeNode represents a node in the parse tree
type ParseTreeNode interface {
	// Type returns the node type
	Type() string

	// Text returns the text matched by this node
	Text() string

	// Location returns the source location
	Location() Location

	// Children returns child nodes
	Children() []ParseTreeNode

	// ChildCount returns the number of children
	ChildCount() int

	// Child returns the child at the given index
	Child(int) ParseTreeNode
}

// SourceReader is an interface for reading source files
//
// This abstraction allows for different file system implementations
// and enables testing without actual file I/O.
type SourceReader interface {
	// Read reads the content of a file
	Read(path string) ([]byte, error)

	// Exists checks if a file exists
	Exists(path string) bool

	// IsDir checks if a path is a directory
	IsDir(path string) bool
}

// ASTVisitor is an interface for traversing AST nodes
//
// This enables visitor pattern operations on the AST such as
// analysis, transformation, and code generation.
type ASTVisitor interface {
	// Visit visits a node and returns whether to continue traversal
	Visit(node Node) (recurse bool, err error)

	// VisitBefore is called before visiting children
	VisitBefore(node Node) error

	// VisitAfter is called after visiting children
	VisitAfter(node Node) error
}

// VisitorFunc is a function adapter for ASTVisitor
type VisitorFunc func(node Node) error

// Visit implements ASTVisitor for a simple function
func (f VisitorFunc) Visit(node Node) (bool, error) {
	return true, f(node)
}

func (f VisitorFunc) VisitBefore(node Node) error {
	return f(node)
}

func (f VisitorFunc) VisitAfter(node Node) error {
	return nil
}

// Walk traverses the AST using the given visitor
func Walk(visitor ASTVisitor, node Node) error {
	if err := visitor.VisitBefore(node); err != nil {
		return err
	}

	recurse, err := visitor.Visit(node)
	if err != nil {
		return err
	}
	if recurse {
		for _, child := range node.Children() {
			if err := Walk(visitor, child); err != nil {
				return err
			}
		}
	}

	return visitor.VisitAfter(node)
}
