package models

import "encoding/json"

// Station is a train station.
type Station struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Address     string  `json:"address"`
	CountryCode string  `json:"country_code"`
	Timezone    *string `json:"timezone,omitempty"`
}

// Trip is a train trip.
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

// Booking is a booking for a train trip.
type Booking struct {
	// Read-only: the API returns this value; requests should not send it.
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

type BookingPayment_Status string

// BookingPayment is a payment for a booking.
type BookingPayment struct {
	// Read-only: the API returns this value; requests should not send it.
	ID       *string                  `json:"id,omitempty"`
	Amount   *float64                 `json:"amount,omitempty"`
	Currency *BookingPayment_Currency `json:"currency,omitempty"`
	// The payment source to take the payment from.
	Source *BookingPayment_SourceUnion `json:"source,omitempty"`
	// Read-only: the API returns this value; requests should not send it.
	Status *BookingPayment_Status `json:"status,omitempty"`
}
