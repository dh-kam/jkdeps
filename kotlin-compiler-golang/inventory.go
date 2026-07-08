package kotlincompilergolang

import (
	"encoding/json"
	"os"
	"sort"
)

type InventoryPackage struct {
	Package string `json:"package"`
	Count   int    `json:"count"`
}

type EmbeddableInventory struct {
	JarPath         string             `json:"jar_path"`
	SourceJars      []string           `json:"source_jars,omitempty"`
	ClassFiles      int                `json:"class_files"`
	TopLevelClasses int                `json:"top_level_classes"`
	BuiltinsFiles   int                `json:"builtins_files"`
	MetadataFiles   int                `json:"metadata_files,omitempty"`
	Packages        []InventoryPackage `json:"packages"`
	Symbols         []string           `json:"symbols,omitempty"`
}

type ExternalIndex struct {
	Packages map[string]struct{}
	Symbols  map[string]struct{}
}

func NewExternalIndex() ExternalIndex {
	return ExternalIndex{
		Packages: map[string]struct{}{},
		Symbols:  map[string]struct{}{},
	}
}

func (i ExternalIndex) HasPackage(pkg string) bool {
	if len(i.Packages) == 0 || pkg == "" {
		return false
	}
	_, ok := i.Packages[pkg]
	return ok
}

func (i ExternalIndex) HasSymbol(symbol string) bool {
	if len(i.Symbols) == 0 || symbol == "" {
		return false
	}
	_, ok := i.Symbols[symbol]
	return ok
}

func (i ExternalIndex) PackageNames() []string {
	names := make([]string, 0, len(i.Packages))
	for name := range i.Packages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (i ExternalIndex) SymbolNames() []string {
	names := make([]string, 0, len(i.Symbols))
	for name := range i.Symbols {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func MergeExternalIndices(indices ...ExternalIndex) ExternalIndex {
	merged := NewExternalIndex()
	for _, index := range indices {
		for pkg := range index.Packages {
			merged.Packages[pkg] = struct{}{}
		}
		for symbol := range index.Symbols {
			merged.Symbols[symbol] = struct{}{}
		}
	}
	return merged
}

func LoadExternalIndex(path string) (ExternalIndex, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return ExternalIndex{}, err
	}

	var inv EmbeddableInventory
	if err := json.Unmarshal(payload, &inv); err != nil {
		return ExternalIndex{}, err
	}

	index := NewExternalIndex()
	for _, stat := range inv.Packages {
		name := normalizeImportPath(stat.Package)
		if name == "" {
			continue
		}
		index.Packages[name] = struct{}{}
	}
	for _, symbol := range inv.Symbols {
		name := normalizeImportPath(symbol)
		if name == "" {
			continue
		}
		index.Symbols[name] = struct{}{}
	}
	return index, nil
}

func LoadExternalIndices(paths []string) (ExternalIndex, error) {
	if len(paths) == 0 {
		return NewExternalIndex(), nil
	}
	indices := make([]ExternalIndex, 0, len(paths))
	for _, path := range paths {
		index, err := LoadExternalIndex(path)
		if err != nil {
			return ExternalIndex{}, err
		}
		indices = append(indices, index)
	}
	return MergeExternalIndices(indices...), nil
}

func LoadInventoryPackages(path string) (map[string]struct{}, error) {
	index, err := LoadExternalIndex(path)
	if err != nil {
		return nil, err
	}
	return index.Packages, nil
}
