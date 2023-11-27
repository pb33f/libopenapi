package libopenapi

import (
	"os"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

type loopFrame struct {
	Type       string
	Restricted bool
}

type context struct {
	visited []string
	stack   []loopFrame
}

func Test_Speakeasy_Document_Iteration(t *testing.T) {
	spec, err := os.ReadFile("test_specs/speakeasy-test.yaml")
	require.NoError(t, err)

	doc, err := NewDocumentWithConfiguration(spec, &datamodel.DocumentConfiguration{
		BasePath:                            "./test_specs",
		IgnorePolymorphicCircularReferences: true,
		IgnoreArrayCircularReferences:       true,
		AllowFileReferences:                 true,
	})
	require.NoError(t, err)

	m, errs := doc.BuildV3Model()
	require.Empty(t, errs)

	for pair := orderedmap.First(m.Model.Paths.PathItems); pair != nil; pair = pair.Next() {
		t.Log(pair.Key())

		iterateOperations(t, pair.Value().GetOperations())
	}

	for pair := orderedmap.First(m.Model.Webhooks); pair != nil; pair = pair.Next() {
		t.Log(pair.Key())

		iterateOperations(t, pair.Value().GetOperations())
	}

	for pair := orderedmap.First(m.Model.Components.Schemas); pair != nil; pair = pair.Next() {
		t.Log(pair.Key())

		handleSchema(t, pair.Value(), context{})
	}
}

func iterateOperations(t *testing.T, ops map[string]*v3.Operation) {
	t.Helper()

	for method, op := range ops {
		t.Log(method)

		for _, param := range op.Parameters {
			if param.Schema != nil {
				handleSchema(t, param.Schema, context{})
			}
		}

		if op.RequestBody != nil {
			for pair := orderedmap.First(op.RequestBody.Content); pair != nil; pair = pair.Next() {
				t.Log(pair.Key())

				mediaType := pair.Value()

				if mediaType.Schema != nil {
					handleSchema(t, mediaType.Schema, context{})
				}
			}
		}

		for codePair := orderedmap.First(op.Responses.Codes); codePair != nil; codePair = codePair.Next() {
			t.Log(codePair.Key())

			for contentPair := orderedmap.First(codePair.Value().Content); contentPair != nil; contentPair = contentPair.Next() {
				t.Log(contentPair.Key())

				mediaType := contentPair.Value()

				if mediaType.Schema != nil {
					handleSchema(t, mediaType.Schema, context{})
				}
			}
		}

		for callacksPair := orderedmap.First(op.Callbacks); callacksPair != nil; callacksPair = callacksPair.Next() {
			t.Log(callacksPair.Key())

			for expressionPair := orderedmap.First(callacksPair.Value().Expression); expressionPair != nil; expressionPair = expressionPair.Next() {
				t.Log(expressionPair.Key())

				iterateOperations(t, expressionPair.Value().GetOperations())
			}
		}
	}
}

func handleSchema(t *testing.T, schProxy *base.SchemaProxy, ctx context) {
	t.Helper()

	if checkCircularReference(t, &ctx, schProxy) {
		return
	}

	sch, err := schProxy.BuildSchema()
	require.NoError(t, err)

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

		if sch.Properties.Len() > 0 {
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
	t.Helper()

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
	t.Helper()

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
	t.Helper()

	for pair := orderedmap.First(sch.Properties); pair != nil; pair = pair.Next() {
		ctx.stack = append(ctx.stack, loopFrame{Type: "object", Restricted: slices.Contains(sch.Required, pair.Key())})
		handleSchema(t, pair.Value(), ctx)
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

			require.False(t, isRestricted, "circular reference: %s", append(ctx.visited, loopRef))
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
