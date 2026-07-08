package parser

import "bytes"

func normalizeJavaSourceForANTLR(source []byte) []byte {
	if len(source) == 0 ||
		(!bytes.Contains(source, []byte(`"""`)) &&
			!bytes.Contains(source, []byte("when")) &&
			!bytes.Contains(source, []byte("_"))) {
		return source
	}

	var out bytes.Buffer
	out.Grow(len(source))
	changed := false
	inLineComment := false
	inBlockComment := false
	inString := false
	inChar := false

	for i := 0; i < len(source); {
		ch := source[i]
		next := byte(0)
		if i+1 < len(source) {
			next = source[i+1]
		}

		if inLineComment {
			out.WriteByte(ch)
			i++
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			out.WriteByte(ch)
			if ch == '*' && next == '/' {
				out.WriteByte(next)
				i += 2
				inBlockComment = false
				continue
			}
			i++
			continue
		}
		if inString {
			out.WriteByte(ch)
			i++
			if ch == '\\' && i < len(source) {
				out.WriteByte(source[i])
				i++
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if inChar {
			out.WriteByte(ch)
			i++
			if ch == '\\' && i < len(source) {
				out.WriteByte(source[i])
				i++
				continue
			}
			if ch == '\'' {
				inChar = false
			}
			continue
		}

		if ch == '/' && next == '/' {
			out.WriteByte(ch)
			out.WriteByte(next)
			i += 2
			inLineComment = true
			continue
		}
		if ch == '/' && next == '*' {
			out.WriteByte(ch)
			out.WriteByte(next)
			i += 2
			inBlockComment = true
			continue
		}
		if ch == '"' && i+2 < len(source) && source[i+1] == '"' && source[i+2] == '"' {
			out.WriteString(`""`)
			i += 3
			for i+2 < len(source) {
				if source[i] == '"' && source[i+1] == '"' && source[i+2] == '"' {
					i += 3
					break
				}
				i++
			}
			changed = true
			continue
		}
		if ch == '"' {
			out.WriteByte(ch)
			i++
			inString = true
			continue
		}
		if ch == '\'' {
			out.WriteByte(ch)
			i++
			inChar = true
			continue
		}
		if isJavaIdentifierStart(ch) {
			start := i
			i++
			for i < len(source) && isJavaIdentifierPart(source[i]) {
				i++
			}
			word := source[start:i]
			if bytes.Equal(word, []byte("when")) && !isJavaWhenGuard(source, start) {
				out.WriteString("when_")
				changed = true
				continue
			}
			if bytes.Equal(word, []byte("_")) {
				out.WriteString("_ignored")
				changed = true
				continue
			}
			out.Write(word)
			continue
		}

		out.WriteByte(ch)
		i++
	}

	if !changed {
		return source
	}
	return out.Bytes()
}

func isJavaIdentifierStart(ch byte) bool {
	return ch == '_' || ch == '$' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func isJavaIdentifierPart(ch byte) bool {
	return isJavaIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

func isJavaWhenGuard(source []byte, whenStart int) bool {
	lineStart := whenStart
	for lineStart > 0 && source[lineStart-1] != '\n' && source[lineStart-1] != '\r' {
		lineStart--
	}
	prefix := source[lineStart:whenStart]
	if !bytes.Contains(prefix, []byte("case ")) {
		return false
	}
	if bytes.Contains(prefix, []byte("->")) {
		return false
	}
	return true
}
