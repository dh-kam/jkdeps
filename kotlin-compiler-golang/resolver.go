package kotlincompilergolang

import (
	"sort"
	"strings"
)

type ImportResolution struct {
	FilePath       string `json:"file_path"`
	ImportPath     string `json:"import_path"`
	Resolved       bool   `json:"resolved"`
	External       bool   `json:"external,omitempty"`
	TargetSymbolID int    `json:"target_symbol_id,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

type ResolutionReport struct {
	TotalImports      int                `json:"total_imports"`
	ResolvedImports   int                `json:"resolved_imports"`
	UnresolvedImports int                `json:"unresolved_imports"`
	Items             []ImportResolution `json:"items"`
}

func (r RepositoryResult) ResolveImports(table SymbolTable) ResolutionReport {
	return ResolveImports(r, table)
}

func ResolveImports(result RepositoryResult, table SymbolTable) ResolutionReport {
	return ResolveImportsWithExternal(result, table, ExternalIndex{})
}

func ResolveImportsWithPackages(result RepositoryResult, table SymbolTable, externalPackages map[string]struct{}) ResolutionReport {
	return ResolveImportsWithExternal(result, table, ExternalIndex{Packages: externalPackages})
}

func ResolveImportsWithExternal(result RepositoryResult, table SymbolTable, external ExternalIndex) ResolutionReport {
	report := ResolutionReport{Items: make([]ImportResolution, 0, result.TotalFiles*4)}

	symbolByFQN := make(map[string]Symbol, len(table.Symbols))
	packageByName := make(map[string]Symbol, len(table.Symbols))
	for _, symbol := range table.Symbols {
		symbolByFQN[symbol.FQN] = symbol
		if symbol.Kind == SymbolPackage {
			packageByName[symbol.Name] = symbol
		}
	}

	for _, file := range result.Files {
		for _, rawImport := range file.Imports {
			report.TotalImports++

			importPath := normalizeImportPath(rawImport)
			item := ImportResolution{FilePath: file.Path, ImportPath: importPath}
			if importPath == "" {
				item.Reason = "empty import"
				report.Items = append(report.Items, item)
				continue
			}

			if wildcardPkg := wildcardPackage(importPath); wildcardPkg != "" {
				if symbol, ok := packageByName[wildcardPkg]; ok {
					item.Resolved = true
					item.TargetSymbolID = symbol.ID
					report.ResolvedImports++
				} else if symbol, ok := symbolByFQN[wildcardPkg]; ok {
					item.Resolved = true
					item.TargetSymbolID = symbol.ID
					report.ResolvedImports++
				} else if external.HasPackage(wildcardPkg) {
					item.Resolved = true
					item.External = true
					report.ResolvedImports++
				} else if external.HasSymbol(wildcardPkg) {
					item.Resolved = true
					item.External = true
					report.ResolvedImports++
				} else if resolved, externalResolved, symbolID := resolveImportPrefix(wildcardPkg, packageByName, symbolByFQN, external); resolved {
					item.Resolved = true
					item.External = externalResolved
					item.TargetSymbolID = symbolID
					report.ResolvedImports++
				} else if isKnownExternalImport(wildcardPkg) {
					item.Resolved = true
					item.External = true
					report.ResolvedImports++
				} else {
					item.Reason = "wildcard package not found"
					report.UnresolvedImports++
				}
				report.Items = append(report.Items, item)
				continue
			}

			if symbol, ok := symbolByFQN[importPath]; ok {
				item.Resolved = true
				item.TargetSymbolID = symbol.ID
				report.ResolvedImports++
				report.Items = append(report.Items, item)
				continue
			}
			if external.HasSymbol(importPath) {
				item.Resolved = true
				item.External = true
				report.ResolvedImports++
				report.Items = append(report.Items, item)
				continue
			}

			pkg := inferImportPackage(importPath)
			if symbol, ok := packageByName[pkg]; ok {
				item.Resolved = true
				item.TargetSymbolID = symbol.ID
				report.ResolvedImports++
				report.Items = append(report.Items, item)
				continue
			}
			if external.HasPackage(importPath) || external.HasPackage(pkg) {
				item.Resolved = true
				item.External = true
				report.ResolvedImports++
				report.Items = append(report.Items, item)
				continue
			}
			if resolved, externalResolved, symbolID := resolveImportPrefix(importPath, packageByName, symbolByFQN, external); resolved {
				item.Resolved = true
				item.External = externalResolved
				item.TargetSymbolID = symbolID
				report.ResolvedImports++
				report.Items = append(report.Items, item)
				continue
			}
			if isKnownExternalImport(importPath) || isKnownExternalImport(pkg) {
				item.Resolved = true
				item.External = true
				report.ResolvedImports++
				report.Items = append(report.Items, item)
				continue
			}

			item.Reason = "symbol/package not found"
			report.UnresolvedImports++
			report.Items = append(report.Items, item)
		}
	}

	sort.Slice(report.Items, func(i, j int) bool {
		if report.Items[i].FilePath != report.Items[j].FilePath {
			return report.Items[i].FilePath < report.Items[j].FilePath
		}
		return report.Items[i].ImportPath < report.Items[j].ImportPath
	})
	return report
}

func wildcardPackage(importPath string) string {
	if len(importPath) < 3 {
		return ""
	}
	if importPath[len(importPath)-2:] != ".*" {
		return ""
	}
	return importPath[:len(importPath)-2]
}

func resolveImportPrefix(importPath string, packageByName map[string]Symbol, symbolByFQN map[string]Symbol, external ExternalIndex) (resolved bool, externalResolved bool, symbolID int) {
	path := normalizeImportPath(importPath)
	if path == "" {
		return false, false, 0
	}

	parts := strings.Split(path, ".")
	for i := len(parts); i >= 1; i-- {
		candidate := strings.Join(parts[:i], ".")
		if symbol, ok := symbolByFQN[candidate]; ok {
			return true, false, symbol.ID
		}
		if symbol, ok := packageByName[candidate]; ok {
			return true, false, symbol.ID
		}
		if external.HasSymbol(candidate) || external.HasPackage(candidate) {
			return true, true, 0
		}
		if isKnownExternalImport(candidate) {
			return true, true, 0
		}
	}
	return false, false, 0
}
