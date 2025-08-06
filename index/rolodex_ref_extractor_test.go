// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractedRef_GetFile(t *testing.T) {
	a := &ExtractedRef{Location: "#/components/schemas/One", Type: Local}
	assert.Equal(t, "#/components/schemas/One", a.GetFile())

	a = &ExtractedRef{Location: "pizza.yaml#/components/schemas/One", Type: File}
	assert.Equal(t, "pizza.yaml", a.GetFile())

	a = &ExtractedRef{Location: "https://api.pb33f.io/openapi.yaml#/components/schemas/One", Type: File}
	assert.Equal(t, "https://api.pb33f.io/openapi.yaml", a.GetFile())
}

func TestExtractedRef_GetReference(t *testing.T) {
	a := &ExtractedRef{Location: "#/components/schemas/One", Type: Local}
	assert.Equal(t, "#/components/schemas/One", a.GetReference())

	a = &ExtractedRef{Location: "pizza.yaml#/components/schemas/One", Type: File}
	assert.Equal(t, "#/components/schemas/One", a.GetReference())

	a = &ExtractedRef{Location: "https://api.pb33f.io/openapi.yaml#/components/schemas/One", Type: File}
	assert.Equal(t, "#/components/schemas/One", a.GetReference())
}

func TestExtractFileType(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want FileExtension
	}{
		{
			name: "yaml file with .yaml",
			ref:  "config.yaml",
			want: YAML,
		},
		{
			name: "yaml file with .yml",
			ref:  "config.yml",
			want: YAML,
		},
		{
			name: "JSON file",
			ref:  "data.json",
			want: JSON,
		},
		{
			name: "JS file",
			ref:  "script.js",
			want: JS,
		},
		{
			name: "Go file",
			ref:  "main.go",
			want: GO,
		},
		{
			name: "TS file",
			ref:  "app.ts",
			want: TS,
		},
		{
			name: "C# file",
			ref:  "Program.cs",
			want: CS,
		},
		{
			name: "C file",
			ref:  "hello.c",
			want: C,
		},
		{
			name: "C++ file",
			ref:  "hello.cpp",
			want: CPP,
		},
		{
			name: "PHP file",
			ref:  "index.php",
			want: PHP,
		},
		{
			name: "Python file",
			ref:  "script.py",
			want: PY,
		},
		{
			name: "HTML file",
			ref:  "index.html",
			want: HTML,
		},
		{
			name: "Markdown file",
			ref:  "README.md",
			want: MD,
		},
		{
			name: "Java file",
			ref:  "HelloWorld.java",
			want: JAVA,
		},
		{
			name: "Rust file",
			ref:  "main.rs",
			want: RS,
		},
		{
			name: "Zig file",
			ref:  "main.zig",
			want: ZIG,
		},
		{
			name: "Ruby file",
			ref:  "app.rb",
			want: RB,
		},
		{
			name: "Unknown extension",
			ref:  "unknown.xyz",
			want: UNKNOWN,
		},
		{
			name: "No extension at all",
			ref:  "Makefile",
			want: UNKNOWN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractFileType(tt.ref)
			assert.Equal(t, tt.want, got)
		})
	}
}
