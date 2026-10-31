package model

import (
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low/base"
)

// A comparison owns its visited pairs. No state is shared with other callers or
// union candidate scoring. Properties and compositions visit pairs in stable order.
type schemaComparison struct {
	visited map[schemaComparisonPair]struct{}
}

type schemaComparisonPair struct {
	left, right *yaml.Node
}

func (c *schemaComparison) visit(left, right *base.SchemaProxy) bool {
	pair := schemaComparisonPair{schemaDefinitionNode(left), schemaDefinitionNode(right)}
	if _, exists := c.visited[pair]; exists {
		return false
	}
	if c.visited == nil {
		c.visited = make(map[schemaComparisonPair]struct{})
	}
	c.visited[pair] = struct{}{}
	return true
}
