package localproxy

import (
	"context"
	"testing"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/model"
	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/internal/localproxytest"
	"github.com/stretchr/testify/require"
)

func Test_ReserveSelf(t *testing.T) {
	t.Parallel()

	svr, err := localproxytest.NewLocalProxy()
	require.NoError(t, err)
	defer svr.Close()

	chanError := make(chan error, 1)
	c, err := New(svr.Host, 1, chanError)
	require.NoError(t, err)

	resp, err := c.ReserveSelf(context.Background(), &model.ReserveRequest{})
	require.NoError(t, err)
	require.Equal(t, svr.ReserveResponse, resp)
}
