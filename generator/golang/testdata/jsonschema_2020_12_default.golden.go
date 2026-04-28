package models

import (
	"encoding/json"
	"fmt"
)

type TortureDocument_MultiValueUnion struct {
	Raw json.RawMessage
}

func (u *TortureDocument_MultiValueUnion) UnmarshalJSON(data []byte) error {
	u.Raw = append(u.Raw[:0], data...)
	return nil
}

func (u TortureDocument_MultiValueUnion) MarshalJSON() ([]byte, error) {
	if len(u.Raw) == 0 {
		return []byte("null"), nil
	}
	return u.Raw, nil
}

func (u TortureDocument_MultiValueUnion) IsZero() bool {
	return len(u.Raw) == 0
}

func (u TortureDocument_MultiValueUnion) Bytes() []byte {
	return append([]byte(nil), u.Raw...)
}

type TortureDocument struct {
	// ID readOnly.
	ID   string `json:"id"`
	Kind string `json:"kind"`
	// MultiValue A nullable multi-type value.
	MultiValue     *TortureDocument_MultiValueUnion `json:"multi_value,omitempty"`
	NullableStatus *NullableStatus                  `json:"nullable_status,omitempty"`
	MixedEnum      *MixedEnum                       `json:"mixed_enum,omitempty"`
	StringEnum     *StringEnum                      `json:"string_enum,omitempty"`
	IntEnum        *IntEnum                         `json:"int_enum,omitempty"`
	FloatEnum      *FloatEnum                       `json:"float_enum,omitempty"`
	BoolEnum       *BoolEnum                        `json:"bool_enum,omitempty"`
	ClosedConfig   *ClosedConfig                    `json:"closed_config,omitempty"`
	Labels         *StringMap                       `json:"labels,omitempty"`
	Tuple          *TupleProbe                      `json:"tuple,omitempty"`
	ObjectRules    *ObjectRules                     `json:"object_rules,omitempty"`
	EncodedPayload *EncodedPayload                  `json:"encoded_payload,omitempty"`
	Payment        PaymentSourceUnion               `json:"payment"`
	LooseChoice    *LooseChoiceUnion                `json:"loose_choice,omitempty"`
	DynamicNode    *TreeNode                        `json:"dynamic_node,omitempty"`
}

type StringEnum string

type IntEnum int

type FloatEnum float64

type BoolEnum bool

type NullableStatus string

type MixedEnum any

type ClosedConfig struct {
	Enabled   bool     `json:"enabled"`
	Threshold *float64 `json:"threshold,omitempty"`
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

type TupleProbe []any

type ObjectRules struct {
	Name  *string `json:"name,omitempty"`
	Count *int    `json:"count,omitempty"`
}

type EncodedPayload string

type TreeNode struct {
	Name     *string    `json:"name,omitempty"`
	Children []TreeNode `json:"children,omitempty"`
}

type PaymentSource interface {
	isPaymentSource()
}

func (CardSource) isPaymentSource() {}

func (BankSource) isPaymentSource() {}

type PaymentSourceUnion struct {
	Value PaymentSource
}

func (u PaymentSourceUnion) MarshalJSON() ([]byte, error) {
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

func (u PaymentSourceUnion) IsZero() bool {
	return u.Value == nil
}

func (u *PaymentSourceUnion) UnmarshalJSON(data []byte) error {
	var discriminator struct {
		Value string `json:"object"`
	}
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return err
	}
	switch discriminator.Value {
	case "bank_account":
		var v BankSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = v
	case "card":
		var v CardSource
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = v
	default:
		return fmt.Errorf("unknown object discriminator value %q", discriminator.Value)
	}
	return nil
}

type CardSource struct {
	Object string `json:"object"`
	Number string `json:"number"`
	// CVC writeOnly.
	CVC string `json:"cvc"`
}

type BankSource struct {
	Object        string  `json:"object"`
	AccountNumber string  `json:"account_number"`
	BankName      *string `json:"bank_name,omitempty"`
}

type LooseChoiceUnion struct {
	Raw json.RawMessage
}

func (u *LooseChoiceUnion) UnmarshalJSON(data []byte) error {
	u.Raw = append(u.Raw[:0], data...)
	return nil
}

func (u LooseChoiceUnion) MarshalJSON() ([]byte, error) {
	if len(u.Raw) == 0 {
		return []byte("null"), nil
	}
	return u.Raw, nil
}

func (u LooseChoiceUnion) IsZero() bool {
	return len(u.Raw) == 0
}

func (u LooseChoiceUnion) Bytes() []byte {
	return append([]byte(nil), u.Raw...)
}
