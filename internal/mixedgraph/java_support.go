package mixedgraph

import (
	"regexp"
	"sort"

	"github.com/antlr4-go/antlr/v4"
)

var javaPackagePattern = regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z_][A-Za-z0-9_\.]*)\s*;`)
var javaImportPattern = regexp.MustCompile(`(?m)^\s*import\s+(?:static\s+)?([A-Za-z_][A-Za-z0-9_\.\*]*)\s*;`)
var pathIgnoreTable = [256]bool{
	'`':  true,
	' ':  true,
	'\t': true,
	'\n': true,
	'\r': true,
	'\f': true,
	'\v': true,
}

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
	return pathIgnoreTable[ch]
}

type syntaxErrorListener struct {
	*antlr.DefaultErrorListener
	path       string
	maxErrors  int
	diagnostic []Diagnostic
}

func newSyntaxErrorListener(path string, maxErrors int) *syntaxErrorListener {
	if maxErrors <= 0 {
		maxErrors = 10
	}
	return &syntaxErrorListener{
		DefaultErrorListener: &antlr.DefaultErrorListener{},
		path:                 path,
		maxErrors:            maxErrors,
		diagnostic:           make([]Diagnostic, 0, maxErrors),
	}
}

func (l *syntaxErrorListener) SyntaxError(_ antlr.Recognizer, _ interface{}, line, column int, msg string, _ antlr.RecognitionException) {
	l.addMessage(line, column+1, msg)
}

func (l *syntaxErrorListener) addMessage(line, column int, msg string) {
	if len(l.diagnostic) >= l.maxErrors {
		return
	}
	l.diagnostic = append(l.diagnostic, Diagnostic{
		Path:    l.path,
		Line:    line,
		Column:  column,
		Message: msg,
	})
}

func (l *syntaxErrorListener) Diagnostics() []Diagnostic {
	if len(l.diagnostic) == 0 {
		return nil
	}
	out := make([]Diagnostic, len(l.diagnostic))
	copy(out, l.diagnostic)
	return out
}
