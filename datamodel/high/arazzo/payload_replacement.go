// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// PayloadReplacement represents a high-level Arazzo Payload Replacement Object.
// https://spec.openapis.org/arazzo/v1.1.0#payload-replacement-object
type PayloadReplacement struct {
	Target                       string                              `json:"target,omitempty" yaml:"target,omitempty"`
	TargetSelectorType           string                              `json:"-" yaml:"-"`
	TargetSelectorExpressionType *ExpressionType                     `json:"-" yaml:"-"`
	Value                        *yaml.Node                          `json:"value,omitempty" yaml:"value,omitempty"`
	Extensions                   *orderedmap.Map[string, *yaml.Node] `json:"-" yaml:"-"`
	low                          *low.PayloadReplacement
}

// NewPayloadReplacement creates a new high-level PayloadReplacement instance from a low-level one.
func NewPayloadReplacement(pr *low.PayloadReplacement) *PayloadReplacement {
	p := new(PayloadReplacement)
	p.low = pr
	if !pr.Target.IsEmpty() {
		p.Target = pr.Target.Value
	}
	if !pr.TargetSelectorType.IsEmpty() && pr.TargetSelectorType.Value != nil {
		node := dereferenceAliasNode(pr.TargetSelectorType.Value)
		if node != nil {
			switch node.Kind {
			case yaml.ScalarNode:
				p.TargetSelectorType = node.Value
			case yaml.MappingNode:
				expressionType := new(low.ExpressionType)
				if err := lowmodel.BuildModel(node, expressionType); err == nil {
					if err = expressionType.Build(context.Background(), nil, node, nil); err == nil {
						p.TargetSelectorExpressionType = NewExpressionType(expressionType)
					}
				}
			}
		}
	}
	if !pr.Value.IsEmpty() {
		p.Value = pr.Value.Value
	}
	p.Extensions = high.ExtractExtensions(pr.Extensions)
	return p
}

// IsRuntimeExpression reports whether the replacement value is a runtime-expression scalar.
func (p *PayloadReplacement) IsRuntimeExpression() bool {
	if p == nil || p.Value == nil || p.Value.Kind != yaml.ScalarNode {
		return false
	}
	return strings.HasPrefix(p.Value.Value, "$") || strings.Contains(p.Value.Value, "{$")
}

// IsSelector reports whether the replacement value is a Selector Object.
func (p *PayloadReplacement) IsSelector() bool {
	return p != nil && selectorFromNode(p.Value) != nil
}

// GetSelector returns the typed Selector Object when the value has selector shape.
func (p *PayloadReplacement) GetSelector() (*Selector, bool) {
	if p == nil {
		return nil, false
	}
	selector := selectorFromNode(p.Value)
	return selector, selector != nil
}

// GetSelectors returns every Selector Object nested in the replacement value.
func (p *PayloadReplacement) GetSelectors() []*Selector {
	if p == nil {
		return nil
	}
	return selectorsFromNode(p.Value)
}

// GetValueNode returns the original replacement value node.
func (p *PayloadReplacement) GetValueNode() *yaml.Node {
	if p == nil {
		return nil
	}
	return p.Value
}

// GoLow returns the low-level PayloadReplacement instance used to create the high-level one.
func (p *PayloadReplacement) GoLow() *low.PayloadReplacement {
	return p.low
}

// GoLowUntyped returns the low-level PayloadReplacement instance with no type.
func (p *PayloadReplacement) GoLowUntyped() any {
	return p.low
}

// Render returns a YAML representation of the PayloadReplacement object as a byte slice.
func (p *PayloadReplacement) Render() ([]byte, error) {
	return yaml.Marshal(p)
}

// MarshalYAML creates a ready to render YAML representation of the PayloadReplacement object.
func (p *PayloadReplacement) MarshalYAML() (any, error) {
	m := orderedmap.New[string, any]()
	if p.Target != "" {
		m.Set(low.TargetLabel, p.Target)
	}
	if p.TargetSelectorExpressionType != nil {
		if p.TargetSelectorType != "" {
			return nil, fmt.Errorf("payload replacement cannot contain both target selector type variants")
		}
		m.Set(low.TargetSelectorTypeLabel, p.TargetSelectorExpressionType)
	} else if p.TargetSelectorType != "" {
		m.Set(low.TargetSelectorTypeLabel, p.TargetSelectorType)
	}
	if p.Value != nil {
		m.Set(low.ValueLabel, p.Value)
	}
	marshalExtensions(m, p.Extensions)
	return m, nil
}
