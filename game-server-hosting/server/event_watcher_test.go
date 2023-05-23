package server

import (
	"testing"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/internal/localproxytest"
	"github.com/stretchr/testify/require"
)

func Test_listenForEvents(t *testing.T) {
	t.Parallel()

	s, err := New(TypeAllocation)
	require.NoError(t, err)

	svr, err := localproxytest.NewLocalProxy()
	require.NoError(t, err)
	defer svr.Close()

	s.currentConfig = Config{
		LocalProxyURL: svr.Host,
		ServerID:      "1234",
	}

	go s.listenForEvents()
	<-s.eventWatcherReady

	go func() {
		channel := "server#1234"

		// Publish an allocation
		_, err = svr.Node.Publish(channel, []byte(`{"EventType":"AllocateEventType", "EventID": "event-id", "ServerID": 1, "AllocationID": "alloc-id"}`))
		require.NoError(t, err)

		// Publish a deallocation
		_, err = svr.Node.Publish(channel, []byte(`{"EventType":"DeallocateEventType", "EventID": "event-id", "ServerID": 1, "AllocationID": "alloc-id"}`))
		require.NoError(t, err)
	}()

	require.Equal(t, "alloc-id", <-s.OnAllocate())
	require.Equal(t, "alloc-id", <-s.OnDeallocate())
	close(s.done)
}

func Test_listenForEvents_badServerID(t *testing.T) {
	t.Parallel()

	s, err := New(TypeAllocation)
	require.NoError(t, err)

	s.currentConfig = Config{
		ServerID: "NaN",
	}

	go s.listenForEvents()
	err = <-s.eventWatcherReady
	require.ErrorContains(t, err, "error parsing server ID")
}
