package localproxy

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/internal/localproxytest"
	"github.com/stretchr/testify/require"
)

func Test_Client_Lifecycle(t *testing.T) {
	t.Parallel()

	svr, err := localproxytest.NewLocalProxy()
	require.NoError(t, err)
	defer svr.Close()

	chanError := make(chan error, 1)
	c, err := New(svr.Host, 1, chanError)
	require.NoError(t, err)

	allocateCalls := int32(0)
	c.RegisterCallback(AllocateEventType, func(ev Event) {
		atomic.AddInt32(&allocateCalls, 1)
	})

	deallocateCalls := int32(0)
	c.RegisterCallback(DeallocateEventType, func(ev Event) {
		atomic.AddInt32(&deallocateCalls, 1)
	})

	require.NoError(t, c.Start())

	channel := "server#1"

	// Publish an allocation
	_, err = svr.Node.Publish(channel, []byte(`{"EventType":"AllocateEventType", "EventID": "event-id", "ServerID": 1, "AllocationID": "alloc-id"}`))
	require.NoError(t, err)

	// Publish a deallocation
	_, err = svr.Node.Publish(channel, []byte(`{"EventType":"DeallocateEventType", "EventID": "event-id", "ServerID": 1, "AllocationID": "alloc-id"}`))
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&allocateCalls) == 1
	}, 2*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&deallocateCalls) == 1
	}, 2*time.Second, 100*time.Millisecond)

	require.NoError(t, c.Stop())
}
