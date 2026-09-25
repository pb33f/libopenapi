// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"unsafe"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

//go:linkname v3LowBuildModelFieldCache github.com/pb33f/libopenapi/datamodel/low.buildModelFieldCache
var v3LowBuildModelFieldCache sync.Map

// forceOperationBuildModelError swaps the cached field layout low.BuildModel uses for Operation with a
// single bool field mapped to key, so building any operation holding that key fails with
// "unable to parse unsupported type". BuildModel cannot fail for the real Operation type, so this is
// the only way to prove PathItem.Build propagates that error.
func forceOperationBuildModelError(t *testing.T, key string) {
	t.Helper()
	opType := reflect.TypeOf(Operation{})
	original, ok := v3LowBuildModelFieldCache.Load(opType)
	require.True(t, ok)

	origType := reflect.TypeOf(original)
	replacement := reflect.MakeSlice(origType, 1, 1)
	elem := reflect.New(origType.Elem()).Elem()
	setV3UnexportedField(elem.FieldByName("lookupKey"), key)
	setV3UnexportedField(elem.FieldByName("index"), 0)
	setV3UnexportedField(elem.FieldByName("kind"), reflect.Bool)
	replacement.Index(0).Set(elem)

	v3LowBuildModelFieldCache.Store(opType, replacement.Interface())
	t.Cleanup(func() {
		v3LowBuildModelFieldCache.Store(opType, original)
	})
}

func setV3UnexportedField(field reflect.Value, value any) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func buildPathItem(t *testing.T, yml string) error {
	t.Helper()
	var idxNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &idxNode))
	idx := index.NewSpecIndex(&idxNode)

	var n PathItem
	return n.Build(context.Background(), nil, idxNode.Content[0], idx)
}

func TestPathItem_Build_OperationBuildModelError(t *testing.T) {
	yml := "get:\n  summary: fetch\n"
	require.NoError(t, buildPathItem(t, yml)) // seeds the BuildModel field cache for Operation

	forceOperationBuildModelError(t, "summary")

	err := buildPathItem(t, yml)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse unsupported type")
}

// The additionalOperations map is itself passed through BuildModel as an Operation before each entry
// is built, so the forced key sits only on the nested operation to reach the per-entry build.
func TestPathItem_Build_AdditionalOperationBuildModelError(t *testing.T) {
	yml := "additionalOperations:\n  COPY:\n    summary: copy\n"
	require.NoError(t, buildPathItem(t, yml))

	forceOperationBuildModelError(t, "summary")

	err := buildPathItem(t, yml)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse unsupported type")
}
