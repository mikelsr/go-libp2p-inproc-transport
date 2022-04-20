package inproc

import (
	"context"
	"errors"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/transport"
	"github.com/multiformats/go-multiaddr"
)

var _ transport.CapableConn = (*conn)(nil)

type conn struct {
	l      *listener
	remote *conn

	cq     chan struct{}
	accept chan *pipe
}

func (remote *listener) newConnPair(local *listener) (*conn, *conn) {
	lc, rc := newConn(local), newConn(remote)
	lc.remote = rc
	rc.remote = lc

	return lc, rc
}

func newConn(l *listener) *conn {
	return &conn{
		l:      l,
		cq:     make(chan struct{}),
		accept: make(chan *pipe),
	}
}

/* MuxedConn */

// Close closes the stream muxer and the the underlying net.Conn.
func (c *conn) Close() error {
	select {
	case <-c.cq:
	default:
		close(c.cq)
	}
	return nil
}

func (c *conn) IsClosed() bool {
	select {
	case <-c.cq:
		return true
	default:
		return false
	}
}

// OpenStream creates a new stream.
func (c *conn) OpenStream(ctx context.Context) (network.MuxedStream, error) {
	local, remote := newPipe()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case c.remote.accept <- remote:
		return local, nil
	}
}

// AcceptStream accepts a stream opened by the other side.
func (c *conn) AcceptStream() (network.MuxedStream, error) {
	select {
	case <-c.cq:
		return nil, errors.New("closed")
	case s := <-c.accept:
		return s, nil
	}
}

func (c *conn) Scope() network.ConnScope {
	return network.NullScope
}

/* ConnSecurity */

func (c *conn) LocalPeer() peer.ID  { return c.l.t.h.ID() }
func (c *conn) RemotePeer() peer.ID { return c.remote.LocalPeer() }

func (c *conn) LocalPrivateKey() crypto.PrivKey { return c.l.t.pk }
func (c *conn) RemotePublicKey() crypto.PubKey  { return c.remote.l.t.pk.GetPublic() }

/* ConnMultiaddrs */

func (c *conn) LocalMultiaddr() multiaddr.Multiaddr  { return c.l.Multiaddr() }
func (c *conn) RemoteMultiaddr() multiaddr.Multiaddr { return c.remote.l.Multiaddr() }

/* Transport */

func (c *conn) Transport() transport.Transport { return c.l.t }
