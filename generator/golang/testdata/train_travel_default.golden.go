package models

import "encoding/json"

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

type BookingPayment_SourceUnion struct {
	Raw json.RawMessage
}

func (u *BookingPayment_SourceUnion) UnmarshalJSON(data []byte) error {
	u.Raw = append(u.Raw[:0], data...)
	return nil
}

func (u BookingPayment_SourceUnion) MarshalJSON() ([]byte, error) {
	if len(u.Raw) == 0 {
		return []byte("null"), nil
	}
	return u.Raw, nil
}

func (u BookingPayment_SourceUnion) IsZero() bool {
	return len(u.Raw) == 0
}

func (u BookingPayment_SourceUnion) Bytes() []byte {
	return append([]byte(nil), u.Raw...)
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
