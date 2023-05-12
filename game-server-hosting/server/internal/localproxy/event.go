package localproxy

import "encoding/json"

type (
	// Event represents the interface for any event received by the subscriber.
	Event interface {
		Type() EventType
	}

	// EventType is a type alias for an event received by the subscriber.
	EventType string

	// BaseEvent represents the base structure of all events received by the subscriber.
	BaseEvent struct {
		Typ      EventType `json:"EventType"`
		EventID  string    `json:"EventID"`
		ServerID int64     `json:"ServerID"`
	}

	// AllocateEvent represents the data received on an allocation event.
	AllocateEvent struct {
		*BaseEvent
		AllocationID string `json:"AllocationID"`
	}

	// DeallocateEvent represents the data received on a deallocation event.
	DeallocateEvent struct {
		*BaseEvent
		AllocationID string `json:"AllocationID"`
	}
)

const (
	// AllocateEventType represents an event received by the server when it is allocated.
	AllocateEventType = EventType("AllocateEventType")

	// DeallocateEventType represents an event received by the server when it is deallocated.
	DeallocateEventType = EventType("DeallocateEventType")
)

// Type returns the type of the event.
func (b *BaseEvent) Type() EventType {
	return b.Typ
}

// unmarshalEvent unmarshals the provided data into a data structure based upon its type. If the type is not supported,
// a BaseEvent is returned instead of an error.
func unmarshalEvent(data []byte) (Event, error) {
	var event *BaseEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}

	switch event.Type() {
	case AllocateEventType:
		var ae *AllocateEvent
		if err := json.Unmarshal(data, &ae); err != nil {
			return nil, err
		}

		return ae, nil

	case DeallocateEventType:
		var de *DeallocateEvent
		if err := json.Unmarshal(data, &de); err != nil {
			return nil, err
		}

		return de, nil

	default:
		return event, nil
	}
}
