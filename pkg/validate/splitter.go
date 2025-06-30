package validate

import (
	"strings"
	"unicode"
)

// GenericSplit splits SQL text into individual statements respecting quoted
// strings, comments and dollar-quoted sections. Dialects may override this
// if needed.
func GenericSplit(sqlStr string) ([]string, error) {
	var stmts []string
	var sb strings.Builder
	var inSQuote, inDQuote bool
	var inLineComment, inBlockComment bool
	var dollarTag string

	flush := func() {
		stmt := strings.TrimSpace(sb.String())
		if stmt != "" {
			stmts = append(stmts, stmt)
		}
		sb.Reset()
	}

	for i := 0; i < len(sqlStr); i++ {
		c := sqlStr[i]
		next := byte(0)
		if i+1 < len(sqlStr) {
			next = sqlStr[i+1]
		}

		switch {
		case inLineComment:
			if c == '\n' {
				inLineComment = false
			}
			sb.WriteByte(c)
			continue
		case inBlockComment:
			if c == '*' && next == '/' {
				inBlockComment = false
				sb.WriteByte(c)
				sb.WriteByte(next)
				i++
				continue
			}
			sb.WriteByte(c)
			continue
		case inSQuote:
			sb.WriteByte(c)
			if c == '\'' {
				if next == '\'' {
					sb.WriteByte(next)
					i++
				} else {
					inSQuote = false
				}
			}
			continue
		case inDQuote:
			sb.WriteByte(c)
			if c == '"' {
				if next == '"' {
					sb.WriteByte(next)
					i++
				} else {
					inDQuote = false
				}
			}
			continue
		case dollarTag != "":
			sb.WriteByte(c)
			if len(sqlStr[i:]) >= len(dollarTag) && strings.HasPrefix(sqlStr[i:], dollarTag) {
				sb.WriteString(dollarTag)
				i += len(dollarTag) - 1
				dollarTag = ""
			}
			continue
		}

		if c == '-' && next == '-' {
			inLineComment = true
			sb.WriteByte(c)
			sb.WriteByte(next)
			i++
			continue
		}
		if c == '/' && next == '*' {
			inBlockComment = true
			sb.WriteByte(c)
			sb.WriteByte(next)
			i++
			continue
		}

		if c == '\'' {
			inSQuote = true
			sb.WriteByte(c)
			continue
		}
		if c == '"' {
			inDQuote = true
			sb.WriteByte(c)
			continue
		}
		if c == '$' {
			j := i + 1
			for j < len(sqlStr) && sqlStr[j] != '$' {
				if !(unicode.IsLetter(rune(sqlStr[j])) || unicode.IsDigit(rune(sqlStr[j])) || sqlStr[j] == '_') {
					j = i
					break
				}
				j++
			}
			if j > i && j < len(sqlStr) && sqlStr[j] == '$' {
				dollarTag = sqlStr[i : j+1]
				sb.WriteString(dollarTag)
				i = j
				continue
			}
		}

		if c == ';' {
			flush()
			continue
		}

		sb.WriteByte(c)
	}
	flush()
	return stmts, nil
}
