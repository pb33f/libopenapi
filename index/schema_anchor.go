// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"regexp"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/utils"
)

// schemaAnchorName is the plain-name form JSON Schema 2020-12 allows for $anchor.
var schemaAnchorName = regexp.MustCompile(`^[A-Za-z_][-A-Za-z0-9._]*$`)

// isSchemaAnchorFragment reports whether a fragment such as "#foo" names an $anchor
// rather than a JSON pointer.
func isSchemaAnchorFragment(fragment string) bool {
	return len(fragment) > 1 && fragment[0] == '#' && fragment[1] != '/'
}

// registerSchemaAnchorAt records an $anchor under the base URI of the schema resource
// that declares it: the nearest $id, or the document itself.
func (index *SpecIndex) registerSchemaAnchorAt(node *yaml.Node, keyIndex int, seenPath []string, baseUri string) {
	if underOpenAPIExamplePath(seenPath) || len(node.Content) <= keyIndex+1 {
		return
	}
	anchorNode := node.Content[keyIndex+1]
	if !utils.IsNodeStringValue(anchorNode) || !schemaAnchorName.MatchString(anchorNode.Value) {
		return
	}

	definitionPath := "#"
	if len(seenPath) > 0 {
		definitionPath = "#/" + strings.Join(seenPath, "/")
	}
	entry := &SchemaIdEntry{
		Id:             anchorNode.Value,
		ResolvedUri:    baseUri + "#" + anchorNode.Value,
		SchemaNode:     node,
		ParentId:       baseUri,
		Index:          index,
		DefinitionPath: definitionPath,
		Line:           anchorNode.Line,
		Column:         anchorNode.Column,
	}

	index.schemaIdRegistryLock.Lock()
	defer index.schemaIdRegistryLock.Unlock()
	if index.schemaAnchorRegistry == nil {
		index.schemaAnchorRegistry = make(map[string]*SchemaIdEntry)
	}
	if _, ok := index.schemaAnchorRegistry[entry.ResolvedUri]; !ok {
		index.schemaAnchorRegistry[entry.ResolvedUri] = entry
	}
}

func (index *SpecIndex) getSchemaAnchor(uri string) *SchemaIdEntry {
	index.schemaIdRegistryLock.RLock()
	defer index.schemaIdRegistryLock.RUnlock()
	return index.schemaAnchorRegistry[uri]
}

// resolveRefViaSchemaAnchor resolves a ref whose fragment is a plain-name $anchor. A ref
// with no base URI, such as "#foo", is looked up in this document.
func (index *SpecIndex) resolveRefViaSchemaAnchor(ref, baseUri, fragment string) *Reference {
	if baseUri == "" {
		baseUri = index.specAbsolutePath
	}
	key := baseUri + fragment
	entry := index.getSchemaAnchor(key)
	if entry == nil && index.rolodex != nil {
		entry = index.rolodex.lookupSchemaAnchor(key)
	}
	if entry == nil {
		return nil
	}

	resolved := buildSchemaIdResolvedReference(index, entry, ref, baseUri, "")
	// name it after the schema it lands on, as a pointer ref would be.
	resolved.Name = entry.Id
	if i := strings.LastIndexByte(entry.DefinitionPath, '/'); i >= 0 && i+1 < len(entry.DefinitionPath) {
		resolved.Name = entry.DefinitionPath[i+1:]
	}
	if entry.Index != nil && entry.ParentId == entry.Index.GetSpecAbsolutePath() {
		// the anchor's resource is the document, not an $id.
		resolved.SchemaIdBase = ""
	}
	return resolved
}
