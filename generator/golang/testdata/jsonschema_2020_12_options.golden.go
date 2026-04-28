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

const (
	StringEnumDraft     StringEnum = "draft"
	StringEnumPublished StringEnum = "published"
)

type IntEnum int

const (
	IntEnumValue1 IntEnum = 1
	IntEnumValue2 IntEnum = 2
)

type FloatEnum float64

const (
	FloatEnumValue15 FloatEnum = 1.5
	FloatEnumValue2  FloatEnum = 2
)

type BoolEnum bool

const (
	BoolEnumTrue  BoolEnum = true
	BoolEnumFalse BoolEnum = false
)

type NullableStatus string

const (
	NullableStatusActive   NullableStatus = "active"
	NullableStatusInactive NullableStatus = "inactive"
)

type MixedEnum any

type ClosedConfig struct {
	Enabled   bool     `json:"enabled"`
	Threshold *float64 `json:"threshold,omitempty"`
}

type StringMap struct {
	AdditionalProperties map[string]string `json:"-"`
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
