package inproc_test

import (
	"testing"

	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	"github.com/multiformats/go-multiaddr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBind(t *testing.T) {
	t.Parallel()

	env := inproc.NewEnv()
	ma := multiaddr.StringCast("/inproc/test")

	t.Run("AddrIsFree", func(t *testing.T) {
		require.True(t, env.Bind(ma, &inproc.Transport{}),
			"failed to bind transport to free address")

		tpt, ok := env.Lookup(ma)
		assert.True(t, ok)
		assert.NotNil(t, tpt)
	})

	t.Run("AddrInUse", func(t *testing.T) {
		require.False(t, env.Bind(ma, &inproc.Transport{}),
			"overwrote bound address")
	})

	t.Run("Free", func(t *testing.T) {
		env.Free(ma)
		tpt, ok := env.Lookup(ma)
		assert.False(t, ok)
		assert.Nil(t, tpt)
	})
}
