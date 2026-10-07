// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

const idiomaticNamesSpec = `openapi: 3.1.0
info:
  title: Idiomatic names
  version: 1.0.0
paths: {}
components:
  schemas:
    Account:
      type: object
      properties:
        status:
          type: string
          enum: [active, closed]
        account_type:
          type: string
          enum: [person, company]
        tags:
          type: array
          items:
            type: object
            properties:
              label:
                type: string
        history:
          type: array
          items:
            type: object
            properties:
              at:
                type: string
        settings:
          type: object
          additionalProperties:
            type: object
            properties:
              value:
                type: string
        pair:
          type: array
          prefixItems:
            - type: object
              properties:
                left:
                  type: string
        extra:
          type: object
        closed:
          type: object
          additionalProperties: false
    Pet:
      type: object
      properties:
        name:
          type: string
    Pets:
      type: array
      items:
        type: object
        properties:
          name:
            type: string
    Dogs:
      type: array
      items:
        type: object
        properties:
          name:
            type: string
    X:
      type: object
      properties:
        id:
          type: string
    DupX:
      type: object
      properties:
        id:
          type: string
    Shape:
      oneOf:
        - title: circle
          type: object
          required: [kind]
          properties:
            kind:
              type: string
              const: circle
            radius:
              type: number
        - type: object
          required: [kind]
          properties:
            kind:
              type: string
              const: square
            width:
              type: number
      properties:
        label:
          type: object
          properties:
            text:
              type: string
    PeopleMatchResponse:
      type: object
      properties:
        person:
          type: object
          properties:
            organization:
              type: object
              properties:
                name:
                  type: string
            contact_emails:
              type: array
              items:
                type: object
                properties:
                  email:
                    type: string
            account:
              type: object
              properties:
                id:
                  type: string
            client:
              type: object
              properties:
                id:
                  type: string
    OtherResponse:
      type: object
      properties:
        organization:
          type: object
          properties:
            id:
              type: integer
    Dup:
      type: object
      properties:
        x:
          type: object
          properties:
            name:
              type: string
`

// assertContainsCode checks for code with gofmt's column alignment collapsed.
func assertContainsCode(t *testing.T, src string, expected ...string) {
	t.Helper()
	collapsed := strings.Join(strings.Fields(src), " ")
	for _, code := range expected {
		if !strings.Contains(collapsed, strings.Join(strings.Fields(code), " ")) {
			t.Fatalf("expected %q in:\n%s", code, src)
		}
	}
}

func renderSpec(t *testing.T, spec string, opts ...Option) *GeneratedFile {
	t.Helper()
	file, err := NewGenerator(opts...).RenderSchemas(componentsFromSpec(t, spec))
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func componentsFromSpec(t *testing.T, spec string) *orderedmap.Map[string, *highbase.SchemaProxy] {
	t.Helper()
	doc, err := libopenapi.NewDocument([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	return model.Model.Components.Schemas
}

func TestIdiomaticNamesQualifyComponentChildren(t *testing.T) {
	file := renderSpec(t, idiomaticNamesSpec, WithNameStyle(NameStyleIdiomatic), WithReservedTypeNames("Client"),
		WithInlineRoots("PeopleMatchResponse", "OtherResponse", "Dup"))
	src := string(file.Source)
	assertContainsCode(t, src,
		"type AccountStatus string",
		"type AccountType string",
		"Tags     []AccountTag ",
		"History  []AccountHistoryItem ",
		"Settings map[string]AccountSettingsValue ",
		"Pair     []any ",
		"Extra    map[string]any ",
		"Closed   *AccountClosed ",
		"type AccountClosed struct {\n}",
		"type Pets []PetsItem",
		"type Dogs []Dog",
		"type ShapeCircle struct",
		"type ShapeVariant2 struct",
		"Person *Person ",
		"Organization  *Organization ",
		"ContactEmails []ContactEmail ",
		"Account       *PersonAccount ",
		"Client        *PersonClient ",
		"Organization *OtherResponseOrganization ",
		"X *DupX2 ",
	)
	for _, generated := range file.Types {
		if strings.Contains(generated.Name, "_") {
			t.Fatalf("type name %q contains an underscore", generated.Name)
		}
	}
	for _, unexpected := range []string{"AccountExtra", "AccountPairTuple1", "type AccountSettings map", "ShapeLabel"} {
		assertNotContains(t, src, unexpected)
	}
	collisions := 0
	for _, diagnostic := range file.Diagnostics {
		if diagnostic.Code == DiagnosticTypeNameCollision {
			collisions++
			if diagnostic.Message != "type name collision resolved as DupX2" {
				t.Fatalf("unexpected collision diagnostic %#v", diagnostic)
			}
		}
	}
	if collisions != 1 {
		t.Fatalf("expected one numbered collision, got %#v", file.Diagnostics)
	}
	assertParsesAndCompiles(t, file.Source)
}

func TestIdiomaticFieldNamesSpellOutLeadingSymbols(t *testing.T) {
	file := renderSpec(t, `openapi: 3.1.0
info:
  title: Fields
  version: 1.0.0
paths: {}
components:
  schemas:
    Base:
      type: object
      properties:
        kind:
          type: string
    Record:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            _id:
              type: string
            id:
              type: string
            "@type":
              type: string
            type:
              type: string
            __v:
              type: integer
            user_id:
              type: string
            userId:
              type: string
            base:
              type: string
      additionalProperties:
        type: string
      properties:
        additionalProperties:
          type: string
`, WithNameStyle(NameStyleIdiomatic))
	src := string(file.Source)
	assertContainsCode(t, src,
		"UnderscoreID *string `json:\"_id,omitempty\"`",
		"ID           *string `json:\"id,omitempty\"`",
		"AtType       *string `json:\"@type,omitempty\"`",
		"Type         *string `json:\"type,omitempty\"`",
		"V            *int    `json:\"__v,omitempty\"`",
		"UserID       *string `json:\"user_id,omitempty\"`",
		"UserID2      *string `json:\"userId,omitempty\"`",
		"Base2        *string `json:\"base,omitempty\"`",
		"AdditionalProperties2 map[string]string `json:\"-\"`",
	)
	var fields []string
	for _, diagnostic := range file.Diagnostics {
		if diagnostic.Code == DiagnosticFieldNameCollision {
			fields = append(fields, diagnostic.Path)
		}
	}
	if want := []string{"Record.userId", "Record.base", "Record.additionalProperties"}; !reflect.DeepEqual(fields, want) {
		t.Fatalf("field collisions = %q, want %q", fields, want)
	}
	assertParsesAndCompiles(t, file.Source)
}

func TestQualifiedFieldNamesKeepEmbeddedTypes(t *testing.T) {
	file := renderSpec(t, `openapi: 3.1.0
info:
  title: Fields
  version: 1.0.0
paths: {}
components:
  schemas:
    Base:
      type: object
      properties:
        kind:
          type: string
    Record:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            base:
              type: string
`)
	assertContains(t, string(file.Source), "Base__2 *string `json:\"base,omitempty\"`")
	assertParsesAndCompiles(t, file.Source)
}

func TestUntypedSchemasAsRawMessage(t *testing.T) {
	file := renderSpec(t, `openapi: 3.0.3
info:
  title: Untyped
  version: 1.0.0
paths: {}
components:
  schemas:
    Loose:
      type: object
      required: [required_anything]
      properties:
        anything: {}
        required_anything: {}
        nullable_anything:
          nullable: true
        object:
          type: object
        open_values:
          type: object
          additionalProperties: {}
        list:
          type: array
        tuple:
          type: array
          items:
            type: string
          prefixItems:
            - type: string
        typed:
          type: object
          properties:
            name:
              type: string
          additionalProperties: {}
    Anything: {}
    Bag:
      type: object
`, WithUntypedAsRawMessage(true), WithOptionalNullableAsDoublePointer(true))
	src := string(file.Source)
	assertContainsCode(t, src,
		"Anything         json.RawMessage   `json:\"anything,omitempty\"`",
		"RequiredAnything json.RawMessage   `json:\"required_anything\"`",
		"NullableAnything json.RawMessage   `json:\"nullable_anything,omitempty\"`",
		"Object           json.RawMessage   `json:\"object,omitempty\"`",
		"OpenValues       json.RawMessage   `json:\"open_values,omitempty\"`",
		"List             []json.RawMessage `json:\"list,omitempty\"`",
		"Tuple            []json.RawMessage `json:\"tuple,omitempty\"`",
		"AdditionalProperties map[string]json.RawMessage `json:\"-\"`",
		"type Anything = json.RawMessage",
		"type Bag = json.RawMessage",
	)
	if regexp.MustCompile(`\bany\b`).MatchString(src) {
		t.Fatalf("untyped schemas rendered as any:\n%s", src)
	}
	assertParsesCompilesAndTests(t, file.Source, `package models

import (
	"encoding/json"
	"testing"
)

func TestRawMessageRoundTrip(t *testing.T) {
	input := []byte(`+"`"+`{"anything":{"a":[1,2]},"required_anything":null,"object":{"b":true},"list":[1,"x"]}`+"`"+`)
	var loose Loose
	if err := json.Unmarshal(input, &loose); err != nil {
		t.Fatal(err)
	}
	if string(loose.Anything) != `+"`"+`{"a":[1,2]}`+"`"+` || string(loose.RequiredAnything) != "null" || len(loose.List) != 2 {
		t.Fatalf("decoded %#v", loose)
	}
	output, err := json.Marshal(loose)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != string(input) {
		t.Fatalf("round trip = %s", output)
	}
	var bag Bag = json.RawMessage(`+"`"+`{"k":1}`+"`"+`)
	if encoded, _ := json.Marshal(bag); string(encoded) != `+"`"+`{"k":1}`+"`"+` {
		t.Fatalf("alias lost RawMessage methods: %s", encoded)
	}
}
`)
}

func TestNullableKeywordInOpenAPI31IsHonouredWithDiagnostic(t *testing.T) {
	for _, test := range []struct {
		version    string
		diagnostic bool
	}{{"3.1.0", true}, {"3.0.3", false}} {
		file := renderSpec(t, `openapi: `+test.version+`
info:
  title: Nullable
  version: 1.0.0
paths: {}
components:
  schemas:
    Person:
      type: object
      required: [confidence]
      properties:
        confidence:
          type: number
          nullable: true
`)
		assertContains(t, string(file.Source), "Confidence *float64 `json:\"confidence\"`")
		if got := hasDiagnosticCode(file.Diagnostics, DiagnosticNullableKeyword); got != test.diagnostic {
			t.Fatalf("OpenAPI %s nullable diagnostic = %v, want %v: %#v", test.version, got, test.diagnostic, file.Diagnostics)
		}
	}
	nullable := true
	built := highbase.CreateSchemaProxy(&highbase.Schema{Type: []string{"number"}, Nullable: &nullable})
	file, err := NewGenerator().RenderSchemas(singleSchemaMap(t, "Built", built))
	if err != nil {
		t.Fatal(err)
	}
	if hasDiagnosticCode(file.Diagnostics, DiagnosticNullableKeyword) {
		t.Fatal("a schema without a source document cannot be judged as OpenAPI 3.1")
	}
}

func TestIdiomaticRenderSchemaNamesRootAndChildren(t *testing.T) {
	source, err := RenderSchema("widget-list", schemaProxyFromYAML(t, `
type: array
items:
  type: object
  properties:
    name:
      type: string
`), WithNameStyle(NameStyleIdiomatic))
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, string(source), "type WidgetList []WidgetListItem")
	assertContains(t, string(source), "type WidgetListItem struct")
}

func TestPublicNameWordsAndInitialisms(t *testing.T) {
	for input, want := range map[string]string{
		"label_ids":          "LabelIDs",
		"emailer_urls":       "EmailerURLs",
		"email_md5":          "EmailMD5",
		"email_sha256":       "EmailSHA256",
		"base64Encoded":      "Base64Encoded",
		"OAuth2Token":        "OAuth2Token",
		"Status200Response":  "Status200Response",
		"employment_history": "EmploymentHistory",
	} {
		if got := PublicName(input); got != want {
			t.Fatalf("PublicName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestItemNameSingularizesConfidentPlurals(t *testing.T) {
	for input, want := range map[string]string{
		"FundingEvents":       "FundingEvent",
		"CurrentTechnologies": "CurrentTechnology",
		"Addresses":           "Address",
		"Boxes":               "Box",
		"Matches":             "Match",
		"Hashes":              "Hash",
		"Buzzes":              "Buzz",
		"People":              "Person",
		"Statuses":            "Status",
		"LabelIDs":            "LabelID",
		"Pets":                "Pet",
		"EmploymentHistory":   "EmploymentHistoryItem",
		"Class":               "ClassItem",
		"Status":              "StatusItem",
		"Analysis":            "AnalysisItem",
		"Data":                "DataItem",
		"Series":              "SeriesItem",
		"Gas":                 "GasItem",
		"Ties":                "Tie",
	} {
		if got := itemName(input); got != want {
			t.Fatalf("itemName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestJoinTypeNameWritesSharedWordsOnce(t *testing.T) {
	for _, test := range []struct{ owner, leaf, want string }{
		{"Contact", "ContactEmail", "ContactEmail"},
		{"BookAccount", "AccountStatus", "BookAccountStatus"},
		{"Account", "Account", "AccountAccount"},
		{"Person", "Organization", "PersonOrganization"},
		{"", "Person", "Person"},
	} {
		if got := joinTypeName(test.owner, test.leaf); got != test.want {
			t.Fatalf("joinTypeName(%q, %q) = %q, want %q", test.owner, test.leaf, got, test.want)
		}
	}
}

func TestNameRegistryClaimFirstAndIdiomaticNumbers(t *testing.T) {
	registry := NewIdiomaticNameRegistry("Client")
	if got := registry.ClaimFirst("Client", "PeopleClient"); got != "PeopleClient" {
		t.Fatalf("ClaimFirst() = %q", got)
	}
	if got := registry.ClaimFirst("Client", "PeopleClient"); got != "PeopleClient2" {
		t.Fatalf("ClaimFirst() numbered = %q", got)
	}
	if got := registry.Claim("Client", ""); got != "Client2" {
		t.Fatalf("Claim() numbered = %q", got)
	}
}

func TestFieldNamesResolvesRepeatedAndSymbolicSources(t *testing.T) {
	got := FieldNames([]string{"_id", "id", "id", "body"}, "Body")
	if want := []string{"UnderscoreID", "ID", "ID2", "Body2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("FieldNames() = %q, want %q", got, want)
	}
}

func TestIdiomaticRegistryLettersNamesEndingInDigits(t *testing.T) {
	registry := NewIdiomaticNameRegistry("PlaybookV2")
	if got := registry.ClaimFirst("PlaybookV2"); got != "PlaybookV2B" {
		t.Fatalf("ClaimFirst() = %q", got)
	}
	for letter := 'C'; letter <= 'Z'; letter++ {
		registry.ClaimFirst("PlaybookV2")
	}
	if got := registry.ClaimFirst("PlaybookV2"); got != "PlaybookV2Z2" {
		t.Fatalf("ClaimFirst() after the alphabet = %q", got)
	}
}

func TestDecodeOnlyRootsAndFallbackDescriptions(t *testing.T) {
	file := renderSpec(t, `openapi: 3.1.0
info:
  title: Roots
  version: 1.0.0
paths: {}
components:
  schemas:
    Sent:
      type: object
      properties:
        note:
          type: [string, "null"]
    Received:
      type: object
      properties:
        note:
          type: [string, "null"]
        nested:
          type: object
          properties:
            note:
              type: [string, "null"]
    Described:
      description: Its own words.
      type: string
`, WithOptionalNullableAsDoublePointer(true), WithDecodeOnlyRoots("Received"),
		WithFallbackDescriptions(map[string]string{"Sent": "The request body.", "Described": "Ignored."}))
	assertContainsCode(t, string(file.Source),
		"// Sent is the request body.\ntype Sent struct { Note **string `json:\"note,omitempty\"` }",
		"type Received struct { Note *string `json:\"note,omitempty\"` Nested *Received_Nested `json:\"nested,omitempty\"` }",
		"type Received_Nested struct { Note *string `json:\"note,omitempty\"` }",
		"// Described Its own words.\ntype Described string",
	)
}

type IdiomaticFieldSchemaOwner struct {
	Entries []any `json:"entries"`
}

func TestIdiomaticNamesFollowReflectedFieldSchemas(t *testing.T) {
	entries := schemaProxyFromYAML(t, `
type: array
items:
  type: object
  properties:
    name:
      type: string
`)
	set, err := SchemasFromTypesWithOptions([]reflect.Type{reflect.TypeOf(IdiomaticFieldSchemaOwner{})},
		WithNameStyle(NameStyleIdiomatic),
		WithFieldSchema(reflect.TypeOf(IdiomaticFieldSchemaOwner{}), "Entries", entries))
	if err != nil {
		t.Fatal(err)
	}
	if set.Root == nil {
		t.Fatal("no root schema")
	}
}
