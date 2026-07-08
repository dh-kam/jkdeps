package javaorig

import "github.com/antlr4-go/antlr/v4"

// JavaParserBase wires custom semantic predicates declared in JavaParser.g4.
type JavaParserBase struct {
	*antlr.BaseParser
}

func (p *JavaParserBase) DoLastRecordComponent() bool {
	ctx, ok := p.GetParserRuleContext().(*RecordComponentListContext)
	if !ok {
		return true
	}

	components := ctx.AllRecordComponent()
	for idx, component := range components {
		if component.ELLIPSIS() != nil && idx+1 < len(components) {
			return false
		}
	}
	return true
}

func (p *JavaParserBase) IsNotIdentifierAssign() bool {
	next := p.GetTokenStream().LA(1)
	switch next {
	case JavaParserIDENTIFIER,
		JavaParserMODULE,
		JavaParserOPEN,
		JavaParserREQUIRES,
		JavaParserEXPORTS,
		JavaParserOPENS,
		JavaParserTO,
		JavaParserUSES,
		JavaParserPROVIDES,
		JavaParserWHEN,
		JavaParserWITH,
		JavaParserTRANSITIVE,
		JavaParserYIELD,
		JavaParserSEALED,
		JavaParserPERMITS,
		JavaParserRECORD,
		JavaParserVAR:
		// Continue.
	default:
		return true
	}

	return p.GetTokenStream().LA(2) != JavaParserASSIGN
}
