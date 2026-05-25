package main

import "testing"

func TestSelectExamplesDefaultIncludesCanonicalRangeSlug(t *testing.T) {
	selected, err := selectExamples("")
	if err != nil {
		t.Fatalf("selectExamples returned error: %v", err)
	}

	found := false
	for _, slug := range selected {
		if slug == "range-over-built-in-types" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected canonical range slug in defaults, got %v", selected)
	}
}

func TestSelectExamplesSupportsRangeAlias(t *testing.T) {
	selected, err := selectExamples("range")
	if err != nil {
		t.Fatalf("selectExamples returned error: %v", err)
	}

	if len(selected) != 1 || selected[0] != "range-over-built-in-types" {
		t.Fatalf("unexpected selected examples: %v", selected)
	}
}
