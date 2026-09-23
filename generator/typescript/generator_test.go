// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package typescript

import (
	"os"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/generator/golang"
	"github.com/pb33f/libopenapi/orderedmap"
)

func TestFeaturesGolden(t *testing.T) {
	file := renderFixture(t, "testdata/features.yaml")
	assertGolden(t, "testdata/features.golden.ts", file.Source)
}

func TestTrainTravelGolden(t *testing.T) {
	file := renderFixture(t, "../golang/testdata/train-travel.yaml")
	assertGolden(t, "testdata/train_travel.golden.ts", file.Source)
}

func TestTypesListFollowsInputOrder(t *testing.T) {
	file := renderFixture(t, "testdata/features.yaml")
	if len(file.Types) < 3 || file.Types[0] != "Account" || file.Types[1] != "Person" || file.Types[2] != "Status" {
		t.Fatalf("types out of input order: %v", file.Types)
	}
}

func TestInvalidComponentNameIsRenamedWithDiagnostic(t *testing.T) {
	file := renderFixture(t, "testdata/features.yaml")
	if !strings.Contains(string(file.Source), "export type Default = string;") {
		t.Fatalf("reserved component name was not renamed:\n%s", file.Source)
	}
	found := false
	for _, d := range file.Diagnostics {
		if d.Code == DiagnosticInvalidIdentifier && d.Path == "default" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing %s diagnostic: %+v", DiagnosticInvalidIdentifier, file.Diagnostics)
	}
}

func TestTypeNameResolverAndHeader(t *testing.T) {
	file := renderFixture(t, "testdata/features.yaml",
		WithHeaderComment(""),
		WithTypeNameResolver(func(name string) string {
			if name == "Person" {
				return "Human"
			}
			return ""
		}),
	)
	source := string(file.Source)
	if strings.HasPrefix(source, "//") {
		t.Fatalf("header was not omitted:\n%s", source)
	}
	if !strings.Contains(source, "export interface Human {") || !strings.Contains(source, "owner?: Human;") {
		t.Fatalf("resolved name was not used for declaration and references:\n%s", source)
	}
}

func TestEmptySchemaSetIsAModule(t *testing.T) {
	file, err := NewGenerator().RenderSchemas(orderedmap.New[string, *highbase.SchemaProxy]())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(file.Source), "export {};\n") {
		t.Fatalf("empty module missing export marker:\n%s", file.Source)
	}
	file, err = NewGenerator().RenderSchemas(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(file.Source), "export {};\n") {
		t.Fatalf("nil module missing export marker:\n%s", file.Source)
	}
}

func renderFixture(t *testing.T, path string, opts ...Option) *GeneratedFile {
	t.Helper()
	spec, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := libopenapi.NewDocument(spec)
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	file, err := RenderSchemas(model.Model.Components.Schemas, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func assertGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	if os.Getenv("LIBOPENAPI_GENERATOR_UPDATE_GOLDENS") == "true" {
		if err := os.WriteFile(path, got, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ReplaceAll(string(want), "\r\n", "\n") != string(got) {
		t.Fatalf("golden mismatch for %s; regenerate with LIBOPENAPI_GENERATOR_UPDATE_GOLDENS=true and review the diff\n%s", path, got)
	}
}

func TestValidNamesWinOverRenamedComponents(t *testing.T) {
	file := renderFixture(t, "testdata/features.yaml")
	source := string(file.Source)
	for _, want := range []string{
		"export type MyType = number;",    // valid name kept verbatim
		"export type MyType__2 = string;", // my-type renamed onto a taken name
		"export interface RecordType {",   // shadowing a global the output uses
		"export type Default = string;",   // reserved word renamed
	} {
		if !strings.Contains(source, want) {
			t.Errorf("missing %q", want)
		}
	}
	collided := false
	for _, d := range file.Diagnostics {
		if d.Code == golang.DiagnosticComponentNameCollision && d.Path == "my-type" {
			collided = true
		}
	}
	if !collided {
		t.Fatalf("missing collision diagnostic for my-type: %+v", file.Diagnostics)
	}
}
