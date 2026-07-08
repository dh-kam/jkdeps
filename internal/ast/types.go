// Package ast provides common Abstract Syntax Tree (AST) node types
// for Java and Kotlin source code parsing.
//
// The design follows these principles:
// - Language-agnostic core types with language-specific extensions
// - Source location tracking for all nodes
// - Minimal and focused node types (Single Responsibility Principle)
// - Interface-based design for flexibility and testability
package ast

import (
	"time"
)

// SourceLanguage represents the programming language
type SourceLanguage string

const (
	LanguageJava   SourceLanguage = "java"
	LanguageKotlin SourceLanguage = "kotlin"
)

// Position represents a source code location
type Position struct {
	Line   int `json:"line"`   // 1-based line number
	Column int `json:"column"` // 1-based column number
}

// Span represents a range in source code
type Span struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents source location information
type Location struct {
	FilePath string `json:"file_path"`
	Span     Span   `json:"span,omitempty"`
}

// Node is the base interface for all AST nodes
type Node interface {
	// Location returns the source location of this node
	Location() Location
	// Children returns child nodes (for traversal)
	Children() []Node
}

// Modifier represents access modifiers and other modifiers
type Modifier string

const (
	ModifierPublic       Modifier = "public"
	ModifierPrivate      Modifier = "private"
	ModifierProtected    Modifier = "protected"
	ModifierInternal     Modifier = "internal"      // Kotlin
	ModifierPackageLocal Modifier = "package-local" // Java (default)
	ModifierStatic       Modifier = "static"
	ModifierFinal        Modifier = "final"
	ModifierAbstract     Modifier = "abstract"
	ModifierOverride     Modifier = "override" // Kotlin
	ModifierOpen         Modifier = "open"     // Kotlin
	ModifierSealed       Modifier = "sealed"
	ModifierData         Modifier = "data"     // Kotlin
	ModifierInline       Modifier = "inline"   // Kotlin
	ModifierInfix        Modifier = "infix"    // Kotlin
	ModifierExternal     Modifier = "external" // Kotlin
	ModifierSuspend      Modifier = "suspend"  // Kotlin
	ModifierTailrec      Modifier = "tailrec"  // Kotlin
	ModifierConst        Modifier = "const"
	ModifierLateinit     Modifier = "lateinit"    // Kotlin
	ModifierVararg       Modifier = "vararg"      // Kotlin
	ModifierNoinline     Modifier = "noinline"    // Kotlin
	ModifierCrossinline  Modifier = "crossinline" // Kotlin
	ModifierReified      Modifier = "reified"     // Kotlin
	ModifierExpect       Modifier = "expect"      // Kotlin (multiplatform)
	ModifierActual       Modifier = "actual"      // Kotlin (multiplatform)
	ModifierCompanion    Modifier = "companion"   // Kotlin
	ModifierEnum         Modifier = "enum"
	ModifierAnnotation   Modifier = "annotation"
	ModifierTransient    Modifier = "transient"    // Java
	ModifierVolatile     Modifier = "volatile"     // Java
	ModifierSynchronized Modifier = "synchronized" // Java
	ModifierNative       Modifier = "native"       // Java
	ModifierStrictfp     Modifier = "strictfp"     // Java
	ModifierDefault      Modifier = "default"      // Java (interface methods)
)

// Import represents an import declaration
type Import struct {
	Loc        Location `json:"location,omitempty"`
	Path       string   `json:"path"`            // e.g., "java.util.List" or "kotlinx.coroutines.launch"
	IsStatic   bool     `json:"is_static"`       // Java static import
	IsAlias    bool     `json:"is_alias"`        // Kotlin import alias (e.g., "import foo as bar")
	Alias      string   `json:"alias,omitempty"` // Alias name if IsAlias is true
	IsWildcard bool     `json:"is_wildcard"`     // e.g., "import java.util.*"
}

// PackageDeclaration represents a package declaration
type PackageDeclaration struct {
	Loc  Location `json:"location,omitempty"`
	Name string   `json:"name"` // e.g., "com.example.package"
}

// Diagnostic represents a parsing or compilation error/warning
type Diagnostic struct {
	Loc      Location `json:"location,omitempty"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Code     string   `json:"code,omitempty"` // Error code for categorization
}

// Severity represents the severity level of a diagnostic
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// SourceFile represents a parsed source file with its metadata
type SourceFile struct {
	Language     SourceLanguage     `json:"language"`
	FilePath     string             `json:"file_path"`
	Package      PackageDeclaration `json:"package,omitempty"`
	Imports      []Import           `json:"imports,omitempty"`
	References   []ReferenceUsage   `json:"references,omitempty"`
	Declarations []Declaration      `json:"declarations,omitempty"`
	Diagnostics  []Diagnostic       `json:"diagnostics,omitempty"`
	Parsed       bool               `json:"parsed"`
	Duration     time.Duration      `json:"duration"`
}

// Declaration represents a top-level declaration (class, interface, function, etc.)
type Declaration interface {
	Node
	// Name returns the declaration name
	Name() string
	// Modifiers returns the declaration modifiers
	Modifiers() []Modifier
	// Kind returns the declaration kind
	Kind() DeclarationKind
}

// DeclarationKind represents the kind of declaration
type DeclarationKind string

const (
	KindClass           DeclarationKind = "class"
	KindInterface       DeclarationKind = "interface"
	KindEnum            DeclarationKind = "enum"
	KindEnumConstant    DeclarationKind = "enum_constant"
	KindAnnotation      DeclarationKind = "annotation"
	KindObject          DeclarationKind = "object"           // Kotlin
	KindCompanionObject DeclarationKind = "companion_object" // Kotlin
	KindFunction        DeclarationKind = "function"
	KindProperty        DeclarationKind = "property"   // Kotlin
	KindTypeAlias       DeclarationKind = "type_alias" // Kotlin
	KindRecord          DeclarationKind = "record"     // Java
	KindSealedClass     DeclarationKind = "sealed_class"
	KindDataClass       DeclarationKind = "data_class"    // Kotlin
	KindValueClass      DeclarationKind = "value_class"   // Kotlin (inline class)
	KindFunInterface    DeclarationKind = "fun_interface" // Kotlin
)

// TypeReference represents a reference to a type
type TypeReference struct {
	Location      Location        `json:"location,omitempty"`
	Name          string          `json:"name"`                     // e.g., "List", "Map"
	Package       string          `json:"package,omitempty"`        // e.g., "java.util"
	IsNullable    bool            `json:"is_nullable"`              // Kotlin nullable type
	TypeArguments []TypeReference `json:"type_arguments,omitempty"` // Generic type arguments
}

// ReferenceUsageKind describes the semantic role of a type/symbol reference.
type ReferenceUsageKind string

const (
	ReferenceUsageConstructorCall      ReferenceUsageKind = "constructor_call"
	ReferenceUsageQualifiedMethodCall  ReferenceUsageKind = "qualified_method_call"
	ReferenceUsageClassLiteral         ReferenceUsageKind = "class_literal"
	ReferenceUsageLocalVariableType    ReferenceUsageKind = "local_variable_type"
	ReferenceUsageCatchType            ReferenceUsageKind = "catch_type"
	ReferenceUsageTypeArgument         ReferenceUsageKind = "type_argument"
	ReferenceUsageMethodReference      ReferenceUsageKind = "method_reference"
	ReferenceUsageConstructorReference ReferenceUsageKind = "constructor_reference"
	ReferenceUsageCastType             ReferenceUsageKind = "cast_type"
	ReferenceUsageInstanceofType       ReferenceUsageKind = "instanceof_type"
)

// ReferenceUsage represents a semantic reference extracted from declarations or bodies.
type ReferenceUsage struct {
	Loc  Location           `json:"location,omitempty"`
	Path string             `json:"path"`
	Kind ReferenceUsageKind `json:"kind"`
}

// ClassDeclaration represents a class or interface declaration
type ClassDeclaration struct {
	Loc             Location        `json:"location,omitempty"`
	Name_           string          `json:"name"`
	Modifiers_      []Modifier      `json:"modifiers,omitempty"`
	Kind_           DeclarationKind `json:"kind"`
	TypeParameters  []TypeParameter `json:"type_parameters,omitempty"`
	SuperClass      *TypeReference  `json:"super_class,omitempty"`
	SuperInterfaces []TypeReference `json:"super_interfaces,omitempty"`
	Members         []Member        `json:"members,omitempty"`
	// Java-specific
	IsRecord         bool                  `json:"is_record,omitempty"`
	RecordComponents []VariableDeclaration `json:"record_components,omitempty"` // Java records
	// Kotlin-specific
	IsData             bool         `json:"is_data,omitempty"`             // Kotlin data class
	IsValue            bool         `json:"is_value,omitempty"`            // Kotlin value class
	IsInline           bool         `json:"is_inline,omitempty"`           // Kotlin inline class
	PrimaryConstructor *Constructor `json:"primary_constructor,omitempty"` // Kotlin
}

func (c *ClassDeclaration) Location() Location { return c.Loc }
func (c *ClassDeclaration) Children() []Node {
	nodes := make([]Node, 0, len(c.Members))
	for _, m := range c.Members {
		if n, ok := m.(Node); ok {
			nodes = append(nodes, n)
		}
	}
	return nodes
}
func (c *ClassDeclaration) Name() string          { return c.Name_ }
func (c *ClassDeclaration) Modifiers() []Modifier { return c.Modifiers_ }
func (c *ClassDeclaration) Kind() DeclarationKind { return c.Kind_ }

// FunctionDeclaration represents a function or method declaration
type FunctionDeclaration struct {
	Loc            Location        `json:"location,omitempty"`
	Name_          string          `json:"name"`
	Modifiers_     []Modifier      `json:"modifiers,omitempty"`
	ReturnType     *TypeReference  `json:"return_type,omitempty"`
	Parameters     []Parameter     `json:"parameters,omitempty"`
	TypeParameters []TypeParameter `json:"type_parameters,omitempty"`
	Body           Node            `json:"body,omitempty"`
	// Kotlin-specific
	IsExtension  bool           `json:"is_extension,omitempty"`  // Kotlin extension function
	ReceiverType *TypeReference `json:"receiver_type,omitempty"` // Kotlin extension receiver
	IsInfix      bool           `json:"is_infix,omitempty"`      // Kotlin infix
	IsInline     bool           `json:"is_inline,omitempty"`     // Kotlin inline
	IsTailrec    bool           `json:"is_tailrec,omitempty"`    // Kotlin tail-recursive
	IsSuspend    bool           `json:"is_suspend,omitempty"`    // Kotlin suspend
	IsExternal   bool           `json:"is_external,omitempty"`   // Kotlin external
	// Java-specific
	IsDefault  bool            `json:"is_default,omitempty"`  // Java interface default method
	IsAbstract bool            `json:"is_abstract,omitempty"` // Java abstract method
	Throws     []TypeReference `json:"throws,omitempty"`      // Java throws clause
}

func (f *FunctionDeclaration) Location() Location    { return f.Loc }
func (f *FunctionDeclaration) Children() []Node      { return []Node{f.Body} }
func (f *FunctionDeclaration) Name() string          { return f.Name_ }
func (f *FunctionDeclaration) Modifiers() []Modifier { return f.Modifiers_ }
func (f *FunctionDeclaration) Kind() DeclarationKind { return KindFunction }

// PropertyDeclaration represents a property declaration (Kotlin)
type PropertyDeclaration struct {
	Loc         Location             `json:"location,omitempty"`
	Name_       string               `json:"name"`
	Modifiers_  []Modifier           `json:"modifiers,omitempty"`
	Type        *TypeReference       `json:"type,omitempty"`
	Initializer Expression           `json:"initializer,omitempty"`
	IsVar       bool                 `json:"is_var"`                // mutable (var) vs read-only (val)
	IsLateinit  bool                 `json:"is_lateinit,omitempty"` // Kotlin lateinit
	IsConst     bool                 `json:"is_const,omitempty"`    // Kotlin const
	IsDelegate  bool                 `json:"is_delegate,omitempty"` // Kotlin delegated property
	Delegate    Expression           `json:"delegate,omitempty"`    // Kotlin delegate expression
	Getter      *FunctionDeclaration `json:"getter,omitempty"`
	Setter      *FunctionDeclaration `json:"setter,omitempty"`
}

func (p *PropertyDeclaration) Location() Location { return p.Loc }
func (p *PropertyDeclaration) Children() []Node {
	children := []Node{}
	if p.Initializer != nil {
		children = append(children, p.Initializer)
	}
	if p.Delegate != nil {
		children = append(children, p.Delegate)
	}
	return children
}
func (p *PropertyDeclaration) Name() string          { return p.Name_ }
func (p *PropertyDeclaration) Modifiers() []Modifier { return p.Modifiers_ }
func (p *PropertyDeclaration) Kind() DeclarationKind { return KindProperty }

// VariableDeclaration represents a variable declaration (Java fields, local variables)
type VariableDeclaration struct {
	Loc       Location       `json:"location,omitempty"`
	Name      string         `json:"name"`
	Modifiers []Modifier     `json:"modifiers,omitempty"`
	Type      *TypeReference `json:"type,omitempty"`
	// Java var type inference
	IsInferredType bool `json:"is_inferred_type,omitempty"` // Java 10+ var
}

func (v *VariableDeclaration) Location() Location { return v.Loc }
func (v *VariableDeclaration) Children() []Node   { return nil }

// Parameter represents a function parameter
type Parameter struct {
	Loc      Location       `json:"location,omitempty"`
	Name     string         `json:"name"`
	Type     *TypeReference `json:"type,omitempty"`
	IsVarArg bool           `json:"is_vararg,omitempty"` // Java varargs, Kotlin vararg
	// Kotlin-specific
	HasDefault bool       `json:"has_default,omitempty"` // Kotlin default parameter value
	Default    Expression `json:"default,omitempty"`     // Kotlin default value expression
}

// TypeParameter represents a generic type parameter
type TypeParameter struct {
	Loc    Location        `json:"location,omitempty"`
	Name   string          `json:"name"`
	Bounds []TypeReference `json:"bounds,omitempty"` // Upper bounds (e.g., <T extends Number>)
}

// Constructor represents a constructor declaration
type Constructor struct {
	Loc            Location        `json:"location,omitempty"`
	Modifiers      []Modifier      `json:"modifiers,omitempty"`
	Parameters     []Parameter     `json:"parameters,omitempty"`
	TypeParameters []TypeParameter `json:"type_parameters,omitempty"`
	// Kotlin-specific
	IsPrimary bool `json:"is_primary,omitempty"` // Kotlin primary constructor
}

func (c *Constructor) Location() Location { return c.Loc }
func (c *Constructor) Children() []Node   { return nil }

// Member represents a class/interface member
type Member interface {
	Node
}

// Expression represents an expression (for AST completeness)
type Expression interface {
	Node
}

// Statement represents a statement (for AST completeness)
type Statement interface {
	Node
}
