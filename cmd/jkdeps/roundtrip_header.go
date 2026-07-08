package main

import (
	"regexp"
	"sort"
	"strings"

	"github.com/dh-kam/jkdeps/internal/mixedgraph"
)

var kotlinAliasImportPattern = regexp.MustCompile(`(?m)^\s*import\s+.+\s+as\s+[A-Za-z_][A-Za-z0-9_]*\s*$`)

func rewriteSourceHeader(source []byte, file mixedgraph.FileUnit) ([]byte, bool) {
	original := string(source)
	preamble, remainder, hadHeader := splitLeadingPreambleAndBody(original)
	header := buildRoundTripHeader(file.Language, file.PackageName, file.Imports, original)
	if header == "" {
		return source, false
	}

	rewritten := preamble + header + remainder
	if !hadHeader && rewritten == original {
		return source, false
	}
	return []byte(rewritten), rewritten != original
}

func isSemanticallyEquivalentHeaderRewrite(original, rewritten []byte, file mixedgraph.FileUnit) bool {
	originalPreamble, _, originalBody, _ := splitSourceSections(string(original))
	rewrittenPreamble, _, rewrittenBody, _ := splitSourceSections(string(rewritten))
	if normalizeRoundTripText([]byte(originalPreamble)) != normalizeRoundTripText([]byte(rewrittenPreamble)) {
		return false
	}
	if normalizeRoundTripText([]byte(originalBody)) != normalizeRoundTripText([]byte(rewrittenBody)) {
		return false
	}

	originalPkg, originalImports := extractHeaderDeclarationsFromSource(file.Language, string(original))
	expectedImports := filterExpectedImportsForSemanticCompare(file.Language, string(original), file.Imports)
	expectedHeader := buildCanonicalHeader(file.Language, file.PackageName, expectedImports)
	originalHeader := buildCanonicalHeader(file.Language, originalPkg, originalImports)
	return normalizeRoundTripText([]byte(originalHeader)) == normalizeRoundTripText([]byte(expectedHeader))
}

func filterExpectedImportsForSemanticCompare(lang mixedgraph.SourceLanguage, source string, imports []string) []string {
	if lang != mixedgraph.LangJava || len(imports) == 0 {
		return imports
	}
	staticOnly := extractJavaStaticImportPaths(source)
	if len(staticOnly) == 0 {
		return imports
	}
	filtered := make([]string, 0, len(imports))
	for _, imp := range imports {
		if _, ok := staticOnly[imp]; ok {
			continue
		}
		filtered = append(filtered, imp)
	}
	return filtered
}

func splitLeadingPreambleAndBody(source string) (string, string, bool) {
	preamble, _, body, hadHeader := splitSourceSections(source)
	return preamble, body, hadHeader
}

func splitSourceSections(source string) (string, string, string, bool) {
	lines := strings.SplitAfter(source, "\n")
	if len(lines) == 0 {
		return "", "", source, false
	}

	idx := 0
	if strings.HasPrefix(lines[0], "#!") {
		idx = 1
	}

	inBlockComment := false
	for idx < len(lines) {
		trimmed := strings.TrimSpace(lines[idx])
		switch {
		case inBlockComment:
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			idx++
		case trimmed == "":
			idx++
		case strings.HasPrefix(trimmed, "//"):
			idx++
		case strings.HasPrefix(trimmed, "/*"):
			inBlockComment = !strings.Contains(trimmed, "*/")
			idx++
		case strings.HasPrefix(trimmed, "@"):
			idx++
		case strings.HasPrefix(trimmed, "@file:"):
			idx++
		default:
			goto headerScan
		}
	}

headerScan:
	preamble := strings.Join(lines[:idx], "")
	headerStart := idx
	hadHeader := false
	inBlockComment = false

	for idx < len(lines) {
		trimmed := strings.TrimSpace(lines[idx])
		switch {
		case inBlockComment:
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			idx++
		case trimmed == "":
			idx++
		case strings.HasPrefix(trimmed, "//"):
			idx++
		case strings.HasPrefix(trimmed, "/*"):
			inBlockComment = !strings.Contains(trimmed, "*/")
			idx++
		case isHeaderLine(trimmed):
			hadHeader = true
			idx++
		default:
			header := strings.Join(lines[headerStart:idx], "")
			body := strings.Join(lines[idx:], "")
			if hadHeader {
				body = strings.TrimLeft(body, "\n")
				return preamble, header, body, hadHeader
			}
			preamble = strings.Join(lines[:headerStart], "")
			body = strings.Join(lines[headerStart:], "")
			return preamble, "", body, hadHeader
		}
	}

	header := strings.Join(lines[headerStart:idx], "")
	return preamble, header, "", hadHeader
}

func isHeaderLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "package ") || strings.HasPrefix(trimmed, "import ")
}

func buildCanonicalHeader(lang mixedgraph.SourceLanguage, packageName string, imports []string) string {
	var parts []string
	canonicalImports := canonicalizeImports(imports)
	if packageName != "" {
		if lang == mixedgraph.LangJava {
			parts = append(parts, "package "+packageName+";")
		} else {
			parts = append(parts, "package "+packageName)
		}
	}
	if len(canonicalImports) > 0 {
		importLines := make([]string, 0, len(canonicalImports))
		for _, imp := range canonicalImports {
			if lang == mixedgraph.LangJava {
				importLines = append(importLines, "import "+imp+";")
			} else {
				importLines = append(importLines, "import "+imp)
			}
		}
		if len(parts) > 0 {
			parts = append(parts, "")
		}
		parts = append(parts, importLines...)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n") + "\n\n"
}

func buildRoundTripHeader(lang mixedgraph.SourceLanguage, packageName string, imports []string, originalSource string) string {
	if lang == mixedgraph.LangJava && strings.Contains(originalSource, "import static ") {
		if header, ok := buildJavaHeaderPreservingStaticImports(packageName, originalSource); ok {
			return header
		}
	}
	if lang == mixedgraph.LangKotlin && kotlinAliasImportPattern.MatchString(originalSource) {
		if header, ok := buildKotlinHeaderPreservingAliasImports(packageName, originalSource); ok {
			return header
		}
	}
	return buildCanonicalHeader(lang, packageName, imports)
}

func buildJavaHeaderPreservingStaticImports(packageName string, source string) (string, bool) {
	_, header, _, hadHeader := splitSourceSections(source)
	if !hadHeader {
		return "", false
	}
	lines := strings.Split(header, "\n")
	out := make([]string, 0, len(lines))
	packageWritten := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "package ") {
			if packageName == "" {
				continue
			}
			out = append(out, "package "+packageName+";")
			packageWritten = true
			continue
		}
		if strings.HasPrefix(trimmed, "import ") {
			out = append(out, trimmed)
		}
	}
	if packageName != "" && !packageWritten {
		out = append([]string{"package " + packageName + ";"}, out...)
	}
	if len(out) == 0 {
		return "", false
	}
	final := make([]string, 0, len(out)+1)
	if packageName != "" {
		final = append(final, out[0])
		if len(out) > 1 {
			final = append(final, "")
			final = append(final, out[1:]...)
		}
	} else {
		final = append(final, out...)
	}
	return strings.Join(final, "\n") + "\n\n", true
}

func buildKotlinHeaderPreservingAliasImports(packageName string, source string) (string, bool) {
	_, header, _, hadHeader := splitSourceSections(source)
	if !hadHeader {
		return "", false
	}
	lines := strings.Split(header, "\n")
	out := make([]string, 0, len(lines))
	packageWritten := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "package ") {
			if packageName == "" {
				continue
			}
			out = append(out, "package "+packageName)
			packageWritten = true
			continue
		}
		if strings.HasPrefix(trimmed, "import ") {
			out = append(out, trimmed)
		}
	}
	if packageName != "" && !packageWritten {
		out = append([]string{"package " + packageName}, out...)
	}
	if len(out) == 0 {
		return "", false
	}
	final := make([]string, 0, len(out)+1)
	if packageName != "" {
		final = append(final, out[0])
		if len(out) > 1 {
			final = append(final, "")
			final = append(final, out[1:]...)
		}
	} else {
		final = append(final, out...)
	}
	return strings.Join(final, "\n") + "\n\n", true
}

func canonicalizeImports(imports []string) []string {
	if len(imports) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(imports))
	out := make([]string, 0, len(imports))
	for _, imp := range imports {
		imp = strings.TrimSpace(imp)
		if imp == "" {
			continue
		}
		if _, ok := seen[imp]; ok {
			continue
		}
		seen[imp] = struct{}{}
		out = append(out, imp)
	}
	sort.Strings(out)
	return out
}

func extractHeaderDeclarationsFromSource(lang mixedgraph.SourceLanguage, source string) (string, []string) {
	_, header, _, _ := splitSourceSections(source)
	lines := strings.Split(header, "\n")
	pkg := ""
	imports := make([]string, 0, 8)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "package "):
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "package "))
			pkg = strings.TrimSuffix(value, ";")
		case lang == mixedgraph.LangKotlin && strings.HasPrefix(trimmed, "import ") && strings.Contains(trimmed, " as "):
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "import "))
			value = strings.TrimSpace(strings.SplitN(value, " as ", 2)[0])
			imports = append(imports, value)
		case lang == mixedgraph.LangJava && strings.HasPrefix(trimmed, "import static "):
			continue
		case strings.HasPrefix(trimmed, "import "):
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "import "))
			imports = append(imports, strings.TrimSuffix(value, ";"))
		}
	}
	return pkg, imports
}

func extractJavaStaticImportPaths(source string) map[string]struct{} {
	_, header, _, _ := splitSourceSections(source)
	lines := strings.Split(header, "\n")
	out := make(map[string]struct{})
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "import static ") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, "import static "))
		value = strings.TrimSuffix(value, ";")
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}
