package parser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	ast "github.com/dh-kam/jkdeps/internal/ast"
	java20parser "github.com/dh-kam/jkdeps/internal/parsers/java20"
)

func BuildJavaSourceFile(source []byte, grammar JavaGrammar) (*ast.SourceFile, error) {
	switch grammar {
	case "", JavaGrammar11, JavaGrammar17, JavaGrammar20, JavaGrammar21, JavaGrammar25:
		return buildJava20SourceFile(source)
	default:
		pkgName, imports := extractJavaHeader(string(source))
		return &ast.SourceFile{
			Language: ast.LanguageJava,
			Package:  ast.PackageDeclaration{Name: pkgName},
			Imports:  convertImports(imports),
			Parsed:   true,
		}, nil
	}
}

func buildJava20SourceFile(source []byte) (*ast.SourceFile, error) {
	source = normalizeJavaSourceForANTLR(source)
	var (
		tree java20parser.ICompilationUnitContext
		err  error
	)

	err = safeParse(func() error {
		listener := newSyntaxErrorListener(fileErrorMessageLimit)
		input := antlr.NewInputStream(string(source))
		lexer := java20parser.NewJava20Lexer(input)
		lexer.RemoveErrorListeners()
		lexer.AddErrorListener(listener)

		tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
		parser := java20parser.NewJava20Parser(tokens)
		parser.RemoveErrorListeners()
		parser.AddErrorListener(listener)
		parser.BuildParseTrees = true
		parser.GetInterpreter().SetPredictionMode(antlr.PredictionModeLL)
		tree = parser.CompilationUnit()
		if parseErr := listener.Err(); parseErr != nil {
			return parseErr
		}
		if parser.HasError() {
			return parserRecognitionError(parser.GetError())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	collector := newJava20SourceFileCollector()
	antlr.ParseTreeWalkerDefault.Walk(collector, tree)
	collector.finalizeImports()
	return collector.file, nil
}

type java20SourceFileCollector struct {
	java20parser.BaseJava20ParserListener
	file      *ast.SourceFile
	declStack []*ast.ClassDeclaration
}

func newJava20SourceFileCollector() *java20SourceFileCollector {
	return &java20SourceFileCollector{
		file: &ast.SourceFile{
			Language: ast.LanguageJava,
			Parsed:   true,
		},
	}
}

func (c *java20SourceFileCollector) EnterPackageDeclaration(ctx *java20parser.PackageDeclarationContext) {
	idents := ctx.AllIdentifier()
	if len(idents) == 0 {
		return
	}

	parts := make([]string, 0, len(idents))
	for _, ident := range idents {
		parts = append(parts, ident.GetText())
	}
	c.file.Package = ast.PackageDeclaration{
		Loc:  locationFromRule(ctx),
		Name: strings.Join(parts, "."),
	}
}

func (c *java20SourceFileCollector) EnterImportDeclaration(ctx *java20parser.ImportDeclarationContext) {
	switch {
	case ctx.SingleTypeImportDeclaration() != nil:
		importCtx := ctx.SingleTypeImportDeclaration()
		c.file.Imports = append(c.file.Imports, ast.Import{
			Loc:  locationFromRule(importCtx),
			Path: normalizePath(importCtx.TypeName().GetText()),
		})
	case ctx.TypeImportOnDemandDeclaration() != nil:
		importCtx := ctx.TypeImportOnDemandDeclaration()
		c.file.Imports = append(c.file.Imports, ast.Import{
			Loc:        locationFromRule(importCtx),
			Path:       normalizePath(importCtx.PackageOrTypeName().GetText()) + ".*",
			IsWildcard: true,
		})
	case ctx.SingleStaticImportDeclaration() != nil:
		importCtx := ctx.SingleStaticImportDeclaration()
		c.file.Imports = append(c.file.Imports, ast.Import{
			Loc:      locationFromRule(importCtx),
			Path:     normalizePath(importCtx.TypeName().GetText() + "." + importCtx.Identifier().GetText()),
			IsStatic: true,
		})
	case ctx.StaticImportOnDemandDeclaration() != nil:
		importCtx := ctx.StaticImportOnDemandDeclaration()
		c.file.Imports = append(c.file.Imports, ast.Import{
			Loc:        locationFromRule(importCtx),
			Path:       normalizePath(importCtx.TypeName().GetText()) + ".*",
			IsStatic:   true,
			IsWildcard: true,
		})
	}
}

func (c *java20SourceFileCollector) EnterClassDeclaration(ctx *java20parser.ClassDeclarationContext) {
	decl := buildJavaClassDeclaration(ctx)
	if decl == nil {
		return
	}
	c.pushClassDeclaration(decl)
}

func (c *java20SourceFileCollector) ExitClassDeclaration(_ *java20parser.ClassDeclarationContext) {
	c.popClassDeclaration()
}

func (c *java20SourceFileCollector) EnterInterfaceDeclaration(ctx *java20parser.InterfaceDeclarationContext) {
	decl := buildJavaInterfaceDeclaration(ctx)
	if decl == nil {
		return
	}
	c.pushClassDeclaration(decl)
}

func (c *java20SourceFileCollector) ExitInterfaceDeclaration(_ *java20parser.InterfaceDeclarationContext) {
	c.popClassDeclaration()
}

func (c *java20SourceFileCollector) EnterFieldDeclaration(ctx *java20parser.FieldDeclarationContext) {
	c.addFieldMembers(
		locationFromRule(ctx),
		convertJavaFieldModifiers(ctx.AllFieldModifier()),
		parseJavaTypeReferenceText(ctx.UnannType().GetText()),
		ctx.VariableDeclaratorList(),
	)
}

func (c *java20SourceFileCollector) EnterConstantDeclaration(ctx *java20parser.ConstantDeclarationContext) {
	c.addFieldMembers(
		locationFromRule(ctx),
		convertJavaConstantModifiers(ctx.AllConstantModifier()),
		parseJavaTypeReferenceText(ctx.UnannType().GetText()),
		ctx.VariableDeclaratorList(),
	)
}

func (c *java20SourceFileCollector) EnterMethodDeclaration(ctx *java20parser.MethodDeclarationContext) {
	method := buildJavaMethodDeclaration(
		locationFromRule(ctx),
		convertJavaMethodModifiers(ctx.AllMethodModifier()),
		ctx.MethodHeader(),
		false,
	)
	if method == nil {
		return
	}
	c.addMember(method)
}

func (c *java20SourceFileCollector) EnterInterfaceMethodDeclaration(ctx *java20parser.InterfaceMethodDeclarationContext) {
	method := buildJavaMethodDeclaration(
		locationFromRule(ctx),
		convertJavaInterfaceMethodModifiers(ctx.AllInterfaceMethodModifier()),
		ctx.MethodHeader(),
		true,
	)
	if method == nil {
		return
	}
	c.addMember(method)
}

func (c *java20SourceFileCollector) EnterConstructorDeclaration(ctx *java20parser.ConstructorDeclarationContext) {
	if len(c.declStack) == 0 {
		return
	}

	constructor := &ast.Constructor{
		Loc:        locationFromRule(ctx),
		Modifiers:  convertJavaConstructorModifiers(ctx.AllConstructorModifier()),
		Parameters: extractJavaParameters(ctx.ConstructorDeclarator().FormalParameterList()),
	}
	c.addMember(constructor)
}

func (c *java20SourceFileCollector) pushClassDeclaration(decl *ast.ClassDeclaration) {
	if len(c.declStack) == 0 {
		c.file.Declarations = append(c.file.Declarations, decl)
	} else {
		c.declStack[len(c.declStack)-1].Members = append(c.declStack[len(c.declStack)-1].Members, decl)
	}
	c.declStack = append(c.declStack, decl)
}

func (c *java20SourceFileCollector) popClassDeclaration() {
	if len(c.declStack) == 0 {
		return
	}
	c.declStack = c.declStack[:len(c.declStack)-1]
}

func (c *java20SourceFileCollector) addMember(member ast.Member) {
	if len(c.declStack) == 0 {
		return
	}
	c.declStack[len(c.declStack)-1].Members = append(c.declStack[len(c.declStack)-1].Members, member)
}

func (c *java20SourceFileCollector) addFieldMembers(
	loc ast.Location,
	modifiers []ast.Modifier,
	baseType *ast.TypeReference,
	list java20parser.IVariableDeclaratorListContext,
) {
	if len(c.declStack) == 0 || list == nil {
		return
	}
	for _, decl := range list.AllVariableDeclarator() {
		c.addMember(&ast.VariableDeclaration{
			Loc:       loc,
			Name:      decl.VariableDeclaratorId().Identifier().GetText(),
			Modifiers: append([]ast.Modifier(nil), modifiers...),
			Type:      cloneTypeReference(baseType),
		})
	}
}

func (c *java20SourceFileCollector) finalizeImports() {
	if len(c.file.Imports) == 0 {
		return
	}
	sort.Slice(c.file.Imports, func(i, j int) bool {
		if c.file.Imports[i].Path != c.file.Imports[j].Path {
			return c.file.Imports[i].Path < c.file.Imports[j].Path
		}
		if c.file.Imports[i].IsStatic != c.file.Imports[j].IsStatic {
			return !c.file.Imports[i].IsStatic
		}
		return !c.file.Imports[i].IsWildcard && c.file.Imports[j].IsWildcard
	})
}

func buildJavaClassDeclaration(ctx *java20parser.ClassDeclarationContext) *ast.ClassDeclaration {
	switch {
	case ctx.NormalClassDeclaration() != nil:
		return buildNormalJavaClassDeclaration(ctx.NormalClassDeclaration())
	case ctx.EnumDeclaration() != nil:
		return buildJavaEnumDeclaration(ctx.EnumDeclaration())
	case ctx.RecordDeclaration() != nil:
		return buildJavaRecordDeclaration(ctx.RecordDeclaration())
	default:
		return nil
	}
}

func buildJavaInterfaceDeclaration(ctx *java20parser.InterfaceDeclarationContext) *ast.ClassDeclaration {
	switch {
	case ctx.NormalInterfaceDeclaration() != nil:
		return buildNormalJavaInterfaceDeclaration(ctx.NormalInterfaceDeclaration())
	case ctx.AnnotationInterfaceDeclaration() != nil:
		return buildJavaAnnotationDeclaration(ctx.AnnotationInterfaceDeclaration())
	default:
		return nil
	}
}

func buildNormalJavaClassDeclaration(ctx java20parser.INormalClassDeclarationContext) *ast.ClassDeclaration {
	decl := &ast.ClassDeclaration{
		Loc:        locationFromRule(ctx),
		Name_:      ctx.TypeIdentifier().GetText(),
		Modifiers_: convertJavaClassModifiers(ctx.AllClassModifier()),
		Kind_:      ast.KindClass,
	}
	if extends := ctx.ClassExtends(); extends != nil {
		decl.SuperClass = parseJavaTypeReferenceText(extends.ClassType().GetText())
	}
	if impl := ctx.ClassImplements(); impl != nil {
		decl.SuperInterfaces = extractJavaInterfaceTypeList(impl.InterfaceTypeList())
	}
	return decl
}

func buildJavaEnumDeclaration(ctx java20parser.IEnumDeclarationContext) *ast.ClassDeclaration {
	decl := &ast.ClassDeclaration{
		Loc:        locationFromRule(ctx),
		Name_:      ctx.TypeIdentifier().GetText(),
		Modifiers_: convertJavaClassModifiers(ctx.AllClassModifier()),
		Kind_:      ast.KindEnum,
	}
	if impl := ctx.ClassImplements(); impl != nil {
		decl.SuperInterfaces = extractJavaInterfaceTypeList(impl.InterfaceTypeList())
	}
	return decl
}

func buildJavaRecordDeclaration(ctx java20parser.IRecordDeclarationContext) *ast.ClassDeclaration {
	decl := &ast.ClassDeclaration{
		Loc:              locationFromRule(ctx),
		Name_:            ctx.TypeIdentifier().GetText(),
		Modifiers_:       convertJavaClassModifiers(ctx.AllClassModifier()),
		Kind_:            ast.KindRecord,
		IsRecord:         true,
		RecordComponents: extractJavaRecordComponents(ctx.RecordHeader()),
	}
	if impl := ctx.ClassImplements(); impl != nil {
		decl.SuperInterfaces = extractJavaInterfaceTypeList(impl.InterfaceTypeList())
	}
	return decl
}

func buildNormalJavaInterfaceDeclaration(ctx java20parser.INormalInterfaceDeclarationContext) *ast.ClassDeclaration {
	decl := &ast.ClassDeclaration{
		Loc:        locationFromRule(ctx),
		Name_:      ctx.TypeIdentifier().GetText(),
		Modifiers_: convertJavaInterfaceModifiers(ctx.AllInterfaceModifier()),
		Kind_:      ast.KindInterface,
	}
	if extends := ctx.InterfaceExtends(); extends != nil {
		decl.SuperInterfaces = extractJavaInterfaceTypeList(extends.InterfaceTypeList())
	}
	return decl
}

func buildJavaAnnotationDeclaration(ctx java20parser.IAnnotationInterfaceDeclarationContext) *ast.ClassDeclaration {
	return &ast.ClassDeclaration{
		Loc:        locationFromRule(ctx),
		Name_:      ctx.TypeIdentifier().GetText(),
		Modifiers_: convertJavaInterfaceModifiers(ctx.AllInterfaceModifier()),
		Kind_:      ast.KindAnnotation,
	}
}

func buildJavaMethodDeclaration(
	loc ast.Location,
	modifiers []ast.Modifier,
	header java20parser.IMethodHeaderContext,
	fromInterface bool,
) *ast.FunctionDeclaration {
	if header == nil || header.MethodDeclarator() == nil {
		return nil
	}

	method := &ast.FunctionDeclaration{
		Loc:        loc,
		Name_:      header.MethodDeclarator().Identifier().GetText(),
		Modifiers_: append([]ast.Modifier(nil), modifiers...),
		Parameters: extractJavaParameters(header.MethodDeclarator().FormalParameterList()),
		Throws:     extractJavaThrows(header.ThrowsT()),
	}
	if header.Result() != nil && header.Result().UnannType() != nil {
		method.ReturnType = parseJavaTypeReferenceText(header.Result().UnannType().GetText())
	}
	if fromInterface {
		method.IsDefault = hasModifier(modifiers, ast.ModifierDefault)
		method.IsAbstract = !method.IsDefault && !hasModifier(modifiers, ast.ModifierStatic) && !hasModifier(modifiers, ast.ModifierPrivate)
	} else {
		method.IsAbstract = hasModifier(modifiers, ast.ModifierAbstract)
	}
	return method
}

func extractJavaParameters(list java20parser.IFormalParameterListContext) []ast.Parameter {
	if list == nil {
		return nil
	}

	out := make([]ast.Parameter, 0, len(list.AllFormalParameter())+1)
	for _, param := range list.AllFormalParameter() {
		if varArg := param.VariableArityParameter(); varArg != nil {
			out = append(out, ast.Parameter{
				Loc:      locationFromRule(varArg),
				Name:     varArg.Identifier().GetText(),
				Type:     parseJavaTypeReferenceText(varArg.UnannType().GetText()),
				IsVarArg: true,
			})
			continue
		}
		out = append(out, ast.Parameter{
			Loc:  locationFromRule(param),
			Name: param.VariableDeclaratorId().Identifier().GetText(),
			Type: parseJavaTypeReferenceText(param.UnannType().GetText()),
		})
	}
	return out
}

func extractJavaThrows(throws java20parser.IThrowsTContext) []ast.TypeReference {
	if throws == nil || throws.ExceptionTypeList() == nil {
		return nil
	}

	out := make([]ast.TypeReference, 0, len(throws.ExceptionTypeList().AllExceptionType()))
	for _, exceptionType := range throws.ExceptionTypeList().AllExceptionType() {
		ref := parseJavaTypeReferenceText(exceptionType.GetText())
		if ref == nil {
			continue
		}
		out = append(out, *ref)
	}
	return out
}

func extractJavaInterfaceTypeList(list java20parser.IInterfaceTypeListContext) []ast.TypeReference {
	if list == nil {
		return nil
	}

	out := make([]ast.TypeReference, 0, len(list.AllInterfaceType()))
	for _, interfaceType := range list.AllInterfaceType() {
		ref := parseJavaTypeReferenceText(interfaceType.GetText())
		if ref == nil {
			continue
		}
		out = append(out, *ref)
	}
	return out
}

func extractJavaRecordComponents(header java20parser.IRecordHeaderContext) []ast.VariableDeclaration {
	if header == nil || header.RecordComponentList() == nil {
		return nil
	}

	out := make([]ast.VariableDeclaration, 0, len(header.RecordComponentList().AllRecordComponent()))
	for _, component := range header.RecordComponentList().AllRecordComponent() {
		if variableArity := component.VariableArityRecordComponent(); variableArity != nil {
			out = append(out, ast.VariableDeclaration{
				Loc:  locationFromRule(component),
				Name: variableArity.Identifier().GetText(),
				Type: parseJavaTypeReferenceText(variableArity.UnannType().GetText()),
			})
			continue
		}
		out = append(out, ast.VariableDeclaration{
			Loc:  locationFromRule(component),
			Name: component.Identifier().GetText(),
			Type: parseJavaTypeReferenceText(component.UnannType().GetText()),
		})
	}
	return out
}

func locationFromRule(ctx antlr.ParserRuleContext) ast.Location {
	if ctx == nil || ctx.GetStart() == nil {
		return ast.Location{}
	}

	start := ctx.GetStart()
	end := ctx.GetStop()
	endLine := start.GetLine()
	endColumn := start.GetColumn() + len(ctx.GetText())
	if end != nil {
		endLine = end.GetLine()
		endColumn = end.GetColumn() + len(end.GetText())
	}

	return ast.Location{
		Span: ast.Span{
			Start: ast.Position{Line: start.GetLine(), Column: start.GetColumn() + 1},
			End:   ast.Position{Line: endLine, Column: endColumn + 1},
		},
	}
}

func convertJavaClassModifiers(modifiers []java20parser.IClassModifierContext) []ast.Modifier {
	out := make([]ast.Modifier, 0, len(modifiers))
	for _, modifier := range modifiers {
		switch {
		case modifier.PUBLIC() != nil:
			out = append(out, ast.ModifierPublic)
		case modifier.PROTECTED() != nil:
			out = append(out, ast.ModifierProtected)
		case modifier.PRIVATE() != nil:
			out = append(out, ast.ModifierPrivate)
		case modifier.ABSTRACT() != nil:
			out = append(out, ast.ModifierAbstract)
		case modifier.STATIC() != nil:
			out = append(out, ast.ModifierStatic)
		case modifier.FINAL() != nil:
			out = append(out, ast.ModifierFinal)
		case modifier.SEALED() != nil:
			out = append(out, ast.ModifierSealed)
		case modifier.NONSEALED() != nil:
			out = append(out, ast.ModifierOpen)
		case modifier.STRICTFP() != nil:
			out = append(out, ast.ModifierStrictfp)
		}
	}
	return out
}

func convertJavaInterfaceModifiers(modifiers []java20parser.IInterfaceModifierContext) []ast.Modifier {
	out := make([]ast.Modifier, 0, len(modifiers))
	for _, modifier := range modifiers {
		switch {
		case modifier.PUBLIC() != nil:
			out = append(out, ast.ModifierPublic)
		case modifier.PROTECTED() != nil:
			out = append(out, ast.ModifierProtected)
		case modifier.PRIVATE() != nil:
			out = append(out, ast.ModifierPrivate)
		case modifier.ABSTRACT() != nil:
			out = append(out, ast.ModifierAbstract)
		case modifier.STATIC() != nil:
			out = append(out, ast.ModifierStatic)
		case modifier.SEALED() != nil:
			out = append(out, ast.ModifierSealed)
		case modifier.NONSEALED() != nil:
			out = append(out, ast.ModifierOpen)
		case modifier.STRICTFP() != nil:
			out = append(out, ast.ModifierStrictfp)
		}
	}
	return out
}

func convertJavaFieldModifiers(modifiers []java20parser.IFieldModifierContext) []ast.Modifier {
	out := make([]ast.Modifier, 0, len(modifiers))
	for _, modifier := range modifiers {
		switch {
		case modifier.PUBLIC() != nil:
			out = append(out, ast.ModifierPublic)
		case modifier.PROTECTED() != nil:
			out = append(out, ast.ModifierProtected)
		case modifier.PRIVATE() != nil:
			out = append(out, ast.ModifierPrivate)
		case modifier.STATIC() != nil:
			out = append(out, ast.ModifierStatic)
		case modifier.FINAL() != nil:
			out = append(out, ast.ModifierFinal)
		case modifier.TRANSIENT() != nil:
			out = append(out, ast.ModifierTransient)
		case modifier.VOLATILE() != nil:
			out = append(out, ast.ModifierVolatile)
		}
	}
	return out
}

func convertJavaConstantModifiers(modifiers []java20parser.IConstantModifierContext) []ast.Modifier {
	out := make([]ast.Modifier, 0, len(modifiers))
	for _, modifier := range modifiers {
		switch {
		case modifier.PUBLIC() != nil:
			out = append(out, ast.ModifierPublic)
		case modifier.STATIC() != nil:
			out = append(out, ast.ModifierStatic)
		case modifier.FINAL() != nil:
			out = append(out, ast.ModifierFinal)
		}
	}
	return out
}

func convertJavaMethodModifiers(modifiers []java20parser.IMethodModifierContext) []ast.Modifier {
	out := make([]ast.Modifier, 0, len(modifiers))
	for _, modifier := range modifiers {
		switch {
		case modifier.PUBLIC() != nil:
			out = append(out, ast.ModifierPublic)
		case modifier.PROTECTED() != nil:
			out = append(out, ast.ModifierProtected)
		case modifier.PRIVATE() != nil:
			out = append(out, ast.ModifierPrivate)
		case modifier.ABSTRACT() != nil:
			out = append(out, ast.ModifierAbstract)
		case modifier.STATIC() != nil:
			out = append(out, ast.ModifierStatic)
		case modifier.FINAL() != nil:
			out = append(out, ast.ModifierFinal)
		case modifier.SYNCHRONIZED() != nil:
			out = append(out, ast.ModifierSynchronized)
		case modifier.NATIVE() != nil:
			out = append(out, ast.ModifierNative)
		case modifier.STRICTFP() != nil:
			out = append(out, ast.ModifierStrictfp)
		}
	}
	return out
}

func convertJavaInterfaceMethodModifiers(modifiers []java20parser.IInterfaceMethodModifierContext) []ast.Modifier {
	out := make([]ast.Modifier, 0, len(modifiers))
	for _, modifier := range modifiers {
		switch {
		case modifier.PUBLIC() != nil:
			out = append(out, ast.ModifierPublic)
		case modifier.PRIVATE() != nil:
			out = append(out, ast.ModifierPrivate)
		case modifier.ABSTRACT() != nil:
			out = append(out, ast.ModifierAbstract)
		case modifier.DEFAULT() != nil:
			out = append(out, ast.ModifierDefault)
		case modifier.STATIC() != nil:
			out = append(out, ast.ModifierStatic)
		case modifier.STRICTFP() != nil:
			out = append(out, ast.ModifierStrictfp)
		}
	}
	return out
}

func convertJavaConstructorModifiers(modifiers []java20parser.IConstructorModifierContext) []ast.Modifier {
	out := make([]ast.Modifier, 0, len(modifiers))
	for _, modifier := range modifiers {
		switch {
		case modifier.PUBLIC() != nil:
			out = append(out, ast.ModifierPublic)
		case modifier.PROTECTED() != nil:
			out = append(out, ast.ModifierProtected)
		case modifier.PRIVATE() != nil:
			out = append(out, ast.ModifierPrivate)
		}
	}
	return out
}

func hasModifier(modifiers []ast.Modifier, want ast.Modifier) bool {
	for _, modifier := range modifiers {
		if modifier == want {
			return true
		}
	}
	return false
}

func cloneTypeReference(in *ast.TypeReference) *ast.TypeReference {
	if in == nil {
		return nil
	}
	cloned := *in
	if len(in.TypeArguments) != 0 {
		cloned.TypeArguments = append([]ast.TypeReference(nil), in.TypeArguments...)
	}
	return &cloned
}

func (b *javaASTBuilder) Build(source []byte, lang ast.SourceLanguage) (*ast.SourceFile, error) {
	sourceFile, err := BuildJavaSourceFile(source, b.grammar)
	if err != nil {
		return nil, fmt.Errorf("build java ast: %w", err)
	}
	sourceFile.Language = lang
	return sourceFile, nil
}
