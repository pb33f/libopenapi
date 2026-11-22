// Copyright 2023-2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"io"
	"log/slog"
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

// IsSchemaAnchorFragment reports whether fragment, such as "#foo", names a JSON Schema $anchor
// rather than a JSON pointer such as "#/components/schemas/Foo".
func IsSchemaAnchorFragment(fragment string) bool {
	return isSchemaAnchorFragment(fragment)
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
	// a resource under an extension keeps its anchors, see inSchemaData.
	if inSchemaData(pointer, scope.resourceDepth, true) {
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

// recordSchemaIdBase records the base URI of node, a mapping inside an $id resource. A node the
// walk reaches again through a YAML alias keeps the base it was first seen with.
func (index *SpecIndex) recordSchemaIdBase(node *yaml.Node, baseUri string) {
	index.schemaIdRegistryLock.Lock()
	defer index.schemaIdRegistryLock.Unlock()
	if index.schemaIdBaseByNode == nil {
		index.schemaIdBaseByNode = make(map[*yaml.Node]string)
	}
	if _, ok := index.schemaIdBaseByNode[node]; !ok {
		index.schemaIdBaseByNode[node] = baseUri
	}
}

// GetSchemaIdBaseForNode returns the base URI of the innermost $id resource holding node, a mapping
// in this index's document, including node's own $id. It returns "" when node is outside every
// $id resource, where the document is the base. A schema reached by a JSON pointer, such as
// "#/components/schemas/R/$defs/A", resolves its "#foo" and relative refs against this base.
func (index *SpecIndex) GetSchemaIdBaseForNode(node *yaml.Node) string {
	if index == nil || node == nil {
		return ""
	}
	index.schemaIdRegistryLock.RLock()
	defer index.schemaIdRegistryLock.RUnlock()
	return index.schemaIdBaseByNode[node]
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
	if entry.DefinitionPath == "#" && resolved.RemoteLocation != "" {
		// an anchor on the document root names the whole file, as a whole-file ref does.
		resolved.FullDefinition = resolved.RemoteLocation
	}
	// name it after the schema it lands on, as a pointer ref would be.
	resolved.Name = entry.Id
	if i := strings.LastIndexByte(entry.DefinitionPath, '/'); i >= 0 && i+1 < len(entry.DefinitionPath) {
		resolved.Name = entry.DefinitionPath[i+1:]
	}
	// the circular check follows the schema's required refs. inside an $id resource they resolve
	// against it, elsewhere against the document, as a pointer ref's do.
	requiredBase := resolved.FullDefinition
	if entry.Index != nil && entry.ParentId == entry.Index.GetSpecAbsolutePath() {
		// the anchor's resource is the document, not an $id.
		resolved.SchemaIdBase = ""
	} else {
		requiredBase = entry.ParentId
	}
	resolved.RequiredRefProperties = extractDefinitionRequiredRefProperties(entry.SchemaNode, map[string][]string{}, requiredBase, entry.Index)
	// errors about the reference, such as an infinite loop, point at where it was used.
	resolved.ParentNode = lookupComponentParentNode(index, ref, key)
	return resolved
}

// findSchemaAnchorInFile resolves ref, such as "other.yaml#foo", to the $anchor called name in
// file. The anchor belongs to the file's root resource: its $id, or the file itself.
func (index *SpecIndex) findSchemaAnchorInFile(ctx context.Context, file, ref, name string) *Reference {
	if file == index.specAbsolutePath {
		return index.resolveRefViaSchemaAnchor(ref, "", name)
	}
	fileRef := index.lookupRolodex(ctx, []string{file})
	if fileRef == nil || fileRef.Node == nil {
		return nil
	}
	fileIndex := fileRef.Index
	if fileIndex == nil {
		// the file is still being indexed elsewhere, and waiting for that can deadlock files that
		// refer to each other. Its anchors come from the same walk its index runs.
		fileIndex = index.scanSchemaIdentifiers(fileRef.RemoteLocation, fileRef.Node)
	}
	base := fileIndex.GetSchemaIdBaseForNode(fileRef.Node)
	resolved := fileIndex.resolveRefViaSchemaAnchor(ref, base, name)
	if resolved == nil {
		return nil
	}
	if fileRef.Index == nil {
		// as with a pointer ref into a file still being indexed, there is no index to name yet.
		resolved.Index = nil
	}
	if base != "" {
		// a search resolves the raw ref against the $id, so name the anchor by its URI there.
		resolved.RawRef = base + "#" + name
	}
	// the ref was used in this document.
	resolved.ParentNode = lookupComponentParentNode(index, ref, ref)
	return resolved
}

// scanSchemaIdentifiers walks node, the root mapping of file, for its $id and $anchor declarations
// on an index of its own, which records nothing else of use and is then discarded.
func (index *SpecIndex) scanSchemaIdentifiers(file string, node *yaml.Node) *SpecIndex {
	config := SpecIndexConfig{}
	if index.config != nil {
		config = *index.config
	}
	config.SpecAbsolutePath = file
	config.Rolodex = nil
	scan := &SpecIndex{config: &config, specAbsolutePath: file, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	bootstrapIndexCollections(scan)
	scan.ExtractRefs(context.Background(), node, node, []string{}, 0, false, "")
	return scan
}
