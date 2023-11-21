package model

// PatchAllocationRequest defines the model for the request to patch a server allocation.
type PatchAllocationRequest struct {
	// Ready is the ready state of the server.
	Ready bool `json:"ready"`
}
