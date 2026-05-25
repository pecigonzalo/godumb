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

// TranspileWithMapResult contains reconstructed Go source and position mapping.
type TranspileWithMapResult struct {
	Source string
	Mapper PositionMapper
}

// PositionMapper maps generated Go positions back to GoDumb source lines.
type PositionMapper struct {
	spansByGeneratedLine map[int][]tokenSpan
}

type tokenSpan struct {
	startCol   int
	endCol     int
	sourceLine int
}

// Map returns the originating GoDumb source line for a generated Go position.
func (m PositionMapper) Map(generatedLine, generatedColumn int) (int, bool) {
	spans := m.spansByGeneratedLine[generatedLine]
	if len(spans) == 0 {
		return 0, false
	}

	if generatedColumn <= 0 {
		generatedColumn = 1
	}

	for i, span := range spans {
		if generatedColumn < span.startCol {
			if i == 0 {
				return span.sourceLine, true
			}
			return spans[i-1].sourceLine, true
		}
		if generatedColumn <= span.endCol {
			return span.sourceLine, true
		}
	}

	return spans[len(spans)-1].sourceLine, true
}

// Transpile converts GoDumb source (one token per line) into canonical Go.
func Transpile(src []byte, opts TranspileOptions) (string, error) {
	result, err := TranspileWithSourceMap(src, opts)
	if err != nil {
		return "", err
	}

	formatted, err := format.Source([]byte(result.Source))
	if err != nil {
		return "", fmt.Errorf("invalid GoDumb source: %w", err)
	}

	return string(formatted), nil
}

// TranspileWithSourceMap reconstructs Go source and returns source position mapping.
func TranspileWithSourceMap(src []byte, _ TranspileOptions) (TranspileWithMapResult, error) {
	tokens, err := readGoDumbTokens(src)
	if err != nil {
		return TranspileWithMapResult{}, err
	}

	reconstructed, mapper := reconstructGoSource(tokens)
	return TranspileWithMapResult{
		Source: reconstructed,
		Mapper: mapper,
	}, nil
}

type sourceToken struct {
	text       string
	sourceLine int
}

func readGoDumbTokens(src []byte) ([]sourceToken, error) {
	var tokens []sourceToken

	s := bufio.NewScanner(bytes.NewReader(src))
	lineNo := 0
	for s.Scan() {
		lineNo++
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		tokens = append(tokens, sourceToken{text: line, sourceLine: lineNo})
	}

	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty GoDumb input")
	}

	return tokens, nil
}

func reconstructGoSource(tokens []sourceToken) (string, PositionMapper) {
	var out strings.Builder
	out.Grow(len(tokens) * 3)

	mapper := PositionMapper{spansByGeneratedLine: make(map[int][]tokenSpan)}
	generatedLine := 1
	generatedCol := 1

	for i := range tokens {
		curr := tokens[i]
		startCol := generatedCol
		out.WriteString(curr.text)
		generatedCol += len(curr.text)
		mapper.spansByGeneratedLine[generatedLine] = append(mapper.spansByGeneratedLine[generatedLine], tokenSpan{
			startCol:   startCol,
			endCol:     generatedCol - 1,
			sourceLine: curr.sourceLine,
		})

		if i == len(tokens)-1 {
			break
		}

		next := tokens[i+1]
		if shouldInsertLineBreak(curr.text, next.text) {
			out.WriteByte('\n')
			generatedLine++
			generatedCol = 1
		} else {
			out.WriteByte(' ')
			generatedCol++
		}
	}

	out.WriteByte('\n')
	return out.String(), mapper
}

func shouldInsertLineBreak(curr, next string) bool {
	if strings.HasPrefix(curr, "//") {
		return true
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
	if tok == ")" || tok == "}" || tok == "++" || tok == "--" {
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
