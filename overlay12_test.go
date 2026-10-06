package libopenapi

import (
	"fmt"
	"testing"

	highoverlay "github.com/pb33f/libopenapi/datamodel/high/overlay"
	"github.com/pb33f/libopenapi/overlay"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestOverlayScalarVersionCompatibility(t *testing.T) {
	target := []byte("openapi: 3.1.0\ninfo: {title: Original, version: '1'}\npaths: {}")
	for _, version := range []string{"1", "1.0", "'1.0'"} {
		for _, dialect := range []string{"1.0", "'1.2'", "1.3.0", "1.99.7"} {
			t.Run(dialect+"/"+version, func(t *testing.T) {
				raw := []byte(fmt.Sprintf("overlay: %s\ninfo: {title: Compatibility, version: %s}\nactions: [{target: '$.info.title', update: Changed}]", dialect, version))
				ov, err := NewOverlayDocument(raw)
				require.NoError(t, err)
				rendered, err := ov.Render()
				require.NoError(t, err)
				reparsed, err := NewOverlayDocument(rendered)
				require.NoError(t, err)
				assert.Equal(t, ov.Info.Version, reparsed.Info.Version)
				assert.Equal(t, ov.Overlay, reparsed.Overlay)
				for _, input := range [][]byte{raw, rendered} {
					result, err := ApplyOverlayFromBytesToSpecBytes(target, input)
					require.NoError(t, err)
					assert.Contains(t, string(result.Bytes), "title: Changed")
				}
			})
		}
	}
}

func TestOverlayApplyDoesNotResolveUnusedDocumentURIs(t *testing.T) {
	target := []byte("openapi: 3.1.0\ninfo: {title: Original, version: '1'}\npaths: {}")
	for _, field := range []string{"extends", "$self"} {
		for _, uri := range []string{"file:///C:/My Specs/api.yaml", "api.yaml#fragment", "%zz"} {
			t.Run(field+"/"+uri, func(t *testing.T) {
				raw := []byte(fmt.Sprintf("overlay: 1.2.0\n%s: '%s'\ninfo: {title: Compatibility, version: '1'}\nactions: [{target: '$.info.title', update: Changed}]", field, uri))
				result, err := ApplyOverlayFromBytesToSpecBytes(target, raw)
				require.NoError(t, err)
				assert.Contains(t, string(result.Bytes), "title: Changed")
				ov, err := NewOverlayDocument(raw)
				require.NoError(t, err)
				_, err = ov.ResolveExtends("")
				require.Error(t, err)
			})
		}
	}
}

func TestOverlay12PublicEntryPoints(t *testing.T) {
	overlays := []string{
		`overlay: 1.2.0
$self: https://example.com/overlays/shared.yaml
extends: ../api.yaml
info: {title: Error responses, version: '1', description: Shared errors}
components:
  x-source: docs
  actions:
    errorResponse:
      description: Error response
      x-owner: docs
      fields:
        description: Add missing response
        remove: true
        update:
          '404': {description: Not Found}
actions:
  - $ref: '#/components/actions/errorResponse'
    target: $.paths.*.*.responses
    description: ''
    remove: false
`,
		`{"overlay":"1.2.0","$self":"https://example.com/overlays/shared.yaml","extends":"../api.yaml","info":{"title":"Error responses","version":"1","description":"Shared errors"},"components":{"x-source":"docs","actions":{"errorResponse":{"description":"Error response","x-owner":"docs","fields":{"description":"Add missing response","remove":true,"update":{"404":{"description":"Not Found"}}}}}},"actions":[{"$ref":"#/components/actions/errorResponse","target":"$.paths.*.*.responses","description":"","remove":false}]}`,
	}
	targets := []string{
		"openapi: 3.1.0\ninfo: {title: Items, version: '1'}\npaths:\n  /items:\n    get:\n      responses:\n        '200': {description: OK}",
		`{"openapi":"3.1.0","info":{"title":"Items","version":"1"},"paths":{"/items":{"get":{"responses":{"200":{"description":"OK"}}}}}}`,
	}
	for _, raw := range overlays {
		for _, target := range targets {
			ov, err := NewOverlayDocument([]byte(raw))
			require.NoError(t, err)
			assert.Equal(t, "https://example.com/overlays/shared.yaml", ov.Self)
			resolved, err := ov.ResolveExtends("file:///local/overlay.yaml")
			require.NoError(t, err)
			assert.Equal(t, "https://example.com/api.yaml", resolved)
			require.NotNil(t, ov.Components.GoLow())
			assert.Equal(t, ov.Components.GoLow(), ov.Components.GoLowUntyped())
			reusable, ok := ov.Components.Actions.Get("errorResponse")
			require.True(t, ok)
			require.NotNil(t, reusable.GoLow())
			assert.Equal(t, reusable.GoLow(), reusable.GoLowUntyped())
			_, err = ov.Components.Render()
			require.NoError(t, err)
			_, err = reusable.Render()
			require.NoError(t, err)
			rendered, err := ov.Render()
			require.NoError(t, err)
			reparsed, err := NewOverlayDocument(rendered)
			require.NoError(t, err)
			assert.Equal(t, ov.GoLow().Hash(), reparsed.GoLow().Hash())
			assert.True(t, reparsed.Actions[0].HasRemove())
			assert.False(t, reparsed.Actions[0].Remove)
			doc, err := NewDocument([]byte(target))
			require.NoError(t, err)
			calls := []func() (*OverlayResult, error){
				func() (*OverlayResult, error) { return ApplyOverlay(doc, ov) },
				func() (*OverlayResult, error) { return ApplyOverlayFromBytes(doc, rendered) },
				func() (*OverlayResult, error) { return ApplyOverlayToSpecBytes([]byte(target), reparsed) },
				func() (*OverlayResult, error) { return ApplyOverlayFromBytesToSpecBytes([]byte(target), []byte(raw)) },
			}
			for _, call := range calls {
				result, err := call()
				require.NoError(t, err)
				require.Empty(t, result.Warnings)
				model, err := result.OverlayDocument.BuildV3Model()
				require.NoError(t, err)
				path, ok := model.Model.Paths.PathItems.Get("/items")
				require.True(t, ok)
				response, ok := path.Get.Responses.Codes.Get("404")
				require.True(t, ok)
				assert.Equal(t, "Not Found", response.Description)
				_, ok = path.Get.Responses.Codes.Get("200")
				assert.True(t, ok)
			}
		}
	}
}

func TestOverlay12ConstructedModelsRender(t *testing.T) {
	action := &highoverlay.Action{Ref: "#/components/actions/noop", Target: "$"}
	action.SetRemove(false)
	ov := &highoverlay.Overlay{Overlay: "1.2.0", Info: &highoverlay.Info{Title: "Constructed", Version: "1"}, Components: &highoverlay.Components{}, Actions: []*highoverlay.Action{action}}
	raw, err := ov.Render()
	require.NoError(t, err)
	parsed, err := NewOverlayDocument(raw)
	require.NoError(t, err)
	require.NotNil(t, parsed.Components)
	assert.True(t, parsed.Actions[0].HasRemove())
	assert.False(t, parsed.Actions[0].Remove)
}

func TestOverlay12EmptyAndAbsentInfo(t *testing.T) {
	target := []byte("openapi: 3.1.0\ninfo: {title: Original, version: '1'}\npaths: {}")
	for _, info := range []string{"{title: '', version: '1'}", "{title: Example, version: ''}", "{title: '', version: ''}"} {
		ov, err := NewOverlayDocument([]byte("overlay: 1.2.0\ninfo: " + info + "\nactions: [{target: '$.info', update: {title: Changed}}]"))
		require.NoError(t, err)
		rendered, err := ov.Render()
		require.NoError(t, err)
		result, err := ApplyOverlayFromBytesToSpecBytes(target, rendered)
		require.NoError(t, err)
		assert.Contains(t, string(result.Bytes), "title: Changed")
	}
	for _, info := range []string{"{version: '1'}", "{title: Example}"} {
		_, err := ApplyOverlayFromBytesToSpecBytes(target, []byte("overlay: 1.2.0\ninfo: "+info+"\nactions: [{target: '$'}]"))
		require.ErrorIs(t, err, overlay.ErrInvalidInfo)
	}
}
