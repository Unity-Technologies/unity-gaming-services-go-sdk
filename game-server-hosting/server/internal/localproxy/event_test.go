package localproxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_unmarshalEvent(t *testing.T) {
	t.Parallel()

	ev, err := unmarshalEvent([]byte(`{"EventType":"AllocateEventType", "EventID": "event-id", "ServerID": 1, "AllocationID": "alloc-id"}`))
	require.NoError(t, err)
	require.Equal(t, &AllocateEvent{
		BaseEvent: &BaseEvent{
			Typ:      AllocateEventType,
			ServerID: 1,
			EventID:  "event-id",
		},
		AllocationID: "alloc-id",
	}, ev)

	ev, err = unmarshalEvent([]byte(`{"EventType":"DeallocateEventType", "EventID": "event-id", "ServerID": 1, "AllocationID": "alloc-id"}`))
	require.NoError(t, err)
	require.Equal(t, &DeallocateEvent{
		BaseEvent: &BaseEvent{
			Typ:      DeallocateEventType,
			ServerID: 1,
			EventID:  "event-id",
		},
		AllocationID: "alloc-id",
	}, ev)
}
