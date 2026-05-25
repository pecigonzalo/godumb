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
task check
task check:examples
task build
task gen:examples
task sync:gobyexample
task transpile:hello
task build:hello
task run:hello
```

Format a file into GoDumb style:

```bash
go run ./cmd/godumb fmt examples/hello.go
```

Write in place:

```bash
go run ./cmd/godumb fmt -w examples/hello.go
```

Transpile GoDumb back to Go:

```bash
go run ./cmd/godumb transpile examples/hello.gdb
```

Write transpiled output to a `.go` file:

```bash
go run ./cmd/godumb transpile -w examples/hello.gdb
```

Build a GoDumb source directly into a binary:

```bash
go run ./cmd/godumb build -o ./bin/hello examples/hello.gdb
```

Run a GoDumb source directly (transpile + build + execute):

```bash
go run ./cmd/godumb run examples/hello.gdb
```

Check GoDumb source with diagnostics mapped to `.gdb` lines:

```bash
go run ./cmd/godumb check examples/hello.gdb
```

Sync curated examples from Go by Example:

```bash
task sync:gobyexample
```

## Repo layout

- `cmd/godumb`: CLI entrypoint
- `cmd/godumb-examples`: Go by Example sync helper
- `internal/godumb`: formatter and transpiler core
- `examples`: sample `.go` and `.gdb` files

## Next steps

- Improve multi-file package checking/building from `.gdb` directories
- Add richer diagnostics (column-level/token-level mapping)
- Explore a dedicated parser pipeline inspired by Thorsten Ball when needed
