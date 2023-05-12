package localproxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_unmarshalEvent(t *testing.T) {
	t.Parallel()

	ev, err := unmarshalEvent([]byte(`{"EventType":"ServerInfoEvent", "EventID": "event-id", "ServerID": 1}`))
	require.NoError(t, err)
	require.Equal(t, &BaseEvent{
		Typ:      ServerInfoEvent,
		ServerID: 1,
		EventID:  "event-id",
	}, ev)

	ev, err = unmarshalEvent([]byte(`{"EventType":"ServerAllocateEvent", "EventID": "event-id", "ServerID": 1, "AllocationID": "alloc-id"}`))
	require.NoError(t, err)
	require.Equal(t, &AllocateEvent{
		BaseEvent: &BaseEvent{
			Typ:      ServerAllocateEvent,
			ServerID: 1,
			EventID:  "event-id",
		},
		AllocationID: "alloc-id",
	}, ev)

	ev, err = unmarshalEvent([]byte(`{"EventType":"ServerDeallocateEvent", "EventID": "event-id", "ServerID": 1, "AllocationID": "alloc-id"}`))
	require.NoError(t, err)
	require.Equal(t, &DeallocateEvent{
		BaseEvent: &BaseEvent{
			Typ:      ServerDeallocateEvent,
			ServerID: 1,
			EventID:  "event-id",
		},
		AllocationID: "alloc-id",
	}, ev)
}
