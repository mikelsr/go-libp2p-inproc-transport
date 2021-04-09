package inproc

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/transport"
)

// Factory type for inproc.Transport.  The factory type is suitable
// for passing to the libp2p.Transport function.
type Factory func(host.Host, crypto.PrivKey) transport.Transport

// New transport constructor that is suitable for passing to the
// libp2p.Transport function.
func New(opt ...Option) Factory {
	return func(h host.Host, pk crypto.PrivKey) transport.Transport {
		t := &Transport{
			h:  h,
			pk: pk,
			ls: make(map[string]*listener),
		}

		for _, option := range withDefaults(opt) {
			option(t)
		}

		return t
	}
}

// Option type for Transport.
type Option func(*Transport)

// WithEnv sets the transport's environment.
func WithEnv(env Env) Option {
	return func(t *Transport) {
		t.env = env
	}
}

func withDefaults(opt []Option) []Option {
	return append([]Option{
		WithEnv(globalEnv),
	}, opt...)
}
