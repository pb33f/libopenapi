// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import "testing"

func TestJSONSchema202012GeneratedConformanceDefault(t *testing.T) {
	file := renderJSONSchema202012(t)
	assertParsesCompilesAndTests(t, file.Source, `package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONSchema202012GeneratedModels(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"id":              "018f70f9-506f-7c68-b9ff-4b0d80dc8c31",
		"kind":            "torture",
		"multi_value":     42,
		"nullable_status": "active",
		"mixed_enum":      true,
		"string_enum":     "draft",
		"int_enum":        2,
		"float_enum":      1.5,
		"bool_enum":       false,
		"closed_config": map[string]any{
			"enabled":   true,
			"threshold": 12.5,
		},
		"labels": map[string]any{
			"region": "west",
			"tier":   "gold",
		},
		"tuple": []any{"seat", 3},
		"object_rules": map[string]any{
			"name":  "sample",
			"count": 2,
		},
		"encoded_payload": "eyJwYXlsb2FkX2lkIjoiYSJ9",
		"payment": map[string]any{
			"object": "card",
			"number": "4242424242424242",
			"cvc":    "123",
		},
		"loose_choice": 7,
		"dynamic_node": map[string]any{
			"name": "root",
			"children": []any{
				map[string]any{"name": "leaf"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var doc TortureDocument
	if err := json.Unmarshal(payload, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.ID != "018f70f9-506f-7c68-b9ff-4b0d80dc8c31" || doc.Kind != "torture" {
		t.Fatalf("unexpected scalar fields: %#v", doc)
	}
	if doc.MultiValue == nil || string(doc.MultiValue.Bytes()) != "42" {
		t.Fatalf("raw multi-type value did not decode: %#v", doc.MultiValue)
	}
	copied := doc.MultiValue.Bytes()
	copied[0] = '9'
	if string(doc.MultiValue.Bytes()) != "42" {
		t.Fatal("raw multi-type Bytes should return a copy")
	}
	if doc.NullableStatus == nil || *doc.NullableStatus != NullableStatus("active") {
		t.Fatalf("nullable enum did not decode: %#v", doc.NullableStatus)
	}
	if doc.MixedEnum == nil || any(*doc.MixedEnum) != true {
		t.Fatalf("mixed enum did not decode: %#v", doc.MixedEnum)
	}
	if doc.StringEnum == nil || *doc.StringEnum != StringEnum("draft") {
		t.Fatalf("string enum did not decode: %#v", doc.StringEnum)
	}
	if doc.IntEnum == nil || *doc.IntEnum != IntEnum(2) {
		t.Fatalf("integer enum did not decode: %#v", doc.IntEnum)
	}
	if doc.FloatEnum == nil || *doc.FloatEnum != FloatEnum(1.5) {
		t.Fatalf("number enum did not decode: %#v", doc.FloatEnum)
	}
	if doc.BoolEnum == nil || *doc.BoolEnum != BoolEnum(false) {
		t.Fatalf("boolean enum did not decode: %#v", doc.BoolEnum)
	}
	if doc.ClosedConfig == nil || !doc.ClosedConfig.Enabled || doc.ClosedConfig.Threshold == nil || *doc.ClosedConfig.Threshold != 12.5 {
		t.Fatalf("closed config did not decode: %#v", doc.ClosedConfig)
	}
	if doc.Labels == nil || doc.Labels.AdditionalProperties["region"] != "west" || doc.Labels.AdditionalProperties["tier"] != "gold" {
		t.Fatalf("additional properties did not decode: %#v", doc.Labels)
	}
	if doc.Tuple == nil || len(*doc.Tuple) != 2 || (*doc.Tuple)[0] != "seat" || (*doc.Tuple)[1] != float64(3) {
		t.Fatalf("tuple probe did not decode: %#v", doc.Tuple)
	}
	if doc.ObjectRules == nil || doc.ObjectRules.Name == nil || *doc.ObjectRules.Name != "sample" || doc.ObjectRules.Count == nil || *doc.ObjectRules.Count != 2 {
		t.Fatalf("object rules did not decode: %#v", doc.ObjectRules)
	}
	if doc.EncodedPayload == nil || *doc.EncodedPayload != EncodedPayload("eyJwYXlsb2FkX2lkIjoiYSJ9") {
		t.Fatalf("encoded payload did not decode: %#v", doc.EncodedPayload)
	}
	card, ok := doc.Payment.Value.(CardSource)
	if !ok || card.Object != "card" || card.Number != "4242424242424242" || card.CVC != "123" {
		t.Fatalf("payment union did not decode card: %#v", doc.Payment.Value)
	}
	if doc.LooseChoice == nil || string(doc.LooseChoice.Bytes()) != "7" {
		t.Fatalf("anyOf raw union did not decode: %#v", doc.LooseChoice)
	}
	if doc.DynamicNode == nil || doc.DynamicNode.Name == nil || *doc.DynamicNode.Name != "root" || len(doc.DynamicNode.Children) != 1 || doc.DynamicNode.Children[0].Name == nil || *doc.DynamicNode.Children[0].Name != "leaf" {
		t.Fatalf("dynamic recursive node did not decode: %#v", doc.DynamicNode)
	}

	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"\"object\":\"card\"",
		"\"region\":\"west\"",
		"\"loose_choice\":7",
		"\"multi_value\":42",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("missing %s in marshal output: %s", want, out)
		}
	}

	var labels StringMap
	if err := json.Unmarshal([]byte("{\"region\":\"west\",\"tier\":\"gold\"}"), &labels); err != nil {
		t.Fatal(err)
	}
	if labels.AdditionalProperties["region"] != "west" || labels.AdditionalProperties["tier"] != "gold" {
		t.Fatalf("additional property map did not decode: %#v", labels.AdditionalProperties)
	}
	out, err = json.Marshal(labels)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "\"region\":\"west\"") || !strings.Contains(string(out), "\"tier\":\"gold\"") {
		t.Fatalf("additional property map did not marshal: %s", out)
	}

	var payment PaymentSourceUnion
	if err := json.Unmarshal([]byte("{\"object\":\"bank_account\",\"account_number\":\"abc\",\"bank_name\":\"Bank\"}"), &payment); err != nil {
		t.Fatal(err)
	}
	bank, ok := payment.Value.(BankSource)
	if !ok || bank.Object != "bank_account" || bank.AccountNumber != "abc" || bank.BankName == nil || *bank.BankName != "Bank" {
		t.Fatalf("payment union did not decode bank source: %#v", payment.Value)
	}
	if err := json.Unmarshal([]byte("{\"object\":\"cash\"}"), &payment); err == nil || !strings.Contains(err.Error(), "unknown object discriminator") {
		t.Fatalf("expected unknown discriminator error, got %v", err)
	}

	var loose LooseChoiceUnion
	if !loose.IsZero() {
		t.Fatal("zero raw union should report IsZero")
	}
	out, err = json.Marshal(loose)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "null" {
		t.Fatalf("zero raw union should marshal null, got %s", out)
	}
	if err := json.Unmarshal([]byte("\"abc\""), &loose); err != nil {
		t.Fatal(err)
	}
	if string(loose.Bytes()) != "\"abc\"" {
		t.Fatalf("raw union did not retain bytes: %s", loose.Bytes())
	}
}
`)
}

func TestJSONSchema202012GeneratedConformanceOptions(t *testing.T) {
	file := renderJSONSchema202012(t,
		WithAdditionalPropertiesMethods(false),
		WithEnumConstants(true),
	)
	assertParsesCompilesAndTests(t, file.Source, `package models

import (
	"encoding/json"
	"testing"
)

func TestJSONSchema202012GeneratedOptions(t *testing.T) {
	if StringEnumDraft != StringEnum("draft") || StringEnumPublished != StringEnum("published") {
		t.Fatalf("unexpected string enum constants: %q %q", StringEnumDraft, StringEnumPublished)
	}
	if IntEnumValue1 != IntEnum(1) || IntEnumValue2 != IntEnum(2) {
		t.Fatalf("unexpected integer enum constants: %d %d", IntEnumValue1, IntEnumValue2)
	}
	if FloatEnumValue15 != FloatEnum(1.5) || FloatEnumValue2 != FloatEnum(2) {
		t.Fatalf("unexpected number enum constants: %v %v", FloatEnumValue15, FloatEnumValue2)
	}
	if BoolEnumTrue != BoolEnum(true) || BoolEnumFalse != BoolEnum(false) {
		t.Fatalf("unexpected boolean enum constants: %v %v", BoolEnumTrue, BoolEnumFalse)
	}
	if NullableStatusActive != NullableStatus("active") || NullableStatusInactive != NullableStatus("inactive") {
		t.Fatalf("unexpected nullable enum constants: %q %q", NullableStatusActive, NullableStatusInactive)
	}

	var doc TortureDocument
	if err := json.Unmarshal([]byte("{\"id\":\"018f70f9-506f-7c68-b9ff-4b0d80dc8c31\",\"kind\":\"torture\",\"payment\":{\"object\":\"card\",\"number\":\"4242424242424242\",\"cvc\":\"123\"},\"string_enum\":\"published\"}"), &doc); err != nil {
		t.Fatal(err)
	}
	card, ok := doc.Payment.Value.(CardSource)
	if !ok || card.Object != "card" || card.Number != "4242424242424242" {
		t.Fatalf("payment union did not decode with options: %#v", doc.Payment.Value)
	}
	if doc.StringEnum == nil || *doc.StringEnum != StringEnumPublished {
		t.Fatalf("enum constant value did not decode: %#v", doc.StringEnum)
	}

	var labels StringMap
	if err := json.Unmarshal([]byte("{\"region\":\"west\"}"), &labels); err != nil {
		t.Fatal(err)
	}
	if labels.AdditionalProperties != nil {
		t.Fatalf("additional property methods should be disabled: %#v", labels.AdditionalProperties)
	}
	out, err := json.Marshal(StringMap{AdditionalProperties: map[string]string{"region": "west"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "{}" {
		t.Fatalf("additional properties should be ignored without generated methods, got %s", out)
	}
}
`)
}

func TestNameCollisionGeneratedConformanceDefault(t *testing.T) {
	file := renderNameCollisions(t, WithEnumConstants(true))
	if !hasDiagnosticCode(file.Diagnostics, DiagnosticComponentNameCollision) {
		t.Fatalf("expected component collision diagnostic: %#v", file.Diagnostics)
	}
	if !hasDiagnosticCode(file.Diagnostics, DiagnosticFieldNameCollision) {
		t.Fatalf("expected field collision diagnostic: %#v", file.Diagnostics)
	}
	if !hasDiagnosticCode(file.Diagnostics, DiagnosticTypeNameCollision) {
		t.Fatalf("expected nested type collision diagnostic: %#v", file.Diagnostics)
	}
	assertParsesCompilesAndTests(t, file.Source, `package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNameCollisionGeneratedModels(t *testing.T) {
	var root CollisionRoot
	if err := json.Unmarshal([]byte(`+"`"+`{
		"type": "root",
		"user-id": {"id": "first"},
		"user_id": {"id": 2},
		"UserID": {"id": true},
		"map": {"func": "call", "type": "mapping", "interface": "shape"},
		"choice": {"type": "card_duplicate", "value": 9},
		"recursive": {"next": {"children": [{"next": {}}]}},
		"map_ref": {"region": "west"},
		"alias": "alias-1",
		"enum": "1-5",
		"additional_properties": "known",
		"nested": {
			"value-id": "nested-id",
			"value_id": 5,
			"dup-obj": {"name": "first"},
			"dup_obj": {"count": 2},
			"inline-item": {"func": "inner"}
		},
		"x-extra": "extra"
	}`+"`"+`), &root); err != nil {
		t.Fatal(err)
	}
	if root.Type != "root" {
		t.Fatalf("keyword property did not decode: %#v", root)
	}
	if root.UserID.ID != "first" || root.UserID__2.ID != 2 || !root.UserID__3.ID {
		t.Fatalf("component refs did not resolve collision-safe names: %#v %#v %#v", root.UserID, root.UserID__2, root.UserID__3)
	}
	if root.Map.Func != "call" || root.Map.Type != "mapping" || root.Map.Interface != "shape" {
		t.Fatalf("keyword-like fields did not decode: %#v", root.Map)
	}
	duplicate, ok := root.Choice.Value.(ChoiceCard__2)
	if !ok || duplicate.Type != "card_duplicate" || duplicate.Value != 9 {
		t.Fatalf("discriminator mapping did not resolve collided variant: %#v", root.Choice.Value)
	}
	if root.Recursive == nil || root.Recursive.Next == nil || len(root.Recursive.Next.Children) != 1 || root.Recursive.Next.Children[0].Next == nil {
		t.Fatalf("recursive ref did not decode: %#v", root.Recursive)
	}
	if root.MapRef.AdditionalProperties["region"] != "west" {
		t.Fatalf("map ref did not decode: %#v", root.MapRef)
	}
	if root.Alias != AliasValue("alias-1") || root.Enum != EnumCollisionValue15__3 {
		t.Fatalf("alias or enum collision constants did not decode: %q %q", root.Alias, root.Enum)
	}
	if root.AdditionalProperties == nil || *root.AdditionalProperties != "known" {
		t.Fatalf("known additional_properties field did not decode: %#v", root.AdditionalProperties)
	}
	if root.AdditionalProperties__2["x-extra"] != "extra" {
		t.Fatalf("unknown additional property did not decode through collision-safe field: %#v", root.AdditionalProperties__2)
	}
	if root.Nested == nil || root.Nested.ValueID != "nested-id" || root.Nested.ValueID__2 != 5 || root.Nested.InlineItem == nil || root.Nested.InlineItem.Func == nil || *root.Nested.InlineItem.Func != "inner" {
		t.Fatalf("nested collided fields did not decode: %#v", root.Nested)
	}
	if root.Nested.DupObj == nil || root.Nested.DupObj.Name == nil || *root.Nested.DupObj.Name != "first" || root.Nested.DupObj__2 == nil || root.Nested.DupObj__2.Count == nil || *root.Nested.DupObj__2.Count != 2 {
		t.Fatalf("nested type-name collisions did not decode: %#v", root.Nested)
	}

	out, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"\"user-id\":{\"id\":\"first\"}",
		"\"user_id\":{\"id\":2}",
		"\"UserID\":{\"id\":true}",
		"\"x-extra\":\"extra\"",
		"\"type\":\"card_duplicate\"",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("missing %s in output: %s", want, out)
		}
	}
}
`)
}

func TestNameCollisionGeneratedConformanceCompactDelimiter(t *testing.T) {
	file := renderNameCollisions(t,
		WithNestedTypeNameDelimiter(""),
		WithEnumConstants(true),
	)
	assertParsesCompilesAndTests(t, file.Source, `package models

import "testing"

func TestCompactNestedDelimiterGeneratedModels(t *testing.T) {
	nested := CollisionRootNested{
		ValueID:    "nested-id",
		ValueID__2: 5,
		DupObj:     &CollisionRootNestedDupObj{},
		DupObj__2:  &CollisionRootNestedDupObj__2{},
		InlineItem: &CollisionRootNestedInlineItem{
			Func: stringPtr("inner"),
		},
	}
	if nested.InlineItem == nil || nested.InlineItem.Func == nil || *nested.InlineItem.Func != "inner" {
		t.Fatalf("compact delimiter nested names did not compile: %#v", nested)
	}
}

func stringPtr(value string) *string {
	return &value
}
`)
}
