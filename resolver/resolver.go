// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package resolver

import (
	"fmt"

	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// ResolvingError represents an issue the resolver had trying to stitch the tree together.
type ResolvingError struct {
	// ErrorRef is the error thrown by the resolver
	ErrorRef error

	// Node is the *yaml.Node reference that contains the resolving error
	Node *yaml.Node

	// Path is the shortened journey taken by the resolver
	Path string

	// CircularReference is set if the error is a reference to the circular reference.
	CircularReference *index.CircularReferenceResult
}

func (r *ResolvingError) Error() string {
	return fmt.Sprintf("%s: %s [%d:%d]", r.ErrorRef.Error(),
		r.Path, r.Node.Line, r.Node.Column)
}

// Resolver will use a *index.SpecIndex to stitch together a resolved root tree using all the discovered
// references in the doc.
type Resolver struct {
	specIndex          *index.SpecIndex
	resolvedRoot       *yaml.Node
	resolvingErrors    []*ResolvingError
	circularReferences []*index.CircularReferenceResult
	referencesVisited  int
	indexesVisited     int
	journeysTaken      int
	relativesSeen      int
}

// NewResolver will create a new resolver from a *index.SpecIndex
func NewResolver(index *index.SpecIndex) *Resolver {
	if index == nil {
		return nil
	}
	return &Resolver{
		specIndex:    index,
		resolvedRoot: index.GetRootNode(),
	}
}

// GetResolvingErrors returns all errors found during resolving
func (resolver *Resolver) GetResolvingErrors() []*ResolvingError {
	return resolver.resolvingErrors
}

// GetCircularErrors returns all circular reference errors found.
func (resolver *Resolver) GetCircularErrors() []*index.CircularReferenceResult {
	return resolver.circularReferences
}

// GetPolymorphicCircularErrors returns all circular errors that stem from polymorphism
func (resolver *Resolver) GetPolymorphicCircularErrors() []*index.CircularReferenceResult {
	var res []*index.CircularReferenceResult
	for i := range resolver.circularReferences {
		if !resolver.circularReferences[i].IsInfiniteLoop {
			continue
		}
		if !resolver.circularReferences[i].IsPolymorphicResult {
			continue
		}
		res = append(res, resolver.circularReferences[i])
	}
	return res
}

// GetNonPolymorphicCircularErrors returns all circular errors that DO NOT stem from polymorphism
func (resolver *Resolver) GetNonPolymorphicCircularErrors() []*index.CircularReferenceResult {
	var res []*index.CircularReferenceResult
	for i := range resolver.circularReferences {
		if !resolver.circularReferences[i].IsInfiniteLoop {
			continue
		}

		if !resolver.circularReferences[i].IsPolymorphicResult {
			res = append(res, resolver.circularReferences[i])
		}
	}

	return res
}

// GetJourneysTaken returns the number of journeys taken by the resolver
func (resolver *Resolver) GetJourneysTaken() int {
	return resolver.journeysTaken
}

// GetReferenceVisited returns the number of references visited by the resolver
func (resolver *Resolver) GetReferenceVisited() int {
	return resolver.referencesVisited
}

// GetIndexesVisited returns the number of indexes visited by the resolver
func (resolver *Resolver) GetIndexesVisited() int {
	return resolver.indexesVisited
}

// GetRelativesSeen returns the number of siblings (nodes at the same level) seen for each reference found.
func (resolver *Resolver) GetRelativesSeen() int {
	return resolver.relativesSeen
}

// Resolve will resolve the specification, everything that is not polymorphic and not circular, will be resolved.
// this data can get big, it results in a massive duplication of data. This is a destructive method and will permanently
// re-organize the node tree. Make sure you have copied your original tree before running this (if you want to preserve
// original data)
func (resolver *Resolver) Resolve() []*ResolvingError {

	visitIndex(resolver, resolver.specIndex)

	for _, circRef := range resolver.circularReferences {
		// If the circular reference is not required, we can ignore it, as it's a terminable loop rather than an infinite one
		if !circRef.IsInfiniteLoop {
			continue
		}

		resolver.resolvingErrors = append(resolver.resolvingErrors, &ResolvingError{
			ErrorRef: fmt.Errorf("Infinite circular reference detected: %s", circRef.Start.Name),
			Node:     circRef.LoopPoint.Node,
			Path:     circRef.GenerateJourneyPath(),
		})
	}

	return resolver.resolvingErrors
}

// CheckForCircularReferences Check for circular references, without resolving, a non-destructive run.
func (resolver *Resolver) CheckForCircularReferences() []*ResolvingError {
	visitIndexWithoutDamagingIt(resolver, resolver.specIndex)
	for _, circRef := range resolver.circularReferences {
		// If the circular reference is not required, we can ignore it, as it's a terminable loop rather than an infinite one
		if !circRef.IsInfiniteLoop {
			continue
		}

		resolver.resolvingErrors = append(resolver.resolvingErrors, &ResolvingError{
			ErrorRef:          fmt.Errorf("Infinite circular reference detected: %s", circRef.Start.Name),
			Node:              circRef.LoopPoint.Node,
			Path:              circRef.GenerateJourneyPath(),
			CircularReference: circRef,
		})
	}
	// update our index with any circular refs we found.
	resolver.specIndex.SetCircularReferences(resolver.circularReferences)
	return resolver.resolvingErrors
}

func visitIndexWithoutDamagingIt(res *Resolver, idx *index.SpecIndex) {
	mapped := idx.GetMappedReferencesSequenced()
	mappedIndex := idx.GetMappedReferences()
	res.indexesVisited++
	for _, ref := range mapped {
		seenReferences := make(map[string]bool)
		var journey []*index.Reference
		res.journeysTaken++
		res.VisitReference(ref.Reference, seenReferences, journey, false)
	}
	schemas := idx.GetAllComponentSchemas()
	for s, schemaRef := range schemas {
		if mappedIndex[s] == nil {
			seenReferences := make(map[string]bool)
			var journey []*index.Reference
			res.journeysTaken++
			res.VisitReference(schemaRef, seenReferences, journey, false)
		}
	}
	for _, c := range idx.GetChildren() {
		visitIndexWithoutDamagingIt(res, c)
	}
}

func visitIndex(res *Resolver, idx *index.SpecIndex) {
	mapped := idx.GetMappedReferencesSequenced()
	mappedIndex := idx.GetMappedReferences()
	res.indexesVisited++

	for _, ref := range mapped {
		seenReferences := make(map[string]bool)
		var journey []*index.Reference
		res.journeysTaken++
		if ref != nil && ref.Reference != nil {
			ref.Reference.Node.Content = res.VisitReference(ref.Reference, seenReferences, journey, true)
		}
	}

	schemas := idx.GetAllComponentSchemas()
	for s, schemaRef := range schemas {
		if mappedIndex[s] == nil {
			seenReferences := make(map[string]bool)
			var journey []*index.Reference
			res.journeysTaken++
			schemaRef.Node.Content = res.VisitReference(schemaRef, seenReferences, journey, true)
		}
	}

	// map everything
	for _, sequenced := range idx.GetAllSequencedReferences() {
		locatedDef := mappedIndex[sequenced.Definition]
		if locatedDef != nil {
			if !locatedDef.Circular && locatedDef.Seen {
				sequenced.Node.Content = locatedDef.Node.Content
			}
		}
	}
	for _, c := range idx.GetChildren() {
		visitIndex(res, c)
	}
}

// VisitReference will visit a reference as part of a journey and will return resolved nodes.
func (resolver *Resolver) VisitReference(ref *index.Reference, seen map[string]bool, journey []*index.Reference, resolve bool) []*yaml.Node {
	resolver.referencesVisited++
	if ref.Resolved || ref.Seen {
		return ref.Node.Content
	}

	journey = append(journey, ref)
	relatives := resolver.extractRelatives(ref.Node, seen, journey, resolve)

	seen = make(map[string]bool)

	seen[ref.Definition] = true
	for _, r := range relatives {
		// check if we have seen this on the journey before, if so! it's circular
		skip := false
		for i, j := range journey {
			if j.Definition == r.Definition {

				var foundDup *index.Reference
				foundRefs := resolver.specIndex.SearchIndexForReference(r.Definition)
				if len(foundRefs) > 0 {
					foundDup = foundRefs[0]
				}

				var circRef *index.CircularReferenceResult
				if !foundDup.Circular {
					loop := append(journey, foundDup)

					visitedDefinitions := map[string]bool{}
					isInfiniteLoop, _ := resolver.isInfiniteCircularDependency(foundDup, visitedDefinitions, nil)
					circRef = &index.CircularReferenceResult{
						Journey:        loop,
						Start:          foundDup,
						LoopIndex:      i,
						LoopPoint:      foundDup,
						IsInfiniteLoop: isInfiniteLoop,
					}
					resolver.circularReferences = append(resolver.circularReferences, circRef)

					foundDup.Seen = true
					foundDup.Circular = true
				}
				skip = true
			}
		}

		if !skip {
			var original *index.Reference
			foundRefs := resolver.specIndex.SearchIndexForReference(r.Definition)
			if len(foundRefs) > 0 {
				original = foundRefs[0]
			}
			resolved := resolver.VisitReference(original, seen, journey, resolve)
			if resolve {
				r.Node.Content = resolved // this is where we perform the actual resolving.
			}
			r.Seen = true
			ref.Seen = true
		}
	}
	ref.Resolved = true
	ref.Seen = true

	return ref.Node.Content
}

func (resolver *Resolver) isInfiniteCircularDependency(ref *index.Reference, visitedDefinitions map[string]bool, initialRef *index.Reference) (bool, map[string]bool) {
	if ref == nil {
		return false, visitedDefinitions
	}

	for refDefinition := range ref.RequiredRefProperties {
		r := resolver.specIndex.GetMappedReferences()[refDefinition]
		if initialRef != nil && initialRef.Definition == r.Definition {
			return true, visitedDefinitions
		}

		if visitedDefinitions[r.Definition] {
			continue
		}
		visitedDefinitions[r.Definition] = true

		ir := initialRef
		if ir == nil {
			ir = ref
		}

		var isChildICD bool
		isChildICD, visitedDefinitions = resolver.isInfiniteCircularDependency(r, visitedDefinitions, ir)
		if isChildICD {
			return true, visitedDefinitions
		}
	}

	return false, visitedDefinitions
}

func (resolver *Resolver) extractRelatives(node *yaml.Node,
	foundRelatives map[string]bool,
	journey []*index.Reference, resolve bool) []*index.Reference {

	if len(journey) > 100 {
		return nil
	}

	var found []*index.Reference
	if len(node.Content) > 0 {
		for i, n := range node.Content {
			if utils.IsNodeMap(n) || utils.IsNodeArray(n) {
				found = append(found, resolver.extractRelatives(n, foundRelatives, journey, resolve)...)
			}

			if i%2 == 0 && n.Value == "$ref" {

				if !utils.IsNodeStringValue(node.Content[i+1]) {
					continue
				}

				value := node.Content[i+1].Value

				ref := resolver.specIndex.SearchIndexForReference(value)

				if ref == nil {
					_, path := utils.ConvertComponentIdIntoFriendlyPathSearch(value)
					err := &ResolvingError{
						ErrorRef: fmt.Errorf("cannot resolve reference `%s`, it's missing", value),
						Node:     n,
						Path:     path,
					}
					resolver.resolvingErrors = append(resolver.resolvingErrors, err)
					continue
				}

				r := &index.Reference{
					Definition: value,
					Name:       value,
					Node:       node,
				}

				found = append(found, r)

				foundRelatives[value] = true
			}

			if i%2 == 0 && n.Value != "$ref" && n.Value != "" {

				if n.Value == "allOf" ||
					n.Value == "oneOf" ||
					n.Value == "anyOf" {

					// if this is a polymorphic link, we want to follow it and see if it becomes circular
					if utils.IsNodeMap(node.Content[i+1]) { // check for nested items
						// check if items is present, to indicate an array
						if _, v := utils.FindKeyNodeTop("items", node.Content[i+1].Content); v != nil {
							if utils.IsNodeMap(v) {
								if d, _, l := utils.IsNodeRefValue(v); d {
									ref := resolver.specIndex.GetMappedReferences()[l]
									if ref != nil && !ref.Circular {
										circ := false
										for f := range journey {
											if journey[f].Definition == ref.Definition {
												circ = true
												break
											}
										}
										if !circ {
											resolver.VisitReference(ref, foundRelatives, journey, resolve)
										} else {
											loop := append(journey, ref)
											circRef := &index.CircularReferenceResult{
												Journey:             loop,
												Start:               ref,
												LoopIndex:           i,
												LoopPoint:           ref,
												IsPolymorphicResult: true,
											}

											ref.Seen = true
											ref.Circular = true
											resolver.circularReferences = append(resolver.circularReferences, circRef)
										}
									}
								}
							}
						}
					}
					// for array based polymorphic items
					if utils.IsNodeArray(node.Content[i+1]) { // check for nested items
						// check if items is present, to indicate an array
						for q := range node.Content[i+1].Content {
							v := node.Content[i+1].Content[q]
							if utils.IsNodeMap(v) {
								if d, _, l := utils.IsNodeRefValue(v); d {
									ref := resolver.specIndex.GetMappedReferences()[l]
									if ref != nil && !ref.Circular {
										circ := false
										for f := range journey {
											if journey[f].Definition == ref.Definition {
												circ = true
												break
											}
										}
										if !circ {
											resolver.VisitReference(ref, foundRelatives, journey, resolve)
										} else {
											loop := append(journey, ref)
											circRef := &index.CircularReferenceResult{
												Journey:             loop,
												Start:               ref,
												LoopIndex:           i,
												LoopPoint:           ref,
												IsPolymorphicResult: true,
											}

											ref.Seen = true
											ref.Circular = true
											resolver.circularReferences = append(resolver.circularReferences, circRef)
										}
									}
								}
							}
						}
					}
					break
				}
			}
		}
	}
	resolver.relativesSeen += len(found)
	return found
}
