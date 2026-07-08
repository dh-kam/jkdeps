package kotlincompilergolang

import (
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	kotlinparser "github.com/dh-kam/jkdeps/internal/parsers/kotlin"
)

type parseTreeCollector struct {
	*kotlinparser.BaseKotlinParserListener
	packageName   string
	imports       map[string]struct{}
	declarations  []TopLevelDeclaration
	typeScopes    []string
	functionDepth int
}

func newParseTreeCollector() *parseTreeCollector {
	return &parseTreeCollector{
		BaseKotlinParserListener: &kotlinparser.BaseKotlinParserListener{},
		imports:                  make(map[string]struct{}, 16),
		declarations:             make([]TopLevelDeclaration, 0, 8),
		typeScopes:               make([]string, 0, 4),
	}
}

func (c *parseTreeCollector) EnterPackageHeader(ctx *kotlinparser.PackageHeaderContext) {
	if c.packageName != "" {
		return
	}
	if id := ctx.Identifier(); id != nil {
		c.packageName = normalizeQualifiedName(id.GetText())
	}
}

func (c *parseTreeCollector) EnterImportHeader(ctx *kotlinparser.ImportHeaderContext) {
	id := ctx.Identifier()
	if id == nil {
		return
	}
	imp := normalizeQualifiedName(id.GetText())
	if imp == "" {
		return
	}
	if ctx.MULT() != nil {
		imp += ".*"
	}
	c.imports[imp] = struct{}{}
}

func (c *parseTreeCollector) EnterTopLevelObject(ctx *kotlinparser.TopLevelObjectContext) {
	if functionDecl := ctx.FunctionDeclaration(); functionDecl != nil {
		name := ""
		if id := functionDecl.Identifier(); id != nil {
			name = normalizeIdentifier(id.GetText())
		}
		c.addDeclaration(DeclFunction, name, startLine(functionDecl), extractModifiers(functionDecl.ModifierList()))
		return
	}
	if propertyDecl := ctx.PropertyDeclaration(); propertyDecl != nil {
		modifiers := extractModifiers(propertyDecl.ModifierList())
		if variableDecl := propertyDecl.VariableDeclaration(); variableDecl != nil {
			c.addDeclaration(DeclProperty, extractSimpleIdentifier(variableDecl.SimpleIdentifier()), startLine(propertyDecl), modifiers)
			return
		}
		if multiDecl := propertyDecl.MultiVariableDeclaration(); multiDecl != nil {
			for _, variableDecl := range multiDecl.AllVariableDeclaration() {
				c.addDeclaration(DeclProperty, extractSimpleIdentifier(variableDecl.SimpleIdentifier()), startLine(variableDecl), modifiers)
			}
		}
	}
}

func (c *parseTreeCollector) EnterClassDeclaration(ctx *kotlinparser.ClassDeclarationContext) {
	name := extractSimpleIdentifier(ctx.SimpleIdentifier())
	if name == "" {
		return
	}
	qualified := c.qualifiedTypeName(name)
	if c.functionDepth == 0 {
		kind := DeclClass
		if ctx.INTERFACE() != nil {
			kind = DeclInterface
		}
		c.addDeclaration(kind, qualified, startLine(ctx), extractModifiers(ctx.ModifierList()))
	}
	c.typeScopes = append(c.typeScopes, name)
}

func (c *parseTreeCollector) ExitClassDeclaration(ctx *kotlinparser.ClassDeclarationContext) {
	name := extractSimpleIdentifier(ctx.SimpleIdentifier())
	if name == "" {
		return
	}
	c.popTypeScope(name)
}

func (c *parseTreeCollector) EnterObjectDeclaration(ctx *kotlinparser.ObjectDeclarationContext) {
	name := extractSimpleIdentifier(ctx.SimpleIdentifier())
	if name == "" {
		return
	}
	qualified := c.qualifiedTypeName(name)
	if c.functionDepth == 0 {
		c.addDeclaration(DeclObject, qualified, startLine(ctx), extractModifiers(ctx.ModifierList()))
	}
	c.typeScopes = append(c.typeScopes, name)
}

func (c *parseTreeCollector) ExitObjectDeclaration(ctx *kotlinparser.ObjectDeclarationContext) {
	name := extractSimpleIdentifier(ctx.SimpleIdentifier())
	if name == "" {
		return
	}
	c.popTypeScope(name)
}

func (c *parseTreeCollector) EnterCompanionObject(ctx *kotlinparser.CompanionObjectContext) {
	name := extractSimpleIdentifier(ctx.SimpleIdentifier())
	if name == "" {
		name = "Companion"
	}
	qualified := c.qualifiedTypeName(name)
	if c.functionDepth == 0 {
		c.addDeclaration(DeclObject, qualified, startLine(ctx), extractCompanionModifiers(ctx))
	}
	c.typeScopes = append(c.typeScopes, name)
}

func (c *parseTreeCollector) ExitCompanionObject(ctx *kotlinparser.CompanionObjectContext) {
	name := extractSimpleIdentifier(ctx.SimpleIdentifier())
	if name == "" {
		name = "Companion"
	}
	c.popTypeScope(name)
}

func (c *parseTreeCollector) EnterTypeAlias(ctx *kotlinparser.TypeAliasContext) {
	name := extractSimpleIdentifier(ctx.SimpleIdentifier())
	if name == "" {
		return
	}
	if c.functionDepth > 0 || len(c.typeScopes) > 0 {
		return
	}
	qualified := c.qualifiedTypeName(name)
	c.addDeclaration(DeclTypeAlias, qualified, startLine(ctx), extractModifiers(ctx.ModifierList()))
}

func (c *parseTreeCollector) EnterFunctionDeclaration(_ *kotlinparser.FunctionDeclarationContext) {
	c.functionDepth++
}

func (c *parseTreeCollector) ExitFunctionDeclaration(_ *kotlinparser.FunctionDeclarationContext) {
	if c.functionDepth > 0 {
		c.functionDepth--
	}
}

func (c *parseTreeCollector) PackageName() string {
	return c.packageName
}

func (c *parseTreeCollector) Imports() []string {
	if len(c.imports) == 0 {
		return nil
	}
	out := make([]string, 0, len(c.imports))
	for imp := range c.imports {
		out = append(out, imp)
	}
	sort.Strings(out)
	return out
}

func (c *parseTreeCollector) Declarations() []TopLevelDeclaration {
	if len(c.declarations) == 0 {
		return nil
	}
	out := make([]TopLevelDeclaration, len(c.declarations))
	copy(out, c.declarations)
	return out
}

func (c *parseTreeCollector) addDeclaration(kind DeclarationKind, name string, line int, modifiers []string) {
	name = normalizeIdentifier(name)
	if name == "" {
		return
	}
	c.declarations = append(c.declarations, TopLevelDeclaration{
		Kind:      kind,
		Name:      name,
		Line:      line,
		Modifiers: copySlice(modifiers),
	})
}

func (c *parseTreeCollector) qualifiedTypeName(name string) string {
	name = normalizeIdentifier(name)
	if name == "" {
		return ""
	}
	if len(c.typeScopes) == 0 {
		return name
	}
	return strings.Join(append(copySlice(c.typeScopes), name), ".")
}

func (c *parseTreeCollector) popTypeScope(name string) {
	if len(c.typeScopes) == 0 {
		return
	}
	last := c.typeScopes[len(c.typeScopes)-1]
	if last == name {
		c.typeScopes = c.typeScopes[:len(c.typeScopes)-1]
		return
	}
	// parser recovery can desynchronize enter/exit events; remove nearest match.
	for i := len(c.typeScopes) - 1; i >= 0; i-- {
		if c.typeScopes[i] == name {
			c.typeScopes = append(c.typeScopes[:i], c.typeScopes[i+1:]...)
			return
		}
	}
}

func startLine(ctx antlr.ParserRuleContext) int {
	if ctx == nil {
		return 0
	}
	start := ctx.GetStart()
	if start == nil {
		return 0
	}
	return start.GetLine()
}

func extractSimpleIdentifier(ctx kotlinparser.ISimpleIdentifierContext) string {
	if ctx == nil {
		return ""
	}
	return normalizeIdentifier(ctx.GetText())
}

func extractModifiers(ctx kotlinparser.IModifierListContext) []string {
	if ctx == nil {
		return nil
	}
	modifiers := ctx.AllModifier()
	if len(modifiers) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(modifiers))
	out := make([]string, 0, len(modifiers))
	for _, modifier := range modifiers {
		if modifier == nil {
			continue
		}
		value := strings.ToLower(strings.TrimSpace(modifier.GetText()))
		if value == "" {
			continue
		}
		value = strings.Join(strings.Fields(value), "")
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func extractCompanionModifiers(ctx *kotlinparser.CompanionObjectContext) []string {
	if ctx == nil {
		return nil
	}
	modifierLists := ctx.AllModifierList()
	if len(modifierLists) == 0 {
		return nil
	}
	merged := make([]string, 0, len(modifierLists))
	for _, list := range modifierLists {
		merged = mergeUniqueStrings(merged, extractModifiers(list))
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

func normalizeQualifiedName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "`", "")
	value = strings.Join(strings.Fields(value), "")
	value = strings.Trim(value, ".")
	return value
}

func normalizeIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "`", "")
	value = strings.Join(strings.Fields(value), "")
	return value
}
