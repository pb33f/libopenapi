// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"net/url"
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
	_, ok := schemaAnchorFragmentName(fragment)
	return ok
}

// schemaAnchorFragmentName returns the anchor name a fragment such as "#foo" names. A URI
// fragment is percent-encoded, so "#f%6Fo" names foo as well, and "#%2Ffoo" is the JSON
// pointer "/foo", not an anchor.
func schemaAnchorFragmentName(fragment string) (string, bool) {
	if len(fragment) < 2 || fragment[0] != '#' {
		return "", false
	}
	name, err := url.PathUnescape(fragment[1:])
	if err != nil || !schemaAnchorName.MatchString(name) {
		return "", false
	}
	return name, true
}

// schemaAnchorDefinition returns the full definition of a plain-name fragment such as "#foo"
// found inside ref: the anchor is looked up in the document that holds ref.
func schemaAnchorDefinition(ref *Reference, fragment string) string {
	base := ref.FullDefinition
	if ref.RemoteLocation != "" {
		base = ref.RemoteLocation
	}
	document, _ := SplitRefFragment(base)
	return document + fragment
}

// registerSchemaAnchorAt records an $anchor under the base URI of the schema resource
// that declares it: the nearest $id, or the document itself. pointer is the JSON pointer
// segments to node, and scope is the walk's $id scope there.
func (index *SpecIndex) registerSchemaAnchorAt(node *yaml.Node, keyIndex int, pointer []string, scope *SchemaIdScope) {
	// in a sequence, such as required: [$anchor, name], "$anchor" is an entry, not a key.
	if len(node.Content) <= keyIndex+1 || utils.IsNodeArray(node) {
		return
	}
	anchorNode := node.Content[keyIndex+1]
	if !utils.IsNodeStringValue(anchorNode) || !schemaAnchorName.MatchString(anchorNode.Value) {
		return
	}
	// within an $id resource, whose root is a schema, the path from that root decides, so a
	// resource under an extension or a default keeps its anchors.
	if depth := scope.resourceDepth; depth > 0 && depth <= len(pointer) {
		if underSchemaDataPath(pointer[depth:], true) {
			return
		}
	} else if underSchemaDataPath(pointer, false) {
		return
	}
	baseUri := scope.BaseUri

	entry := &SchemaIdEntry{
		Id:             anchorNode.Value,
		ResolvedUri:    baseUri + "#" + anchorNode.Value,
		SchemaNode:     node,
		ParentId:       baseUri,
		Index:          index,
		DefinitionPath: schemaPointer(pointer),
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

// resolveRefViaSchemaAnchor resolves ref to the $anchor called name, already decoded from the
// ref's fragment. A ref with no base URI, such as "#foo", is looked up in this document.
func (index *SpecIndex) resolveRefViaSchemaAnchor(ref, baseUri, name string) *Reference {
	if baseUri == "" {
		baseUri = index.specAbsolutePath
	}
	key := baseUri + "#" + name
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
