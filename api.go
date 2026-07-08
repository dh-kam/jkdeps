package jkdeps

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dh-kam/jkdeps/internal/mixedgraph"
	internalparser "github.com/dh-kam/jkdeps/internal/parser"
	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

type Language string

const (
	LanguageJava   Language = "java"
	LanguageKotlin Language = "kotlin"
)

type JavaGrammar string

const (
	JavaGrammarDefault JavaGrammar = "java20"
	JavaGrammarOrig    JavaGrammar = "java"
	JavaGrammar7       JavaGrammar = "java7"
	JavaGrammar8       JavaGrammar = "java8"
	JavaGrammar9       JavaGrammar = "java9"
	JavaGrammar11      JavaGrammar = "java11"
	JavaGrammar17      JavaGrammar = "java17"
	JavaGrammar20      JavaGrammar = "java20"
	JavaGrammar21      JavaGrammar = "java21"
	JavaGrammar25      JavaGrammar = "java25"
)

type JavaParseMode string

const (
	JavaParseModeHeaderOnly JavaParseMode = "header-only"
	JavaParseModeFull       JavaParseMode = "full"
)

type KotlinScripts string

const (
	KotlinScriptsDefault KotlinScripts = ""
	KotlinScriptsNone    KotlinScripts = "none"
	KotlinScriptsRegular KotlinScripts = "regular"
	KotlinScriptsBuild   KotlinScripts = "build"
	KotlinScriptsAll     KotlinScripts = "all"
)

type GroupBy string

const (
	GroupByPackage GroupBy = "package"
	GroupByDir     GroupBy = "dir"
)

type NodeKind string

const (
	NodeInternal NodeKind = "internal"
	NodeExternal NodeKind = "external"
	NodeUnknown  NodeKind = "unknown"
)

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

type Options struct {
	Parse ParseOptions `json:"parse,omitempty"`
	Graph GraphOptions `json:"graph,omitempty"`
}

type ParseOptions struct {
	JavaGrammar      JavaGrammar   `json:"java_grammar,omitempty"`
	JavaParseMode    JavaParseMode `json:"java_parse_mode,omitempty"`
	Workers          int           `json:"workers,omitempty"`
	KotlinScripts    KotlinScripts `json:"kotlin_scripts,omitempty"`
	MaxErrorsPerFile int           `json:"max_errors_per_file,omitempty"`
	LenientSyntax    bool          `json:"lenient_syntax,omitempty"`
	ParseTimeout     time.Duration `json:"parse_timeout,omitempty"`
}

type GraphOptions struct {
	GroupBy  GroupBy       `json:"group_by,omitempty"`
	External ExternalIndex `json:"external,omitempty"`
	Filter   GraphFilter   `json:"filter,omitempty"`
}

type GraphFilter struct {
	MinEdgeCount  int      `json:"min_edge_count,omitempty"`
	IncludePrefix []string `json:"include_prefix,omitempty"`
	ExcludePrefix []string `json:"exclude_prefix,omitempty"`
}

type ExternalIndex struct {
	Packages []string `json:"packages,omitempty"`
	Symbols  []string `json:"symbols,omitempty"`
}

type Report struct {
	Repository   Repository       `json:"repository"`
	Graph        Graph            `json:"graph"`
	Dependencies DependencyReport `json:"dependencies"`
}

type Repository struct {
	Root        string        `json:"root"`
	TotalFiles  int           `json:"total_files"`
	JavaFiles   int           `json:"java_files"`
	KotlinFiles int           `json:"kotlin_files"`
	ParsedFiles int           `json:"parsed_files"`
	FailedFiles int           `json:"failed_files"`
	Files       []File        `json:"files,omitempty"`
	Duration    time.Duration `json:"duration"`
}

type File struct {
	Path         string        `json:"path"`
	RelativePath string        `json:"relative_path"`
	Language     Language      `json:"language"`
	Package      string        `json:"package,omitempty"`
	Imports      []string      `json:"imports,omitempty"`
	References   []Reference   `json:"references,omitempty"`
	Parsed       bool          `json:"parsed"`
	Diagnostics  []Diagnostic  `json:"diagnostics,omitempty"`
	Duration     time.Duration `json:"duration"`
}

type Reference struct {
	Path string        `json:"path"`
	Kind ReferenceKind `json:"kind"`
}

type Diagnostic struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

type ParseFileTiming struct {
	Path         string        `json:"path"`
	RelativePath string        `json:"relative_path,omitempty"`
	Language     Language      `json:"language"`
	Parsed       bool          `json:"parsed"`
	Duration     time.Duration `json:"duration"`
}

type Node struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Kind      NodeKind `json:"kind"`
	InDegree  int      `json:"in_degree"`
	OutDegree int      `json:"out_degree"`
}

type Edge struct {
	FromID int `json:"from_id"`
	ToID   int `json:"to_id"`
	Count  int `json:"count"`
}

type Graph struct {
	Root    string  `json:"root"`
	GroupBy GroupBy `json:"group_by"`
	Nodes   []Node  `json:"nodes"`
	Edges   []Edge  `json:"edges"`
}

type FileDependency struct {
	FilePath      string        `json:"file_path"`
	FromPackage   string        `json:"from_package"`
	ImportPath    string        `json:"import_path"`
	ToPackage     string        `json:"to_package"`
	ReferenceKind ReferenceKind `json:"reference_kind,omitempty"`
	Kind          NodeKind      `json:"kind"`
}

type DependencyReport struct {
	Root                 string           `json:"root"`
	KnownPackages        int              `json:"known_packages"`
	TotalDependencies    int              `json:"total_dependencies"`
	InternalDependencies int              `json:"internal_dependencies"`
	ExternalDependencies int              `json:"external_dependencies"`
	UnknownDependencies  int              `json:"unknown_dependencies"`
	Dependencies         []FileDependency `json:"dependencies"`
	UnresolvedReferences []FileDependency `json:"unresolved_references,omitempty"`
	UnresolvedImports    []FileDependency `json:"unresolved_imports"`
}

func Analyze(ctx context.Context, root string, opts Options) (Report, error) {
	repo, err := ParseRepository(ctx, root, opts.Parse)
	if err != nil {
		return Report{}, err
	}
	if err := contextError(ctx); err != nil {
		return Report{}, err
	}

	graph := BuildGraph(repo, opts.Graph)
	deps := BuildDependencyReport(repo, opts.Graph.External)
	return Report{
		Repository:   repo,
		Graph:        graph,
		Dependencies: deps,
	}, nil
}

func DefaultOptions() Options {
	return Options{
		Parse: DefaultParseOptions(),
		Graph: DefaultGraphOptions(),
	}
}

func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		JavaGrammar:   JavaGrammarDefault,
		JavaParseMode: JavaParseModeHeaderOnly,
		KotlinScripts: KotlinScriptsRegular,
	}
}

func DefaultGraphOptions() GraphOptions {
	return GraphOptions{
		GroupBy: GroupByPackage,
	}
}

func ParseRepository(ctx context.Context, root string, opts ParseOptions) (Repository, error) {
	if err := contextError(ctx); err != nil {
		return Repository{}, err
	}

	mixedOpts, err := opts.toMixedGraph()
	if err != nil {
		return Repository{}, err
	}

	result, err := mixedgraph.ParseRepository(root, mixedOpts)
	if err != nil {
		return Repository{}, err
	}
	if err := contextError(ctx); err != nil {
		return Repository{}, err
	}
	return fromMixedRepository(result), nil
}

func BuildGraph(repo Repository, opts GraphOptions) Graph {
	var graph mixedgraph.Graph
	if opts.Filter.isActive() {
		graph = mixedgraph.BuildFilteredGraph(toMixedRepository(repo), opts.External.toKCG(), opts.groupBy(), opts.Filter.toMixedGraph())
	} else {
		graph = mixedgraph.BuildGraph(toMixedRepository(repo), opts.External.toKCG(), opts.groupBy())
	}
	return fromMixedGraph(graph)
}

func FilterGraph(graph Graph, filter GraphFilter) Graph {
	return fromMixedGraph(mixedgraph.FilterGraph(toMixedGraph(graph), filter.toMixedGraph()))
}

func BuildDependencyReport(repo Repository, external ExternalIndex) DependencyReport {
	report := mixedgraph.BuildFileDependencyReport(toMixedRepository(repo), external.toKCG())
	return fromMixedDependencyReport(report)
}

func LoadExternalIndex(path string) (ExternalIndex, error) {
	index, err := kcg.LoadExternalIndex(path)
	if err != nil {
		return ExternalIndex{}, err
	}
	return fromKCGExternalIndex(index), nil
}

func LoadExternalIndices(paths ...string) (ExternalIndex, error) {
	index, err := kcg.LoadExternalIndices(paths)
	if err != nil {
		return ExternalIndex{}, err
	}
	return fromKCGExternalIndex(index), nil
}

func NewExternalIndex(packages, symbols []string) ExternalIndex {
	return ExternalIndex{
		Packages: normalizeNames(packages),
		Symbols:  normalizeNames(symbols),
	}
}

func MergeExternalIndices(indices ...ExternalIndex) ExternalIndex {
	packages := map[string]struct{}{}
	symbols := map[string]struct{}{}
	for _, index := range indices {
		for _, pkg := range index.Packages {
			if name := cleanName(pkg); name != "" {
				packages[name] = struct{}{}
			}
		}
		for _, symbol := range index.Symbols {
			if name := cleanName(symbol); name != "" {
				symbols[name] = struct{}{}
			}
		}
	}
	return ExternalIndex{
		Packages: sortedKeys(packages),
		Symbols:  sortedKeys(symbols),
	}
}

func (i ExternalIndex) HasPackage(pkg string) bool {
	name := cleanName(pkg)
	for _, candidate := range i.Packages {
		if cleanName(candidate) == name {
			return true
		}
	}
	return false
}

func (i ExternalIndex) HasSymbol(symbol string) bool {
	name := cleanName(symbol)
	for _, candidate := range i.Symbols {
		if cleanName(candidate) == name {
			return true
		}
	}
	return false
}

func (r Repository) BuildGraph(opts GraphOptions) Graph {
	return BuildGraph(r, opts)
}

func (g Graph) Filter(filter GraphFilter) Graph {
	return FilterGraph(g, filter)
}

func (r Repository) BuildDependencyReport(external ExternalIndex) DependencyReport {
	return BuildDependencyReport(r, external)
}

func (r Repository) SlowestFiles(limit int) []ParseFileTiming {
	if limit <= 0 || len(r.Files) == 0 {
		return nil
	}

	timings := make([]ParseFileTiming, 0, len(r.Files))
	for _, file := range r.Files {
		timings = append(timings, ParseFileTiming{
			Path:         file.Path,
			RelativePath: file.RelativePath,
			Language:     file.Language,
			Parsed:       file.Parsed,
			Duration:     file.Duration,
		})
	}

	sort.Slice(timings, func(i, j int) bool {
		if timings[i].Duration == timings[j].Duration {
			if timings[i].RelativePath == timings[j].RelativePath {
				return timings[i].Path < timings[j].Path
			}
			return timings[i].RelativePath < timings[j].RelativePath
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

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func (o ParseOptions) toMixedGraph() (mixedgraph.ParseOptions, error) {
	grammar := internalparser.JavaGrammar(o.JavaGrammar)
	if grammar == "" {
		grammar = internalparser.JavaGrammar(JavaGrammarDefault)
	}
	if !grammar.IsValid() {
		return mixedgraph.ParseOptions{}, fmt.Errorf("unsupported java grammar: %q", grammar)
	}

	mode := mixedgraph.JavaParseMode(o.JavaParseMode)
	if mode == "" {
		mode = mixedgraph.JavaParseModeHeaderOnly
	}
	if !mode.IsValid() {
		return mixedgraph.ParseOptions{}, fmt.Errorf("unsupported java parse mode: %q", mode)
	}

	includeKTS, includeBuildScripts, err := o.KotlinScripts.includes()
	if err != nil {
		return mixedgraph.ParseOptions{}, err
	}

	return mixedgraph.ParseOptions{
		JavaGrammar:         grammar,
		JavaParseMode:       mode,
		Workers:             o.Workers,
		IncludeKTS:          includeKTS,
		IncludeBuildScripts: includeBuildScripts,
		MaxErrorsPerFile:    o.MaxErrorsPerFile,
		LenientSyntax:       o.LenientSyntax,
		ParseTimeout:        o.ParseTimeout,
	}, nil
}

func (s KotlinScripts) includes() (includeKTS bool, includeBuildScripts bool, err error) {
	switch s {
	case KotlinScriptsDefault, KotlinScriptsRegular:
		return true, false, nil
	case KotlinScriptsNone:
		return false, false, nil
	case KotlinScriptsBuild:
		return false, true, nil
	case KotlinScriptsAll:
		return true, true, nil
	default:
		return false, false, fmt.Errorf("unsupported kotlin scripts option: %q", s)
	}
}

func (o GraphOptions) groupBy() mixedgraph.GroupBy {
	switch o.GroupBy {
	case GroupByDir:
		return mixedgraph.GroupByDir
	default:
		return mixedgraph.GroupByPackage
	}
}

func (f GraphFilter) isActive() bool {
	return f.MinEdgeCount > 0 || len(f.IncludePrefix) > 0 || len(f.ExcludePrefix) > 0
}

func (f GraphFilter) toMixedGraph() mixedgraph.GraphFilter {
	return mixedgraph.GraphFilter{
		MinEdgeCount:  f.MinEdgeCount,
		IncludePrefix: copyStrings(f.IncludePrefix),
		ExcludePrefix: copyStrings(f.ExcludePrefix),
	}
}

func fromMixedRepository(in mixedgraph.RepositoryResult) Repository {
	out := Repository{
		Root:        in.Root,
		TotalFiles:  in.TotalFiles,
		JavaFiles:   in.JavaFiles,
		KotlinFiles: in.KotlinFiles,
		ParsedFiles: in.ParsedFiles,
		FailedFiles: in.FailedFiles,
		Files:       make([]File, 0, len(in.Files)),
		Duration:    in.Duration,
	}
	for _, file := range in.Files {
		out.Files = append(out.Files, fromMixedFile(file))
	}
	sortFiles(out.Files)
	return out
}

func fromMixedFile(in mixedgraph.FileUnit) File {
	return File{
		Path:         in.Path,
		RelativePath: in.Relative,
		Language:     Language(in.Language),
		Package:      in.PackageName,
		Imports:      copyStrings(in.Imports),
		References:   fromMixedReferences(in.References),
		Parsed:       in.Parsed,
		Diagnostics:  fromMixedDiagnostics(in.Diagnostics),
		Duration:     in.Duration,
	}
}

func fromMixedReferences(in []mixedgraph.Reference) []Reference {
	if len(in) == 0 {
		return nil
	}
	out := make([]Reference, 0, len(in))
	for _, ref := range in {
		out = append(out, Reference{
			Path: ref.Path,
			Kind: ReferenceKind(ref.Kind),
		})
	}
	return out
}

func fromMixedDiagnostics(in []mixedgraph.Diagnostic) []Diagnostic {
	if len(in) == 0 {
		return nil
	}
	out := make([]Diagnostic, 0, len(in))
	for _, diag := range in {
		out = append(out, Diagnostic{
			Path:    diag.Path,
			Line:    diag.Line,
			Column:  diag.Column,
			Message: diag.Message,
		})
	}
	return out
}

func toMixedRepository(in Repository) mixedgraph.RepositoryResult {
	out := mixedgraph.RepositoryResult{
		Root:        in.Root,
		TotalFiles:  in.TotalFiles,
		JavaFiles:   in.JavaFiles,
		KotlinFiles: in.KotlinFiles,
		ParsedFiles: in.ParsedFiles,
		FailedFiles: in.FailedFiles,
		Files:       make([]mixedgraph.FileUnit, 0, len(in.Files)),
		Duration:    in.Duration,
	}
	for _, file := range in.Files {
		out.Files = append(out.Files, toMixedFile(file))
	}
	return out
}

func toMixedFile(in File) mixedgraph.FileUnit {
	return mixedgraph.FileUnit{
		Path:        in.Path,
		Relative:    in.RelativePath,
		Language:    mixedgraph.SourceLanguage(in.Language),
		PackageName: in.Package,
		Imports:     copyStrings(in.Imports),
		References:  toMixedReferences(in.References),
		Parsed:      in.Parsed,
		Diagnostics: toMixedDiagnostics(in.Diagnostics),
		Duration:    in.Duration,
	}
}

func toMixedReferences(in []Reference) []mixedgraph.Reference {
	if len(in) == 0 {
		return nil
	}
	out := make([]mixedgraph.Reference, 0, len(in))
	for _, ref := range in {
		out = append(out, mixedgraph.Reference{
			Path: ref.Path,
			Kind: mixedgraph.ReferenceKind(ref.Kind),
		})
	}
	return out
}

func toMixedDiagnostics(in []Diagnostic) []mixedgraph.Diagnostic {
	if len(in) == 0 {
		return nil
	}
	out := make([]mixedgraph.Diagnostic, 0, len(in))
	for _, diag := range in {
		out = append(out, mixedgraph.Diagnostic{
			Path:    diag.Path,
			Line:    diag.Line,
			Column:  diag.Column,
			Message: diag.Message,
		})
	}
	return out
}

func fromMixedGraph(in mixedgraph.Graph) Graph {
	out := Graph{
		Root:    in.Root,
		GroupBy: GroupBy(in.GroupBy),
		Nodes:   make([]Node, 0, len(in.Nodes)),
		Edges:   make([]Edge, 0, len(in.Edges)),
	}
	for _, node := range in.Nodes {
		out.Nodes = append(out.Nodes, Node{
			ID:        node.ID,
			Name:      node.Name,
			Kind:      NodeKind(node.Kind),
			InDegree:  node.InDegree,
			OutDegree: node.OutDegree,
		})
	}
	for _, edge := range in.Edges {
		out.Edges = append(out.Edges, Edge{
			FromID: edge.FromID,
			ToID:   edge.ToID,
			Count:  edge.Count,
		})
	}
	return out
}

func toMixedGraph(in Graph) mixedgraph.Graph {
	out := mixedgraph.Graph{
		Root:    in.Root,
		GroupBy: mixedgraph.GroupBy(in.GroupBy),
		Nodes:   make([]mixedgraph.Node, 0, len(in.Nodes)),
		Edges:   make([]mixedgraph.Edge, 0, len(in.Edges)),
	}
	for _, node := range in.Nodes {
		out.Nodes = append(out.Nodes, mixedgraph.Node{
			ID:        node.ID,
			Name:      node.Name,
			Kind:      mixedgraph.NodeKind(node.Kind),
			InDegree:  node.InDegree,
			OutDegree: node.OutDegree,
		})
	}
	for _, edge := range in.Edges {
		out.Edges = append(out.Edges, mixedgraph.Edge{
			FromID: edge.FromID,
			ToID:   edge.ToID,
			Count:  edge.Count,
		})
	}
	return out
}

func fromMixedDependencyReport(in mixedgraph.DependencyReport) DependencyReport {
	return DependencyReport{
		Root:                 in.Root,
		KnownPackages:        in.KnownPackages,
		TotalDependencies:    in.TotalDependencies,
		InternalDependencies: in.InternalDependencies,
		ExternalDependencies: in.ExternalDependencies,
		UnknownDependencies:  in.UnknownDependencies,
		Dependencies:         fromMixedFileDependencies(in.Dependencies),
		UnresolvedReferences: fromMixedFileDependencies(in.UnresolvedReferences),
		UnresolvedImports:    fromMixedFileDependencies(in.UnresolvedImports),
	}
}

func fromMixedFileDependencies(in []mixedgraph.FileDependency) []FileDependency {
	if len(in) == 0 {
		return nil
	}
	out := make([]FileDependency, 0, len(in))
	for _, dep := range in {
		out = append(out, FileDependency{
			FilePath:      dep.FilePath,
			FromPackage:   dep.FromPackage,
			ImportPath:    dep.ImportPath,
			ToPackage:     dep.ToPackage,
			ReferenceKind: ReferenceKind(dep.ReferenceKind),
			Kind:          NodeKind(dep.Kind),
		})
	}
	return out
}

func fromKCGExternalIndex(in kcg.ExternalIndex) ExternalIndex {
	return ExternalIndex{
		Packages: in.PackageNames(),
		Symbols:  in.SymbolNames(),
	}
}

func (i ExternalIndex) toKCG() kcg.ExternalIndex {
	index := kcg.NewExternalIndex()
	for _, pkg := range i.Packages {
		if name := cleanName(pkg); name != "" {
			index.Packages[name] = struct{}{}
		}
	}
	for _, symbol := range i.Symbols {
		if name := cleanName(symbol); name != "" {
			index.Symbols[name] = struct{}{}
		}
	}
	return index
}

func sortFiles(files []File) {
	sort.Slice(files, func(i, j int) bool {
		if files[i].RelativePath != files[j].RelativePath {
			return files[i].RelativePath < files[j].RelativePath
		}
		return files[i].Path < files[j].Path
	})
}

func copyStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func normalizeNames(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if name := cleanName(value); name != "" {
			set[name] = struct{}{}
		}
	}
	return sortedKeys(set)
}

func cleanName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "`", "")
	value = strings.Join(strings.Fields(value), "")
	value = strings.Trim(value, ".")
	return value
}

func sortedKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
