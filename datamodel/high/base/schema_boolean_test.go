// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/require"
)

func TestBooleanSchemaRendering(t *testing.T) {
	for _, value := range []bool{true, false} {
		t.Run(strconv.FormatBool(value), func(t *testing.T) {
			var node yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(strconv.FormatBool(value)+" # boolean schema\n"), &node))
			idx := index.NewSpecIndex(&node)
			proxy := new(lowbase.SchemaProxy)
			require.NoError(t, proxy.Build(context.Background(), nil, node.Content[0], idx))
			high := NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{Value: proxy, ValueNode: node.Content[0]})
			schema := high.Schema()
			require.NotNil(t, schema)
			require.NotNil(t, schema.BooleanSchema)
			require.Equal(t, value, *schema.BooleanSchema)
			for name, render := range map[string]func() ([]byte, error){"yaml": schema.Render, "inline YAML": schema.RenderInline, "json": schema.MarshalJSON, "inline JSON": schema.MarshalJSONInline, "proxy": high.Render} {
				t.Run(name, func(t *testing.T) {
					b, err := render()
					require.NoError(t, err)
					var actual any
					require.NoError(t, yaml.Unmarshal(b, &actual))
					require.Equal(t, value, actual)
				})
			}
			opposite := !value
			schema.BooleanSchema = &opposite
			b, err := json.Marshal(schema)
			require.NoError(t, err)
			require.Equal(t, strconv.FormatBool(opposite), string(b))
			require.Equal(t, value, schema.GoLow().BooleanSchema.Value)
			schema.BooleanSchema = nil
			schema.Type = []string{"string"}
			b, err = high.Render()
			require.NoError(t, err)
			var object map[string]any
			require.NoError(t, yaml.Unmarshal(b, &object))
			require.Equal(t, "string", object["type"])
			require.Equal(t, "!!bool", node.Content[0].Tag)
			require.Equal(t, "# boolean schema", node.Content[0].LineComment)
		})
	}
}

func TestProgrammaticBooleanSchemaRendering(t *testing.T) {
	value := false
	schema := &Schema{BooleanSchema: &value}
	for _, render := range []func() ([]byte, error){schema.Render, schema.RenderInline, schema.MarshalJSON, schema.MarshalJSONInline} {
		b, err := render()
		require.NoError(t, err)
		var actual any
		require.NoError(t, yaml.Unmarshal(b, &actual))
		require.Equal(t, false, actual)
	}
}

func TestObjectSchemaChangesToBooleanThroughProxy(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("type: string"), &node))
	proxy := new(lowbase.SchemaProxy)
	require.NoError(t, proxy.Build(context.Background(), nil, node.Content[0], index.NewSpecIndex(&node)))
	high := NewSchemaProxy(&low.NodeReference[*lowbase.SchemaProxy]{Value: proxy, ValueNode: node.Content[0]})
	value := false
	high.Schema().BooleanSchema = &value
	data, err := high.Render()
	require.NoError(t, err)
	var actual any
	require.NoError(t, yaml.Unmarshal(data, &actual))
	require.Equal(t, false, actual)
	require.True(t, high.Schema().GoLow().BooleanSchema.IsEmpty())
}
