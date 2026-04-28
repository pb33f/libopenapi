// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"os"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
)

func TestTrainTravelGoldenDefaultRawUnion(t *testing.T) {
	assertGolden(t, "testdata/train_travel_default.golden.go", renderTrainTravel(t).Source)
}

func TestTrainTravelGoldenOptionalConstDiscriminatorTypedUnion(t *testing.T) {
	assertGolden(t, "testdata/train_travel_typed_union.golden.go", renderTrainTravel(t, WithOptionalConstDiscriminatorUnions(true)).Source)
}

func TestJSONSchema202012GoldenDefault(t *testing.T) {
	assertGolden(t, "testdata/jsonschema_2020_12_default.golden.go", renderJSONSchema202012(t).Source)
}

func TestJSONSchema202012GoldenOptions(t *testing.T) {
	assertGolden(t, "testdata/jsonschema_2020_12_options.golden.go", renderJSONSchema202012(t,
		WithAdditionalPropertiesMethods(false),
		WithEnumConstants(true),
	).Source)
}

func TestNameCollisionsGoldenDefault(t *testing.T) {
	assertGolden(t, "testdata/name_collisions_default.golden.go", renderNameCollisions(t).Source)
}

func TestNameCollisionsGoldenCompactDelimiter(t *testing.T) {
	assertGolden(t, "testdata/name_collisions_compact_delimiter.golden.go", renderNameCollisions(t,
		WithNestedTypeNameDelimiter(""),
		WithEnumConstants(true),
	).Source)
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
	if string(got) != string(want) {
		t.Fatalf("golden mismatch for %s at %s", path, firstDiff(string(want), string(got)))
	}
}

func firstDiff(want, got string) string {
	max := len(want)
	if len(got) < max {
		max = len(got)
	}
	for i := 0; i < max; i++ {
		if want[i] != got[i] {
			return diffLocation(want, i)
		}
	}
	if len(want) != len(got) {
		return diffLocation(want, max)
	}
	return "no difference"
}

func renderJSONSchema202012(t *testing.T, opts ...Option) *GeneratedFile {
	t.Helper()
	spec, err := os.ReadFile("testdata/jsonschema-2020-12.yaml")
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
	file, err := NewGenerator(opts...).RenderSchemas(model.Model.Components.Schemas)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func renderNameCollisions(t *testing.T, opts ...Option) *GeneratedFile {
	t.Helper()
	spec, err := os.ReadFile("testdata/name-collisions.yaml")
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
	file, err := NewGenerator(opts...).RenderSchemas(model.Model.Components.Schemas)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func diffLocation(text string, offset int) string {
	line := 1 + strings.Count(text[:offset], "\n")
	column := offset - strings.LastIndex(text[:offset], "\n")
	return "line " + intString(line) + ", column " + intString(column)
}
