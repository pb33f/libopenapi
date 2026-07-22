// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTrainTravelFullCircleCanonicalRoundTrip(t *testing.T) {
	file := renderTrainTravel(t, trainTravelFullCircleOptions()...)

	repoRoot := repoRootDir(t)
	dir := t.TempDir()
	writeTempModule(t, dir, repoRoot)
	writeTempFile(t, filepath.Join(dir, "internal", "trainmodels", "models_gen.go"), file.Source)
	if file.SchemaMetadata == nil {
		t.Fatal("expected schema metadata sidecar")
	}
	writeTempFile(t, filepath.Join(dir, "internal", "trainmodels", file.SchemaMetadata.Name), file.SchemaMetadata.Source)
	writeTempFile(t, filepath.Join(dir, "cmd", "roundtrip", "main.go"), []byte(trainTravelFullCircleProgram))

	cmd := exec.Command("go", "run", "./cmd/roundtrip")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GO111MODULE=on",
		"GOWORK=off",
		"GOFLAGS=-mod=mod",
		"TRAIN_TRAVEL_SPEC="+filepath.Join(repoRoot, "generator", "golang", "testdata", "train-travel.yaml"),
		"GENERATED_MODELS="+filepath.Join(dir, "internal", "trainmodels", "models_gen.go"),
		"GENERATED_METADATA="+filepath.Join(dir, "internal", "trainmodels", file.SchemaMetadata.Name),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("full-circle command failed: %v\n%s", err, out)
	}
	assertContains(t, string(out), "canonical schema equal: true")
	assertContains(t, string(out), "model source equal: true")
	assertContains(t, string(out), "metadata source equal: true")
}

func trainTravelFullCircleOptions() []Option {
	return []Option{
		WithPackageName("trainmodels"),
		WithGeneratedComment(true),
		WithOptionalConstDiscriminatorUnions(true),
		WithOpenAPITags(true),
		WithSchemaMetadataSidecar(true),
	}
}

func repoRootDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join(wd, "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func writeTempModule(t *testing.T, dir, repoRoot string) {
	t.Helper()
	writeTempFile(t, filepath.Join(dir, "go.mod"), []byte("module trainfullcircle\n\ngo 1.25.0\n\nrequire github.com/pb33f/libopenapi v0.0.0\n\nreplace github.com/pb33f/libopenapi => "+repoRoot+"\n"))
	sum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	writeTempFile(t, filepath.Join(dir, "go.sum"), sum)
}

func writeTempFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

const trainTravelFullCircleProgram = `package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	gogenerator "github.com/pb33f/libopenapi/generator/golang"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"

	"trainfullcircle/internal/trainmodels"
)

var order = []string{"Station", "Trip", "Booking", "BookingPayment"}

func main() {
	set, err := gogenerator.SchemasFromTypes(
		reflect.TypeOf(trainmodels.Station{}),
		reflect.TypeOf(trainmodels.Trip{}),
		reflect.TypeOf(trainmodels.Booking{}),
		reflect.TypeOf(trainmodels.BookingPayment{}),
	)
	if err != nil {
		panic(err)
	}
	originalCanonical, err := canonicalOriginal(os.Getenv("TRAIN_TRAVEL_SPEC"))
	if err != nil {
		panic(err)
	}
	reflectedCanonical, err := canonicalReflected(set)
	if err != nil {
		panic(err)
	}
	canonicalEqual := bytes.Equal(originalCanonical, reflectedCanonical)
	fmt.Printf("canonical schema equal: %v\n", canonicalEqual)
	if !canonicalEqual {
		fmt.Printf("--- original\n%s\n--- reflected\n%s\n", originalCanonical, reflectedCanonical)
	}

	schemas, err := schemasInOrder(set)
	if err != nil {
		panic(err)
	}
	regenerated, err := gogenerator.NewGenerator(trainTravelOptions()...).RenderSchemas(schemas)
	if err != nil {
		panic(err)
	}
	originalModels, err := os.ReadFile(os.Getenv("GENERATED_MODELS"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("model source equal: %v\n", bytes.Equal(originalModels, regenerated.Source))
	originalMetadata, err := os.ReadFile(os.Getenv("GENERATED_METADATA"))
	if err != nil {
		panic(err)
	}
	if regenerated.SchemaMetadata == nil {
		panic("missing regenerated schema metadata")
	}
	fmt.Printf("metadata source equal: %v\n", bytes.Equal(originalMetadata, regenerated.SchemaMetadata.Source))
}

func trainTravelOptions() []gogenerator.Option {
	return []gogenerator.Option{
		gogenerator.WithPackageName("trainmodels"),
		gogenerator.WithGeneratedComment(true),
		gogenerator.WithOptionalConstDiscriminatorUnions(true),
		gogenerator.WithOpenAPITags(true),
		gogenerator.WithSchemaMetadataSidecar(true),
	}
}

func schemasInOrder(set *gogenerator.SchemaSet) (*orderedmap.Map[string, *base.SchemaProxy], error) {
	schemas := orderedmap.New[string, *base.SchemaProxy]()
	for _, name := range order {
		schema, ok := set.Components.Get(name)
		if !ok {
			return nil, fmt.Errorf("missing reflected component %q", name)
		}
		schemas.Set(name, schema)
	}
	return schemas, nil
}

func canonicalOriginal(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, err
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		return nil, err
	}
	schemas := make(map[string]any, len(order))
	for _, name := range order {
		schema, ok := model.Model.Components.Schemas.Get(name)
		if !ok {
			return nil, fmt.Errorf("missing original component %q", name)
		}
		rendered, err := schema.Render()
		if err != nil {
			return nil, err
		}
		decoded, err := canonicalSchemaValue(rendered)
		if err != nil {
			return nil, err
		}
		schemas[name] = decoded
	}
	return json.Marshal(schemas)
}

func canonicalReflected(set *gogenerator.SchemaSet) ([]byte, error) {
	schemas := make(map[string]any, len(order))
	for _, name := range order {
		schema, ok := set.Components.Get(name)
		if !ok {
			return nil, fmt.Errorf("missing reflected component %q", name)
		}
		rendered, err := schema.Render()
		if err != nil {
			return nil, err
		}
		decoded, err := canonicalSchemaValue(rendered)
		if err != nil {
			return nil, err
		}
		schemas[name] = decoded
	}
	return json.Marshal(schemas)
}

func canonicalSchemaValue(rendered []byte) (any, error) {
	var value any
	if err := yaml.Unmarshal(rendered, &value); err != nil {
		return nil, err
	}
	return value, nil
}
`
