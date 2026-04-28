package models

import (
	"encoding/json"
	"fmt"
)

type CollisionRootNestedDupObj struct {
	Name *string `json:"name,omitempty"`
}

type CollisionRootNestedDupObj__2 struct {
	Count *int `json:"count,omitempty"`
}

type CollisionRootNestedInlineItem struct {
	Func *string `json:"func,omitempty"`
}

type CollisionRootNested struct {
	ValueID    string                         `json:"value-id"`
	ValueID__2 int                            `json:"value_id"`
	DupObj     *CollisionRootNestedDupObj     `json:"dup-obj,omitempty"`
	DupObj__2  *CollisionRootNestedDupObj__2  `json:"dup_obj,omitempty"`
	InlineItem *CollisionRootNestedInlineItem `json:"inline-item,omitempty"`
}

type CollisionRoot struct {
	Type                    string               `json:"type"`
	UserID                  UserID               `json:"user-id"`
	UserID__2               UserID__2            `json:"user_id"`
	UserID__3               UserID__3            `json:"UserID"`
	Map                     Map                  `json:"map"`
	Choice                  ChoiceUnion          `json:"choice"`
	Recursive               *RecursiveNode       `json:"recursive,omitempty"`
	MapRef                  StringMap            `json:"map_ref"`
	Alias                   AliasValue           `json:"alias"`
	Enum                    EnumCollision        `json:"enum"`
	AdditionalProperties    *string              `json:"additional_properties,omitempty"`
	Nested                  *CollisionRootNested `json:"nested,omitempty"`
	AdditionalProperties__2 map[string]string    `json:"-"`
}

func (m *CollisionRoot) UnmarshalJSON(data []byte) error {
	type Alias CollisionRoot
	var known Alias
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	*m = CollisionRoot(known)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	delete(raw, "type")
	delete(raw, "user-id")
	delete(raw, "user_id")
	delete(raw, "UserID")
	delete(raw, "map")
	delete(raw, "choice")
	delete(raw, "recursive")
	delete(raw, "map_ref")
	delete(raw, "alias")
	delete(raw, "enum")
	delete(raw, "additional_properties")
	delete(raw, "nested")
	if len(raw) == 0 {
		return nil
	}
	m.AdditionalProperties__2 = make(map[string]string, len(raw))
	for key, value := range raw {
		var decoded string
		if err := json.Unmarshal(value, &decoded); err != nil {
			return err
		}
		m.AdditionalProperties__2[key] = decoded
	}
	return nil
}

func (m CollisionRoot) MarshalJSON() ([]byte, error) {
	type Alias CollisionRoot
	encoded, err := json.Marshal(Alias(m))
	if err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &object); err != nil {
		return nil, err
	}
	for key, value := range m.AdditionalProperties__2 {
		encodedValue, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		object[key] = encodedValue
	}
	return json.Marshal(object)
}

type UserID struct {
	ID string `json:"id"`
}

type UserID__2 struct {
	ID int `json:"id"`
}

type UserID__3 struct {
	ID bool `json:"id"`
}

type Map struct {
	Func      string `json:"func"`
	Type      string `json:"type"`
	Interface string `json:"interface"`
}

type Choice interface {
	isChoice()
}

func (ChoiceCard) isChoice() {}

func (ChoiceCard__2) isChoice() {}

type ChoiceUnion struct {
	Value Choice
}

func (u ChoiceUnion) MarshalJSON() ([]byte, error) {
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

func (u ChoiceUnion) IsZero() bool {
	return u.Value == nil
}

func (u *ChoiceUnion) UnmarshalJSON(data []byte) error {
	var discriminator struct {
		Value string `json:"type"`
	}
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return err
	}
	switch discriminator.Value {
	case "card":
		var v ChoiceCard
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = v
	case "card_duplicate":
		var v ChoiceCard__2
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = v
	default:
		return fmt.Errorf("unknown type discriminator value %q", discriminator.Value)
	}
	return nil
}

type ChoiceCard struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ChoiceCard__2 struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

type RecursiveNode struct {
	Next     *RecursiveNode  `json:"next,omitempty"`
	Children []RecursiveNode `json:"children,omitempty"`
}

type StringMap struct {
	AdditionalProperties map[string]string `json:"-"`
}

func (m *StringMap) UnmarshalJSON(data []byte) error {
	type Alias StringMap
	var known Alias
	if err := json.Unmarshal(data, &known); err != nil {
		return err
	}
	*m = StringMap(known)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		return nil
	}
	m.AdditionalProperties = make(map[string]string, len(raw))
	for key, value := range raw {
		var decoded string
		if err := json.Unmarshal(value, &decoded); err != nil {
			return err
		}
		m.AdditionalProperties[key] = decoded
	}
	return nil
}

func (m StringMap) MarshalJSON() ([]byte, error) {
	type Alias StringMap
	encoded, err := json.Marshal(Alias(m))
	if err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &object); err != nil {
		return nil, err
	}
	for key, value := range m.AdditionalProperties {
		encodedValue, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		object[key] = encodedValue
	}
	return json.Marshal(object)
}

type AliasValue string

type EnumCollision string

const (
	EnumCollisionEmpty         EnumCollision = ""
	EnumCollisionInProgress    EnumCollision = "in-progress"
	EnumCollisionInProgress__2 EnumCollision = "in progress"
	EnumCollisionTrue          EnumCollision = "true"
	EnumCollisionTrue__2       EnumCollision = "True"
	EnumCollisionValue15       EnumCollision = "1.5"
	EnumCollisionValue15__2    EnumCollision = "1_5"
	EnumCollisionValue15__3    EnumCollision = "1-5"
)
