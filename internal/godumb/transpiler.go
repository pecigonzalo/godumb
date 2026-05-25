package godumb

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"
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
	tokenIndex int
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

func (m PositionMapper) tokenIndexAt(generatedLine, generatedColumn int) (int, bool) {
	spans := m.spansByGeneratedLine[generatedLine]
	if len(spans) == 0 {
		return 0, false
	}

	if generatedColumn <= 0 {
		generatedColumn = 1
	}

	if generatedColumn < spans[0].startCol {
		return spans[0].tokenIndex, true
	}

	for i, span := range spans {
		if generatedColumn <= span.endCol {
			return span.tokenIndex, true
		}
		if i+1 < len(spans) && generatedColumn < spans[i+1].startCol {
			return spans[i+1].tokenIndex, true
		}
	}

	return spans[len(spans)-1].tokenIndex, true
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
	if len(tokens) == 1 {
		source, mapper := renderTokens(tokens, nil)
		return source, mapper
	}

	separators := make([]byte, len(tokens)-1)
	for i := range separators {
		if strings.HasPrefix(tokens[i].text, "//") {
			separators[i] = '\n'
			continue
		}
		separators[i] = ' '
	}

	source, mapper := renderTokens(tokens, separators)

	maxIterations := len(tokens)*2 + 8
	for i := 0; i < maxIterations; i++ {
		err, hasError := firstParseError(source)
		if !hasError {
			return source, mapper
		}

		if !isRecoverableBoundaryError(err.message) {
			return source, mapper
		}

		tokenIdx, ok := mapper.tokenIndexAt(err.line, err.column)
		if !ok || tokenIdx == 0 {
			return source, mapper
		}

		sepIdx := tokenIdx - 1
		if sepIdx < 0 || sepIdx >= len(separators) {
			return source, mapper
		}
		if separators[sepIdx] == '\n' {
			return source, mapper
		}

		separators[sepIdx] = '\n'
		source, mapper = renderTokens(tokens, separators)
	}

	return source, mapper
}

func renderTokens(tokens []sourceToken, separators []byte) (string, PositionMapper) {
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
			tokenIndex: i,
		})

		if i == len(tokens)-1 {
			break
		}

		sep := byte(' ')
		if len(separators) > 0 {
			sep = separators[i]
		}

		if sep == '\n' {
			out.WriteByte('\n')
			generatedLine++
			generatedCol = 1
			continue
		}

		out.WriteByte(' ')
		generatedCol++
	}

	out.WriteByte('\n')
	return out.String(), mapper
}

type parseError struct {
	line    int
	column  int
	message string
}

func firstParseError(source string) (parseError, bool) {
	_, err := parser.ParseFile(token.NewFileSet(), "generated.go", source, parser.AllErrors)
	if err == nil {
		return parseError{}, false
	}

	switch parsed := err.(type) {
	case scanner.ErrorList:
		if len(parsed) == 0 {
			return parseError{message: err.Error()}, true
		}
		first := parsed[0]
		return parseError{line: first.Pos.Line, column: first.Pos.Column, message: first.Msg}, true
	case *scanner.ErrorList:
		if len(*parsed) == 0 {
			return parseError{message: err.Error()}, true
		}
		first := (*parsed)[0]
		return parseError{line: first.Pos.Line, column: first.Pos.Column, message: first.Msg}, true
	default:
		return parseError{message: err.Error()}, true
	}
}

func isRecoverableBoundaryError(msg string) bool {
	return strings.Contains(msg, "expected ';'") || strings.Contains(msg, "missing import path")
}
