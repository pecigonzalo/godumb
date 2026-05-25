# godumb-writer

Teach an AI coding harness to create and maintain GoDumb source files (`.gdb`).

## Use this skill when

- Writing new GoDumb examples or programs.
- Converting Go snippets into GoDumb format.
- Reviewing `.gdb` files for style/correctness.

## Core rules

1. GoDumb is Go by tokens, not by pretty indentation.
2. Put **one lexical token per non-empty line**.
3. Keep Go syntax valid after transpilation.
4. Keep explicit `;` tokens when present in source.
5. Use `.gdb` extension for GoDumb source files.

## Tooling-agnostic workflow for agents

1. Write or update code in normal Go first.
2. Convert it to GoDumb by splitting into one token per line.
3. Validate by transpiling/parsing/building with whatever toolchain the target repo provides.

If a project has a `godumb` CLI, typical commands are:

- `godumb fmt <file.go>`
- `godumb check <file.gdb>`
- `godumb run <file.gdb>`

If it does not, follow local project conventions and scripts.

## Minimal example

Go:

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

## Common mistakes to avoid

- Joining multiple tokens on one line (except a single comment token line).
- Assuming every repo uses the same commands/tasks.
- Skipping parse/build validation after edits.
