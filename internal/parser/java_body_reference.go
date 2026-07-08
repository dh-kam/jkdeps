package parser

import (
	ast "github.com/dh-kam/jkdeps/internal/ast"
	java20parser "github.com/dh-kam/jkdeps/internal/parsers/java20"
)

func (c *java20SourceFileCollector) EnterUnqualifiedClassInstanceCreationExpression(ctx *java20parser.UnqualifiedClassInstanceCreationExpressionContext) {
	if ctx == nil || ctx.ClassOrInterfaceTypeToInstantiate() == nil {
		return
	}
	c.addReferenceUsage(
		locationFromRule(ctx),
		ast.ReferenceUsageConstructorCall,
		parseJavaTypeReferenceText(ctx.ClassOrInterfaceTypeToInstantiate().GetText()),
	)
}

func (c *java20SourceFileCollector) EnterMethodInvocation(ctx *java20parser.MethodInvocationContext) {
	if ctx == nil || ctx.TypeName() == nil {
		return
	}
	c.addReferenceUsage(
		locationFromRule(ctx),
		ast.ReferenceUsageQualifiedMethodCall,
		parseJavaTypeReferenceText(ctx.TypeName().GetText()),
	)
}

func (c *java20SourceFileCollector) EnterClassLiteral(ctx *java20parser.ClassLiteralContext) {
	if ctx == nil || ctx.TypeName() == nil {
		return
	}
	c.addReferenceUsage(
		locationFromRule(ctx),
		ast.ReferenceUsageClassLiteral,
		parseJavaTypeReferenceText(ctx.TypeName().GetText()),
	)
}

func (c *java20SourceFileCollector) EnterLocalVariableDeclaration(ctx *java20parser.LocalVariableDeclarationContext) {
	if ctx == nil || ctx.LocalVariableType() == nil || ctx.LocalVariableType().UnannType() == nil {
		return
	}
	c.addReferenceUsage(
		locationFromRule(ctx),
		ast.ReferenceUsageLocalVariableType,
		parseJavaTypeReferenceText(ctx.LocalVariableType().UnannType().GetText()),
	)
}

func (c *java20SourceFileCollector) EnterCatchFormalParameter(ctx *java20parser.CatchFormalParameterContext) {
	if ctx == nil || ctx.CatchType() == nil {
		return
	}

	catchType := ctx.CatchType()
	if catchType.UnannClassType() != nil {
		c.addReferenceUsage(
			locationFromRule(ctx),
			ast.ReferenceUsageCatchType,
			parseJavaTypeReferenceText(catchType.UnannClassType().GetText()),
		)
	}
	for _, classType := range catchType.AllClassType() {
		c.addReferenceUsage(
			locationFromRule(ctx),
			ast.ReferenceUsageCatchType,
			parseJavaTypeReferenceText(classType.GetText()),
		)
	}
}

func (c *java20SourceFileCollector) EnterMethodReference(ctx *java20parser.MethodReferenceContext) {
	if ctx == nil {
		return
	}

	kind := ast.ReferenceUsageMethodReference
	if ctx.NEW() != nil {
		kind = ast.ReferenceUsageConstructorReference
	}

	switch {
	case ctx.ReferenceType() != nil:
		c.addReferenceUsage(locationFromRule(ctx), kind, parseJavaTypeReferenceText(ctx.ReferenceType().GetText()))
	case ctx.TypeName() != nil:
		c.addReferenceUsage(locationFromRule(ctx), kind, parseJavaTypeReferenceText(ctx.TypeName().GetText()))
	case ctx.ClassType() != nil:
		c.addReferenceUsage(locationFromRule(ctx), kind, parseJavaTypeReferenceText(ctx.ClassType().GetText()))
	case ctx.ArrayType() != nil:
		c.addReferenceUsage(locationFromRule(ctx), kind, parseJavaTypeReferenceText(ctx.ArrayType().GetText()))
	}
}

func (c *java20SourceFileCollector) EnterPrimaryNoNewArray(ctx *java20parser.PrimaryNoNewArrayContext) {
	if ctx == nil || ctx.COLONCOLON() == nil {
		return
	}

	kind := ast.ReferenceUsageMethodReference
	if ctx.NEW() != nil {
		kind = ast.ReferenceUsageConstructorReference
	}

	switch {
	case ctx.ReferenceType() != nil:
		c.addReferenceUsage(locationFromRule(ctx), kind, parseJavaTypeReferenceText(ctx.ReferenceType().GetText()))
	case ctx.ClassType() != nil:
		c.addReferenceUsage(locationFromRule(ctx), kind, parseJavaTypeReferenceText(ctx.ClassType().GetText()))
	case ctx.TypeName() != nil:
		c.addReferenceUsage(locationFromRule(ctx), kind, parseJavaTypeReferenceText(ctx.TypeName().GetText()))
	case ctx.ArrayType() != nil:
		c.addReferenceUsage(locationFromRule(ctx), kind, parseJavaTypeReferenceText(ctx.ArrayType().GetText()))
	case ctx.ExpressionName() != nil:
		text := ctx.ExpressionName().GetText()
		if looksQualifiedJavaReferenceText(text) {
			c.addReferenceUsage(locationFromRule(ctx), kind, parseJavaTypeReferenceText(text))
		}
	}
}

func (c *java20SourceFileCollector) EnterCastExpression(ctx *java20parser.CastExpressionContext) {
	if ctx == nil || ctx.ReferenceType() == nil {
		return
	}

	c.addReferenceUsage(
		locationFromRule(ctx),
		ast.ReferenceUsageCastType,
		parseJavaTypeReferenceText(ctx.ReferenceType().GetText()),
	)
	for _, bound := range ctx.AllAdditionalBound() {
		if bound == nil || bound.InterfaceType() == nil {
			continue
		}
		c.addReferenceUsage(
			locationFromRule(ctx),
			ast.ReferenceUsageCastType,
			parseJavaTypeReferenceText(bound.InterfaceType().GetText()),
		)
	}
}

func (c *java20SourceFileCollector) EnterRelationalExpression(ctx *java20parser.RelationalExpressionContext) {
	if ctx == nil || ctx.INSTANCEOF() == nil {
		return
	}

	if ctx.ReferenceType() != nil {
		c.addReferenceUsage(
			locationFromRule(ctx),
			ast.ReferenceUsageInstanceofType,
			parseJavaTypeReferenceText(ctx.ReferenceType().GetText()),
		)
	}
	if ctx.Pattern() != nil {
		c.addPatternReferenceUsages(locationFromRule(ctx), ctx.Pattern())
	}
}

func (c *java20SourceFileCollector) addPatternReferenceUsages(loc ast.Location, pattern java20parser.IPatternContext) {
	if pattern == nil {
		return
	}

	if typePattern := pattern.TypePattern(); typePattern != nil && typePattern.LocalVariableDeclaration() != nil {
		localType := typePattern.LocalVariableDeclaration().LocalVariableType()
		if localType != nil && localType.UnannType() != nil {
			c.addReferenceUsage(loc, ast.ReferenceUsageInstanceofType, parseJavaTypeReferenceText(localType.UnannType().GetText()))
		}
	}

	recordPattern := pattern.RecordPattern()
	if recordPattern == nil || recordPattern.UnannType() == nil {
		return
	}
	c.addReferenceUsage(loc, ast.ReferenceUsageInstanceofType, parseJavaTypeReferenceText(recordPattern.UnannType().GetText()))
}

func (c *java20SourceFileCollector) addReferenceUsage(loc ast.Location, kind ast.ReferenceUsageKind, ref *ast.TypeReference) {
	if c == nil || ref == nil || ref.Name == "" || shouldSkipJavaTypeReference(ref) {
		return
	}

	path := ref.Name
	if ref.Package != "" {
		path = ref.Package + "." + ref.Name
	}

	c.file.References = append(c.file.References, ast.ReferenceUsage{
		Loc:  loc,
		Path: path,
		Kind: kind,
	})
	for i := range ref.TypeArguments {
		c.addReferenceUsage(loc, ast.ReferenceUsageTypeArgument, &ref.TypeArguments[i])
	}
}

func looksQualifiedJavaReferenceText(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] != '.' {
			continue
		}
		if i == 0 {
			return false
		}
		first := text[:i]
		return first[0] >= 'a' && first[0] <= 'z'
	}
	return false
}
