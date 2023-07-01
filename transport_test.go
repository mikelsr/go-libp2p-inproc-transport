package inproc_test

import (
	"context"
	"testing"

	"github.com/mikelsr/go-libp2p"
	inproc "github.com/mikelsr/go-libp2p-inproc-transport"
	"github.com/mikelsr/go-libp2p/core/host"
	"github.com/mikelsr/go-libp2p/core/network"
	"github.com/stretchr/testify/require"
)

func TestReset(t *testing.T) {
	t.Parallel()

	env := inproc.NewEnv()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h0, err := newTestHost(env)
	require.NoError(t, err)
	defer h0.Close()

	h1, err := newTestHost(env)
	require.NoError(t, err)
	defer h1.Close()

	err = h1.Connect(ctx, *host.InfoFromHost(h0))
	require.NoError(t, err)

	sync := make(chan struct{})

	t.Run("ListenerReset", func(t *testing.T) {
		h0.SetStreamHandler("/test/reset/listener", func(s network.Stream) {
			defer s.Close()
			<-sync
			require.NoError(t, s.Reset())
			<-ctx.Done() // block until test finishes
		})

		s, err := h1.NewStream(ctx, h0.ID(), "/test/reset/listener")
		require.NoError(t, err)

		close(sync)

		_, err = s.Read(make([]byte, 1))
		require.ErrorIs(t, err, network.ErrReset)

		_, err = s.Write(make([]byte, 1))
		require.ErrorIs(t, err, network.ErrReset)
	})

	t.Run("DialerReset", func(t *testing.T) {
		h0.SetStreamHandler("/test/reset/dialer", func(s network.Stream) {
			defer s.Close()

			_, err = s.Read(make([]byte, 1))
			require.ErrorIs(t, err, network.ErrReset)

			_, err = s.Write(make([]byte, 1))
			require.ErrorIs(t, err, network.ErrReset)
		})

		s, err := h1.NewStream(ctx, h0.ID(), "/test/reset/dialer")
		require.NoError(t, err)
		require.NoError(t, s.Reset())
	})
}

func newTestHost(env inproc.Env) (host.Host, error) {
	return libp2p.New(
		libp2p.NoTransports,
		libp2p.Transport(inproc.New(inproc.WithEnv(env))),
		libp2p.ListenAddrStrings("/inproc/~")) // auto-bind
}
