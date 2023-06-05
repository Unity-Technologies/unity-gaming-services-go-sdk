package model

import "time"

type (
	// ReserveRequest defines the model for the request to reserve a server.
	ReserveRequest struct{}

	// ReserveResponse defines the model for a successful response to a reservation request.
	ReserveResponse struct {
		// BuildConfigurationID is the build configuration this server is using
		BuildConfigurationID int64 `json:"buildConfigurationId"`
		// Creates is the time at which the reservation was made
		Created time.Time `json:"created"`
		// Fulfilled is the time at which the reservation was fulfilled
		Fulfilled time.Time `json:"fulfilled"`
		// GamePort is the port of the server on the requested machine
		GamePort int64 `json:"gamePort"`
		// Ipv4 address of the machine the server is running on
		Ipv4 *string `json:"ipv4,omitempty"`
		// Ipv6 address of the machine the server is running on
		Ipv6 *string `json:"ipv6,omitempty"`
		// Requested is the time at which the reservation was requested
		Requested time.Time `json:"requested"`
		// ReservationID is the UUID of the reservation generated by the service
		ReservationID string `json:"reservationId"`
	}
)
