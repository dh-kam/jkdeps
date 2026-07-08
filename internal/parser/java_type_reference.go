package parser

import (
	"strings"
	"unicode"

	ast "github.com/dh-kam/jkdeps/internal/ast"
)

func parseJavaTypeReferenceText(text string) *ast.TypeReference {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	text = stripJavaTypeSuffixes(text)
	text = unwrapJavaWildcardBound(text)
	text = strings.TrimSpace(text)
	if text == "" || text == "?" {
		return nil
	}

	baseText, argText, hasArgs := splitJavaTypeArguments(text)
	ref := newJavaBaseTypeReference(baseText)
	if ref == nil {
		return nil
	}
	if !hasArgs {
		return ref
	}

	parts := splitJavaTypeArgumentList(argText)
	if len(parts) == 0 {
		return ref
	}

	ref.TypeArguments = make([]ast.TypeReference, 0, len(parts))
	for _, part := range parts {
		arg := parseJavaTypeReferenceText(part)
		if arg == nil {
			continue
		}
		ref.TypeArguments = append(ref.TypeArguments, *arg)
	}
	return ref
}

func stripJavaTypeSuffixes(text string) string {
	text = strings.TrimSpace(text)
	for strings.HasSuffix(text, "...") {
		text = strings.TrimSpace(strings.TrimSuffix(text, "..."))
	}
	for strings.HasSuffix(text, "[]") {
		text = strings.TrimSpace(strings.TrimSuffix(text, "[]"))
	}
	return text
}

func unwrapJavaWildcardBound(text string) string {
	switch {
	case strings.HasPrefix(text, "?extends"):
		return text[len("?extends"):]
	case strings.HasPrefix(text, "?super"):
		return text[len("?super"):]
	case strings.HasPrefix(text, "?"):
		return ""
	default:
		return text
	}
}

func splitJavaTypeArguments(text string) (base string, args string, hasArgs bool) {
	depth := 0
	start := -1
	for i, r := range text {
		switch r {
		case '<':
			if depth == 0 {
				start = i
			}
			depth++
		case '>':
			depth--
			if depth == 0 && start >= 0 {
				return text[:start], text[start+1 : i], true
			}
		}
	}
	return text, "", false
}

func splitJavaTypeArgumentList(text string) []string {
	if text == "" {
		return nil
	}

	parts := make([]string, 0, 4)
	start := 0
	depth := 0
	for i, r := range text {
		switch r {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				part := strings.TrimSpace(text[start:i])
				if part != "" {
					parts = append(parts, part)
				}
				start = i + 1
			}
		}
	}

	if tail := strings.TrimSpace(text[start:]); tail != "" {
		parts = append(parts, tail)
	}
	return parts
}

func newJavaBaseTypeReference(text string) *ast.TypeReference {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	segments := strings.Split(text, ".")
	if len(segments) == 0 {
		return nil
	}

	if len(segments) == 1 {
		return &ast.TypeReference{Name: text}
	}

	pkgEnd := 0
	for pkgEnd < len(segments) && looksLikeJavaPackageSegment(segments[pkgEnd]) {
		pkgEnd++
	}
	if pkgEnd == 0 || pkgEnd == len(segments) {
		return &ast.TypeReference{Name: text}
	}

	return &ast.TypeReference{
		Name:    strings.Join(segments[pkgEnd:], "."),
		Package: strings.Join(segments[:pkgEnd], "."),
	}
}

func looksLikeJavaPackageSegment(segment string) bool {
	if segment == "" {
		return false
	}
	r, _ := utf8DecodeRuneInString(segment)
	return unicode.IsLower(r)
}

func shouldSkipJavaTypeReference(ref *ast.TypeReference) bool {
	if ref == nil || ref.Name == "" {
		return true
	}
	switch ref.Name {
	case "boolean", "byte", "char", "double", "float", "int", "long", "short", "void":
		return true
	}
	return len(ref.Name) == 1 && ref.Package == "" && ref.Name[0] >= 'A' && ref.Name[0] <= 'Z'
}

func utf8DecodeRuneInString(s string) (rune, int) {
	for _, r := range s {
		return r, len(string(r))
	}
	return rune(0), 0
}
