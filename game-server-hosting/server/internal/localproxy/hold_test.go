package localproxy

import (
	"context"
	"testing"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/model"
	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/internal/localproxytest"
	"github.com/stretchr/testify/require"
)

func Test_HoldSelf(t *testing.T) {
	t.Parallel()

	svr, err := localproxytest.NewLocalProxy()
	require.NoError(t, err)
	defer svr.Close()

	chanError := make(chan error, 1)
	c, err := New(svr.Host, 1, chanError)
	require.NoError(t, err)

	resp, err := c.HoldSelf(context.Background(), &model.HoldRequest{
		Timeout: "10m",
	})
	require.NoError(t, err)
	require.Equal(t, svr.HoldStatus, resp)
}

func Test_HoldStatus(t *testing.T) {
	t.Parallel()

	svr, err := localproxytest.NewLocalProxy()
	require.NoError(t, err)
	defer svr.Close()

	chanError := make(chan error, 1)
	c, err := New(svr.Host, 1, chanError)
	require.NoError(t, err)

	resp, err := c.HoldStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, svr.HoldStatus, resp)
}
