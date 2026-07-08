package kotlincompilergolang

import (
	"bytes"
	"strings"
)

func normalizeKotlinSourceForANTLR(source []byte) []byte {
	if len(source) == 0 {
		return source
	}

	text := string(source)
	normalized := platformModifierNormalizationPattern.ReplaceAllString(text, "$1")
	normalized = valueClassNormalizationPattern.ReplaceAllString(normalized, "$1")
	normalized = funInterfaceNormalizationPattern.ReplaceAllString(normalized, "$1")
	normalized = contextReceiverNormalizationPattern.ReplaceAllString(normalized, "")
	normalized = contextFunctionTypeNormalizationPattern.ReplaceAllString(normalized, "$1")
	normalized = sealedInterfaceNormalizationPattern.ReplaceAllString(normalized, "$1")
	normalized = lambdaLabelNormalizationPattern.ReplaceAllString(normalized, "{")
	normalized = functionTypeCastNormalizationPattern.ReplaceAllString(normalized, "")
	normalized = extensionFunctionTypeCastNormalizationPattern.ReplaceAllString(normalized, "")
	normalized = arrayExtensionFunctionTypeCastNormalizationPattern.ReplaceAllString(normalized, " as Array")
	normalized = suspendReceiverFunctionTypeNormalizationPattern.ReplaceAllString(normalized, "() ->")
	normalized = receiverFunctionTypeNormalizationPattern.ReplaceAllString(normalized, "() ->")
	normalized = definitelyNonNullTypeNormalizationPattern.ReplaceAllString(normalized, "")
	normalized = nullableTypeCastNormalizationPattern.ReplaceAllString(normalized, "as $1")
	normalized = extensionStarReceiverNormalizationPattern.ReplaceAllString(normalized, "$1.")
	normalized = extensionGenericReceiverNormalizationPattern.ReplaceAllString(normalized, "$1.")
	normalized = genericSupertypeNormalizationPattern.ReplaceAllString(normalized, "$1$2$3")
	normalized = objectExpressionSupertypeNormalizationPattern.ReplaceAllString(normalized, "object {")
	normalized = anonymousFunctionAssignmentPattern.ReplaceAllString(normalized, "= { $1 ->")
	normalized = rangeUntilOperatorNormalizationPattern.ReplaceAllString(normalized, " until ")
	normalized = standaloneUnderscoreNormalizationPattern.ReplaceAllString(normalized, "${1}ignored${2}")
	normalized = normalizeStringTemplateExpressions(normalized)
	normalized = stringTemplateExpressionNormalizationPattern.ReplaceAllString(normalized, "0")
	normalized = stripAnnotationArguments(normalized)
	if normalized == text {
		return source
	}
	return []byte(normalized)
}

func normalizeStringTemplateExpressions(text string) string {
	if text == "" || !strings.Contains(text, "${") {
		return text
	}

	var out strings.Builder
	out.Grow(len(text))
	changed := false

	for i := 0; i < len(text); {
		ch := text[i]
		if ch != '"' {
			out.WriteByte(ch)
			i++
			continue
		}

		// Keep multiline strings untouched.
		if i+2 < len(text) && text[i+1] == '"' && text[i+2] == '"' {
			start := i
			i += 3
			for i+2 < len(text) {
				if text[i] == '"' && text[i+1] == '"' && text[i+2] == '"' {
					i += 3
					break
				}
				i++
			}
			if i > len(text) {
				i = len(text)
			}
			out.WriteString(text[start:i])
			continue
		}

		out.WriteByte('"')
		i++
		for i < len(text) {
			switch text[i] {
			case '\\':
				if i+1 < len(text) {
					out.WriteByte(text[i])
					out.WriteByte(text[i+1])
					i += 2
				} else {
					out.WriteByte(text[i])
					i++
				}
			case '"':
				out.WriteByte('"')
				i++
				goto nextString
			case '$':
				if i+1 < len(text) && text[i+1] == '{' {
					out.WriteByte('0')
					i += 2
					i = skipStringTemplateExpression(text, i)
					changed = true
					continue
				}
				out.WriteByte(text[i])
				i++
			default:
				out.WriteByte(text[i])
				i++
			}
		}
	nextString:
	}

	if !changed {
		return text
	}
	return out.String()
}

func shouldNormalizeKotlinSourceForANTLR(source []byte) bool {
	if len(source) == 0 {
		return false
	}
	return bytes.Contains(source, []byte("expect")) ||
		bytes.Contains(source, []byte("actual")) ||
		bytes.Contains(source, []byte("value class")) ||
		bytes.Contains(source, []byte("fun interface")) ||
		bytes.Contains(source, []byte("context(")) ||
		bytes.Contains(source, []byte(".() ->")) ||
		bytes.Contains(source, []byte("& Any")) ||
		bytes.Contains(source, []byte("sealed interface")) ||
		bytes.Contains(source, []byte("${")) ||
		bytes.Contains(source, []byte("@{")) ||
		bytes.Contains(source, []byte(" as ")) ||
		bytes.Contains(source, []byte(".<")) ||
		bytes.Contains(source, []byte(">.")) ||
		bytes.Contains(source, []byte("..<")) ||
		bytes.Contains(source, []byte("_")) ||
		genericSupertypeNormalizationPattern.Match(source) ||
		bytes.Contains(source, []byte("(suspend ")) ||
		bytes.Contains(source, []byte("= fun(")) ||
		bytes.Contains(source, []byte("object :")) ||
		bytes.Contains(source, []byte("@file:")) ||
		hasLikelyAnnotationArgumentInSource(source)
}

func hasLikelyAnnotationArgumentInSource(source []byte) bool {
	if len(source) < 2 {
		return false
	}

	inLineComment := false
	inBlockComment := false
	inSingleQuote := false
	inDoubleQuote := false
	inRawString := false

	for i := 0; i < len(source); i++ {
		ch := source[i]
		next := byte(0)
		if i+1 < len(source) {
			next = source[i+1]
		}

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if inSingleQuote {
			if ch == '\\' && i+1 < len(source) {
				i++
				continue
			}
			if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}
		if inDoubleQuote {
			if ch == '\\' && i+1 < len(source) {
				i++
				continue
			}
			if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}
		if inRawString {
			if ch == '"' && i+2 < len(source) && source[i+1] == '"' && source[i+2] == '"' {
				inRawString = false
				i += 2
			}
			continue
		}

		if ch == '/' && next == '/' {
			inLineComment = true
			i++
			continue
		}
		if ch == '/' && next == '*' {
			inBlockComment = true
			i++
			continue
		}
		if ch == '\'' {
			inSingleQuote = true
			continue
		}
		if ch == '"' {
			if i+2 < len(source) && next == '"' && source[i+2] == '"' {
				inRawString = true
				i += 2
				continue
			}
			inDoubleQuote = true
			continue
		}
		if ch != '@' {
			continue
		}

		j := i + 1
		for j < len(source) {
			if source[j] == ' ' || source[j] == '\t' || source[j] == '\r' || source[j] == '\n' {
				j++
				continue
			}
			break
		}
		if j >= len(source) || !isAnnotationNameStart(source[j]) {
			continue
		}
		for j < len(source) && isIdentifierByte(source[j]) {
			j++
		}
		for j < len(source) {
			if source[j] == ' ' || source[j] == '\t' || source[j] == '\r' || source[j] == '\n' {
				j++
				continue
			}
			break
		}
		if j < len(source) && source[j] == '(' {
			return true
		}
	}
	return false
}
