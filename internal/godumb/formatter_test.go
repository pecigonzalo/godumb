package godumb_test

import (
	"testing"

	"github.com/pecigonzalo/godumb/internal/godumb"
)

func TestFormatHelloWorld(t *testing.T) {
	input := `package main

import "fmt"

func main() {
	fmt.Println("hello world")
}
`

	got, err := godumb.Format([]byte(input), godumb.FormatOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `package
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

	if got != want {
		t.Fatalf("formatted output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatPreservesExplicitSemicolons(t *testing.T) {
	input := `package main
func main() {
	for i := 0; i < 3; i++ {}
}
`

	got, err := godumb.Format([]byte(input), godumb.FormatOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `package
main
func
main
(
)
{
for
i
:=
0
;
i
<
3
;
i
++
{
}
}
`

	if got != want {
		t.Fatalf("formatted output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatOptionKeepComments(t *testing.T) {
	input := `package main
// line counts matter
func main() {}
`

	got, err := godumb.Format([]byte(input), godumb.FormatOptions{KeepComments: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `package
main
// line counts matter
func
main
(
)
{
}
`

	if got != want {
		t.Fatalf("formatted output mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}
