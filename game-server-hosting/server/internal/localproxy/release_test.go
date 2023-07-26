package localproxy

import (
	"context"
	"testing"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/internal/localproxytest"
	"github.com/stretchr/testify/require"
)

func Test_ReleaseSelf(t *testing.T) {
	t.Parallel()

	svr, err := localproxytest.NewLocalProxy()
	require.NoError(t, err)
	defer svr.Close()

	chanError := make(chan error, 1)
	c, err := New(svr.Host, 1, chanError)
	require.NoError(t, err)

	require.NoError(t, c.ReleaseSelf(context.Background()))
}
