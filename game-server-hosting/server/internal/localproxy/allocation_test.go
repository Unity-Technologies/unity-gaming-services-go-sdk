package localproxy

import (
	"context"
	"testing"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/model"
	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/internal/localproxytest"
	"github.com/stretchr/testify/require"
)

func Test_PatchAllocation(t *testing.T) {
	t.Parallel()

	proxy, err := localproxytest.NewLocalProxy()
	require.NoError(t, err, "creating local proxy")
	defer proxy.Close()

	chanError := make(chan error, 1)

	c, err := New(proxy.Host, 1, chanError)
	require.NoError(t, err)

	args := &model.PatchAllocationRequest{
		Ready: true,
	}

	ctx := context.Background()
	alloc := "00000001-0000-0000-0000-000000000000"

	require.NoError(t, c.PatchAllocation(ctx, alloc, args), "patching allocation")
	require.NotNil(t, proxy.PatchAllocationRequest, "nil patch allocation request")
	require.Equal(t, true, proxy.PatchAllocationRequest.Ready, "unexpected ready value")
}
