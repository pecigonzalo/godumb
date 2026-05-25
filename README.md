# GoDumb

GoDumb is **Go, but hyperoptimized for the AI line-count era**.

Core idea: same Go syntax, but written as one token per line.

## Example

Normal Go:

```go
package main

import "fmt"

func main() {
    fmt.Println("hello world")
}
```

GoDumb:

```go
package
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
```

## Quickstart

```bash
nix develop
```

Then:

```bash
task test
task build
task gen:examples
```

Format a file into GoDumb style:

```bash
go run ./cmd/godumb fmt examples/hello.go
```

Write in place:

```bash
go run ./cmd/godumb fmt -w examples/hello.go
```

## Repo layout

- `cmd/godumb`: CLI entrypoint
- `internal/godumb`: GoDumb formatter/token-line converter
- `examples`: sample `.go` and `.gdb` files

## Next steps

- Add a dedicated lexer/parser pipeline
- Define `.gdb` / `.godumb` source conventions
- Add transpile + execute pipeline (`godumb run`)
