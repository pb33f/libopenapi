// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"encoding/json"
	"fmt"

	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"go.yaml.in/yaml/v4"
)

type outputValueVariant uint8

const (
	outputValueUnset outputValueVariant = iota
	outputValueExpression
	outputValueSelector
)

// OutputValue represents a runtime-expression scalar or Selector Object.
// Exactly one variant must be active when the value is rendered.
type OutputValue struct {
	expression string
	selector   *Selector
	low        *low.OutputValue
	variant    outputValueVariant
}

// NewOutputValue creates a high-level output union from a low-level value.
func NewOutputValue(value *low.OutputValue) *OutputValue {
	o := &OutputValue{low: value}
	if value.IsExpression() {
		o.expression = value.Expression.Value
		o.variant = outputValueExpression
	}
	if value.IsSelector() {
		o.selector = NewSelector(value.Selector.Value)
		o.variant = outputValueSelector
	}
	return o
}

// NewExpressionOutputValue creates a scalar runtime-expression output.
func NewExpressionOutputValue(expression string) *OutputValue {
	return &OutputValue{expression: expression, variant: outputValueExpression}
}

// NewSelectorOutputValue creates a Selector Object output.
func NewSelectorOutputValue(selector *Selector) *OutputValue {
	if selector == nil {
		return &OutputValue{}
	}
	return &OutputValue{selector: selector, variant: outputValueSelector}
}

// SetExpression switches the output to the runtime-expression variant.
func (o *OutputValue) SetExpression(expression string) {
	if o == nil {
		return
	}
	o.expression = expression
	o.selector = nil
	o.variant = outputValueExpression
}

// SetSelector switches the output to the Selector Object variant. A nil selector
// clears the output so it cannot accidentally render as a valid union member.
func (o *OutputValue) SetSelector(selector *Selector) {
	if o == nil {
		return
	}
	o.expression = ""
	o.selector = selector
	if selector == nil {
		o.variant = outputValueUnset
		return
	}
	o.variant = outputValueSelector
}

// IsExpression reports whether exactly the expression variant is active.
func (o *OutputValue) IsExpression() bool {
	return o != nil &&
		o.selector == nil &&
		o.variant == outputValueExpression
}

// IsSelector reports whether exactly the selector variant is active.
func (o *OutputValue) IsSelector() bool {
	return o != nil &&
		o.selector != nil &&
		o.variant == outputValueSelector
}

// GetExpression returns the expression and whether it is the active variant.
func (o *OutputValue) GetExpression() (string, bool) {
	if !o.IsExpression() {
		return "", false
	}
	return o.expression, true
}

// GetSelector returns the Selector and whether it is the active variant.
func (o *OutputValue) GetSelector() (*Selector, bool) {
	if !o.IsSelector() {
		return nil, false
	}
	return o.selector, true
}

// GoLow returns the low-level output value used to create this model.
func (o *OutputValue) GoLow() *low.OutputValue {
	return o.low
}

// GoLowUntyped returns the low-level output value without a concrete type.
func (o *OutputValue) GoLowUntyped() any {
	return o.low
}

// Render returns the active output variant as YAML.
func (o *OutputValue) Render() ([]byte, error) {
	return yaml.Marshal(o)
}

// MarshalYAML renders the active union variant.
func (o *OutputValue) MarshalYAML() (any, error) {
	switch {
	case o.IsExpression():
		return o.expression, nil
	case o.IsSelector():
		return o.selector, nil
	default:
		return nil, fmt.Errorf("output value must contain exactly one expression or selector variant")
	}
}

// MarshalJSON renders the active output union variant.
func (o *OutputValue) MarshalJSON() ([]byte, error) {
	value, err := o.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}
