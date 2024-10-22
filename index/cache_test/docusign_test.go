package libopenapi

import (
	"github.com/pb33f/libopenapi"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/require"
)

type loopFrame struct {
	Type       string
	Restricted bool
}

type context struct {
	visited []string
	stack   []loopFrame
}

func Test_Docusign_Document_Iteration(t *testing.T) {
	runTest(t, "../../test_specs/docusignv3.1.json")
}

func runTest(t *testing.T, specLocation string) {
	spec, err := os.ReadFile(specLocation)
	if t != nil {
		require.NoError(t, err)
	}

	doc, err := libopenapi.NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
		BasePath:                            "../../test_specs",
		IgnorePolymorphicCircularReferences: true,
		IgnoreArrayCircularReferences:       true,
		AllowFileReferences:                 true,
	})
	if t != nil {
		require.NoError(t, err)
	}

	m, errs := doc.BuildV3Model()
	if t != nil {
		require.Empty(t, errs)
	}

	for _, pathItem := range m.Model.Paths.PathItems.FromOldest() {

		iterateOperations(t, pathItem.GetOperations())
	}

	for _, pathItem := range m.Model.Webhooks.FromOldest() {

		iterateOperations(t, pathItem.GetOperations())
	}

	for _, schemaProxy := range m.Model.Components.Schemas.FromOldest() {
		handleSchema(t, schemaProxy, context{})
	}

	require.Equal(t, uint64(1008), m.Index.GetHighCacheMisses())
	require.Equal(t, uint64(4049727), m.Index.GetHighCacheHits())
}

func iterateOperations(t *testing.T, ops *orderedmap.Map[string, *v3.Operation]) {
	for _, op := range ops.FromOldest() {

		for _, param := range op.Parameters {

			if param.Schema != nil {
				handleSchema(t, param.Schema, context{})
			}
		}

		if op.RequestBody != nil {

			for _, mediaType := range op.RequestBody.Content.FromOldest() {

				if mediaType.Schema != nil {
					handleSchema(t, mediaType.Schema, context{})
				}
			}
		}

		if orderedmap.Len(op.Responses.Codes) > 0 {
		}

		for _, response := range op.Responses.Codes.FromOldest() {

			for _, mediaType := range response.Content.FromOldest() {

				if mediaType.Schema != nil {
					handleSchema(t, mediaType.Schema, context{})
				}
			}
		}

		if orderedmap.Len(op.Responses.Codes) > 0 {

		}

		for _, callback := range op.Callbacks.FromOldest() {

			for _, pathItem := range callback.Expression.FromOldest() {

				iterateOperations(t, pathItem.GetOperations())
			}
		}
	}
}

func handleSchema(t *testing.T, schProxy *base.SchemaProxy, ctx context) {
	if checkCircularReference(t, &ctx, schProxy) {
		return
	}

	sch, err := schProxy.BuildSchema()
	if t != nil {
		require.NoError(t, err)
	}

	typ, subTypes := getResolvedType(sch)

	if len(sch.Enum) > 0 {
		switch typ {
		case "string":
			return
		case "integer":
			return
		default:
			// handle as base type
		}
	}

	switch typ {
	case "allOf":
		fallthrough
	case "anyOf":
		fallthrough
	case "oneOf":
		if len(subTypes) > 0 {
			return
		}

		handleAllOfAnyOfOneOf(t, sch, ctx)
	case "array":
		handleArray(t, sch, ctx)
	case "object":
		handleObject(t, sch, ctx)
	default:
		return
	}
}

func getResolvedType(sch *base.Schema) (string, []string) {
	subTypes := []string{}

	for _, t := range sch.Type {
		if t == "" { // treat empty type as any
			subTypes = append(subTypes, "any")
		} else if t != "null" {
			subTypes = append(subTypes, t)
		}
	}

	if len(sch.AllOf) > 0 {
		return "allOf", nil
	}

	if len(sch.AnyOf) > 0 {
		return "anyOf", nil
	}

	if len(sch.OneOf) > 0 {
		return "oneOf", nil
	}

	if len(subTypes) == 0 {
		if len(sch.Enum) > 0 {
			return "string", nil
		}

		if orderedmap.Len(sch.Properties) > 0 {
			return "object", nil
		}

		if sch.AdditionalProperties != nil {
			return "object", nil
		}

		if sch.Items != nil {
			return "array", nil
		}

		return "any", nil
	}

	if len(subTypes) == 1 {
		return subTypes[0], nil
	}

	return "oneOf", subTypes
}

func handleAllOfAnyOfOneOf(t *testing.T, sch *base.Schema, ctx context) {
	var schemas []*base.SchemaProxy

	switch {
	case len(sch.AllOf) > 0:
		schemas = sch.AllOf
	case len(sch.AnyOf) > 0:
		schemas = sch.AnyOf
		ctx.stack = append(ctx.stack, loopFrame{Type: "anyOf", Restricted: len(sch.AnyOf) == 1})
	case len(sch.OneOf) > 0:
		schemas = sch.OneOf
		ctx.stack = append(ctx.stack, loopFrame{Type: "oneOf", Restricted: len(sch.OneOf) == 1})
	}

	for _, s := range schemas {
		handleSchema(t, s, ctx)
	}
}

func handleArray(t *testing.T, sch *base.Schema, ctx context) {
	ctx.stack = append(ctx.stack, loopFrame{Type: "array", Restricted: sch.MinItems != nil && *sch.MinItems > 0})

	if sch.Items != nil && sch.Items.IsA() {
		handleSchema(t, sch.Items.A, ctx)
	}

	if sch.Contains != nil {
		handleSchema(t, sch.Contains, ctx)
	}

	if sch.PrefixItems != nil {
		for _, s := range sch.PrefixItems {
			handleSchema(t, s, ctx)
		}
	}
}

func handleObject(t *testing.T, sch *base.Schema, ctx context) {
	for name, schemaProxy := range sch.Properties.FromOldest() {
		ctx.stack = append(ctx.stack, loopFrame{Type: "object", Restricted: slices.Contains(sch.Required, name)})
		handleSchema(t, schemaProxy, ctx)
	}

	if sch.AdditionalProperties != nil && sch.AdditionalProperties.IsA() {
		handleSchema(t, sch.AdditionalProperties.A, ctx)
	}
}

func checkCircularReference(t *testing.T, ctx *context, schProxy *base.SchemaProxy) bool {
	loopRef := getSimplifiedRef(schProxy.GetReference())

	if loopRef != "" {
		if slices.Contains(ctx.visited, loopRef) {
			isRestricted := true
			containsObject := false

			for _, v := range ctx.stack {
				if v.Type == "object" {
					containsObject = true
				}

				if v.Type == "array" && !v.Restricted {
					isRestricted = false
				} else if !v.Restricted {
					isRestricted = false
				}
			}

			if !containsObject {
				isRestricted = true
			}

			if t != nil {
				require.False(t, isRestricted, "circular reference: %s", append(ctx.visited, loopRef))
			}
			return true
		}

		ctx.visited = append(ctx.visited, loopRef)
	}

	return false
}

// getSimplifiedRef will return the reference without the preceding file path
// caveat is that if a spec has the same ref in two different files they include this may identify them incorrectly
// but currently a problem anyway as libopenapi when returning references from an external file won't include the file path
// for a local reference with that file and so we might fail to distinguish between them that way.
// The fix needed is for libopenapi to also track which file the reference is in so we can always prefix them with the file path
func getSimplifiedRef(ref string) string {
	if ref == "" {
		return ""
	}

	refParts := strings.Split(ref, "#/")
	return "#/" + refParts[len(refParts)-1]
}
