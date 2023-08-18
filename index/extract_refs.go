// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi/utils"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// ExtractRefs will return a deduplicated slice of references for every unique ref found in the document.
// The total number of refs, will generally be much higher, you can extract those from GetRawReferenceCount()
func (index *SpecIndex) ExtractRefs(node, parent *yaml.Node, seenPath []string, level int, poly bool, pName string) []*Reference {
	if node == nil {
		return nil
	}
	var found []*Reference
	if len(node.Content) > 0 {
		var prev, polyName string
		for i, n := range node.Content {

			if utils.IsNodeMap(n) || utils.IsNodeArray(n) {
				level++
				// check if we're using  polymorphic values. These tend to create rabbit warrens of circular
				// references if every single link is followed. We don't resolve polymorphic values.
				isPoly, _ := index.checkPolymorphicNode(prev)
				polyName = pName
				if isPoly {
					poly = true
					if prev != "" {
						polyName = prev
					}
				}
				found = append(found, index.ExtractRefs(n, node, seenPath, level, poly, polyName)...)
			}

			// check if we're dealing with an inline schema definition, that isn't part of an array
			// (which means it's being used as a value in an array, and it's not a label)
			// https://github.com/pb33f/libopenapi/issues/76
			schemaContainingNodes := []string{"schema", "items", "additionalProperties", "contains", "not", "unevaluatedItems", "unevaluatedProperties"}
			if i%2 == 0 && slices.Contains(schemaContainingNodes, n.Value) && !utils.IsNodeArray(node) && (i+1 < len(node.Content)) {
				ref := &Reference{
					Node: node.Content[i+1],
					Path: fmt.Sprintf("$.%s.%s", strings.Join(seenPath, "."), n.Value),
				}

				isRef, _, _ := utils.IsNodeRefValue(node.Content[i+1])
				if isRef {
					// record this reference
					index.allRefSchemaDefinitions = append(index.allRefSchemaDefinitions, ref)
					continue
				}

				if n.Value == "additionalProperties" || n.Value == "unevaluatedProperties" {
					if utils.IsNodeBoolValue(node.Content[i+1]) {
						continue
					}
				}

				index.allInlineSchemaDefinitions = append(index.allInlineSchemaDefinitions, ref)

				// check if the schema is an object or an array,
				// and if so, add it to the list of inline schema object definitions.
				k, v := utils.FindKeyNodeTop("type", node.Content[i+1].Content)
				if k != nil && v != nil {
					if v.Value == "object" || v.Value == "array" {
						index.allInlineSchemaObjectDefinitions = append(index.allInlineSchemaObjectDefinitions, ref)
					}
				}
			}

			// Perform the same check for all maps of schemas like properties and patternProperties
			// https://github.com/pb33f/libopenapi/issues/76
			mapOfSchemaContainingNodes := []string{"properties", "patternProperties"}
			if i%2 == 0 && slices.Contains(mapOfSchemaContainingNodes, n.Value) && !utils.IsNodeArray(node) && (i+1 < len(node.Content)) {
				// for each property add it to our schema definitions
				label := ""
				for h, prop := range node.Content[i+1].Content {

					if h%2 == 0 {
						label = prop.Value
						continue
					}
					ref := &Reference{
						Node: prop,
						Path: fmt.Sprintf("$.%s.%s.%s", strings.Join(seenPath, "."), n.Value, label),
					}

					isRef, _, _ := utils.IsNodeRefValue(prop)
					if isRef {
						// record this reference
						index.allRefSchemaDefinitions = append(index.allRefSchemaDefinitions, ref)
						continue
					}

					index.allInlineSchemaDefinitions = append(index.allInlineSchemaDefinitions, ref)

					// check if the schema is an object or an array,
					// and if so, add it to the list of inline schema object definitions.
					k, v := utils.FindKeyNodeTop("type", prop.Content)
					if k != nil && v != nil {
						if v.Value == "object" || v.Value == "array" {
							index.allInlineSchemaObjectDefinitions = append(index.allInlineSchemaObjectDefinitions, ref)
						}
					}
				}
			}

			// Perform the same check for all arrays of schemas like allOf, anyOf, oneOf
			arrayOfSchemaContainingNodes := []string{"allOf", "anyOf", "oneOf", "prefixItems"}
			if i%2 == 0 && slices.Contains(arrayOfSchemaContainingNodes, n.Value) && !utils.IsNodeArray(node) && (i+1 < len(node.Content)) {
				// for each element in the array, add it to our schema definitions
				for h, element := range node.Content[i+1].Content {
					ref := &Reference{
						Node: element,
						Path: fmt.Sprintf("$.%s.%s[%d]", strings.Join(seenPath, "."), n.Value, h),
					}

					isRef, _, _ := utils.IsNodeRefValue(element)
					if isRef { // record this reference
						index.allRefSchemaDefinitions = append(index.allRefSchemaDefinitions, ref)
						continue
					}
					index.allInlineSchemaDefinitions = append(index.allInlineSchemaDefinitions, ref)

					// check if the schema is an object or an array,
					// and if so, add it to the list of inline schema object definitions.
					k, v := utils.FindKeyNodeTop("type", element.Content)
					if k != nil && v != nil {
						if v.Value == "object" || v.Value == "array" {
							index.allInlineSchemaObjectDefinitions = append(index.allInlineSchemaObjectDefinitions, ref)
						}
					}
				}
			}

			if i%2 == 0 && n.Value == "$ref" {

				// only look at scalar values, not maps (looking at you k8s)
				if !utils.IsNodeStringValue(node.Content[i+1]) {
					continue
				}

				index.linesWithRefs[n.Line] = true

				fp := make([]string, len(seenPath))
				for x, foundPathNode := range seenPath {
					fp[x] = foundPathNode
				}

				value := node.Content[i+1].Value

				segs := strings.Split(value, "/")
				name := segs[len(segs)-1]
				_, p := utils.ConvertComponentIdIntoFriendlyPathSearch(value)
				ref := &Reference{
					Definition: value,
					Name:       name,
					Node:       node,
					Path:       p,
				}

				// add to raw sequenced refs
				index.rawSequencedRefs = append(index.rawSequencedRefs, ref)

				// add ref by line number
				refNameIndex := strings.LastIndex(value, "/")
				refName := value[refNameIndex+1:]
				if len(index.refsByLine[refName]) > 0 {
					index.refsByLine[refName][n.Line] = true
				} else {
					v := make(map[int]bool)
					v[n.Line] = true
					index.refsByLine[refName] = v
				}

				// if this ref value has any siblings (node.Content is larger than two elements)
				// then add to refs with siblings
				if len(node.Content) > 2 {
					copiedNode := *node
					copied := Reference{
						Definition: ref.Definition,
						Name:       ref.Name,
						Node:       &copiedNode,
						Path:       p,
					}
					// protect this data using a copy, prevent the resolver from destroying things.
					index.refsWithSiblings[value] = copied
				}

				// if this is a polymorphic reference, we're going to leave it out
				// allRefs. We don't ever want these resolved, so instead of polluting
				// the timeline, we will keep each poly ref in its own collection for later
				// analysis.
				if poly {
					index.polymorphicRefs[value] = ref

					// index each type
					switch pName {
					case "anyOf":
						index.polymorphicAnyOfRefs = append(index.polymorphicAnyOfRefs, ref)
					case "allOf":
						index.polymorphicAllOfRefs = append(index.polymorphicAllOfRefs, ref)
					case "oneOf":
						index.polymorphicOneOfRefs = append(index.polymorphicOneOfRefs, ref)
					}
					continue
				}

				// check if this is a dupe, if so, skip it, we don't care now.
				if index.allRefs[value] != nil { // seen before, skip.
					continue
				}

				if value == "" {

					completedPath := fmt.Sprintf("$.%s", strings.Join(fp, "."))

					indexError := &IndexingError{
						Err:  errors.New("schema reference is empty and cannot be processed"),
						Node: node.Content[i+1],
						Path: completedPath,
					}

					index.refErrors = append(index.refErrors, indexError)

					continue
				}

				index.allRefs[value] = ref
				found = append(found, ref)
			}

			if i%2 == 0 && n.Value != "$ref" && n.Value != "" {

				nodePath := fmt.Sprintf("$.%s", strings.Join(seenPath, "."))

				// capture descriptions and summaries
				if n.Value == "description" {

					// if the parent is a sequence, ignore.
					if utils.IsNodeArray(node) {
						continue
					}

					ref := &DescriptionReference{
						Content:   node.Content[i+1].Value,
						Path:      nodePath,
						Node:      node.Content[i+1],
						IsSummary: false,
					}

					if !utils.IsNodeMap(ref.Node) {
						index.allDescriptions = append(index.allDescriptions, ref)
						index.descriptionCount++
					}
				}

				if n.Value == "summary" {

					var b *yaml.Node
					if len(node.Content) == i+1 {
						b = node.Content[i]
					} else {
						b = node.Content[i+1]
					}
					ref := &DescriptionReference{
						Content:   b.Value,
						Path:      nodePath,
						Node:      b,
						IsSummary: true,
					}

					index.allSummaries = append(index.allSummaries, ref)
					index.summaryCount++
				}

				// capture security requirement references (these are not traditional references, but they
				// are used as a look-up. This is the only exception to the design.
				if n.Value == "security" {
					var b *yaml.Node
					if len(node.Content) == i+1 {
						b = node.Content[i]
					} else {
						b = node.Content[i+1]
					}
					if utils.IsNodeArray(b) {
						var secKey string
						for k := range b.Content {
							if utils.IsNodeMap(b.Content[k]) {
								for g := range b.Content[k].Content {
									if g%2 == 0 {
										secKey = b.Content[k].Content[g].Value
										continue
									}
									if utils.IsNodeArray(b.Content[k].Content[g]) {
										var refMap map[string][]*Reference
										if index.securityRequirementRefs[secKey] == nil {
											index.securityRequirementRefs[secKey] = make(map[string][]*Reference)
											refMap = index.securityRequirementRefs[secKey]
										} else {
											refMap = index.securityRequirementRefs[secKey]
										}
										for r := range b.Content[k].Content[g].Content {
											var refs []*Reference
											if refMap[b.Content[k].Content[g].Content[r].Value] != nil {
												refs = refMap[b.Content[k].Content[g].Content[r].Value]
											}

											refs = append(refs, &Reference{
												Definition: b.Content[k].Content[g].Content[r].Value,
												Path:       fmt.Sprintf("%s.security[%d].%s[%d]", nodePath, k, secKey, r),
												Node:       b.Content[k].Content[g].Content[r],
											})

											index.securityRequirementRefs[secKey][b.Content[k].Content[g].Content[r].Value] = refs
										}
									}
								}
							}
						}
					}
				}
				// capture enums
				if n.Value == "enum" {

					// all enums need to have a type, extract the type from the node where the enum was found.
					_, enumKeyValueNode := utils.FindKeyNodeTop("type", node.Content)

					if enumKeyValueNode != nil {
						ref := &EnumReference{
							Path:       nodePath,
							Node:       node.Content[i+1],
							Type:       enumKeyValueNode,
							SchemaNode: node,
							ParentNode: parent,
						}

						index.allEnums = append(index.allEnums, ref)
						index.enumCount++
					}
				}
				// capture all objects with properties
				if n.Value == "properties" {
					_, typeKeyValueNode := utils.FindKeyNodeTop("type", node.Content)

					if typeKeyValueNode != nil {
						isObject := false

						if typeKeyValueNode.Value == "object" {
							isObject = true
						}

						for _, v := range typeKeyValueNode.Content {
							if v.Value == "object" {
								isObject = true
							}
						}

						if isObject {
							index.allObjectsWithProperties = append(index.allObjectsWithProperties, &ObjectReference{
								Path:       nodePath,
								Node:       node,
								ParentNode: parent,
							})
						}
					}
				}

				seenPath = append(seenPath, n.Value)
				prev = n.Value
			}

			// if next node is map, don't add segment.
			if i < len(node.Content)-1 {
				next := node.Content[i+1]

				if i%2 != 0 && next != nil && !utils.IsNodeArray(next) && !utils.IsNodeMap(next) && len(seenPath) > 0 {
					seenPath = seenPath[:len(seenPath)-1]
				}
			}
		}
		if len(seenPath) > 0 {
			seenPath = seenPath[:len(seenPath)-1]
		}

	}
	if len(seenPath) > 0 {
		seenPath = seenPath[:len(seenPath)-1]
	}

	index.refCount = len(index.allRefs)

	return found
}

// ExtractComponentsFromRefs returns located components from references. The returned nodes from here
// can be used for resolving as they contain the actual object properties.
func (index *SpecIndex) ExtractComponentsFromRefs(refs []*Reference) []*Reference {
	var found []*Reference

	// run this async because when things get recursive, it can take a while
	c := make(chan bool)

	locate := func(ref *Reference, refIndex int, sequence []*ReferenceMapped) {
		located := index.FindComponent(ref.Definition, ref.Node)
		if located != nil {
			index.refLock.Lock()
			if index.allMappedRefs[ref.Definition] == nil {
				found = append(found, located)
				index.allMappedRefs[ref.Definition] = located
				sequence[refIndex] = &ReferenceMapped{
					Reference:  located,
					Definition: ref.Definition,
				}
			}
			index.refLock.Unlock()
		} else {

			_, path := utils.ConvertComponentIdIntoFriendlyPathSearch(ref.Definition)
			indexError := &IndexingError{
				Err:  fmt.Errorf("component '%s' does not exist in the specification", ref.Definition),
				Node: ref.Node,
				Path: path,
			}
			index.errorLock.Lock()
			index.refErrors = append(index.refErrors, indexError)
			index.errorLock.Unlock()
		}
		c <- true
	}

	var refsToCheck []*Reference
	for _, ref := range refs {

		// check reference for backslashes (hah yeah seen this too!)
		if strings.Contains(ref.Definition, "\\") { // this was from blazemeter.com haha!
			_, path := utils.ConvertComponentIdIntoFriendlyPathSearch(ref.Definition)
			indexError := &IndexingError{
				Err:  fmt.Errorf("component '%s' contains a backslash '\\'. It's not valid", ref.Definition),
				Node: ref.Node,
				Path: path,
			}
			index.refErrors = append(index.refErrors, indexError)
			continue

		}
		refsToCheck = append(refsToCheck, ref)
	}
	mappedRefsInSequence := make([]*ReferenceMapped, len(refsToCheck))

	for r := range refsToCheck {
		// expand our index of all mapped refs
		go locate(refsToCheck[r], r, mappedRefsInSequence)
		// locate(refsToCheck[r], r, mappedRefsInSequence) // used for sync testing.
	}

	completedRefs := 0
	for completedRefs < len(refsToCheck) {
		select {
		case <-c:
			completedRefs++
		}
	}
	for m := range mappedRefsInSequence {
		if mappedRefsInSequence[m] != nil {
			index.allMappedRefsSequenced = append(index.allMappedRefsSequenced, mappedRefsInSequence[m])
		}
	}
	return found
}
