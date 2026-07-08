package kotlincompilergolang

import (
	"path/filepath"
	"sort"
	"strings"
)

type SymbolKind string

const (
	SymbolPackage   SymbolKind = "package"
	SymbolFile      SymbolKind = "file"
	SymbolClass     SymbolKind = "class"
	SymbolInterface SymbolKind = "interface"
	SymbolObject    SymbolKind = "object"
	SymbolFunction  SymbolKind = "function"
	SymbolProperty  SymbolKind = "property"
	SymbolTypeAlias SymbolKind = "typealias"
)

type Symbol struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	FQN       string     `json:"fqn"`
	Kind      SymbolKind `json:"kind"`
	Path      string     `json:"path,omitempty"`
	Package   string     `json:"package,omitempty"`
	ParentID  int        `json:"parent_id,omitempty"`
	Line      int        `json:"line,omitempty"`
	Modifiers []string   `json:"modifiers,omitempty"`
}

type SymbolTable struct {
	Symbols []Symbol `json:"symbols"`
}

func (r RepositoryResult) BuildSymbolTable() SymbolTable {
	return BuildSymbolTable(r)
}

func BuildSymbolTable(result RepositoryResult) SymbolTable {
	symbols := make([]Symbol, 0, result.TotalFiles*2)
	nextID := 1
	symbolIDByFQN := map[string]int{}

	files := make([]FileUnit, len(result.Files))
	copy(files, result.Files)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	packageIDs := map[string]int{}
	for _, file := range files {
		pkgName := file.PackageName
		if pkgName != "" {
			if _, ok := packageIDs[pkgName]; !ok {
				pkgSymbol := Symbol{
					ID:   nextID,
					Name: pkgName,
					FQN:  pkgName,
					Kind: SymbolPackage,
				}
				symbols = append(symbols, pkgSymbol)
				packageIDs[pkgName] = nextID
				symbolIDByFQN[pkgSymbol.FQN] = nextID
				nextID++
			}
		}

		fileName := filepath.Base(file.Path)
		fileFQN := file.Path
		if pkgName != "" {
			fileFQN = pkgName + ":" + fileName
		}
		fileSymbol := Symbol{
			ID:       nextID,
			Name:     fileName,
			FQN:      fileFQN,
			Kind:     SymbolFile,
			Path:     file.Path,
			Package:  pkgName,
			ParentID: packageIDs[pkgName],
		}
		fileID := nextID
		symbols = append(symbols, fileSymbol)
		symbolIDByFQN[fileSymbol.FQN] = fileID
		nextID++

		for _, decl := range file.Declarations {
			declFQN := joinFQN(pkgName, decl.Name)
			parentID := fileID
			if parentFQN := declarationParentFQN(pkgName, decl.Name); parentFQN != "" {
				if id, ok := symbolIDByFQN[parentFQN]; ok {
					parentID = id
				}
			}
			symbol := Symbol{
				ID:        nextID,
				Name:      decl.Name,
				FQN:       declFQN,
				Kind:      mapDeclarationKindToSymbolKind(decl.Kind),
				Path:      file.Path,
				Package:   pkgName,
				ParentID:  parentID,
				Line:      decl.Line,
				Modifiers: copySlice(decl.Modifiers),
			}
			symbols = append(symbols, symbol)
			symbolIDByFQN[declFQN] = nextID
			nextID++
		}
	}

	return SymbolTable{Symbols: symbols}
}

func declarationParentFQN(pkgName, declarationName string) string {
	declarationName = strings.TrimSpace(declarationName)
	if declarationName == "" {
		return ""
	}
	idx := strings.LastIndex(declarationName, ".")
	if idx <= 0 {
		return ""
	}
	parentName := declarationName[:idx]
	if parentName == "" {
		return ""
	}
	if pkgName == "" {
		return parentName
	}
	return pkgName + "." + parentName
}

func mapDeclarationKindToSymbolKind(kind DeclarationKind) SymbolKind {
	switch kind {
	case DeclClass:
		return SymbolClass
	case DeclInterface:
		return SymbolInterface
	case DeclObject:
		return SymbolObject
	case DeclFunction:
		return SymbolFunction
	case DeclProperty:
		return SymbolProperty
	case DeclTypeAlias:
		return SymbolTypeAlias
	default:
		return SymbolFile
	}
}

func joinFQN(pkg, name string) string {
	if pkg == "" {
		return name
	}
	if name == "" {
		return pkg
	}
	return pkg + "." + name
}

func copySlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func (t SymbolTable) CountByKind() map[SymbolKind]int {
	counts := map[SymbolKind]int{}
	for _, symbol := range t.Symbols {
		counts[symbol.Kind]++
	}
	return counts
}

func (t SymbolTable) FindByPrefix(prefix string) []Symbol {
	if prefix == "" {
		out := make([]Symbol, len(t.Symbols))
		copy(out, t.Symbols)
		return out
	}

	filtered := make([]Symbol, 0, len(t.Symbols))
	for _, symbol := range t.Symbols {
		if strings.HasPrefix(symbol.FQN, prefix) {
			filtered = append(filtered, symbol)
		}
	}
	return filtered
}
