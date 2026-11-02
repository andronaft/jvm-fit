package flags

import (
	"os"
	"strings"
)

// Split breaks a string into words the way a POSIX shell does for simple
// cases: whitespace separates words, single and double quotes group them and
// a backslash escapes the next character. It does not run commands or expand
// globs.
func Split(s string) []string {
	var (
		words  []string
		cur    strings.Builder
		inWord bool
		quote  rune
		escape bool
	)
	for _, r := range s {
		switch {
		case escape:
			cur.WriteRune(r)
			escape = false
		case r == '\\' && quote != '\'':
			escape = true
			inWord = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words
}

// ExpandKube replaces $(VAR) references the way Kubernetes does in command,
// args and env values. $$(VAR) is an escape for a literal $(VAR). Unknown
// variables are left as they are.
func ExpandKube(s string, env map[string]string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '$' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		if s[i+1] == '$' {
			b.WriteByte('$')
			i++
			continue
		}
		if s[i+1] == '(' {
			if end := strings.IndexByte(s[i+2:], ')'); end >= 0 {
				name := s[i+2 : i+2+end]
				if v, ok := env[name]; ok {
					b.WriteString(v)
				} else {
					b.WriteString(s[i : i+3+end])
				}
				i += 2 + end
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// ExpandShell replaces $VAR and ${VAR} references inside a shell script.
// Unknown variables expand to nothing, like in a shell.
func ExpandShell(s string, env map[string]string) string {
	return os.Expand(s, func(name string) string { return env[name] })
}
