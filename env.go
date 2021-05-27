package inproc

import (
	"bytes"
	"sync"

	"github.com/multiformats/go-multiaddr"
)

var globalEnv = NewEnv()

// Env encapsulates bindings in an isolated address space.
// The caller is responsible for explicit locking during calls
// to 'Bind', 'Lookup' and 'Free'.
//
// Calling 'List' while holding a lock on Env will cause a deadlock.
type Env interface {
	sync.Locker
	Bind(multiaddr.Multiaddr, *Transport) bool
	Lookup(multiaddr.Multiaddr) (*Transport, bool)
	Free(multiaddr.Multiaddr)
	List() AddrSlice
}

// NewEnv returns a new instance of the default Env implementation.
func NewEnv() Env { return &mapEnv{bs: make(map[string]*record)} }

type mapEnv struct {
	sync.RWMutex
	bs map[string]*record
}

func (env *mapEnv) Bind(ma multiaddr.Multiaddr, t *Transport) bool {
	if _, ok := env.bs[ma.String()]; ok {
		return false
	}

	env.bs[ma.String()] = &record{Addr: ma, T: t}
	return true
}

func (env *mapEnv) Lookup(ma multiaddr.Multiaddr) (*Transport, bool) {
	if rec, ok := env.bs[ma.String()]; ok {
		return rec.T, ok
	}

	return nil, false
}

func (env *mapEnv) Free(ma multiaddr.Multiaddr) { delete(env.bs, ma.String()) }

func (env *mapEnv) List() AddrSlice {
	env.RLock()
	defer env.RUnlock()

	addrs := make(AddrSlice, 0, len(env.bs))
	for _, rec := range env.bs {
		addrs = append(addrs, rec.Addr)
	}

	return addrs
}

type record struct {
	Addr multiaddr.Multiaddr
	T    *Transport
}

type AddrSlice []multiaddr.Multiaddr

func (as AddrSlice) Len() int      { return len(as) }
func (as AddrSlice) Swap(i, j int) { as[i], as[j] = as[j], as[i] }
func (as AddrSlice) Less(i, j int) bool {
	return bytes.Compare(as[i].Bytes(), as[j].Bytes()) < 0
}
