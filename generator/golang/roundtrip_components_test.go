// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	whatchanged "github.com/pb33f/libopenapi/what-changed/model"
)

// TestTrainTravelComponentsRoundTrip codifies the bi-directional identity of the
// generator over the train-travel components.schemas:
//
//	components.schemas --RenderSchemas--> Go classes (+ metadata sidecar)
//	Go classes         --SchemasFromTypes--> reconstructed components.schemas
//
// The reconstructed schemas must be semantically identical to the originals.
// "Identical" is judged by libopenapi's own what-changed diff engine rather than
// raw bytes: the back half rebuilds and re-renders each schema, so key ordering
// is not preserved, but the meaning must be. A perfect round trip yields zero
// changes; any drift is reported per component.
//
// The backward leg needs the generated types as runtime reflect types, so this
// test compiles the generated package in a temp module and runs it (it depends
// on the Go toolchain being available, like TestTrainTravelFullCircle...).
func TestTrainTravelComponentsRoundTrip(t *testing.T) {
	file := renderTrainTravel(t, trainTravelFullCircleOptions()...)
	if file.SchemaMetadata == nil {
		t.Fatal("expected a schema metadata sidecar for a high-fidelity round trip")
	}

	reconstructed := reflectComponentsRoundTrip(t, file)

	spec, err := os.ReadFile("testdata/train-travel.yaml")
	if err != nil {
		t.Fatal(err)
	}
	specDoc, err := libopenapi.NewDocument(spec)
	if err != nil {
		t.Fatalf("cannot parse train-travel spec: %v", err)
	}
	model, err := specDoc.BuildV3Model()
	if err != nil {
		t.Fatalf("cannot build train-travel model: %v", err)
	}
	original := assembleComponentsDoc(t, model.Model.Components.Schemas)

	originalDoc, err := libopenapi.NewDocument(original)
	if err != nil {
		t.Fatalf("cannot parse original components doc: %v", err)
	}
	reconstructedDoc, err := libopenapi.NewDocument(reconstructed)
	if err != nil {
		t.Fatalf("cannot parse reconstructed components doc:\n%s\nerror: %v", reconstructed, err)
	}

	changes, err := libopenapi.CompareDocuments(originalDoc, reconstructedDoc)
	if err != nil {
		t.Fatalf("cannot compare documents: %v", err)
	}
	if changes == nil || changes.TotalChanges() == 0 {
		return // perfect round trip
	}

	// There are differences; codify them per component for a precise report.
	schemaChanges := changes.ComponentsChanges.SchemaChanges
	for name := range model.Model.Components.Schemas.FromOldest() {
		component := name
		t.Run(component, func(t *testing.T) {
			sc := schemaChanges[component]
			if sc != nil && sc.TotalChanges() > 0 {
				t.Fatalf("component %q is not identical after a round trip:\n%s", component, formatChanges(sc.GetAllChanges()))
			}
		})
	}
	t.Run("document", func(t *testing.T) {
		if changes.TotalChanges() > 0 {
			t.Fatalf("round trip introduced %d change(s) beyond the original components:\n%s",
				changes.TotalChanges(), formatChanges(changes.GetAllChanges()))
		}
	})
}

// reflectComponentsRoundTrip compiles the generated package in a throwaway
// module and reflects its types back into a minimal OpenAPI document containing
// just the reconstructed components.schemas.
func reflectComponentsRoundTrip(t *testing.T, file *GeneratedFile) []byte {
	t.Helper()
	repoRoot := repoRootDir(t)
	dir := t.TempDir()
	writeTempModule(t, dir, repoRoot)
	writeTempFile(t, filepath.Join(dir, "internal", "trainmodels", "models_gen.go"), file.Source)
	writeTempFile(t, filepath.Join(dir, "internal", "trainmodels", file.SchemaMetadata.Name), file.SchemaMetadata.Source)
	program := strings.Replace(roundTripDriverProgram, "__TYPES__", reflectTypeList(file), 1)
	writeTempFile(t, filepath.Join(dir, "cmd", "roundtrip", "main.go"), []byte(program))

	out := filepath.Join(dir, "reconstructed.yaml")
	cmd := exec.Command("go", "run", "./cmd/roundtrip")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "ROUNDTRIP_OUT="+out)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("reflect round-trip command failed: %v\n%s", err, combined)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// reflectTypeList emits the reflect.TypeOf lines for every generated root object
// type, so the driver always reflects exactly the components that were emitted.
func reflectTypeList(file *GeneratedFile) string {
	var b strings.Builder
	for _, ty := range file.Types {
		if ty.Kind == KindObject || ty.Kind == KindAllOf {
			fmt.Fprintf(&b, "\t\treflect.TypeOf(trainmodels.%s{}),\n", ty.Name)
		}
	}
	return b.String()
}

// assembleComponentsDoc wraps a set of schemas into a minimal OpenAPI 3.1
// document so two component sets can be diffed like for like.
func assembleComponentsDoc(t *testing.T, schemas *orderedmap.Map[string, *highbase.SchemaProxy]) []byte {
	t.Helper()
	var b strings.Builder
	b.WriteString("openapi: 3.1.0\ninfo:\n  title: roundtrip\n  version: 1.0.0\npaths: {}\ncomponents:\n  schemas:\n")
	for name, proxy := range schemas.FromOldest() {
		rendered, err := proxy.Render()
		if err != nil {
			t.Fatalf("cannot render schema %q: %v", name, err)
		}
		b.WriteString("    ")
		b.WriteString(name)
		b.WriteString(":\n")
		b.WriteString(indentSchemaYAML(string(rendered), "      "))
	}
	return []byte(b.String())
}

func formatChanges(changes []*whatchanged.Change) string {
	var b strings.Builder
	for _, c := range changes {
		if c == nil {
			continue
		}
		fmt.Fprintf(&b, "  - %s: %q -> %q (breaking=%v)\n", c.Property, c.Original, c.New, c.Breaking)
	}
	return b.String()
}

const roundTripDriverProgram = `package main

import (
	"os"
	"strings"

	gogenerator "github.com/pb33f/libopenapi/generator/golang"
	"reflect"

	"trainfullcircle/internal/trainmodels"
)

func main() {
	set, err := gogenerator.SchemasFromTypes(
__TYPES__	)
	if err != nil {
		panic(err)
	}
	var b strings.Builder
	b.WriteString("openapi: 3.1.0\ninfo:\n  title: roundtrip\n  version: 1.0.0\npaths: {}\ncomponents:\n  schemas:\n")
	for name, proxy := range set.Components.FromOldest() {
		rendered, err := proxy.Render()
		if err != nil {
			panic(err)
		}
		b.WriteString("    ")
		b.WriteString(name)
		b.WriteString(":\n")
		for _, line := range strings.Split(strings.TrimRight(string(rendered), "\n"), "\n") {
			b.WriteString("      ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	if err := os.WriteFile(os.Getenv("ROUNDTRIP_OUT"), []byte(b.String()), 0o600); err != nil {
		panic(err)
	}
}
`
