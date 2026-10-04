package main

import "testing"

func TestExtractBuildID(t *testing.T) {
	page := []byte(`<script id="__NEXT_DATA__" type="application/json">{"props":{},"page":"/community","buildId":"5iEA0hBH9_5-ZIKnbLmi4","isFallback":false}</script>`)

	got, err := extractBuildID(page)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "5iEA0hBH9_5-ZIKnbLmi4" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractBuildIDMissing(t *testing.T) {
	if _, err := extractBuildID([]byte(`<html></html>`)); err == nil {
		t.Fatal("expected error for page without buildId")
	}
}
