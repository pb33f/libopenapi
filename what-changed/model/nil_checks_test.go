// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
)

// TestAllChangesModels_NilChecks tests that all *Changes models handle nil properly
// This comprehensive test ensures that nil checks added to prevent panics have full coverage
func TestAllChangesModels_NilChecks(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
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

// TestComparisonFunctions_NilReturnPatterns tests the TotalChanges() <= 0 return nil pattern
// This ensures coverage for the optimization that returns nil when there are no changes
func TestComparisonFunctions_NilReturnPatterns(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	// Test all comparison functions that have TotalChanges() <= 0 nil return patterns
	
	// Create identical objects for comparison (should result in no changes)
	
	t.Run("Components_NoChanges_ReturnsNil", func(t *testing.T) {
		components := &v3.Components{}
		result := CompareComponents(components, components)
		assert.Nil(t, result, "CompareComponents should return nil when there are no changes")
	})
	
	t.Run("Header_NoChanges_ReturnsNil", func(t *testing.T) {
		header := &v3.Header{}
		result := CompareHeaders(header, header)
		assert.Nil(t, result, "CompareHeaders should return nil when there are no changes")
	})
	
	t.Run("Paths_NoChanges_ReturnsNil", func(t *testing.T) {
		paths := &v3.Paths{}
		result := ComparePaths(paths, paths)
		assert.Nil(t, result, "ComparePaths should return nil when there are no changes")
	})
	
	t.Run("OAuthFlows_NoChanges_ReturnsNil", func(t *testing.T) {
		flows := &v3.OAuthFlows{}
		result := CompareOAuthFlows(flows, flows)
		assert.Nil(t, result, "CompareOAuthFlows should return nil when there are no changes")
	})
	
	t.Run("OAuthFlow_NoChanges_ReturnsNil", func(t *testing.T) {
		flow := &v3.OAuthFlow{}
		result := CompareOAuthFlow(flow, flow)
		assert.Nil(t, result, "CompareOAuthFlow should return nil when there are no changes")
	})
	
	t.Run("RequestBody_NoChanges_ReturnsNil", func(t *testing.T) {
		requestBody := &v3.RequestBody{}
		result := CompareRequestBodies(requestBody, requestBody)
		assert.Nil(t, result, "CompareRequestBodies should return nil when there are no changes")
	})
	
	t.Run("XML_NoChanges_ReturnsNil", func(t *testing.T) {
		xml := &base.XML{}
		result := CompareXML(xml, xml)
		assert.Nil(t, result, "CompareXML should return nil when there are no changes")
	})
	
	t.Run("ServerVariable_NoChanges_ReturnsNil", func(t *testing.T) {
		serverVar := &v3.ServerVariable{}
		result := CompareServerVariables(serverVar, serverVar)
		assert.Nil(t, result, "CompareServerVariables should return nil when there are no changes")
	})
	
	t.Run("Responses_NoChanges_ReturnsNil", func(t *testing.T) {
		responses := &v3.Responses{}
		result := CompareResponses(responses, responses)
		assert.Nil(t, result, "CompareResponses should return nil when there are no changes")
	})
	
	t.Run("Items_NoChanges_ReturnsNil", func(t *testing.T) {
		items := &v2.Items{}
		result := CompareItems(items, items)
		assert.Nil(t, result, "CompareItems should return nil when there are no changes")
	})
	
	t.Run("Response_NoChanges_ReturnsNil", func(t *testing.T) {
		response := &v3.Response{}
		result := CompareResponseV3(response, response)
		assert.Nil(t, result, "CompareResponseV3 should return nil when there are no changes")
	})
	
	t.Run("Info_NoChanges_ReturnsNil", func(t *testing.T) {
		info := &base.Info{}
		result := CompareInfo(info, info)
		assert.Nil(t, result, "CompareInfo should return nil when there are no changes")
	})
	
	t.Run("Server_NoChanges_ReturnsNil", func(t *testing.T) {
		server := &v3.Server{}
		result := CompareServers(server, server)
		assert.Nil(t, result, "CompareServers should return nil when there are no changes")
	})
	
	t.Run("Discriminator_NoChanges_ReturnsNil", func(t *testing.T) {
		discriminator := &base.Discriminator{}
		result := CompareDiscriminator(discriminator, discriminator)
		assert.Nil(t, result, "CompareDiscriminator should return nil when there are no changes")
	})
	
	t.Run("Extensions_NoChanges_ReturnsNil", func(t *testing.T) {
		result := CompareExtensions(nil, nil)
		assert.Nil(t, result, "CompareExtensions should return nil when there are no changes")
	})
	
	t.Run("SecurityScheme_NoChanges_ReturnsNil", func(t *testing.T) {
		securityScheme := &v3.SecurityScheme{}
		result := CompareSecuritySchemes(securityScheme, securityScheme)
		assert.Nil(t, result, "CompareSecuritySchemes should return nil when there are no changes")
	})
	
	t.Run("Contact_NoChanges_ReturnsNil", func(t *testing.T) {
		contact := &base.Contact{}
		result := CompareContact(contact, contact)
		assert.Nil(t, result, "CompareContact should return nil when there are no changes")
	})
	
	t.Run("Encoding_NoChanges_ReturnsNil", func(t *testing.T) {
		encoding := &v3.Encoding{}
		result := CompareEncoding(encoding, encoding)
		assert.Nil(t, result, "CompareEncoding should return nil when there are no changes")
	})
	
	t.Run("ExternalDocs_NoChanges_ReturnsNil", func(t *testing.T) {
		externalDocs := &base.ExternalDoc{}
		result := CompareExternalDocs(externalDocs, externalDocs)
		assert.Nil(t, result, "CompareExternalDocs should return nil when there are no changes")
	})
	
	t.Run("MediaType_NoChanges_ReturnsNil", func(t *testing.T) {
		mediaType := &v3.MediaType{}
		result := CompareMediaTypes(mediaType, mediaType)
		assert.Nil(t, result, "CompareMediaTypes should return nil when there are no changes")
	})
	
	t.Run("Parameter_NoChanges_ReturnsNil", func(t *testing.T) {
		parameter := &v3.Parameter{}
		result := CompareParametersV3(parameter, parameter)
		assert.Nil(t, result, "CompareParametersV3 should return nil when there are no changes")
	})
	
	t.Run("SecurityRequirement_NoChanges_ReturnsNil", func(t *testing.T) {
		securityReq := &base.SecurityRequirement{}
		result := CompareSecurityRequirement(securityReq, securityReq)
		assert.Nil(t, result, "CompareSecurityRequirement should return nil when there are no changes")
	})
	
	t.Run("Example_NoChanges_ReturnsNil", func(t *testing.T) {
		example := &base.Example{}
		result := CompareExamples(example, example)
		assert.Nil(t, result, "CompareExamples should return nil when there are no changes")
	})
	
	t.Run("Scopes_NoChanges_ReturnsNil", func(t *testing.T) {
		scopes := &v2.Scopes{}
		result := CompareScopes(scopes, scopes)
		assert.Nil(t, result, "CompareScopes should return nil when there are no changes")
	})
	
	t.Run("Operation_NoChanges_ReturnsNil", func(t *testing.T) {
		operation := &v3.Operation{}
		result := CompareOperations(operation, operation)
		assert.Nil(t, result, "CompareOperations should return nil when there are no changes")
	})
	
	t.Run("Examples_NoChanges_ReturnsNil", func(t *testing.T) {
		examples := &v2.Examples{}
		result := CompareExamplesV2(examples, examples)
		assert.Nil(t, result, "CompareExamplesV2 should return nil when there are no changes")
	})
	
	t.Run("Callback_NoChanges_ReturnsNil", func(t *testing.T) {
		callback := &v3.Callback{}
		result := CompareCallback(callback, callback)
		assert.Nil(t, result, "CompareCallback should return nil when there are no changes")
	})
	
	t.Run("Document_NoChanges_ReturnsNil", func(t *testing.T) {
		document := &v3.Document{}
		result := CompareDocuments(document, document)
		assert.Nil(t, result, "CompareDocuments should return nil when there are no changes")
	})
	
	t.Run("PathItem_NoChanges_ReturnsNil", func(t *testing.T) {
		pathItem := &v3.PathItem{}
		result := ComparePathItems(pathItem, pathItem)
		assert.Nil(t, result, "ComparePathItems should return nil when there are no changes")
	})
	
	t.Run("Link_NoChanges_ReturnsNil", func(t *testing.T) {
		link := &v3.Link{}
		result := CompareLinks(link, link)
		assert.Nil(t, result, "CompareLinks should return nil when there are no changes")
	})
	
	t.Run("License_NoChanges_ReturnsNil", func(t *testing.T) {
		license := &base.License{}
		result := CompareLicense(license, license)
		assert.Nil(t, result, "CompareLicense should return nil when there are no changes")
	})
}

// TestTotalChangesZeroReturnNil tests ALL instances of the TotalChanges() <= 0 pattern
// This ensures coverage for every single case found in the what-changed/model package
func TestTotalChangesZeroReturnNil(t *testing.T) {
	// Clear hash cache to ensure deterministic results in concurrent test environments
	low.ClearHashCache()
	// Test all 31 instances of the TotalChanges() <= 0 pattern found by grep
	
	// 1. components.go:203
	t.Run("ComponentsChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		components1 := &v3.Components{}
		components2 := &v3.Components{}
		result := CompareComponents(components1, components2)
		assert.Nil(t, result, "CompareComponents should return nil when TotalChanges() <= 0")
	})
	
	// 2. server_variable.go:84
	t.Run("ServerVariableChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		serverVar1 := &v3.ServerVariable{}
		serverVar2 := &v3.ServerVariable{}
		result := CompareServerVariables(serverVar1, serverVar2)
		assert.Nil(t, result, "CompareServerVariables should return nil when TotalChanges() <= 0")
	})
	
	// 3. header.go:285
	t.Run("HeaderChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		// Create v2 headers that will bypass equality check but have no changes
		header1 := &v2.Header{Description: low.NodeReference[string]{Value: "", ValueNode: nil}}
		header2 := &v2.Header{Description: low.NodeReference[string]{Value: "", ValueNode: nil}}
		result := CompareHeaders(header1, header2)
		assert.Nil(t, result, "CompareHeaders should return nil when TotalChanges() <= 0")
	})
	
	// 4. request_body.go:98
	t.Run("RequestBodyChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		requestBody1 := &v3.RequestBody{}
		requestBody2 := &v3.RequestBody{}
		result := CompareRequestBodies(requestBody1, requestBody2)
		assert.Nil(t, result, "CompareRequestBodies should return nil when TotalChanges() <= 0")
	})
	
	// 5. paths.go:228
	t.Run("PathsChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		paths1 := &v3.Paths{}
		paths2 := &v3.Paths{}
		result := ComparePaths(paths1, paths2)
		assert.Nil(t, result, "ComparePaths should return nil when TotalChanges() <= 0")
	})
	
	// 6. link.go:165
	t.Run("LinkChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		link1 := &v3.Link{}
		link2 := &v3.Link{}
		result := CompareLinks(link1, link2)
		assert.Nil(t, result, "CompareLinks should return nil when TotalChanges() <= 0")
	})
	
	// 7. xml.go:116
	t.Run("XMLChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		xml1 := &base.XML{}
		xml2 := &base.XML{}
		result := CompareXML(xml1, xml2)
		assert.Nil(t, result, "CompareXML should return nil when TotalChanges() <= 0")
	})
	
	// 8. oauth_flows.go:159 (CompareOAuthFlows)
	t.Run("OAuthFlowsChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		flows1 := &v3.OAuthFlows{}
		flows2 := &v3.OAuthFlows{}
		result := CompareOAuthFlows(flows1, flows2)
		assert.Nil(t, result, "CompareOAuthFlows should return nil when TotalChanges() <= 0")
	})
	
	// 9. oauth_flows.go:267 (CompareOAuthFlow)
	t.Run("OAuthFlowChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		flow1 := &v3.OAuthFlow{}
		flow2 := &v3.OAuthFlow{}
		result := CompareOAuthFlow(flow1, flow2)
		assert.Nil(t, result, "CompareOAuthFlow should return nil when TotalChanges() <= 0")
	})
	
	// 10. extensions.go:87
	t.Run("ExtensionChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		result := CompareExtensions(nil, nil)
		assert.Nil(t, result, "CompareExtensions should return nil when TotalChanges() <= 0")
	})
	
	// 11. scopes.go:81
	t.Run("ScopesChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		scopes1 := &v2.Scopes{}
		scopes2 := &v2.Scopes{}
		result := CompareScopes(scopes1, scopes2)
		assert.Nil(t, result, "CompareScopes should return nil when TotalChanges() <= 0")
	})
	
	// 12. response.go:200
	t.Run("ResponseChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		response1 := &v3.Response{}
		response2 := &v3.Response{}
		result := CompareResponseV3(response1, response2)
		assert.Nil(t, result, "CompareResponseV3 should return nil when TotalChanges() <= 0")
	})
	
	// 13. document.go:295
	t.Run("DocumentChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		document1 := &v3.Document{}
		document2 := &v3.Document{}
		result := CompareDocuments(document1, document2)
		assert.Nil(t, result, "CompareDocuments should return nil when TotalChanges() <= 0")
	})
	
	// 14. example.go:212
	t.Run("ExampleChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		example1 := &base.Example{}
		example2 := &base.Example{}
		result := CompareExamples(example1, example2)
		assert.Nil(t, result, "CompareExamples should return nil when TotalChanges() <= 0")
	})
	
	// 15. items.go:88
	t.Run("ItemsChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		items1 := &v2.Items{}
		items2 := &v2.Items{}
		result := CompareItems(items1, items2)
		assert.Nil(t, result, "CompareItems should return nil when TotalChanges() <= 0")
	})
	
	// 16. callback.go:116
	t.Run("CallbackChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		callback1 := &v3.Callback{}
		callback2 := &v3.Callback{}
		result := CompareCallback(callback1, callback2)
		assert.Nil(t, result, "CompareCallback should return nil when TotalChanges() <= 0")
	})
	
	// 17. license.go:94
	t.Run("LicenseChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		license1 := &base.License{}
		license2 := &base.License{}
		result := CompareLicense(license1, license2)
		assert.Nil(t, result, "CompareLicense should return nil when TotalChanges() <= 0")
	})
	
	// 18. server.go:97
	t.Run("ServerChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		server1 := &v3.Server{}
		server2 := &v3.Server{}
		result := CompareServers(server1, server2)
		assert.Nil(t, result, "CompareServers should return nil when TotalChanges() <= 0")
	})
	
	// 19. encoding.go:100
	t.Run("EncodingChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		encoding1 := &v3.Encoding{}
		encoding2 := &v3.Encoding{}
		result := CompareEncoding(encoding1, encoding2)
		assert.Nil(t, result, "CompareEncoding should return nil when TotalChanges() <= 0")
	})
	
	// 20. external_docs.go:84
	t.Run("ExternalDocsChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		extDocs1 := &base.ExternalDoc{}
		extDocs2 := &base.ExternalDoc{}
		result := CompareExternalDocs(extDocs1, extDocs2)
		assert.Nil(t, result, "CompareExternalDocs should return nil when TotalChanges() <= 0")
	})
	
	// 21. examples.go:88
	t.Run("ExamplesChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		examples1 := &v2.Examples{}
		examples2 := &v2.Examples{}
		result := CompareExamplesV2(examples1, examples2)
		assert.Nil(t, result, "CompareExamplesV2 should return nil when TotalChanges() <= 0")
	})
	
	// 22. contact.go:82
	t.Run("ContactChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		contact1 := &base.Contact{}
		contact2 := &base.Contact{}
		result := CompareContact(contact1, contact2)
		assert.Nil(t, result, "CompareContact should return nil when TotalChanges() <= 0")
	})
	
	// 23. parameter.go:342
	t.Run("ParameterChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		param1 := &v3.Parameter{}
		param2 := &v3.Parameter{}
		result := CompareParametersV3(param1, param2)
		assert.Nil(t, result, "CompareParametersV3 should return nil when TotalChanges() <= 0")
	})
	
	// 24. media_type.go:152
	t.Run("MediaTypeChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		mediaType1 := &v3.MediaType{}
		mediaType2 := &v3.MediaType{}
		result := CompareMediaTypes(mediaType1, mediaType2)
		assert.Nil(t, result, "CompareMediaTypes should return nil when TotalChanges() <= 0")
	})
	
	// 25. discriminator.go:97
	t.Run("DiscriminatorChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		discriminator1 := &base.Discriminator{}
		discriminator2 := &base.Discriminator{}
		result := CompareDiscriminator(discriminator1, discriminator2)
		assert.Nil(t, result, "CompareDiscriminator should return nil when TotalChanges() <= 0")
	})
	
	// 26. security_scheme.go:186
	t.Run("SecuritySchemeChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		secScheme1 := &v3.SecurityScheme{}
		secScheme2 := &v3.SecurityScheme{}
		result := CompareSecuritySchemes(secScheme1, secScheme2)
		assert.Nil(t, result, "CompareSecuritySchemes should return nil when TotalChanges() <= 0")
	})
	
	// 27. path_item.go:223
	t.Run("PathItemChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		pathItem1 := &v3.PathItem{}
		pathItem2 := &v3.PathItem{}
		result := ComparePathItems(pathItem1, pathItem2)
		assert.Nil(t, result, "ComparePathItems should return nil when TotalChanges() <= 0")
	})
	
	// 28. info.go:160
	t.Run("InfoChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		info1 := &base.Info{}
		info2 := &base.Info{}
		result := CompareInfo(info1, info2)
		assert.Nil(t, result, "CompareInfo should return nil when TotalChanges() <= 0")
	})
	
	// 29. security_requirement.go:51
	t.Run("SecurityRequirementChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		secReq1 := &base.SecurityRequirement{}
		secReq2 := &base.SecurityRequirement{}
		result := CompareSecurityRequirement(secReq1, secReq2)
		assert.Nil(t, result, "CompareSecurityRequirement should return nil when TotalChanges() <= 0")
	})
	
	// 30. operation.go:428
	t.Run("OperationChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		operation1 := &v3.Operation{}
		operation2 := &v3.Operation{}
		result := CompareOperations(operation1, operation2)
		assert.Nil(t, result, "CompareOperations should return nil when TotalChanges() <= 0")
	})
	
	// 31. responses.go:145
	t.Run("ResponsesChanges_TotalChangesZero_ReturnsNil", func(t *testing.T) {
		responses1 := &v3.Responses{}
		responses2 := &v3.Responses{}
		result := CompareResponses(responses1, responses2)
		assert.Nil(t, result, "CompareResponses should return nil when TotalChanges() <= 0")
	})
}