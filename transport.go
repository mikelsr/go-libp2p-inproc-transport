package inproc

import (
	"context"
	"errors"
	"sync"

	"github.com/mikelsr/go-libp2p/core/crypto"
	"github.com/mikelsr/go-libp2p/core/host"
	"github.com/mikelsr/go-libp2p/core/peer"
	"github.com/mikelsr/go-libp2p/core/transport"
	"github.com/multiformats/go-multiaddr"
)

var _ transport.Transport = (*Transport)(nil)

var (
	// ErrInUse is returnd when binding to an address that is already in
	// use.
	ErrInUse = errors.New("address in use")

	// ErrRefused is returned when dialing an address on which a peer is
	// not accepting connections.
	ErrRefused = errors.New("connection refused")
)

// Transport for fast in-process communication.
type Transport struct {
	env Env

	h  host.Host
	pk crypto.PrivKey

	mu sync.RWMutex
	ls map[string]*listener
}

// Dial dials a remote peer. It should try to reuse local listener
// addresses if possible but it may choose not to.
func (t *Transport) Dial(ctx context.Context, raddr multiaddr.Multiaddr, _ peer.ID) (transport.CapableConn, error) {
	t.env.Lock() // accept may need to bind a dialback listener
	defer t.env.Unlock()

	if bound, ok := t.env.Lookup(raddr); ok {
		return bound.accept(ctx, raddr, t)
	}

	return nil, ErrRefused
}

// CanDial returns true if this transport knows how to dial the given
// multiaddr.
//
// Returning true does not guarantee that dialing this multiaddr will
// succeed. This function should *only* be used to preemptively filter
// out addresses that we can't dial.
func (t *Transport) CanDial(addr multiaddr.Multiaddr) bool {
	return addr.Protocols()[0].Code == P_INPROC
}

// Listen listens on the passed multiaddr.
func (t *Transport) Listen(laddr multiaddr.Multiaddr) (transport.Listener, error) {
	laddr, err := Resolve(laddr)
	if err != nil {
		return nil, err
	}

	t.env.Lock()
	defer t.env.Unlock()

	if t.env.Bind(laddr, t) {
		return t.newListener(laddr)
	}

	return nil, ErrInUse
}

// Protocol returns the set of protocols handled by this transport.
//
// See the Network interface for an explanation of how this is used.
func (t *Transport) Protocols() []int { return []int{P_INPROC} }

// Proxy returns true if this is a proxy transport.
//
// See the Network interface for an explanation of how this is used.
// TODO: Make this a part of the go-multiaddr protocol instead?
func (t *Transport) Proxy() bool { return false }

/*
 * inproc-specific
 */

func (t *Transport) newListener(laddr multiaddr.Multiaddr) (*listener, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	l := newListener(laddr, t)
	t.ls[laddr.String()] = l

	return l, nil
}

func (t *Transport) accept(ctx context.Context, raddr multiaddr.Multiaddr, dialer *Transport) (*conn, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.ls[raddr.String()].NewConn(ctx, dialer)
}
