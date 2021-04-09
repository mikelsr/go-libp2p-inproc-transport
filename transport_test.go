package inproc_test

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/stretchr/testify/require"
	inproc "github.com/wetware/go-libp2p-inproc-transport"
)

func TestBindInUse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tpt := inproc.New(inproc.WithEnv(inproc.NewEnv()))
	bind := libp2p.ListenAddrStrings("/inproc/test")

	h, err := libp2p.New(ctx, libp2p.Transport(tpt), bind)
	require.NoError(t, err)
	defer func() { require.NoError(t, h.Close()) }()

	_, err = libp2p.New(ctx, libp2p.Transport(tpt), bind)
	require.Error(t, err)
}
