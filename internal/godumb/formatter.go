package godumb

import (
	"go/scanner"
	"go/token"
	"strings"
)

// FormatOptions controls optional GoDumb formatting behaviors.
type FormatOptions struct {
	KeepComments bool
}

// Format converts valid Go source into GoDumb style:
// one lexical token per line.
func Format(src []byte, opts FormatOptions) (string, error) {
	var (
		s    scanner.Scanner
		errs scanner.ErrorList
	)

	fset := token.NewFileSet()
	file := fset.AddFile("input.go", -1, len(src))

	mode := scanner.Mode(0)
	if opts.KeepComments {
		mode |= scanner.ScanComments
	}

	s.Init(file, src, func(pos token.Position, msg string) {
		errs.Add(pos, msg)
	}, mode)

	var out strings.Builder
	first := true

	for {
		_, tok, lit := s.Scan()

		if tok == token.EOF {
			break
		}

		if tok == token.COMMENT && !opts.KeepComments {
			continue
		}

		// Keep explicit semicolons (for-loops), drop compiler-inserted ones.
		if tok == token.SEMICOLON && lit == "\n" {
			continue
		}

		text := lit
		if text == "" {
			text = tok.String()
		}

		if !first {
			out.WriteByte('\n')
		}
		out.WriteString(text)
		first = false
	}

	if len(errs) > 0 {
		return "", errs
	}

	if !first {
		out.WriteByte('\n')
	}

	return out.String(), nil
}
