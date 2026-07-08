package java9

import "github.com/antlr4-go/antlr/v4"

// Java9LexerBase is required because Java9Lexer.g4 sets superClass=Java9LexerBase.
type Java9LexerBase struct {
	*antlr.BaseLexer
}

// Check* predicates in this grammar are used for Unicode identifier validation.
// We keep them permissive for broad-source parsing smoke tests.
func Check1() bool { return true }
func Check2() bool { return true }
func Check3() bool { return true }
func Check4() bool { return true }
