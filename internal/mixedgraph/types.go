package mixedgraph

import (
	"sort"
	"time"

	"github.com/dh-kam/jkdeps/internal/ast"
	"github.com/dh-kam/jkdeps/internal/parser"
)

// Language aliases for convenience
type SourceLanguage = ast.SourceLanguage

const (
	LangJava   SourceLanguage = ast.LanguageJava
	LangKotlin SourceLanguage = ast.LanguageKotlin
)

type ParseOptions struct {
	JavaGrammar         parser.JavaGrammar
	JavaParseMode       JavaParseMode
	Workers             int
	IncludeKTS          bool
	IncludeBuildScripts bool
	MaxErrorsPerFile    int
	LenientSyntax       bool
	ParseTimeout        time.Duration
}

type JavaParseMode string

const (
	JavaParseModeFull       JavaParseMode = "full"
	JavaParseModeHeaderOnly JavaParseMode = "header-only"
)

func (m JavaParseMode) IsValid() bool {
	return m == JavaParseModeFull || m == JavaParseModeHeaderOnly
}

func (o ParseOptions) withDefaults() ParseOptions {
	if o.JavaGrammar == "" {
		o.JavaGrammar = parser.JavaGrammarDefault
	}
	if o.JavaParseMode == "" {
		o.JavaParseMode = JavaParseModeFull
	}
	if o.Workers <= 0 {
		o.Workers = 1
	}
	if o.MaxErrorsPerFile <= 0 {
		o.MaxErrorsPerFile = 10
	}
	return o
}

type Diagnostic struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

type FileUnit struct {
	Path        string         `json:"path"`
	Relative    string         `json:"relative_path"`
	Language    SourceLanguage `json:"language"`
	PackageName string         `json:"package_name,omitempty"`
	Imports     []string       `json:"imports,omitempty"`
	References  []Reference    `json:"references,omitempty"`
	Parsed      bool           `json:"parsed"`
	Diagnostics []Diagnostic   `json:"diagnostics,omitempty"`
	Duration    time.Duration  `json:"duration"`
}

type ReferenceKind string

const (
	ReferenceKindImport               ReferenceKind = "import"
	ReferenceKindExtends              ReferenceKind = "extends"
	ReferenceKindImplements           ReferenceKind = "implements"
	ReferenceKindFieldType            ReferenceKind = "field_type"
	ReferenceKindMethodReturn         ReferenceKind = "method_return"
	ReferenceKindMethodParameter      ReferenceKind = "method_parameter"
	ReferenceKindConstructorParameter ReferenceKind = "constructor_parameter"
	ReferenceKindThrows               ReferenceKind = "throws"
	ReferenceKindTypeArgument         ReferenceKind = "type_argument"
	ReferenceKindConstructorCall      ReferenceKind = "constructor_call"
	ReferenceKindQualifiedMethodCall  ReferenceKind = "qualified_method_call"
	ReferenceKindClassLiteral         ReferenceKind = "class_literal"
	ReferenceKindLocalVariableType    ReferenceKind = "local_variable_type"
	ReferenceKindCatchType            ReferenceKind = "catch_type"
	ReferenceKindMethodReference      ReferenceKind = "method_reference"
	ReferenceKindConstructorReference ReferenceKind = "constructor_reference"
	ReferenceKindCastType             ReferenceKind = "cast_type"
	ReferenceKindInstanceofType       ReferenceKind = "instanceof_type"
)

type Reference struct {
	Path string        `json:"path"`
	Kind ReferenceKind `json:"kind"`
}

type RepositoryResult struct {
	Root        string        `json:"root"`
	TotalFiles  int           `json:"total_files"`
	JavaFiles   int           `json:"java_files"`
	KotlinFiles int           `json:"kotlin_files"`
	ParsedFiles int           `json:"parsed_files"`
	FailedFiles int           `json:"failed_files"`
	Files       []FileUnit    `json:"files,omitempty"`
	Duration    time.Duration `json:"duration"`
}

type ParseFileTiming struct {
	Path     string         `json:"path"`
	Relative string         `json:"relative_path,omitempty"`
	Language SourceLanguage `json:"language"`
	Parsed   bool           `json:"parsed"`
	Duration time.Duration  `json:"duration"`
}

func (r RepositoryResult) SlowestFiles(limit int) []ParseFileTiming {
	if limit <= 0 || len(r.Files) == 0 {
		return nil
	}

	timings := make([]ParseFileTiming, 0, len(r.Files))
	for _, file := range r.Files {
		timings = append(timings, ParseFileTiming{
			Path:     file.Path,
			Relative: file.Relative,
			Language: file.Language,
			Parsed:   file.Parsed,
			Duration: file.Duration,
		})
	}

	sort.Slice(timings, func(i, j int) bool {
		if timings[i].Duration == timings[j].Duration {
			if timings[i].Relative == timings[j].Relative {
				return timings[i].Path < timings[j].Path
			}
			return timings[i].Relative < timings[j].Relative
		}
		return timings[i].Duration > timings[j].Duration
	})

	if limit > len(timings) {
		limit = len(timings)
	}
	out := make([]ParseFileTiming, limit)
	copy(out, timings[:limit])
	return out
}

type GroupBy string

const (
	GroupByPackage GroupBy = "package"
	GroupByDir     GroupBy = "dir"
)

func (g GroupBy) IsValid() bool {
	return g == GroupByPackage || g == GroupByDir
}
