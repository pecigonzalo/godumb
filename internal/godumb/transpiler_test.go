package godumb_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/pecigonzalo/godumb/internal/godumb"
)

func TestTranspileHelloWorld(t *testing.T) {
	input := `package
main
import
"fmt"
func
main
(
)
{
fmt
.
Println
(
"hello world"
)
}
`

	got, err := godumb.Transpile([]byte(input), godumb.TranspileOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := parser.ParseFile(token.NewFileSet(), "hello.go", got, parser.AllErrors); err != nil {
		t.Fatalf("transpiled output should parse as Go: %v\nsource:\n%s", err, got)
	}

	roundtrip, err := godumb.Format([]byte(got), godumb.FormatOptions{})
	if err != nil {
		t.Fatalf("unexpected format error for transpiled output: %v", err)
	}

	if roundtrip != input {
		t.Fatalf("token roundtrip mismatch\nwant:\n%s\ngot:\n%s", input, roundtrip)
	}
}

func TestTranspileRoundTripFormatAndTranspile(t *testing.T) {
	source := `package main

import "fmt"

func main() {
	fmt.Println("hello")
	fmt.Println("world")
	for i := 0; i < 3; i++ {
		fmt.Println(i)
	}
}
`

	godumbSrc, err := godumb.Format([]byte(source), godumb.FormatOptions{})
	if err != nil {
		t.Fatalf("format should succeed: %v", err)
	}

	got, err := godumb.Transpile([]byte(godumbSrc), godumb.TranspileOptions{})
	if err != nil {
		t.Fatalf("transpile should succeed: %v", err)
	}

	if _, err := parser.ParseFile(token.NewFileSet(), "roundtrip.go", got, parser.AllErrors); err != nil {
		t.Fatalf("transpiled output should parse as Go: %v\nsource:\n%s", err, got)
	}

	formattedOriginal, err := godumb.Format([]byte(source), godumb.FormatOptions{})
	if err != nil {
		t.Fatalf("format should succeed for original source: %v", err)
	}

	formattedRoundtrip, err := godumb.Format([]byte(got), godumb.FormatOptions{})
	if err != nil {
		t.Fatalf("format should succeed for roundtrip source: %v", err)
	}

	if formattedOriginal != formattedRoundtrip {
		t.Fatalf("roundtrip token mismatch\nwant:\n%s\ngot:\n%s", formattedOriginal, formattedRoundtrip)
	}
}

func TestTranspileRejectsInvalidInput(t *testing.T) {
	input := `package
main
@
`

	_, err := godumb.Transpile([]byte(input), godumb.TranspileOptions{})
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestTranspileWithSourceMapMapsGeneratedPositions(t *testing.T) {
	input := `package
main

func
main
(
)
{
unknown
=
1
}
`

	result, err := godumb.TranspileWithSourceMap([]byte(input), godumb.TranspileOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "mapped.go", result.Source, parser.AllErrors)
	if err != nil {
		t.Fatalf("reconstructed source should parse: %v\n%s", err, result.Source)
	}

	generatedLine := 0
	generatedCol := 0
	ast.Inspect(file, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok || ident.Name != "unknown" {
			return true
		}
		pos := fset.Position(ident.NamePos)
		generatedLine = pos.Line
		generatedCol = pos.Column
		return false
	})

	if generatedLine == 0 {
		t.Fatal("expected unknown identifier in parsed AST")
	}

	sourceLine, ok := result.Mapper.Map(generatedLine, generatedCol)
	if !ok {
		t.Fatal("expected mapped source line")
	}
	if sourceLine != 9 {
		t.Fatalf("unexpected mapped line: got %d want %d", sourceLine, 9)
	}
}
