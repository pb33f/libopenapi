// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package gocomment

import (
	"reflect"
	"strings"
	"testing"
)

func TestLinesTranslatesInlineHTML(t *testing.T) {
	// Shaped like a real vendor's parameter description.
	got := Lines(`The ID for the person. <br><br>To find IDs, call the <a href="https://docs.example.com/reference/people-api-search" target="_blank">people API search endpoint</a> and identify the values for <code>person_id</code>. <br><br>Example: ` + "`587cf802f65125cad923a266`")
	want := []string{
		"The ID for the person.",
		"",
		"To find IDs, call the people API search endpoint",
		"(https://docs.example.com/reference/people-api-search) and identify the",
		"values for `person_id`.",
		"",
		"Example: `587cf802f65125cad923a266`",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines() =\n%q\nwant\n%q", got, want)
	}
}

func TestLinesTranslatesMarkdown(t *testing.T) {
	got := Lines("## Endpoint essentials\n\n**Credit usage:** see [pricing](https://docs.example.com/docs/api-pricing), [webhooks](#webhook-details) and ![diagram](https://example.com/d.png).\nSoft wrapped &amp; joined.")
	want := []string{
		"Endpoint essentials",
		"",
		"Credit usage: see pricing (https://docs.example.com/docs/api-pricing),",
		"webhooks and diagram. Soft wrapped & joined.",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines() =\n%q\nwant\n%q", got, want)
	}
}

func TestLinesKeepsListsAsGoDocLists(t *testing.T) {
	got := Lines("Match confidence:\n- `high`: The service matched the information to a person record and every field was verified against several sources.\n- `none`: no match\n  continued item text\n1. first\n\nAfter.<ul><li>html item</li></ul>")
	want := []string{
		"Match confidence:",
		"",
		"  - `high`: The service matched the information to a person record and every",
		"    field was verified against several sources.",
		"  - `none`: no match continued item text",
		"  1. first",
		"",
		"After.",
		"",
		"  - html item",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines() =\n%q\nwant\n%q", got, want)
	}
}

func TestLinesKeepsFencedCode(t *testing.T) {
	got := Lines("Example:\n```json\n{\n  \"id\": 1\n}\n```\n```\n```\nTrailing.\n```\nunclosed")
	want := []string{
		"Example:",
		"",
		"\t{",
		"\t  \"id\": 1",
		"\t}",
		"",
		"Trailing.",
		"",
		"\tunclosed",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines() =\n%q\nwant\n%q", got, want)
	}
}

func TestLinkRendering(t *testing.T) {
	for _, test := range []struct{ text, target, want string }{
		{"docs", "https://example.com", "docs (https://example.com)"},
		{"https://example.com", "https://example.com", "https://example.com"},
		{"", "http://example.com", "http://example.com"},
		{"<b>guide</b>", "doc:guide", "guide"},
	} {
		if got := link(test.text, test.target); got != test.want {
			t.Fatalf("link(%q, %q) = %q, want %q", test.text, test.target, got, test.want)
		}
	}
}

func TestLinesIgnoresBlankAndLeavesOtherAngles(t *testing.T) {
	if got := Lines(" \n<br>\n "); got != nil {
		t.Fatalf("Lines() = %q, want nil", got)
	}
	if got := Lines("Map<String, Object> where a < b"); !reflect.DeepEqual(got, []string{"Map<String, Object> where a < b."}) {
		t.Fatalf("Lines() = %q", got)
	}
}

func TestWrapNeverSplitsLongWords(t *testing.T) {
	long := strings.Repeat("x", Width+5)
	if got := wrap("see "+long+" now", "", ""); !reflect.DeepEqual(got, []string{"see", long, "now"}) {
		t.Fatalf("wrap() = %q", got)
	}
}

func TestWrite(t *testing.T) {
	var b strings.Builder
	Write(&b, "\t", []string{"First.", "", "Second."})
	if got, want := b.String(), "\t// First.\n\t//\n\t// Second.\n"; got != want {
		t.Fatalf("Write() = %q, want %q", got, want)
	}
}
