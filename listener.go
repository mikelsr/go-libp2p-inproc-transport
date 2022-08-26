package inproc

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/transport"
	"github.com/multiformats/go-multiaddr"
)

var _ transport.Listener = (*listener)(nil)

type listener struct {
	t *Transport

	ma multiaddr.Multiaddr
	na net.Addr

	cq     chan struct{}
	accept chan transport.CapableConn
}

func newListener(ma multiaddr.Multiaddr, t *Transport) *listener {
	na, _ := toInprocNetAddr(ma)

	return &listener{
		ma:     ma,
		na:     na,
		t:      t,
		cq:     make(chan struct{}),
		accept: make(chan transport.CapableConn),
	}
}

func (l listener) Accept() (transport.CapableConn, error) {
	select {
	case <-l.cq:
		return nil, errors.New("closed")
	case conn := <-l.accept:
		return conn, nil
	}
}

func (l listener) Close() error {
	select {
	case <-l.cq:
	default:
		l.t.env.Lock()
		close(l.cq)
		l.t.env.Free(l.ma)
		l.t.env.Unlock()
	}
	return nil
}

func (l listener) Addr() net.Addr                 { return l.na }
func (l listener) Multiaddr() multiaddr.Multiaddr { return l.ma }

/*
 * Used by Transport
 */

func (l listener) NewConn(ctx context.Context, dialer *Transport) (*conn, error) {
	d, err := dialer.dialback()
	if err != nil {
		return nil, err
	}

	local, remote := l.newConnPair(d)

	select {
	case <-l.cq:
		return nil, errors.New("closed")
	case <-ctx.Done():
		return nil, ctx.Err()
	case l.accept <- remote:
		return local, nil
	}
}

func (t *Transport) dialback() (l *listener, err error) {
	// use an existing listener for the dialback, if possible
	if l = t.getRandomListener(); l != nil {
		return
	}

	laddr := newRandomAddr()
	t.env.Bind(laddr, t) // caller already holds the lock

	return t.newListener(laddr)
}

func (t *Transport) getRandomListener() (l *listener) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, l = range t.ls {
		break
	}

	return
}

func newRandomAddr() multiaddr.Multiaddr {
	return multiaddr.StringCast(fmt.Sprintf("/%s/%s", prefix, uuid.New()))
}
