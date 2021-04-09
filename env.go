package inproc

import (
	"sync"

	"github.com/multiformats/go-multiaddr"
)

var globalEnv = NewEnv()

type Env interface {
	sync.Locker
	Bind(multiaddr.Multiaddr, *Transport) bool
	Lookup(multiaddr.Multiaddr) (*Transport, bool)
	Free(multiaddr.Multiaddr)
}

func NewEnv() Env { return &mapEnv{bs: make(map[string]*Transport)} }

type mapEnv struct {
	sync.Mutex
	bs map[string]*Transport
}

func (env *mapEnv) Bind(ma multiaddr.Multiaddr, t *Transport) bool {
	if _, ok := env.bs[ma.String()]; ok {
		return false
	}

	env.bs[ma.String()] = t
	return true
}

func (env *mapEnv) Lookup(ma multiaddr.Multiaddr) (*Transport, bool) {
	t, ok := env.bs[ma.String()]
	return t, ok
}

func (env *mapEnv) Free(ma multiaddr.Multiaddr) { delete(env.bs, ma.String()) }
