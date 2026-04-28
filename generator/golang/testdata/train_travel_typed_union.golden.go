package models

import (
	"encoding/json"
	"fmt"
)

// Station A train station.
type Station struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Address     string  `json:"address"`
	CountryCode string  `json:"country_code"`
	Timezone    *string `json:"timezone,omitempty"`
}

// Trip A train trip.
type Trip struct {
	ID              *string  `json:"id,omitempty"`
	Origin          *string  `json:"origin,omitempty"`
	Destination     *string  `json:"destination,omitempty"`
	DepartureTime   *string  `json:"departure_time,omitempty"`
	ArrivalTime     *string  `json:"arrival_time,omitempty"`
	Price           *float64 `json:"price,omitempty"`
	BicyclesAllowed *bool    `json:"bicycles_allowed,omitempty"`
	DogsAllowed     *bool    `json:"dogs_allowed,omitempty"`
}

// Booking A booking for a train trip.
type Booking struct {
	// ID readOnly.
	ID            *string `json:"id,omitempty"`
	TripID        *string `json:"trip_id,omitempty"`
	PassengerName *string `json:"passenger_name,omitempty"`
	HasBicycle    *bool   `json:"has_bicycle,omitempty"`
	HasDog        *bool   `json:"has_dog,omitempty"`
}

type BookingPayment_Currency string

// BookingPayment_Source_Card A card to take payment from.
type BookingPayment_Source_Card struct {
	Object *string `json:"object,omitempty"`
	Name   string  `json:"name"`
	Number string  `json:"number"`
	// CVC writeOnly.
	CVC            string `json:"cvc"`
	ExpMonth       int64  `json:"exp_month"`
	ExpYear        int64  `json:"exp_year"`
	AddressCountry string `json:"address_country"`
}

type BookingPayment_Source_BankAccount_AccountType string

// BookingPayment_Source_BankAccount A bank account to take payment from.
type BookingPayment_Source_BankAccount struct {
	Object      *string                                       `json:"object,omitempty"`
	Name        string                                        `json:"name"`
	Number      string                                        `json:"number"`
	AccountType BookingPayment_Source_BankAccount_AccountType `json:"account_type"`
	BankName    string                                        `json:"bank_name"`
	Country     string                                        `json:"country"`
}

type BookingPayment_Source interface {
	isBookingPayment_Source()
}

func (BookingPayment_Source_Card) isBookingPayment_Source() {}

func (BookingPayment_Source_BankAccount) isBookingPayment_Source() {}

type BookingPayment_SourceUnion struct {
	Value BookingPayment_Source
}

func (u BookingPayment_SourceUnion) MarshalJSON() ([]byte, error) {
	if u.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(u.Value)
}

func (u BookingPayment_SourceUnion) IsZero() bool {
	return u.Value == nil
}

func (u *BookingPayment_SourceUnion) UnmarshalJSON(data []byte) error {
	var discriminator struct {
		Value string `json:"object"`
	}
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return err
	}
	switch discriminator.Value {
	case "bank_account":
		var v BookingPayment_Source_BankAccount
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = v
	case "card":
		var v BookingPayment_Source_Card
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Value = v
	default:
		return fmt.Errorf("unknown object discriminator value %q", discriminator.Value)
	}
	return nil
}

// BookingPayment_Status readOnly.
type BookingPayment_Status string

// BookingPayment A payment for a booking.
type BookingPayment struct {
	// ID readOnly.
	ID       *string                  `json:"id,omitempty"`
	Amount   *float64                 `json:"amount,omitempty"`
	Currency *BookingPayment_Currency `json:"currency,omitempty"`
	// Source The payment source to take the payment from.
	Source *BookingPayment_SourceUnion `json:"source,omitempty"`
	// Status readOnly.
	Status *BookingPayment_Status `json:"status,omitempty"`
}
