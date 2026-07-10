// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

type inlineReferenceTarget struct {
	fullDefinition string
	definition     string
	index          *index.SpecIndex
	node           *yaml.Node
	name           string
}

// prepareInlineReferences lifts external cycle and discriminator targets before
// rendering and records source-qualified preservation and rewrite policy in ctx.
func prepareInlineReferences(model *v3.Document, ctx *highbase.InlineRenderContext, includeDiscriminatorTargets bool) error {
	if model == nil || model.Rolodex == nil || ctx == nil {
		return nil
	}

	targets, err := collectInlineReferenceTargets(model.Rolodex, includeDiscriminatorTargets)
	return prepareCollectedInlineReferences(model, ctx, includeDiscriminatorTargets, targets, err)
}

func prepareCollectedInlineReferences(model *v3.Document, ctx *highbase.InlineRenderContext, includeDiscriminatorTargets bool, targets []*inlineReferenceTarget, collectErr error) error {
	if collectErr != nil {
		return collectErr
	}
	if len(targets) == 0 {
		if includeDiscriminatorTargets {
			return populateInlineDiscriminatorMappingRewrites(model.Rolodex, nil, ctx)
		}
		return nil
	}

	if model.Components == nil {
		model.Components, _ = buildComponents(model.Rolodex.GetRootIndex())
	}
	if model.Components.Schemas == nil {
		model.Components.Schemas = orderedmap.New[string, *highbase.SchemaProxy]()
	}

	existingNames := make(map[string]bool, orderedmap.Len(model.Components.Schemas)+len(targets))
	for pair := model.Components.Schemas.First(); pair != nil; pair = pair.Next() {
		existingNames[pair.Key()] = true
	}

	rewritesByTarget := make(map[string]string, len(targets))
	for _, target := range targets {
		componentName := findInlineTargetComponent(model.Components.Schemas, target)
		if componentName == "" {
			componentName = target.name
			if existingNames[componentName] {
				componentName = calculateCollisionNameInline(componentName, target.fullDefinition, "__", existingNames)
			}
			schema, _ := buildSchema(target.node, target.index)
			model.Components.Schemas.Set(componentName, schema)
			existingNames[componentName] = true
		}
		rewritesByTarget[target.fullDefinition] = "#/components/schemas/" + encodeJSONPointerSegment(componentName)
	}

	populateInlineReferenceContext(model.Rolodex, targets, rewritesByTarget, ctx)
	if includeDiscriminatorTargets {
		if err := populateInlineDiscriminatorMappingRewrites(model.Rolodex, rewritesByTarget, ctx); err != nil {
			return err
		}
	}
	return nil
}

func collectInlineReferenceTargets(rolodex *index.Rolodex, includeDiscriminatorTargets bool) ([]*inlineReferenceTarget, error) {
	if rolodex == nil || rolodex.GetRootIndex() == nil {
		return nil, nil
	}

	rootPath := index.CanonicalReferenceIdentity(rolodex.GetRootIndex().GetSpecAbsolutePath())
	indexes := append(append([]*index.SpecIndex{}, rolodex.GetIndexes()...), rolodex.GetRootIndex())
	indexesByPath := make(map[string]*index.SpecIndex, len(indexes))
	schemaTargets := make(map[string]struct{})
	var circular []*index.CircularReferenceResult
	for _, idx := range indexes {
		indexesByPath[index.CanonicalReferenceIdentity(idx.GetSpecAbsolutePath())] = idx
		circular = append(circular, idx.GetCircularReferences()...)
		schemaReferenceNodes := make(map[*yaml.Node]struct{})
		for _, schemaRef := range idx.GetAllReferenceSchemas() {
			if schemaRef != nil && schemaRef.Node != nil {
				schemaReferenceNodes[schemaRef.Node] = struct{}{}
			}
		}
		for _, mapped := range idx.GetMappedReferencesSequenced() {
			addInlineSchemaTarget(schemaTargets, schemaReferenceNodes, mapped)
		}
	}
	circular = append(circular, rolodex.GetIgnoredCircularReferences()...)
	circular = append(circular, rolodex.GetSafeCircularReferences()...)

	byDefinition := make(map[string]*inlineReferenceTarget)
	for _, result := range circular {
		if result == nil || len(result.Journey) == 0 {
			continue
		}
		start := circularJourneyStart(result)
		for _, ref := range result.Journey[start:] {
			if ref == nil || ref.FullDefinition == "" {
				continue
			}
			fullDefinition := index.CanonicalReferenceIdentity(ref.FullDefinition)
			if _, isSchema := schemaTargets[fullDefinition]; !isSchema {
				continue
			}
			filePath, definition := index.SplitRefFragment(fullDefinition)
			if filePath == "" || filePath == rootPath {
				continue
			}
			if _, exists := byDefinition[fullDefinition]; exists {
				continue
			}
			target, err := buildCircularInlineTarget(ref, fullDefinition, filePath, definition, indexesByPath)
			if err != nil {
				return nil, err
			}
			byDefinition[fullDefinition] = target
		}
	}

	if includeDiscriminatorTargets {
		for _, schema := range collectExternalDiscriminatorSchemas(rolodex, rolodex.GetRootIndex()) {
			fullDefinition := index.CanonicalReferenceIdentity(schema.fullDef)
			if _, exists := byDefinition[fullDefinition]; exists {
				continue
			}
			node := schema.ref.Node
			byDefinition[fullDefinition] = &inlineReferenceTarget{
				fullDefinition: fullDefinition,
				definition:     schema.originalRef,
				index:          schema.idx,
				node:           node,
				name:           schema.schemaName,
			}
		}
	}

	targets := make([]*inlineReferenceTarget, 0, len(byDefinition))
	for _, target := range byDefinition {
		targets = append(targets, target)
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].fullDefinition < targets[j].fullDefinition
	})
	return targets, nil
}

func addInlineSchemaTarget(targets map[string]struct{}, schemaNodes map[*yaml.Node]struct{}, mapped *index.ReferenceMapped) {
	if mapped == nil || mapped.OriginalReference == nil {
		return
	}
	if _, isSchema := schemaNodes[mapped.OriginalReference.Node]; isSchema {
		targets[index.CanonicalReferenceIdentity(mapped.FullDefinition)] = struct{}{}
	}
}

func buildCircularInlineTarget(ref *index.Reference, fullDefinition, filePath, definition string, indexesByPath map[string]*index.SpecIndex) (*inlineReferenceTarget, error) {
	sourceIndex := indexesByPath[filePath]
	if sourceIndex == nil {
		return nil, fmt.Errorf("external circular reference %q has no source index", ref.FullDefinition)
	}
	node := ref.Node
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node == nil {
		return nil, fmt.Errorf("external circular reference %q has no schema node", ref.FullDefinition)
	}
	return &inlineReferenceTarget{
		fullDefinition: fullDefinition,
		definition:     definition,
		index:          sourceIndex,
		node:           node,
		name:           inlineTargetName(filePath, definition),
	}, nil
}

func circularJourneyStart(result *index.CircularReferenceResult) int {
	if result.LoopIndex >= 0 && result.LoopIndex < len(result.Journey) {
		return result.LoopIndex
	}
	if result.LoopPoint != nil {
		loopPoint := index.CanonicalReferenceIdentity(result.LoopPoint.FullDefinition)
		for i := len(result.Journey) - 1; i >= 0; i-- {
			if result.Journey[i] != nil && index.CanonicalReferenceIdentity(result.Journey[i].FullDefinition) == loopPoint {
				return i
			}
		}
	}
	return 0
}

func populateInlineDiscriminatorMappingRewrites(rolodex *index.Rolodex, rewrites map[string]string, ctx *highbase.InlineRenderContext) error {
	for _, mapping := range collectDiscriminatorMappingNodesWithContext(rolodex) {
		ref, targetIndex := mapping.sourceIdx.SearchIndexForReference(mapping.node.Value)
		if ref == nil || targetIndex == nil {
			if utils.IsExternalRef(mapping.node.Value) {
				return fmt.Errorf("unable to resolve external discriminator mapping %q", mapping.node.Value)
			}
			continue
		}
		if targetIndex == rolodex.GetRootIndex() {
			continue
		}
		canonical := index.CanonicalReferenceIdentity(targetIndex.GetSpecAbsolutePath() + ref.Definition)
		replacement := rewrites[canonical]
		if replacement == "" {
			return fmt.Errorf("external discriminator mapping %q was not lifted", mapping.node.Value)
		}
		ctx.SetMappingRewrite(mapping.node, replacement)
	}
	return nil
}

func inlineTargetName(filePath, definition string) string {
	if definition != "" {
		segment := definition[strings.LastIndex(definition, "/")+1:]
		segment = strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
		if decoded, err := url.PathUnescape(segment); err == nil {
			segment = decoded
		}
		if segment != "" {
			return segment
		}
	}
	base := filepath.Base(filePath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" || name == "." {
		return "Schema"
	}
	return name
}

func findInlineTargetComponent(schemas *orderedmap.Map[string, *highbase.SchemaProxy], target *inlineReferenceTarget) string {
	if schemas == nil || target == nil {
		return ""
	}
	for pair := schemas.First(); pair != nil; pair = pair.Next() {
		proxy := pair.Value()
		if proxy == nil || proxy.GoLow() == nil {
			continue
		}
		lowProxy := proxy.GoLow()
		if lowProxy.GetIndex() == target.index && lowProxy.GetValueNode() == target.node {
			return pair.Key()
		}
	}
	return ""
}

func populateInlineReferenceContext(rolodex *index.Rolodex, targets []*inlineReferenceTarget, rewrites map[string]string, ctx *highbase.InlineRenderContext) {
	targetsByDefinition := make(map[string]*inlineReferenceTarget, len(targets))
	for _, target := range targets {
		targetsByDefinition[target.fullDefinition] = target
	}
	indexes := append(append([]*index.SpecIndex{}, rolodex.GetIndexes()...), rolodex.GetRootIndex())
	for _, idx := range indexes {
		for _, mapped := range idx.GetMappedReferencesSequenced() {
			canonicalTarget := index.CanonicalReferenceIdentity(mapped.FullDefinition)
			ctx.SetReferenceNodeIdentity(mapped.OriginalReference.Node, mapped.OriginalReference.Index, canonicalTarget)
			replacement, ok := rewrites[canonicalTarget]
			if !ok {
				continue
			}
			authored := inlineAuthoredReference(mapped.OriginalReference)
			ctx.SetReferenceNodeRewrite(mapped.OriginalReference.Node, replacement)
			ctx.MarkReferenceNodeAsPreserved(mapped.OriginalReference.Node)
			ctx.SetReferenceRewrite(mapped.OriginalReference.Index, authored, replacement)
			ctx.MarkScopedRefAsPreserved(mapped.OriginalReference.Index, authored)
			// Resolved high-level proxies are backed by the target index even though
			// they retain the authored reference string. Record that representation
			// as an alias without dropping target qualification.
			if target := targetsByDefinition[canonicalTarget]; target != nil {
				ctx.SetReferenceRewrite(target.index, authored, replacement)
				ctx.MarkScopedRefAsPreserved(target.index, authored)
			}
		}
	}

	for _, target := range targets {
		replacement := rewrites[target.fullDefinition]
		ctx.SetReferenceRewrite(target.index, target.definition, replacement)
		ctx.MarkScopedRefAsPreserved(target.index, target.definition)
	}
}

func inlineAuthoredReference(ref *index.Reference) string {
	if ref.RawRef != "" {
		return ref.RawRef
	}
	return ref.Definition
}
