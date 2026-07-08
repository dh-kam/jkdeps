package main

import (
	"fmt"
	"os"

	"github.com/antlr4-go/antlr/v4"
	kotlinparser "github.com/dh-kam/jkdeps/internal/parsers/kotlin"
)

type dbg struct {
	*kotlinparser.BaseKotlinParserListener
}

func textOf(node antlr.Tree) string {
	if p, ok := node.(antlr.ParseTree); ok {
		return p.GetText()
	}
	if n, ok := node.(interface{ GetPayload() any }); ok {
		_ = n
	}
	return ""
}

func (d *dbg) EnterImportHeader(ctx *kotlinparser.ImportHeaderContext) {
	fmt.Printf("IMPORT HEADER: %q\n", ctx.GetText())
	if id := ctx.Identifier(); id != nil {
		fmt.Printf("  Identifier: %q\n", id.GetText())
	}
	children := ctx.GetChildren()
	fmt.Printf("  children=%d\n", len(children))
	for i, c := range children {
		fmt.Printf("    %d: %T %q\n", i, c, textOf(c))
	}
	if ctx.DOT() != nil {
		fmt.Printf("  DOT: %q\n", ctx.DOT().GetText())
	}
	if ctx.MULT() != nil {
		fmt.Printf("  MULT: %q\n", ctx.MULT().GetText())
	}
	if alias := ctx.ImportAlias(); alias != nil {
		fmt.Printf("  Alias: %q\n", alias.GetText())
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: debug_import <file>")
		os.Exit(2)
	}
	path := os.Args[1]
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
		os.Exit(1)
	}
	input := antlr.NewInputStream(string(b))
	lexer := kotlinparser.NewKotlinLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := kotlinparser.NewKotlinParser(tokens)
	parser.BuildParseTrees = true
	tree := parser.KotlinFile()
	listener := &dbg{BaseKotlinParserListener: &kotlinparser.BaseKotlinParserListener{}}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
}
