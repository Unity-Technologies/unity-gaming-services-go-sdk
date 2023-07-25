package model

type (
	// HoldRequest defines the model for the request to hold a server.
	HoldRequest struct {
		// The duration of the server hold. Formatted as a duration string with a sequence of numbers and time units (e.g. 2m / 1h).
		// Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Holds are stored at a per-second granularity.
		Timeout string `json:"timeout"`
	}

	// HoldStatus defines the model for the status of server hold, returned from a successful hold request or status request.
	HoldStatus struct {
		// The unix epoch when the hold will automatically expire, in seconds.
		ExpiresAt int64 `json:"expiresAt"`
		// Whether the server is currently held.
		Held bool `json:"held"`
	}
)
