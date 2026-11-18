// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
)

const unknownDiscriminatorSpec = `openapi: 3.1.0
info:
  title: pets
  version: "1"
paths: {}
components:
  schemas:
    Cat:
      type: object
      required: [petType, name]
      properties:
        petType:
          type: string
        name:
          type: string
    Dog:
      type: object
      required: [petType, bark]
      properties:
        petType:
          type: string
        bark:
          type: boolean
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Cat'
        - $ref: '#/components/schemas/Dog'
      discriminator:
        propertyName: petType
        mapping:
          cat: '#/components/schemas/Cat'
          dog: '#/components/schemas/Dog'
    PetList:
      type: object
      required: [pets]
      properties:
        pets:
          type: array
          items:
            $ref: '#/components/schemas/Pet'
`

func renderUnknownDiscriminatorSpec(t *testing.T, spec string, opts ...Option) *GeneratedFile {
	t.Helper()
	doc, err := libopenapi.NewDocument([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	file, err := NewGenerator(append([]Option{WithPackageName("pets")}, opts...)...).RenderSchemas(model.Model.Components.Schemas)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func TestUnknownDiscriminatorVariantsOffKeepsTheError(t *testing.T) {
	src := string(renderUnknownDiscriminatorSpec(t, unknownDiscriminatorSpec).Source)
	if !strings.Contains(src, `unknown petType discriminator value %q`) {
		t.Fatalf("default output lost the unknown-value error:\n%s", src)
	}
	if strings.Contains(src, "PetUnknown") {
		t.Fatalf("default output declares an unknown variant:\n%s", src)
	}
}

func TestTrainTravelGoldenTypedUnionWithUnknownVariants(t *testing.T) {
	assertGolden(t, "testdata/train_travel_typed_union_unknown.golden.go", renderTrainTravel(t,
		WithOptionalConstDiscriminatorUnions(true),
		WithUnknownDiscriminatorVariants(true),
	).Source)
}

func TestUnknownDiscriminatorVariantNameCollision(t *testing.T) {
	spec := unknownDiscriminatorSpec + `    PetUnknown:
      type: object
      properties:
        note:
          type: string
`
	file := renderUnknownDiscriminatorSpec(t, spec, WithUnknownDiscriminatorVariants(true))
	src := string(file.Source)
	if !strings.Contains(src, "type PetUnknown struct {\n\tNote") {
		t.Fatalf("document schema PetUnknown was renamed or replaced:\n%s", src)
	}
	found := false
	for _, d := range file.Diagnostics {
		if d.Code == DiagnosticTypeNameCollision && d.Path == "Pet" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no collision diagnostic for the unknown variant: %+v", file.Diagnostics)
	}
}

// TestUnknownDiscriminatorVariantsDecode compiles the generated models and
// proves the behaviour: an unknown variant inside an array no longer fails the
// array, its bytes round-trip, and a missing discriminator still fails.
func TestUnknownDiscriminatorVariantsDecode(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles generated code")
	}
	file := renderUnknownDiscriminatorSpec(t, unknownDiscriminatorSpec, WithUnknownDiscriminatorVariants(true))
	dir := t.TempDir()
	writeTempFile(t, filepath.Join(dir, "go.mod"), []byte("module unknowncheck\n\ngo 1.24\n"))
	writeTempFile(t, filepath.Join(dir, "pets", "models_gen.go"), file.Source)
	writeTempFile(t, filepath.Join(dir, "main.go"), []byte(unknownDiscriminatorDriver))
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated models failed the unknown-variant checks: %v\n%s\n--- generated ---\n%s", err, out, file.Source)
	}
}

const unknownDiscriminatorDriver = `package main

import (
	"encoding/json"
	"fmt"
	"os"

	"unknowncheck/pets"
)

func fail(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
	os.Exit(1)
}

func main() {
	var list pets.PetList
	in := []byte(` + "`" + `{"pets":[{"petType":"cat","name":"Tom"},{"petType":"lizard","scales":3}]}` + "`" + `)
	if err := json.Unmarshal(in, &list); err != nil {
		fail("decode with an unknown variant: %v", err)
	}
	if len(list.Pets) != 2 {
		fail("want 2 pets, got %d", len(list.Pets))
	}
	if _, ok := list.Pets[0].Value.(pets.Cat); !ok {
		fail("first pet is %T, want pets.Cat", list.Pets[0].Value)
	}
	unknown, ok := list.Pets[1].Value.(pets.PetUnknown)
	if !ok {
		fail("second pet is %T, want pets.PetUnknown", list.Pets[1].Value)
	}
	if unknown.Discriminator != "lizard" {
		fail("discriminator = %q", unknown.Discriminator)
	}
	out, err := json.Marshal(list.Pets[1])
	if err != nil || string(out) != ` + "`" + `{"petType":"lizard","scales":3}` + "`" + ` {
		fail("unknown variant did not round-trip: %s %v", out, err)
	}
	var missing pets.PetUnion
	if err := json.Unmarshal([]byte(` + "`" + `{"name":"Tom"}` + "`" + `), &missing); err == nil {
		fail("missing discriminator decoded without an error")
	}
}
`

// Outside a component render (for example the properties-beside-oneOf pass)
// there is no type name registry; the unknown variant takes its preferred name
// without claiming it, and the option being off leaves the union untouched.
func TestClaimUnknownVariantWithoutTypeRegistry(t *testing.T) {
	on := NewGenerator(WithUnknownDiscriminatorVariants(true))
	union := &SchemaIR{Name: "Pet", Union: &UnionIR{}}
	on.claimUnknownVariant(union)
	if union.Union.UnknownName != "PetUnknown" {
		t.Fatalf("unknown variant name = %q, want PetUnknown", union.Union.UnknownName)
	}
	off := NewGenerator()
	untouched := &SchemaIR{Name: "Pet", Union: &UnionIR{}}
	off.claimUnknownVariant(untouched)
	if untouched.Union.UnknownName != "" {
		t.Fatalf("option off named an unknown variant %q", untouched.Union.UnknownName)
	}
}
