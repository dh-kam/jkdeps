package parser

import "github.com/antlr4-go/antlr/v4"

type panicBailErrorStrategy struct {
	*antlr.DefaultErrorStrategy
}

func newPanicBailErrorStrategy() *panicBailErrorStrategy {
	return &panicBailErrorStrategy{
		DefaultErrorStrategy: antlr.NewDefaultErrorStrategy(),
	}
}

func (b *panicBailErrorStrategy) Recover(recognizer antlr.Parser, e antlr.RecognitionException) {
	context := recognizer.GetParserRuleContext()
	for context != nil {
		context.SetException(e)
		if parent, ok := context.GetParent().(antlr.ParserRuleContext); ok {
			context = parent
		} else {
			context = nil
		}
	}
	panic(antlr.NewParseCancellationException())
}

func (b *panicBailErrorStrategy) RecoverInline(recognizer antlr.Parser) antlr.Token {
	b.Recover(recognizer, antlr.NewInputMisMatchException(recognizer))
	return nil
}

func (b *panicBailErrorStrategy) Sync(_ antlr.Parser) {}
