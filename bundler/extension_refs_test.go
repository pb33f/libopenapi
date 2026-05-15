// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io
// SPDX-License-Identifier: MIT

package bundler

import (
	"context"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/pb33f/libopenapi/index"
	"go.yaml.in/yaml/v4"
)

// Tests for defensive nil checks and edge cases in extension_refs.go
// These tests ensure code coverage for defensive programming paths.

func TestResolveExtensionRefs_NilRolodex(t *testing.T) {
	// Should not panic with nil rolodex
	resolveExtensionRefs(nil)
}

func TestRewriteExtensionRefsForComposedBundle_NilAndRootCases(t *testing.T) {
	rewriteExtensionRefsForComposedBundle(nil)

	rolo := index.NewRolodex(index.CreateOpenAPIIndexConfig())
	rewriteExtensionRefsForComposedBundle(rolo)

	rewriteExtensionRefsForComposedIndex(nil, nil)

	var node yaml.Node
	_ = yaml.Unmarshal([]byte(`openapi: 3.1.0`), &node)
	idx := index.NewSpecIndexWithConfig(&node, index.CreateOpenAPIIndexConfig())
	idx.SetAbsolutePath(filepath.Join(t.TempDir(), "openapi.yaml"))
	rewriteExtensionRefsForComposedIndex(idx, idx)
	walkAndRewriteComposedExtensionRefs(nil, idx, idx, false)
}

func TestRewriteExtensionRefsForComposedBundle_RebasesNestedExtensionRefs(t *testing.T) {
	rootDir := t.TempDir()
	rootIdx := newComposedRefIndex(t, filepath.Join(rootDir, "openapi.yaml"), `openapi: 3.1.0`)
	sourceIdx := newComposedRefIndex(t, filepath.Join(rootDir, "paths", "echo.yaml"), `post:
  x-codeSamples:
    - lang: C#
      source:
        $ref: ../code_samples/C_sharp/echo/post.cs
  x-empty-key-test: keep
`)

	rootNode := sourceIdx.GetRootNode()
	mappingNode := rootNode.Content[0]
	mappingNode.Content = append(mappingNode.Content,
		nil,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "ignored"},
	)

	walkAndRewriteComposedExtensionRefs(&yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{rootNode},
	}, sourceIdx, rootIdx, false)

	var refValue string
	findRefValue(rootNode, &refValue)
	if refValue != "code_samples/C_sharp/echo/post.cs" {
		t.Fatalf("expected rebased ref, got %q", refValue)
	}
}

func TestRebaseExtensionRefForComposed(t *testing.T) {
	rootDir := t.TempDir()
	rootIdx := newComposedRefIndex(t, filepath.Join(rootDir, "openapi.yaml"), `openapi: 3.1.0`)
	sourceIdx := newComposedRefIndex(t, filepath.Join(rootDir, "paths", "echo.yaml"), `post: {}`)

	got := rebaseExtensionRefForComposed("../code_samples/post.md#/snippet", sourceIdx, rootIdx)
	if got != "code_samples/post.md#/snippet" {
		t.Fatalf("expected relative ref with fragment, got %q", got)
	}

	absTarget := filepath.Join(rootDir, "samples", "post.md")
	got = rebaseExtensionRefForComposed(absTarget, sourceIdx, rootIdx)
	if got != "samples/post.md" {
		t.Fatalf("expected absolute ref rebased to root, got %q", got)
	}

	otherRootIdx := newComposedRefIndex(t, "other-root/openapi.yaml", `openapi: 3.1.0`)
	absTarget = filepath.Join(rootDir, "external", "post.md")
	got = rebaseExtensionRefForComposed(absTarget+"#/snippet", sourceIdx, otherRootIdx)
	if got != filepath.ToSlash(absTarget)+"#/snippet" {
		t.Fatalf("expected absolute ref preserved when it cannot be rebased, got %q", got)
	}

	if got = rebaseExtensionRefForComposed("#/components/schemas/Pet", sourceIdx, rootIdx); got != "paths/echo.yaml#/components/schemas/Pet" {
		t.Fatalf("expected local source ref rebased to source file, got %q", got)
	}
	if got = rebaseExtensionRefForComposed("https://example.com/sample.md", sourceIdx, rootIdx); got != "https://example.com/sample.md" {
		t.Fatalf("expected remote ref unchanged, got %q", got)
	}
	if got = rebaseExtensionRefForComposed("sample.md", nil, rootIdx); got != "sample.md" {
		t.Fatalf("expected ref unchanged without source index, got %q", got)
	}
	if got = rebaseExtensionRefForComposed("#/components/schemas/Pet", nil, rootIdx); got != "#/components/schemas/Pet" {
		t.Fatalf("expected local ref unchanged without source index, got %q", got)
	}

	dirIdx := newComposedRefIndex(t, rootDir, `openapi: 3.1.0`)
	if got = specDir(dirIdx); got != rootDir {
		t.Fatalf("expected directory spec path to be preserved, got %q", got)
	}

	remoteIdx := newComposedRefIndex(t, "https://example.com/openapi.yaml", `openapi: 3.1.0`)
	if got = specDir(remoteIdx); got != "" {
		t.Fatalf("expected remote spec dir to be empty, got %q", got)
	}
}

func TestRebaseExtensionRefForComposed_PreservesWindowsAbsoluteRefOnDifferentVolume(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows volume behavior")
	}

	rootIdx := newComposedRefIndex(t, `C:\root\openapi.yaml`, `openapi: 3.1.0`)
	sourceIdx := newComposedRefIndex(t, `D:\paths\echo.yaml`, `post: {}`)

	got := rebaseExtensionRefForComposed(`D:\samples\post.md#/snippet`, sourceIdx, rootIdx)
	if got != "D:/samples/post.md#/snippet" {
		t.Fatalf("expected absolute Windows ref preserved when it cannot be rebased, got %q", got)
	}
}

func newComposedRefIndex(t *testing.T, absPath, spec string) *index.SpecIndex {
	t.Helper()
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(spec), &node); err != nil {
		t.Fatal(err)
	}
	idx := index.NewSpecIndexWithConfig(&node, index.CreateOpenAPIIndexConfig())
	idx.SetAbsolutePath(absPath)
	return idx
}

func findRefValue(node *yaml.Node, out *string) {
	if node == nil || *out != "" {
		return
	}
	if node.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i] != nil && node.Content[i].Value == "$ref" {
				*out = node.Content[i+1].Value
				return
			}
			findRefValue(node.Content[i+1], out)
		}
		return
	}
	for _, child := range node.Content {
		findRefValue(child, out)
	}
}

func TestResolveExtensionRefsFromIndex_NilIndex(t *testing.T) {
	// Should not panic with nil index
	resolveExtensionRefsFromIndex(nil, nil)
}

func TestResolveExtensionRefsFromIndex_NilRef(t *testing.T) {
	// Create a minimal index with no extension refs
	yml := `openapi: 3.1.0`
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	cfg := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&node, cfg)

	// Should handle empty extension refs gracefully
	resolveExtensionRefsFromIndex(idx, nil)
}

func TestResolveExtensionRefsFromIndex_SkipConditions(t *testing.T) {
	// Test the skip conditions: nil Node, empty FullDefinition, Circular

	t.Run("nil node in ref", func(t *testing.T) {
		yml := `openapi: 3.1.0`
		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)

		cfg := index.CreateOpenAPIIndexConfig()
		idx := index.NewSpecIndexWithConfig(&node, cfg)

		// Use reflection to inject a ref with nil Node into the index's rawSequencedRefs
		// This tests the defensive nil check in resolveExtensionRefsFromIndex
		nilNodeRef := &index.Reference{
			Node:           nil, // nil Node - should trigger skip
			FullDefinition: "#/components/schemas/Test",
			IsExtensionRef: true,
		}

		// Access private field rawSequencedRefs via reflection
		idxValue := reflect.ValueOf(idx).Elem()
		rawRefsField := idxValue.FieldByName("rawSequencedRefs")
		if rawRefsField.IsValid() && rawRefsField.CanAddr() {
			// Get the underlying slice and append our test ref
			rawRefsPtr := reflect.NewAt(rawRefsField.Type(), rawRefsField.Addr().UnsafePointer())
			rawRefs := rawRefsPtr.Elem()
			newSlice := reflect.Append(rawRefs, reflect.ValueOf(nilNodeRef))
			rawRefs.Set(newSlice)
		}

		// This should skip the ref with nil Node without panicking
		resolveExtensionRefsFromIndex(idx, nil)
	})

	t.Run("empty FullDefinition in ref", func(t *testing.T) {
		yml := `openapi: 3.1.0`
		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)

		cfg := index.CreateOpenAPIIndexConfig()
		idx := index.NewSpecIndexWithConfig(&node, cfg)

		// Inject a ref with empty FullDefinition
		emptyDefRef := &index.Reference{
			Node:           &yaml.Node{Kind: yaml.ScalarNode, Value: "test"},
			FullDefinition: "", // empty - should trigger skip
			IsExtensionRef: true,
		}

		idxValue := reflect.ValueOf(idx).Elem()
		rawRefsField := idxValue.FieldByName("rawSequencedRefs")
		if rawRefsField.IsValid() && rawRefsField.CanAddr() {
			rawRefsPtr := reflect.NewAt(rawRefsField.Type(), rawRefsField.Addr().UnsafePointer())
			rawRefs := rawRefsPtr.Elem()
			newSlice := reflect.Append(rawRefs, reflect.ValueOf(emptyDefRef))
			rawRefs.Set(newSlice)
		}

		resolveExtensionRefsFromIndex(idx, nil)
	})

	t.Run("circular ref", func(t *testing.T) {
		yml := `openapi: 3.1.0`
		var node yaml.Node
		_ = yaml.Unmarshal([]byte(yml), &node)

		cfg := index.CreateOpenAPIIndexConfig()
		idx := index.NewSpecIndexWithConfig(&node, cfg)

		// Inject a circular ref
		circularRef := &index.Reference{
			Node:           &yaml.Node{Kind: yaml.ScalarNode, Value: "test"},
			FullDefinition: "#/components/schemas/Test",
			IsExtensionRef: true,
			Circular:       true, // circular - should trigger skip
		}

		idxValue := reflect.ValueOf(idx).Elem()
		rawRefsField := idxValue.FieldByName("rawSequencedRefs")
		if rawRefsField.IsValid() && rawRefsField.CanAddr() {
			rawRefsPtr := reflect.NewAt(rawRefsField.Type(), rawRefsField.Addr().UnsafePointer())
			rawRefs := rawRefsPtr.Elem()
			newSlice := reflect.Append(rawRefs, reflect.ValueOf(circularRef))
			rawRefs.Set(newSlice)
		}

		resolveExtensionRefsFromIndex(idx, nil)
	})
}

func TestResolveExtensionRefContent_NilIndex(t *testing.T) {
	ctx := context.Background()

	// Reference with nil Index
	ref := &index.Reference{
		FullDefinition: "#/components/schemas/Test",
		Index:          nil,
	}

	result := resolveExtensionRefContent(ctx, ref, nil)
	if result != nil {
		t.Error("Expected nil result when ref.Index is nil")
	}
}

func TestResolveExtensionRefContent_ComponentNotFound(t *testing.T) {
	ctx := context.Background()

	yml := `openapi: 3.1.0`
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	cfg := index.CreateOpenAPIIndexConfig()
	idx := index.NewSpecIndexWithConfig(&node, cfg)

	// Reference to non-existent component
	ref := &index.Reference{
		FullDefinition: "#/components/schemas/DoesNotExist",
		Index:          idx,
	}

	result := resolveExtensionRefContent(ctx, ref, nil)
	if result != nil {
		t.Error("Expected nil result when component not found")
	}
}

func TestDeepCopyNode_Nil(t *testing.T) {
	result := deepCopyNode(nil)
	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

func TestDeepCopyNode_DocumentNode(t *testing.T) {
	// Test unwrapping of DocumentNode
	innerNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "test",
	}
	docNode := &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{innerNode},
	}

	result := deepCopyNode(docNode)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Kind != yaml.ScalarNode {
		t.Errorf("Expected ScalarNode after unwrap, got %v", result.Kind)
	}
	if result.Value != "test" {
		t.Errorf("Expected value 'test', got '%s'", result.Value)
	}
}

func TestDeepCopyNode_WithContent(t *testing.T) {
	// Test deep copy with children
	child1 := &yaml.Node{Kind: yaml.ScalarNode, Value: "key"}
	child2 := &yaml.Node{Kind: yaml.ScalarNode, Value: "value"}
	parent := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{child1, child2},
	}

	result := deepCopyNode(parent)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if len(result.Content) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(result.Content))
	}
	// Verify it's a deep copy (different pointers)
	if result.Content[0] == child1 {
		t.Error("Expected deep copy, got same pointer for child1")
	}
	if result.Content[1] == child2 {
		t.Error("Expected deep copy, got same pointer for child2")
	}
	// Verify values are copied
	if result.Content[0].Value != "key" {
		t.Errorf("Expected 'key', got '%s'", result.Content[0].Value)
	}
	if result.Content[1].Value != "value" {
		t.Errorf("Expected 'value', got '%s'", result.Content[1].Value)
	}
}

func TestDeepCopyNode_NoContent(t *testing.T) {
	// Test node with no children
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "scalar",
	}

	result := deepCopyNode(node)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Value != "scalar" {
		t.Errorf("Expected 'scalar', got '%s'", result.Value)
	}
	if result.Content != nil {
		t.Error("Expected nil Content for scalar node")
	}
}

func TestDeepCopyNode_AllFields(t *testing.T) {
	// Test that all fields are copied
	node := &yaml.Node{
		Kind:        yaml.ScalarNode,
		Style:       yaml.DoubleQuotedStyle,
		Tag:         "!!str",
		Value:       "test",
		Anchor:      "anchor1",
		HeadComment: "head",
		LineComment: "line",
		FootComment: "foot",
		Line:        10,
		Column:      5,
	}

	result := deepCopyNode(node)
	if result.Kind != yaml.ScalarNode {
		t.Errorf("Kind mismatch")
	}
	if result.Style != yaml.DoubleQuotedStyle {
		t.Errorf("Style mismatch")
	}
	if result.Tag != "!!str" {
		t.Errorf("Tag mismatch")
	}
	if result.Value != "test" {
		t.Errorf("Value mismatch")
	}
	if result.Anchor != "anchor1" {
		t.Errorf("Anchor mismatch")
	}
	if result.HeadComment != "head" {
		t.Errorf("HeadComment mismatch")
	}
	if result.LineComment != "line" {
		t.Errorf("LineComment mismatch")
	}
	if result.FootComment != "foot" {
		t.Errorf("FootComment mismatch")
	}
	if result.Line != 10 {
		t.Errorf("Line mismatch")
	}
	if result.Column != 5 {
		t.Errorf("Column mismatch")
	}
}

func TestReplaceRefNodeWithContent_NilRefNode(t *testing.T) {
	content := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	// Should not panic with nil refNode
	replaceRefNodeWithContent(nil, content)
}

func TestReplaceRefNodeWithContent_NilContent(t *testing.T) {
	refNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "ref"}
	// Should not panic with nil content
	replaceRefNodeWithContent(refNode, nil)
}

func TestReplaceRefNodeWithContent_BothNil(t *testing.T) {
	// Should not panic with both nil
	replaceRefNodeWithContent(nil, nil)
}

func TestReplaceRefNodeWithContent_Success(t *testing.T) {
	refNode := &yaml.Node{
		Kind:  yaml.MappingNode,
		Value: "$ref",
	}
	content := &yaml.Node{
		Kind:        yaml.ScalarNode,
		Style:       yaml.DoubleQuotedStyle,
		Tag:         "!!str",
		Value:       "resolved",
		Anchor:      "anc",
		HeadComment: "head",
		LineComment: "line",
		FootComment: "foot",
		Content:     []*yaml.Node{{Kind: yaml.ScalarNode, Value: "child"}},
	}

	replaceRefNodeWithContent(refNode, content)

	// Verify all fields were replaced
	if refNode.Kind != yaml.ScalarNode {
		t.Errorf("Kind not replaced")
	}
	if refNode.Style != yaml.DoubleQuotedStyle {
		t.Errorf("Style not replaced")
	}
	if refNode.Tag != "!!str" {
		t.Errorf("Tag not replaced")
	}
	if refNode.Value != "resolved" {
		t.Errorf("Value not replaced")
	}
	if refNode.Anchor != "anc" {
		t.Errorf("Anchor not replaced")
	}
	if refNode.HeadComment != "head" {
		t.Errorf("HeadComment not replaced")
	}
	if refNode.LineComment != "line" {
		t.Errorf("LineComment not replaced")
	}
	if refNode.FootComment != "foot" {
		t.Errorf("FootComment not replaced")
	}
	if len(refNode.Content) != 1 {
		t.Errorf("Content not replaced")
	}
}
