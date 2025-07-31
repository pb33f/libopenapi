// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAllChangesModels_NilChecks tests that all *Changes models handle nil properly
// This comprehensive test ensures that nil checks added to prevent panics have full coverage
func TestAllChangesModels_NilChecks(t *testing.T) {
	testCases := []struct {
		name string
		test func(t *testing.T)
	}{
		// Test TotalChanges() nil checks
		{"ComponentsChanges_TotalChanges_Nil", func(t *testing.T) {
			var c *ComponentsChanges
			assert.Equal(t, 0, c.TotalChanges())
		}},
		{"ParameterChanges_TotalChanges_Nil", func(t *testing.T) {
			var p *ParameterChanges
			assert.Equal(t, 0, p.TotalChanges())
		}},
		{"InfoChanges_TotalChanges_Nil", func(t *testing.T) {
			var i *InfoChanges
			assert.Equal(t, 0, i.TotalChanges())
		}},
		{"MediaTypeChanges_TotalChanges_Nil", func(t *testing.T) {
			var m *MediaTypeChanges
			assert.Equal(t, 0, m.TotalChanges())
		}},
		{"ItemsChanges_TotalChanges_Nil", func(t *testing.T) {
			var i *ItemsChanges
			assert.Equal(t, 0, i.TotalChanges())
		}},
		{"DiscriminatorChanges_TotalChanges_Nil", func(t *testing.T) {
			var d *DiscriminatorChanges
			assert.Equal(t, 0, d.TotalChanges())
		}},
		{"ContactChanges_TotalChanges_Nil", func(t *testing.T) {
			var c *ContactChanges
			assert.Equal(t, 0, c.TotalChanges())
		}},
		{"SchemaChanges_TotalChanges_Nil", func(t *testing.T) {
			var s *SchemaChanges
			assert.Equal(t, 0, s.TotalChanges())
		}},
		{"PathItemChanges_TotalChanges_Nil", func(t *testing.T) {
			var p *PathItemChanges
			assert.Equal(t, 0, p.TotalChanges())
		}},
		{"ExtensionChanges_TotalChanges_Nil", func(t *testing.T) {
			var e *ExtensionChanges
			assert.Equal(t, 0, e.TotalChanges())
		}},
		{"ExternalDocChanges_TotalChanges_Nil", func(t *testing.T) {
			var e *ExternalDocChanges
			assert.Equal(t, 0, e.TotalChanges())
		}},
		{"ExampleChanges_TotalChanges_Nil", func(t *testing.T) {
			var e *ExampleChanges
			assert.Equal(t, 0, e.TotalChanges())
		}},
		{"ExamplesChanges_TotalChanges_Nil", func(t *testing.T) {
			var a *ExamplesChanges
			assert.Equal(t, 0, a.TotalChanges())
		}},
		{"DocumentChanges_TotalChanges_Nil", func(t *testing.T) {
			var d *DocumentChanges
			assert.Equal(t, 0, d.TotalChanges())
		}},
		{"LicenseChanges_TotalChanges_Nil", func(t *testing.T) {
			var l *LicenseChanges
			assert.Equal(t, 0, l.TotalChanges())
		}},
		{"XMLChanges_TotalChanges_Nil", func(t *testing.T) {
			var x *XMLChanges
			assert.Equal(t, 0, x.TotalChanges())
		}},
		{"ResponseChanges_TotalChanges_Nil", func(t *testing.T) {
			var r *ResponseChanges
			assert.Equal(t, 0, r.TotalChanges())
		}},
		{"OperationChanges_TotalChanges_Nil", func(t *testing.T) {
			var o *OperationChanges
			assert.Equal(t, 0, o.TotalChanges())
		}},
		{"LinkChanges_TotalChanges_Nil", func(t *testing.T) {
			var l *LinkChanges
			assert.Equal(t, 0, l.TotalChanges())
		}},
		{"ScopesChanges_TotalChanges_Nil", func(t *testing.T) {
			var s *ScopesChanges
			assert.Equal(t, 0, s.TotalChanges())
		}},
		{"CallbackChanges_TotalChanges_Nil", func(t *testing.T) {
			var c *CallbackChanges
			assert.Equal(t, 0, c.TotalChanges())
		}},
		{"EncodingChanges_TotalChanges_Nil", func(t *testing.T) {
			var e *EncodingChanges
			assert.Equal(t, 0, e.TotalChanges())
		}},
		{"TagChanges_TotalChanges_Nil", func(t *testing.T) {
			var tc *TagChanges
			assert.Equal(t, 0, tc.TotalChanges())
		}},
		{"ResponsesChanges_TotalChanges_Nil", func(t *testing.T) {
			var r *ResponsesChanges
			assert.Equal(t, 0, r.TotalChanges())
		}},
		{"PathsChanges_TotalChanges_Nil", func(t *testing.T) {
			var p *PathsChanges
			assert.Equal(t, 0, p.TotalChanges())
		}},
		{"HeaderChanges_TotalChanges_Nil", func(t *testing.T) {
			var h *HeaderChanges
			assert.Equal(t, 0, h.TotalChanges())
		}},
		{"OAuthFlowsChanges_TotalChanges_Nil", func(t *testing.T) {
			var o *OAuthFlowsChanges
			assert.Equal(t, 0, o.TotalChanges())
		}},
		{"OAuthFlowChanges_TotalChanges_Nil", func(t *testing.T) {
			var o *OAuthFlowChanges
			assert.Equal(t, 0, o.TotalChanges())
		}},
		{"ServerChanges_TotalChanges_Nil", func(t *testing.T) {
			var s *ServerChanges
			assert.Equal(t, 0, s.TotalChanges())
		}},
		{"SecurityRequirementChanges_TotalChanges_Nil", func(t *testing.T) {
			var s *SecurityRequirementChanges
			assert.Equal(t, 0, s.TotalChanges())
		}},
		{"SecuritySchemeChanges_TotalChanges_Nil", func(t *testing.T) {
			var ss *SecuritySchemeChanges
			assert.Equal(t, 0, ss.TotalChanges())
		}},
		{"PropertyChanges_TotalChanges_Nil", func(t *testing.T) {
			var p *PropertyChanges
			assert.Equal(t, 0, p.TotalChanges())
		}},
		{"RequestBodyChanges_TotalChanges_Nil", func(t *testing.T) {
			var rb *RequestBodyChanges
			assert.Equal(t, 0, rb.TotalChanges())
		}},

		// Test TotalBreakingChanges() nil checks
		{"SchemaChanges_TotalBreakingChanges_Nil", func(t *testing.T) {
			var s *SchemaChanges
			assert.Equal(t, 0, s.TotalBreakingChanges())
		}},
		{"DocumentChanges_TotalBreakingChanges_Nil", func(t *testing.T) {
			var d *DocumentChanges
			assert.Equal(t, 0, d.TotalBreakingChanges())
		}},

		// Test GetAllChanges() nil checks
		{"ComponentsChanges_GetAllChanges_Nil", func(t *testing.T) {
			var c *ComponentsChanges
			assert.Nil(t, c.GetAllChanges())
		}},
		{"ServerChanges_GetAllChanges_Nil", func(t *testing.T) {
			var s *ServerChanges
			assert.Nil(t, s.GetAllChanges())
		}},
		{"SecurityRequirementChanges_GetAllChanges_Nil", func(t *testing.T) {
			var s *SecurityRequirementChanges
			assert.Nil(t, s.GetAllChanges())
		}},
		{"LinkChanges_GetAllChanges_Nil", func(t *testing.T) {
			var l *LinkChanges
			assert.Nil(t, l.GetAllChanges())
		}},
		{"ServerVariableChanges_GetAllChanges_Nil", func(t *testing.T) {
			var s *ServerVariableChanges
			assert.Nil(t, s.GetAllChanges())
		}},
		{"ParameterChanges_GetAllChanges_Nil", func(t *testing.T) {
			var p *ParameterChanges
			assert.Nil(t, p.GetAllChanges())
		}},
		{"SecuritySchemeChanges_GetAllChanges_Nil", func(t *testing.T) {
			var ss *SecuritySchemeChanges
			assert.Nil(t, ss.GetAllChanges())
		}},
		{"ScopesChanges_GetAllChanges_Nil", func(t *testing.T) {
			var s *ScopesChanges
			assert.Nil(t, s.GetAllChanges())
		}},
		{"InfoChanges_GetAllChanges_Nil", func(t *testing.T) {
			var i *InfoChanges
			assert.Nil(t, i.GetAllChanges())
		}},
		{"OAuthFlowsChanges_GetAllChanges_Nil", func(t *testing.T) {
			var o *OAuthFlowsChanges
			assert.Nil(t, o.GetAllChanges())
		}},
		{"OAuthFlowChanges_GetAllChanges_Nil", func(t *testing.T) {
			var o *OAuthFlowChanges
			assert.Nil(t, o.GetAllChanges())
		}},
		{"MediaTypeChanges_GetAllChanges_Nil", func(t *testing.T) {
			var m *MediaTypeChanges
			assert.Nil(t, m.GetAllChanges())
		}},
		{"CallbackChanges_GetAllChanges_Nil", func(t *testing.T) {
			var c *CallbackChanges
			assert.Nil(t, c.GetAllChanges())
		}},
		{"ResponsesChanges_GetAllChanges_Nil", func(t *testing.T) {
			var r *ResponsesChanges
			assert.Nil(t, r.GetAllChanges())
		}},
		{"ItemsChanges_GetAllChanges_Nil", func(t *testing.T) {
			var i *ItemsChanges
			assert.Nil(t, i.GetAllChanges())
		}},
		{"TagChanges_GetAllChanges_Nil", func(t *testing.T) {
			var tc *TagChanges
			assert.Nil(t, tc.GetAllChanges())
		}},
		{"OperationChanges_GetAllChanges_Nil", func(t *testing.T) {
			var o *OperationChanges
			assert.Nil(t, o.GetAllChanges())
		}},
		{"EncodingChanges_GetAllChanges_Nil", func(t *testing.T) {
			var e *EncodingChanges
			assert.Nil(t, e.GetAllChanges())
		}},
		{"DiscriminatorChanges_GetAllChanges_Nil", func(t *testing.T) {
			var c *DiscriminatorChanges
			assert.Nil(t, c.GetAllChanges())
		}},
		{"ResponseChanges_GetAllChanges_Nil", func(t *testing.T) {
			var r *ResponseChanges
			assert.Nil(t, r.GetAllChanges())
		}},
		{"SchemaChanges_GetAllChanges_Nil", func(t *testing.T) {
			var s *SchemaChanges
			assert.Nil(t, s.GetAllChanges())
		}},
		{"ExampleChanges_GetAllChanges_Nil", func(t *testing.T) {
			var e *ExampleChanges
			assert.Nil(t, e.GetAllChanges())
		}},
		{"PathsChanges_GetAllChanges_Nil", func(t *testing.T) {
			var p *PathsChanges
			assert.Nil(t, p.GetAllChanges())
		}},
		{"ContactChanges_GetAllChanges_Nil", func(t *testing.T) {
			var c *ContactChanges
			assert.Nil(t, c.GetAllChanges())
		}},
		{"DocumentChanges_GetAllChanges_Nil", func(t *testing.T) {
			var d *DocumentChanges
			assert.Nil(t, d.GetAllChanges())
		}},
		{"RequestBodyChanges_GetAllChanges_Nil", func(t *testing.T) {
			var rb *RequestBodyChanges
			assert.Nil(t, rb.GetAllChanges())
		}},
		{"LicenseChanges_GetAllChanges_Nil", func(t *testing.T) {
			var l *LicenseChanges
			assert.Nil(t, l.GetAllChanges())
		}},
		{"ExternalDocChanges_GetAllChanges_Nil", func(t *testing.T) {
			var e *ExternalDocChanges
			assert.Nil(t, e.GetAllChanges())
		}},
		{"XMLChanges_GetAllChanges_Nil", func(t *testing.T) {
			var x *XMLChanges
			assert.Nil(t, x.GetAllChanges())
		}},
		{"ExamplesChanges_GetAllChanges_Nil", func(t *testing.T) {
			var a *ExamplesChanges
			assert.Nil(t, a.GetAllChanges())
		}},
		{"ExtensionChanges_GetAllChanges_Nil", func(t *testing.T) {
			var e *ExtensionChanges
			assert.Nil(t, e.GetAllChanges())
		}},
		{"PathItemChanges_GetAllChanges_Nil", func(t *testing.T) {
			var p *PathItemChanges
			assert.Nil(t, p.GetAllChanges())
		}},
		{"HeaderChanges_GetAllChanges_Nil", func(t *testing.T) {
			var h *HeaderChanges
			assert.Nil(t, h.GetAllChanges())
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}