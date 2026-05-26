// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"bytes"
	"reflect"
	"sync"
	"testing"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

type reuseWidget struct {
	ID   string `json:"id"`
	Size int    `json:"size,omitempty"`
}

func reuseWidgetSchema() *highbase.SchemaProxy {
	props := orderedmap.New[string, *highbase.SchemaProxy]()
	props.Set("id", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"string"}}))
	props.Set("size", highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"integer"}}))
	return highbase.CreateSchemaProxy(&highbase.Schema{
		Type:       []string{"object"},
		Properties: props,
		Required:   []string{"id"},
	})
}

// A configured generator must produce identical output when reused, with no
// per-invocation state leaking between calls.
func TestGeneratorReuseIsStable(t *testing.T) {
	gen := NewGenerator(WithPackageName("models"))
	schema := reuseWidgetSchema()
	first, err := gen.RenderSchema("Widget", schema)
	if err != nil {
		t.Fatalf("first render failed: %v", err)
	}
	second, err := gen.RenderSchema("Widget", schema)
	if err != nil {
		t.Fatalf("second render failed: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("reused generator produced divergent output:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}

	// SchemaFromType previously reset no state; reusing it must stay correct.
	a, err := gen.SchemaFromType(reflect.TypeOf(reuseWidget{}))
	if err != nil || a == nil {
		t.Fatalf("first SchemaFromType failed: %v", err)
	}
	b, err := gen.SchemaFromType(reflect.TypeOf(reuseWidget{}))
	if err != nil || b == nil {
		t.Fatalf("reused SchemaFromType failed: %v", err)
	}
}

// Concurrent use of a single configured generator must not corrupt output.
// Run with -race to exercise the shared-config / fresh-run-state boundary.
func TestGeneratorConcurrentReuse(t *testing.T) {
	gen := NewGenerator(WithPackageName("models"))
	schema := reuseWidgetSchema()
	want, err := gen.RenderSchema("Widget", schema)
	if err != nil {
		t.Fatalf("baseline render failed: %v", err)
	}
	const workers = 16
	var wg sync.WaitGroup
	failures := make(chan []byte, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, renderErr := gen.RenderSchema("Widget", schema)
			if renderErr != nil || !bytes.Equal(got, want) {
				failures <- got
			}
		}()
	}
	wg.Wait()
	close(failures)
	if got, ok := <-failures; ok {
		t.Fatalf("concurrent render diverged from baseline:\n%s", got)
	}
}
