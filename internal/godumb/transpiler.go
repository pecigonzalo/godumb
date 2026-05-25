package godumb

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TranspileOptions controls optional GoDumb transpilation behavior.
type TranspileOptions struct{}

// Transpile converts GoDumb source (one token per line) into canonical Go.
func Transpile(src []byte, _ TranspileOptions) (string, error) {
	tokens, err := readGoDumbTokens(src)
	if err != nil {
		return "", err
	}

	reconstructed := reconstructGoSource(tokens)
	formatted, err := format.Source([]byte(reconstructed))
	if err != nil {
		return "", fmt.Errorf("invalid GoDumb source: %w", err)
	}

	return string(formatted), nil
}

func readGoDumbTokens(src []byte) ([]string, error) {
	var tokens []string

	s := bufio.NewScanner(bytes.NewReader(src))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		tokens = append(tokens, line)
	}

	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty GoDumb input")
	}

	return tokens, nil
}

func reconstructGoSource(tokens []string) string {
	var out strings.Builder
	out.Grow(len(tokens) * 3)

	state := reconstructionState{}

	for i := range tokens {
		curr := tokens[i]
		out.WriteString(curr)

		if i == len(tokens)-1 {
			break
		}

		state = state.after(curr)
		next := tokens[i+1]
		if shouldInsertLineBreak(curr, next, state) {
			out.WriteByte('\n')
		} else {
			out.WriteByte(' ')
		}
	}

	out.WriteByte('\n')
	return out.String()
}

type reconstructionState struct {
	parenDepth   int
	bracketDepth int
}

func (s reconstructionState) after(tok string) reconstructionState {
	switch tok {
	case "(":
		s.parenDepth++
	case ")":
		if s.parenDepth > 0 {
			s.parenDepth--
		}
	case "[":
		s.bracketDepth++
	case "]":
		if s.bracketDepth > 0 {
			s.bracketDepth--
		}
	}
	return s
}

func shouldInsertLineBreak(curr, next string, state reconstructionState) bool {
	if strings.HasPrefix(curr, "//") {
		return true
	}

	if state.parenDepth > 0 || state.bracketDepth > 0 {
		return false
	}

	if curr == "}" {
		switch next {
		case ")", "]", ",", ".", ";", ":":
			return false
		default:
			return true
		}
	}

	if canEndStatement(curr) && canStartStatement(next) {
		return true
	}

	if next == "case" || next == "default" {
		return canEndStatement(curr) || curr == "}" || curr == ":"
	}

	return false
}

func canEndStatement(tok string) bool {
	if tok == ")" || tok == "]" || tok == "}" || tok == "++" || tok == "--" {
		return true
	}

	if endStmtKeywords[tok] {
		return true
	}

	kind := classifyLexeme(tok)
	return kind == lexemeIdentifier || kind == lexemeLiteral
}

func canStartStatement(tok string) bool {
	if startStmtKeywords[tok] || prefixExprStart[tok] {
		return true
	}

	kind := classifyLexeme(tok)
	return kind == lexemeIdentifier || kind == lexemeLiteral
}

type lexemeKind uint8

const (
	lexemeUnknown lexemeKind = iota
	lexemeIdentifier
	lexemeLiteral
)

func classifyLexeme(tok string) lexemeKind {
	if isIdentifier(tok) {
		return lexemeIdentifier
	}
	if isLiteral(tok) {
		return lexemeLiteral
	}
	return lexemeUnknown
}

func isIdentifier(tok string) bool {
	if tok == "" {
		return false
	}

	r, size := utf8.DecodeRuneInString(tok)
	if r == utf8.RuneError {
		return false
	}
	if r != '_' && !unicode.IsLetter(r) {
		return false
	}

	for _, r := range tok[size:] {
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}

	return !allKeywords[tok]
}

func isLiteral(tok string) bool {
	if tok == "" {
		return false
	}

	if strings.HasPrefix(tok, "\"") || strings.HasPrefix(tok, "`") || strings.HasPrefix(tok, "'") {
		return true
	}

	r, _ := utf8.DecodeRuneInString(tok)
	if unicode.IsDigit(r) {
		return true
	}

	return len(tok) > 1 && tok[0] == '.' && tok[1] >= '0' && tok[1] <= '9'
}

var endStmtKeywords = map[string]bool{
	"break":       true,
	"continue":    true,
	"fallthrough": true,
	"return":      true,
}

var startStmtKeywords = map[string]bool{
	"break":       true,
	"case":        true,
	"const":       true,
	"continue":    true,
	"default":     true,
	"defer":       true,
	"fallthrough": true,
	"for":         true,
	"func":        true,
	"go":          true,
	"goto":        true,
	"if":          true,
	"import":      true,
	"package":     true,
	"return":      true,
	"select":      true,
	"switch":      true,
	"type":        true,
	"var":         true,
}

var prefixExprStart = map[string]bool{
	"!":  true,
	"*":  true,
	"+":  true,
	"-":  true,
	"<-": true,
	"&":  true,
	"^":  true,
}

var allKeywords = map[string]bool{
	"break":       true,
	"case":        true,
	"chan":        true,
	"const":       true,
	"continue":    true,
	"default":     true,
	"defer":       true,
	"else":        true,
	"fallthrough": true,
	"for":         true,
	"func":        true,
	"go":          true,
	"goto":        true,
	"if":          true,
	"import":      true,
	"interface":   true,
	"map":         true,
	"package":     true,
	"range":       true,
	"return":      true,
	"select":      true,
	"struct":      true,
	"switch":      true,
	"type":        true,
	"var":         true,
}
