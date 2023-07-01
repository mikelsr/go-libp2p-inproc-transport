package inproc_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/mikelsr/go-libp2p"
	inproc "github.com/mikelsr/go-libp2p-inproc-transport"
	"github.com/mikelsr/go-libp2p/core/host"
	"github.com/mikelsr/go-libp2p/core/network"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	t.Parallel()

	t.Run("SymmetricListen", func(t *testing.T) {
		t.Parallel()

		tpt := inproc.New(inproc.WithEnv(inproc.NewEnv()))

		h0, err := libp2p.New(
			libp2p.Transport(tpt),
			libp2p.ListenAddrStrings("/inproc/h0"))
		require.NoError(t, err)

		h1, err := libp2p.New(
			libp2p.Transport(tpt),
			libp2p.ListenAddrStrings("/inproc/h1"))
		require.NoError(t, err)

		testFunc(t, h0, h1)
	})

	t.Run("AsymmetricListen", func(t *testing.T) {
		t.Parallel()

		tpt := inproc.New(inproc.WithEnv(inproc.NewEnv()))

		h0, err := libp2p.New(
			libp2p.Transport(tpt),
			libp2p.ListenAddrStrings("/inproc/h0"))
		require.NoError(t, err)

		h1, err := libp2p.New(
			libp2p.Transport(tpt),
			libp2p.NoListenAddrs)
		require.NoError(t, err)

		testFunc(t, h0, h1)
	})
}

func testFunc(t *testing.T, h0, h1 host.Host) {
	defer func() {
		require.NoError(t, h0.Close())
		require.NoError(t, h1.Close())
	}()

	h0.SetStreamHandler("/test", func(s network.Stream) {
		defer func() { require.NoError(t, s.Close()) }()
		io.Copy(s, bytes.NewBufferString("hello, world!"))
	})

	err := h1.Connect(context.Background(), *host.InfoFromHost(h0))
	require.NoError(t, err)

	s, err := h1.NewStream(context.Background(), h0.ID(), "/test")
	require.NoError(t, err)
	defer s.Close()

	var buf bytes.Buffer
	io.Copy(&buf, s)
	require.Equal(t, "hello, world!", buf.String())
}
