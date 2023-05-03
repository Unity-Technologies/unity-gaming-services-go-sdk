package server

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_watchConfig(t *testing.T) {
	t.Parallel()

	p := path.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(p, []byte(`{}`), 0o600))

	g, err := New(TypeAllocation)
	g.cfgFile = p
	require.NoError(t, err)
	require.NotNil(t, g)

	go g.processInternalEvents()
	<-g.internalEventProcessorReady

	// Allocate
	require.NoError(t, os.WriteFile(p, []byte(`{
		"allocatedUUID": "alloc-uuid",
		"maxPlayers": "12"
	}`), 0o600))
	ev := <-g.OnAllocate()
	require.Equal(t, "alloc-uuid", ev.AllocatedUUID)
	require.Equal(t, "12", ev.Extra["maxPlayers"])
	require.Equal(t, QueryProtocolSQP, ev.QueryType)

	// Deallocate
	require.NoError(t, os.WriteFile(p, []byte(`{
		"allocatedUUID": ""
	}`), 0o600))
	ev = <-g.OnDeallocate()
	require.Equal(t, QueryProtocolSQP, ev.QueryType)
	require.Equal(t, "", ev.AllocatedUUID)

	close(g.done)
}
