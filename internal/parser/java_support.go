package parser

import (
	"regexp"
	"sort"
)

var javaPackagePattern = regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z_][A-Za-z0-9_\.]*)\s*;`)
var javaImportPattern = regexp.MustCompile(`(?m)^\s*import\s+(?:static\s+)?([A-Za-z_][A-Za-z0-9_\.\*]*)\s*;`)

// Kotlin patterns (no semicolons)
var kotlinPackagePattern = regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z_][A-Za-z0-9_\.]*)\s*(?://.*)?$`)
var kotlinImportPattern = regexp.MustCompile(`(?m)^\s*import\s+([A-Za-z_][A-Za-z0-9_\.\*]*)\s*(?://.*)?$`)
var kotlinAliasImportPattern = regexp.MustCompile(`(?m)^\s*import\s+([A-Za-z_][A-Za-z0-9_\.\*]*)\s+as\s+([A-Za-z_][A-Za-z0-9_]*)\s*(?://.*)?$`)

// extractJavaHeader extracts package name and imports from Java source
func extractJavaHeader(text string) (string, []string) {
	pkg := ""
	if matches := javaPackagePattern.FindStringSubmatch(text); len(matches) == 2 {
		pkg = normalizePath(matches[1])
	}

	matches := javaImportPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return pkg, nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		value := normalizePath(match[1])
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return pkg, out
}

// normalizePath cleans up package/import paths
func normalizePath(value string) string {
	if value == "" {
		return ""
	}

	start, end := 0, len(value)-1
	for start <= end && isPathIgnoreChar(value[start]) {
		start++
	}
	for end >= start && isPathIgnoreChar(value[end]) {
		end--
	}
	if start > end {
		return ""
	}

	trimmed := value[start : end+1]
	needsCleanup := trimmed[0] == '.' || trimmed[len(trimmed)-1] == '.'
	if !needsCleanup {
		for i := 0; i < len(trimmed); i++ {
			if isPathIgnoreChar(trimmed[i]) {
				needsCleanup = true
				break
			}
		}
	}
	if !needsCleanup {
		return trimmed
	}

	buf := make([]byte, 0, len(trimmed))
	lastNonDot := -1
	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]
		if isPathIgnoreChar(ch) {
			continue
		}
		if len(buf) == 0 && ch == '.' {
			continue
		}
		buf = append(buf, ch)
		if ch != '.' {
			lastNonDot = len(buf)
		}
	}
	if lastNonDot < 0 {
		return ""
	}
	return string(buf[:lastNonDot])
}

func isPathIgnoreChar(ch byte) bool {
	return ch == '`' || ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' || ch == '\v'
}

// extractKotlinHeader extracts package name and imports from Kotlin source
// Kotlin imports don't require semicolons and support aliasing (import x as y)
func extractKotlinHeader(text string) (string, []string) {
	pkg := ""
	if matches := kotlinPackagePattern.FindStringSubmatch(text); len(matches) == 2 {
		pkg = normalizePath(matches[1])
	}

	// First check for alias imports (import x as y)
	aliasMatches := kotlinAliasImportPattern.FindAllStringSubmatch(text, -1)
	seenAliases := map[string]struct{}{}
	aliases := make(map[string]string) // alias -> original
	for _, match := range aliasMatches {
		if len(match) != 3 {
			continue
		}
		original := normalizePath(match[1])
		alias := match[2]
		if original == "" || alias == "" {
			continue
		}
		if _, ok := seenAliases[alias]; ok {
			continue
		}
		seenAliases[alias] = struct{}{}
		aliases[alias] = original
	}

	// Then regular imports
	matches := kotlinImportPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 && len(aliases) == 0 {
		return pkg, nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches)+len(aliases))

	// Add regular imports
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		value := normalizePath(match[1])
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	// Add aliased imports as original (the real import)
	for _, original := range aliases {
		if _, ok := seen[original]; ok {
			continue
		}
		seen[original] = struct{}{}
		out = append(out, original)
	}

	sort.Strings(out)
	return pkg, out
}
