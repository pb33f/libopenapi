// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

type Kind int

const (
	KindUnknown Kind = iota
	KindAny
	KindObject
	KindArray
	KindMap
	KindString
	KindInteger
	KindNumber
	KindBoolean
	KindRef
	KindEnum
	KindAllOf
	KindUnion
)

type UnionKind int

const (
	UnionNone UnionKind = iota
	UnionOneOf
	UnionAnyOf
)

type UnionStrategy int

const (
	UnionRawMessage UnionStrategy = iota
	UnionDiscriminator
)

type Discriminator struct {
	PropertyName string
	Mapping      map[string]string
	Optional     bool
}

type Source struct {
	Line   int
	Column int
	Ref    string
}

type SchemaIR struct {
	Name                 string
	Ref                  string
	Kind                 Kind
	Format               string
	Description          string
	Title                string
	Nullable             bool
	Required             map[string]struct{}
	Properties           *orderedmap.Map[string, *SchemaIR]
	PatternProperties    *orderedmap.Map[string, *SchemaIR]
	Items                *SchemaIR
	PrefixItems          []*SchemaIR
	AdditionalProperties *SchemaIR
	AdditionalAllowed    *bool
	Enum                 []*yaml.Node
	Const                *yaml.Node
	AllOf                []*SchemaIR
	Union                *UnionIR
	Extensions           *orderedmap.Map[string, *yaml.Node]
	Source               *Source
	ReadOnly             bool
	WriteOnly            bool
	Deprecated           bool
	Comments             []string
}

type UnionIR struct {
	Kind          UnionKind
	Variants      []*SchemaIR
	Discriminator *Discriminator
	Strategy      UnionStrategy
}

func newObjectIR(name string) *SchemaIR {
	return &SchemaIR{
		Name:       name,
		Kind:       KindObject,
		Required:   make(map[string]struct{}),
		Properties: orderedmap.New[string, *SchemaIR](),
	}
}

func isRequired(ir *SchemaIR, name string) bool {
	if ir == nil || ir.Required == nil {
		return false
	}
	_, ok := ir.Required[name]
	return ok
}
