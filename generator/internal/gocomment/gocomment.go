// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package gocomment turns OpenAPI description text into Go comment lines.
//
// Descriptions are CommonMark and often carry inline HTML. Go comments are
// plain text, so markup is translated to readable text: links keep their text
// and absolute target, line-break tags become paragraph breaks, and other tags
// and emphasis markers are removed. Paragraphs are wrapped to a fixed width so
// generated comments read like hand-written ones.
package gocomment

import (
	"html"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Width is the preferred maximum length of comment text on one line. Words
// longer than the width, such as URLs, are never split.
const Width = 76

var (
	breakTag     = regexp.MustCompile(`(?i)<br\s*/?>`)
	paragraphTag = regexp.MustCompile(`(?i)</?p(?:\s[^<>]*)?>`)
	listTag      = regexp.MustCompile(`(?i)</?(?:ul|ol)(?:\s[^<>]*)?>`)
	itemTag      = regexp.MustCompile(`(?i)<li(?:\s[^<>]*)?>`)
	anchorTag    = regexp.MustCompile(`(?is)<a\s[^<>]*?href\s*=\s*["']([^"']*)["'][^<>]*>(.*?)</a>`)
	codeTag      = regexp.MustCompile(`(?is)<code(?:\s[^<>]*)?>(.*?)</code>`)
	markupTag    = regexp.MustCompile(`(?i)</?(?:a|abbr|b|blockquote|code|del|div|em|h[1-6]|hr|i|img|ins|kbd|li|mark|pre|s|small|span|strong|sub|sup|table|tbody|td|th|thead|tr|u)(?:\s[^<>]*)?/?>`)
	markdownLink = regexp.MustCompile(`!?\[([^\]]*)\]\(([^()\s]*)\)`)
	heading      = regexp.MustCompile(`^#{1,6}\s+`)
	listItem     = regexp.MustCompile(`^(?:[-*+]|\d+[.)])\s+`)
)

// Lines returns text as comment lines without comment markers. Paragraphs are
// wrapped to Width and separated by an empty string. It returns nil when text
// holds nothing printable.
func Lines(text string) []string {
	var lines []string
	for _, block := range blocks(plain(text)) {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, block...)
	}
	return lines
}

// Write appends lines to b as Go line comments, each starting with indent.
// An empty line becomes a bare comment marker.
func Write(b *strings.Builder, indent string, lines []string) {
	for _, line := range lines {
		b.WriteString(indent)
		if line == "" {
			b.WriteString("//\n")
			continue
		}
		b.WriteString("// ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
}

// plain translates inline HTML and Markdown markup to text. Break and
// paragraph tags become blank lines; list item tags become list lines.
func plain(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = anchorTag.ReplaceAllStringFunc(text, func(match string) string {
		parts := anchorTag.FindStringSubmatch(match)
		return link(parts[2], parts[1])
	})
	text = codeTag.ReplaceAllString(text, "`$1`")
	text = breakTag.ReplaceAllString(text, "\n\n")
	text = paragraphTag.ReplaceAllString(text, "\n\n")
	text = listTag.ReplaceAllString(text, "\n")
	text = itemTag.ReplaceAllString(text, "\n- ")
	text = markupTag.ReplaceAllString(text, "")
	text = markdownLink.ReplaceAllStringFunc(text, func(match string) string {
		parts := markdownLink.FindStringSubmatch(match)
		if strings.HasPrefix(match, "!") {
			return parts[1]
		}
		return link(parts[1], parts[2])
	})
	text = strings.ReplaceAll(text, "**", "")
	return html.UnescapeString(text)
}

// link renders a hyperlink as its text followed by its absolute target.
// Relative and document-local targets mean nothing outside the source
// documentation, so only their text is kept.
func link(text, target string) string {
	text = strings.TrimSpace(markupTag.ReplaceAllString(text, ""))
	absolute := strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "http://")
	switch {
	case !absolute || text == target:
		return text
	case text == "":
		return target
	default:
		return text + " (" + target + ")"
	}
}

// blocks splits text into wrapped blocks: paragraphs, headings, lists, and
// fenced code. Lists keep Go doc comment indentation and fenced code keeps
// its own lines, indented as a Go doc code block.
func blocks(text string) [][]string {
	var result [][]string
	var paragraph, items, code []string
	fenced := false
	flushParagraph := func() {
		if len(paragraph) > 0 {
			result = append(result, wrap(sentence(strings.Join(paragraph, " ")), "", ""))
			paragraph = nil
		}
	}
	flush := func() {
		flushParagraph()
		if len(items) > 0 {
			var list []string
			for _, item := range items {
				marker := listItem.FindString(item)
				list = append(list, wrap(item[len(marker):], "  "+strings.TrimSpace(marker)+" ", "    ")...)
			}
			result = append(result, list)
			items = nil
		}
	}
	for raw := range strings.SplitSeq(text, "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "```"):
			flush()
			if fenced && len(code) > 0 {
				result = append(result, code)
			}
			fenced, code = !fenced, nil
		case fenced:
			code = append(code, "\t"+strings.TrimRight(raw, " \t"))
		case line == "":
			flush()
		case heading.MatchString(line):
			flush()
			result = append(result, wrap(heading.ReplaceAllString(line, ""), "", ""))
		case listItem.MatchString(line):
			flushParagraph()
			items = append(items, line)
		case len(items) > 0:
			items[len(items)-1] += " " + line
		default:
			paragraph = append(paragraph, line)
		}
	}
	flush()
	if len(code) > 0 {
		result = append(result, code)
	}
	return result
}

// sentence ends a paragraph that stops on a letter or digit with a period,
// as Go doc comments do.
func sentence(text string) string {
	last, _ := utf8.DecodeLastRuneInString(text)
	if unicode.IsLetter(last) || unicode.IsDigit(last) {
		return text + "."
	}
	return text
}

// wrap breaks non-blank text into lines of at most Width characters. first
// prefixes the first line and rest prefixes each following line.
func wrap(text, first, rest string) []string {
	words := strings.Fields(text)
	var lines []string
	line := first + words[0]
	for _, word := range words[1:] {
		if len(line)+1+len(word) > Width {
			lines = append(lines, line)
			line = rest + word
			continue
		}
		line += " " + word
	}
	return append(lines, line)
}
