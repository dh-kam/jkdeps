package mixedgraph

import "strings"

var knownJavaLangSimpleTypes = map[string]struct{}{
	"Appendable":       {},
	"AutoCloseable":    {},
	"CharSequence":     {},
	"Class":            {},
	"Cloneable":        {},
	"Comparable":       {},
	"Enum":             {},
	"Exception":        {},
	"Iterable":         {},
	"Number":           {},
	"Object":           {},
	"Record":           {},
	"Runnable":         {},
	"RuntimeException": {},
	"String":           {},
	"StringBuilder":    {},
	"StringBuffer":     {},
	"System":           {},
	"Throwable":        {},
	"Void":             {},
}

func (c graphBuildContext) resolveReference(file FileUnit, rawPath string) resolvedImportTarget {
	for _, candidate := range buildReferenceCandidates(file.PackageName, file.Imports, rawPath) {
		target := c.resolveImportNormalized(candidate)
		if target.Package != "" {
			return target
		}
	}
	return resolvedImportTarget{}
}

func buildReferenceCandidates(packageName string, imports []string, rawPath string) []string {
	rawPath = normalizePath(rawPath)
	if rawPath == "" {
		return nil
	}

	seen := make(map[string]struct{}, len(imports)+4)
	candidates := make([]string, 0, len(imports)+4)
	appendCandidate := func(value string) {
		value = normalizePath(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		candidates = append(candidates, value)
	}

	if looksQualifiedReferencePath(rawPath) {
		appendCandidate(rawPath)
	}

	root, suffix := splitReferenceRoot(rawPath)
	for _, imp := range imports {
		normalizedImport := normalizePath(imp)
		if normalizedImport == "" {
			continue
		}
		if strings.HasSuffix(normalizedImport, ".*") {
			appendCandidate(strings.TrimSuffix(normalizedImport, "*") + rawPath)
			continue
		}
		importLeaf := lastImportSegment(normalizedImport)
		if !looksLikeTypeSegment(importLeaf) || importLeaf != root {
			continue
		}
		appendCandidate(normalizedImport + suffix)
	}

	if _, ok := knownJavaLangSimpleTypes[root]; ok {
		appendCandidate("java.lang." + rawPath)
	}
	if pkg := normalizePath(packageName); pkg != "" {
		appendCandidate(pkg + "." + rawPath)
	}
	return candidates
}

func looksQualifiedReferencePath(path string) bool {
	if !strings.Contains(path, ".") {
		return false
	}
	first := path
	if dot := strings.IndexByte(path, '.'); dot >= 0 {
		first = path[:dot]
	}
	return !looksLikeTypeSegment(first)
}

func splitReferenceRoot(path string) (root string, suffix string) {
	if dot := strings.IndexByte(path, '.'); dot >= 0 {
		return path[:dot], path[dot:]
	}
	return path, ""
}
