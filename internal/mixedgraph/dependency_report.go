package mixedgraph

import (
	"sort"

	kcg "github.com/dh-kam/jkdeps/kotlin-compiler-golang"
)

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

type dependencyReportBuilder struct {
	ctx graphBuildContext
	acc dependencyReportAccumulator
}

type dependencyReportAccumulator struct {
	report DependencyReport
}

type dependencyReportSortPolicy struct{}

var deterministicDependencyReportSortPolicy dependencyReportSortPolicy

func BuildFileDependencyReport(result RepositoryResult, external kcg.ExternalIndex) DependencyReport {
	ctx := newGraphBuildContext(result.Files, external)
	builder := newDependencyReportBuilder(ctx, result.Root)
	builder.addFiles(result.Files)
	return deterministicDependencyReportSortPolicy.sort(builder.acc.report)
}

func newDependencyReportBuilder(ctx graphBuildContext, root string) dependencyReportBuilder {
	return dependencyReportBuilder{
		ctx: ctx,
		acc: newDependencyReportAccumulator(root, len(ctx.internalPackages)),
	}
}

func (b *dependencyReportBuilder) addFiles(files []FileUnit) {
	for i, file := range files {
		b.addFileImports(file, b.ctx.filePackages[i])
		b.addFileReferences(file, b.ctx.filePackages[i])
	}
}

func (b *dependencyReportBuilder) addFileImports(file FileUnit, from string) {
	if from == "" {
		return
	}
	for _, rawImport := range file.Imports {
		dep, ok := b.buildDependency(file.Path, from, rawImport, ReferenceKindImport, b.ctx.resolveImport(rawImport))
		if !ok {
			continue
		}
		b.acc.add(dep)
	}
}

func (b *dependencyReportBuilder) addFileReferences(file FileUnit, from string) {
	if from == "" {
		return
	}
	for _, ref := range file.References {
		target := b.ctx.resolveReference(file, ref.Path)
		dep, ok := b.buildDependency(file.Path, from, ref.Path, ref.Kind, target)
		if !ok {
			continue
		}
		b.acc.add(dep)
	}
}

func (b *dependencyReportBuilder) buildDependency(filePath, from, rawImport string, refKind ReferenceKind, target resolvedImportTarget) (FileDependency, bool) {
	importPath := normalizePath(rawImport)
	if importPath == "" {
		return FileDependency{}, false
	}
	to := target.Package
	if to == "" || to == from {
		return FileDependency{}, false
	}

	return FileDependency{
		FilePath:      filePath,
		FromPackage:   from,
		ImportPath:    importPath,
		ToPackage:     to,
		ReferenceKind: refKind,
		Kind:          target.Kind,
	}, true
}

func newDependencyReportAccumulator(root string, knownPackages int) dependencyReportAccumulator {
	return dependencyReportAccumulator{
		report: DependencyReport{
			Root:          root,
			KnownPackages: knownPackages,
		},
	}
}

func (a *dependencyReportAccumulator) add(dep FileDependency) {
	if dep.ReferenceKind == "" {
		dep.ReferenceKind = ReferenceKindImport
	}
	a.report.Dependencies = append(a.report.Dependencies, dep)
	a.report.TotalDependencies++
	switch dep.Kind {
	case NodeInternal:
		a.report.InternalDependencies++
	case NodeExternal:
		a.report.ExternalDependencies++
	case NodeUnknown:
		a.report.UnknownDependencies++
		a.report.UnresolvedReferences = append(a.report.UnresolvedReferences, dep)
		if dep.ReferenceKind == ReferenceKindImport {
			a.report.UnresolvedImports = append(a.report.UnresolvedImports, dep)
		}
	}
}

func (p dependencyReportSortPolicy) sort(report DependencyReport) DependencyReport {
	p.sortDependencies(report.Dependencies)
	p.sortUnresolvedImports(report.UnresolvedReferences)
	p.sortUnresolvedImports(report.UnresolvedImports)
	return report
}

func (dependencyReportSortPolicy) sortDependencies(deps []FileDependency) {
	sort.Slice(deps, func(i, j int) bool {
		if deps[i].FilePath != deps[j].FilePath {
			return deps[i].FilePath < deps[j].FilePath
		}
		if deps[i].ToPackage != deps[j].ToPackage {
			return deps[i].ToPackage < deps[j].ToPackage
		}
		return deps[i].ImportPath < deps[j].ImportPath
	})
}

func (dependencyReportSortPolicy) sortUnresolvedImports(deps []FileDependency) {
	sort.Slice(deps, func(i, j int) bool {
		if deps[i].FilePath != deps[j].FilePath {
			return deps[i].FilePath < deps[j].FilePath
		}
		return deps[i].ImportPath < deps[j].ImportPath
	})
}
