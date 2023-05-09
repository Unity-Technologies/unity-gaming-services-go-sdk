package server

import (
	"errors"
)

type (
	// BackfillTicket represents a backfill ticket.
	// Documentation: https://services.docs.unity.com/matchmaker/v2/index.html#tag/Backfill/operation/approveBackfillTicket
	BackfillTicket struct {
		// ID represents the backfill ticket ID.
		ID string

		// Connection represents the IP address and port of the server that creates the backfill.
		// The IP address format is ip:port.
		Connection string

		// Attributes represents an object that holds a dictionary of attributes (number or string),
		// indexed by the attribute name. The attributes are compared against the corresponding filters
		// defined in the matchmaking config and used to segment the ticket population into pools.
		// The default pool is used if a pool isn't provided.
		Attributes map[string]float64
	}
)

var (
	// ErrBackfillApprove is an error which denotes that backfill approval process has failed.
	ErrBackfillApprove = errors.New("failed to approve backfill ticket")

	// ErrNotAllocated is an error which denotes the action cannot be performed as the server is not yet allocated.
	ErrNotAllocated = errors.New("server is not allocated")
)
