// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package jsonschema loads and compares authored Draft 2020-12 schema resources.
package jsonschema

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
)

// Dialect is the supported JSON Schema dialect.
const Dialect = "https://json-schema.org/draft/2020-12/schema"

// ErrNotSchema identifies an input without a supported standalone schema declaration.
var ErrNotSchema = errors.New("expected a Draft 2020-12 JSON Schema declaration (use --schema for an unmarked schema)")

// Source holds the physical document once, even when several resources share it.
type Source struct {
	relative string
	Name     string
	Data     []byte
	Node     *yaml.Node
	Index    *index.SpecIndex
}

// Resource is a root schema or an embedded schema with its own base URI.
type Resource struct {
	base    *url.URL
	ID      string
	Pointer string
	Source  *Source
	Node    *yaml.Node
}

// Label returns the authored title or a name derived from the resource.
func (r *Resource) Label() string {
	if n := member(r.Node, "title"); n != nil && n.Value != "" {
		return n.Value
	}
	if name := path.Base(r.ID); name != "." && name != "/" && name != "" {
		return name
	}
	return "Schema"
}

// Reference connects an authored reference site to a schema in this snapshot.
type Reference struct {
	From, To *yaml.Node
	Value    string
	// Keyword identifies the authored $ref or $dynamicRef.
	Keyword string
	address string
	context evaluationContext
}

// Document owns the resources and source locations for a comparison input.
type Document struct {
	Root              *Resource
	Resources         []*Resource
	Sources           map[string]*Source
	References        []Reference
	Rolodex           *index.Rolodex
	addresses         map[string]*yaml.Node
	schemas           map[*yaml.Node]*Resource
	schemaPointers    map[*yaml.Node]string
	hasAdverseContext bool
	config            *datamodel.DocumentConfiguration
	base              *url.URL
}

// Load reads one root and all its referenced resources using the supplied source configuration.
// Explicit mode accepts unmarked object and boolean schemas.
func Load(data []byte, source string, config *datamodel.DocumentConfiguration, explicit bool) (*Document, error) {
	if config == nil {
		config = datamodel.NewDocumentConfiguration()
	}
	uri, err := retrievalURI(source, config)
	if err != nil {
		return nil, err
	}
	doc := newDocument(config, uri.ResolveReference(&url.URL{Path: "."}))
	root, err := doc.addSource(data, uri, explicit)
	if err == nil {
		doc.Root = root
		err = doc.resolveReferences()
	}
	if err != nil {
		doc.Release()
		return nil, err
	}
	return doc, nil
}

func newDocument(config *datamodel.DocumentConfiguration, base *url.URL) *Document {
	return &Document{config: config, base: base, Sources: make(map[string]*Source), addresses: make(map[string]*yaml.Node), schemas: make(map[*yaml.Node]*Resource), schemaPointers: make(map[*yaml.Node]string), Rolodex: index.NewRolodex(index.CreateClosedAPIIndexConfig())}
}

// IsSchema reports whether a node occupies a schema-bearing position in this input.
func (d *Document) IsSchema(node *yaml.Node) bool { return d != nil && d.schemas[node] != nil }

// Release frees the indexes and drops references to the retained source buffers.
func (d *Document) Release() {
	if d == nil {
		return
	}
	for _, source := range d.Sources {
		source.Index.Release()
	}
	if d.Rolodex != nil {
		d.Rolodex.Release()
	}
	*d = Document{}
}

func retrievalURI(source string, config *datamodel.DocumentConfiguration) (*url.URL, error) {
	if config.SpecFilePath != "" {
		source = config.SpecFilePath
	}
	if source == "" {
		source = "schema.yaml"
	}
	original, err := url.Parse(source)
	if config.BaseURL != nil {
		base := *config.BaseURL
		base.Path = strings.TrimSuffix(base.Path, "/") + "/"
		filename := filepath.Base(source)
		if err == nil && original.IsAbs() {
			filename = path.Base(original.Path)
		}
		return base.ResolveReference(&url.URL{Path: filename}), nil
	}
	if err == nil && (original.Scheme == "http" || original.Scheme == "https") {
		return original, nil
	}
	if !filepath.IsAbs(source) {
		source = filepath.Join(config.BasePath, source)
	}
	file, err := filepath.Abs(source)
	return fileURL(file), err
}

func (d *Document) resourceID(uri *url.URL) string {
	if uri.Scheme == "file" && d.base.Scheme == "file" && uri.Host == d.base.Host {
		if relative, err := filepath.Rel(filePath(d.base), filePath(uri)); err == nil {
			name := filepath.ToSlash(relative)
			if uri.RawQuery != "" {
				name += "?" + uri.RawQuery
			}
			return name
		}
	}
	identity := *uri
	identity.User = nil
	return identity.String()
}

func (d *Document) sourceName(uri *url.URL) string {
	safe := *uri
	safe.User = nil
	safe.RawQuery = ""
	name := d.resourceID(&safe)
	if uri.RawQuery != "" || uri.User != nil {
		// Signed retrieval URLs must not leak credentials or collide in report source maps.
		digest := sha256.Sum256([]byte(uri.String()))
		name += fmt.Sprintf(" [%x]", digest[:8])
	}
	return name
}

func (d *Document) addSource(data []byte, uri *url.URL, explicit bool) (*Resource, error) {
	doc, err := datamodel.DecodeSchemaNode(data)
	if err != nil {
		return nil, err
	}
	root := doc.Content[0]
	if !explicit && member(root, "$schema") == nil {
		return nil, ErrNotSchema
	}
	for _, marker := range []string{"openapi", "swagger", "asyncapi"} {
		if member(root, marker) != nil {
			return nil, fmt.Errorf("conflicting %s version marker in JSON Schema", marker)
		}
	}
	if strings.EqualFold(path.Ext(uri.Path), ".json") && !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON in %s", d.sourceName(uri))
	}
	name := d.sourceName(uri)
	cfg := index.CreateClosedAPIIndexConfig()
	cfg.SpecAbsolutePath, cfg.Rolodex = name, d.Rolodex
	idx := index.NewSchemaIndex(doc, cfg)
	relative := name
	if uri.Scheme != "file" && uri.Scheme == d.base.Scheme && uri.Host == d.base.Host && uri.RawQuery == "" {
		if rel, err := filepath.Rel(d.base.Path, uri.Path); err == nil {
			relative = filepath.ToSlash(rel)
		}
	}
	source := &Source{Name: name, relative: relative, Data: data, Node: root, Index: idx}
	d.Sources[name] = source
	if len(d.Resources) == 0 {
		d.Rolodex.SetRootNode(doc)
		d.Rolodex.SetRootIndex(idx)
	} else {
		d.Rolodex.AddIndex(idx)
	}
	if err = d.indexSchema(source, root, uri, "", nil, positiveContext); err != nil {
		return nil, err
	}
	if err := validateSource(root); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if err = d.register(uri.String(), root, source); err != nil {
		return nil, err
	}
	return d.schemas[root], nil
}

func member(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func (d *Document) lookup(address string) *yaml.Node {
	if resource := d.Rolodex.LookupSchemaById(address); resource != nil {
		return resource.SchemaNode
	}
	return d.addresses[address]
}

func (d *Document) register(address string, node *yaml.Node, source *Source) error {
	if prior := d.lookup(address); prior != nil {
		if prior == node {
			return nil
		}
		old := d.schemas[prior]
		uri, _ := url.Parse(address)
		return fmt.Errorf("duplicate schema address %q at %s:%d and %s:%d", d.sourceName(uri), old.Source.Name, prior.Line, source.Name, node.Line)
	}
	d.addresses[address] = node
	return nil
}

func (d *Document) indexSchema(source *Source, node *yaml.Node, base *url.URL, pointer string, r *Resource, evaluation evaluationContext) error {
	if node.Kind != yaml.MappingNode && !(node.Kind == yaml.ScalarNode && node.Tag == "!!bool") {
		return fmt.Errorf("%s:%d: schema must be an object or boolean", source.Name, node.Line)
	}
	if dialect := member(node, "$schema"); dialect != nil && strings.TrimSuffix(dialect.Value, "#") != Dialect {
		return fmt.Errorf("%s:%d: unsupported JSON Schema dialect %q", source.Name, dialect.Line, dialect.Value)
	}
	if vocab := member(node, "$vocabulary"); vocab != nil {
		if vocab.Kind != yaml.MappingNode {
			return fmt.Errorf("%s:%d: $vocabulary must be an object", source.Name, vocab.Line)
		}
		for i := 0; i < len(vocab.Content); i += 2 {
			key, required := vocab.Content[i], vocab.Content[i+1]
			if required.Tag != "!!bool" {
				return fmt.Errorf("%s:%d: vocabulary requirement must be boolean", source.Name, required.Line)
			}
			if !strings.EqualFold(required.Value, "true") {
				continue
			}
			switch strings.TrimPrefix(key.Value, "https://json-schema.org/draft/2020-12/vocab/") {
			case "core", "applicator", "unevaluated", "validation", "meta-data", "format-annotation", "content":
				if strings.HasPrefix(key.Value, "https://json-schema.org/draft/2020-12/vocab/") {
					continue
				}
			}
			return fmt.Errorf("%s:%d: unsupported required vocabulary %q", source.Name, key.Line, key.Value)
		}
	}
	id := member(node, "$id")
	if id != nil {
		resolved, err := base.Parse(id.Value)
		if err != nil || id.Tag != "!!str" || resolved.Fragment != "" {
			return fmt.Errorf("%s:%d: invalid schema $id %q", source.Name, id.Line, id.Value)
		}
		base = resolved
	}
	if r == nil || id != nil {
		evaluation |= positiveContext
		identity := source.relative
		if id != nil {
			identity = d.resourceID(base)
		}
		r = &Resource{base: base, ID: identity, Pointer: pointer, Source: source, Node: node}
		d.Resources = append(d.Resources, r)
		if err := d.register(base.String(), node, source); err != nil {
			return err
		}
		if id != nil {
			entry := &index.SchemaIdEntry{Id: id.Value, ResolvedUri: base.String(), SchemaNode: node, Index: source.Index, DefinitionPath: "#" + pointer, Line: id.Line, Column: id.Column}
			if err := source.Index.RegisterSchemaId(entry); err != nil {
				return err
			}
			if err := d.Rolodex.RegisterGlobalSchemaId(entry); err != nil {
				return err
			}
			delete(d.addresses, base.String())
		}
	}
	d.schemas[node] = r
	d.schemaPointers[node] = pointer[len(r.Pointer):]
	d.hasAdverseContext = d.hasAdverseContext || evaluation != positiveContext
	for _, key := range []string{"$anchor", "$dynamicAnchor"} {
		if value := member(node, key); value != nil {
			if value.Tag != "!!str" || !anchorName.MatchString(value.Value) {
				return fmt.Errorf("%s:%d: invalid %s %q", source.Name, value.Line, key, value.Value)
			}
			address := *base
			address.Fragment = value.Value
			if err := d.register(address.String(), node, source); err != nil {
				return err
			}
		}
	}
	for _, keyword := range []string{"$ref", "$dynamicRef"} {
		if value := member(node, keyword); value != nil {
			target, err := base.Parse(value.Value)
			if value.Tag != "!!str" || err != nil {
				return fmt.Errorf("%s:%d: invalid %s %q", source.Name, value.Line, keyword, value.Value)
			}
			d.References = append(d.References, Reference{From: node, Value: value.Value, address: target.String(), Keyword: keyword, context: evaluation})
		}
	}
	return children(node, func(keyword, key string, child *yaml.Node) error {
		childPointer := pointer + "/" + escapePointer(keyword)
		if childKind(keyword) != singleSchema {
			childPointer += "/" + escapePointer(key)
		}
		return d.indexSchema(source, child, base, childPointer, r, evaluation.child(node, keyword))
	})
}

func escapePointer(key string) string {
	return strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
}

const (
	singleSchema = 1
	schemaMap    = 2
	schemaList   = 3
)

func childKind(keyword string) int {
	switch keyword {
	case "$defs", "properties", "patternProperties", "dependentSchemas":
		return schemaMap
	case "prefixItems", "allOf", "anyOf", "oneOf":
		return schemaList
	case "additionalProperties", "unevaluatedProperties", "propertyNames", "contains", "items", "unevaluatedItems", "not", "if", "then", "else", "contentSchema":
		return singleSchema
	}
	return 0
}

func children(node *yaml.Node, visit func(keyword, key string, child *yaml.Node) error) error {
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyword, value := node.Content[i].Value, node.Content[i+1]
		switch childKind(keyword) {
		case singleSchema:
			if err := visit(keyword, "", value); err != nil {
				return err
			}
		case schemaMap:
			if value.Kind != yaml.MappingNode {
				return fmt.Errorf("line %d: %s requires an object", value.Line, keyword)
			}
			for j := 0; j < len(value.Content); j += 2 {
				if value.Content[j].Tag != "!!str" {
					return fmt.Errorf("line %d: schema keys must be strings", value.Content[j].Line)
				}
				if err := visit(keyword, value.Content[j].Value, value.Content[j+1]); err != nil {
					return err
				}
			}
		case schemaList:
			if value.Kind != yaml.SequenceNode {
				return fmt.Errorf("line %d: %s requires an array", value.Line, keyword)
			}
			for j, item := range value.Content {
				if err := visit(keyword, strconv.Itoa(j), item); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
