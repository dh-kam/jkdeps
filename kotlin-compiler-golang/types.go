package kotlincompilergolang

import "time"

type Config struct {
	Workers             int
	MaxErrorsPerFile    int
	IncludeKTS          bool
	IncludeBuildScripts bool
	LenientSyntax       bool
	ParseTimeout        time.Duration
	ParseBackend        ParseBackend
}

type ParseBackend string

const (
	ParseBackendANTLR      ParseBackend = "antlr"
	ParseBackendEmbeddable ParseBackend = "embeddable"
	ParseBackendDefault    ParseBackend = ParseBackendANTLR
)

func ParseBackendFromString(value string) ParseBackend {
	switch ParseBackend(value) {
	case ParseBackendEmbeddable:
		return ParseBackendEmbeddable
	default:
		return ParseBackendANTLR
	}
}

func (c Config) withDefaults() Config {
	if c.Workers <= 0 {
		c.Workers = 1
	}
	if c.MaxErrorsPerFile <= 0 {
		c.MaxErrorsPerFile = 10
	}
	if c.ParseBackend == "" {
		c.ParseBackend = ParseBackendDefault
	}
	return c
}

type DiagnosticSeverity string

const (
	SeverityError DiagnosticSeverity = "error"
)

type Diagnostic struct {
	Path     string             `json:"path"`
	Line     int                `json:"line"`
	Column   int                `json:"column"`
	Message  string             `json:"message"`
	Severity DiagnosticSeverity `json:"severity"`
}

type DeclarationKind string

const (
	DeclClass     DeclarationKind = "class"
	DeclInterface DeclarationKind = "interface"
	DeclObject    DeclarationKind = "object"
	DeclFunction  DeclarationKind = "function"
	DeclProperty  DeclarationKind = "property"
	DeclTypeAlias DeclarationKind = "typealias"
)

type TopLevelDeclaration struct {
	Kind      DeclarationKind `json:"kind"`
	Name      string          `json:"name"`
	Line      int             `json:"line"`
	Modifiers []string        `json:"modifiers,omitempty"`
}

type FileUnit struct {
	Path         string                `json:"path"`
	PackageName  string                `json:"package_name,omitempty"`
	Imports      []string              `json:"imports,omitempty"`
	Declarations []TopLevelDeclaration `json:"declarations,omitempty"`
	Parsed       bool                  `json:"parsed"`
	Diagnostics  []Diagnostic          `json:"diagnostics,omitempty"`
	Duration     time.Duration         `json:"duration"`
}

type RepositoryResult struct {
	Root        string        `json:"root"`
	TotalFiles  int           `json:"total_files"`
	ParsedFiles int           `json:"parsed_files"`
	FailedFiles int           `json:"failed_files"`
	Files       []FileUnit    `json:"files,omitempty"`
	Duration    time.Duration `json:"duration"`
}
