package inproc

import (
	"context"
	"errors"
	"math/rand"
	"sort"
	"sync"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

var _ discovery.Discoverer = (*Discoverer)(nil)

// PeerListProvider returns a slice of currently-bound peers
// in the Discoverer's environment.
type PeerListProvider interface {
	List() AddrSlice
}

// DiscoveryStrategy selects peeers from the environment.
type DiscoveryStrategy interface {
	SetDefaultOptions(*discovery.Options) error
	Select(context.Context, *discovery.Options, PeerListProvider) (AddrSlice, error)
}

// Discoverer satisfies discovery.Discoverer.  It implements dynamic
// a dynamic strategy, which users can use to obtain the desired net
// topology.
type Discoverer struct {
	init     sync.Once
	Env      PeerListProvider
	Strategy DiscoveryStrategy
	validate func(*discovery.Options) error
}

func (d *Discoverer) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	d.init.Do(func() {
		if d.Env == nil {
			d.Env = globalEnv
		}

		if d.Strategy == nil {
			d.Strategy = &SelectRandom{}
		}

		d.validate = func(*discovery.Options) error { return nil }
		if v, ok := d.Strategy.(validator); ok {
			d.validate = v.Validate
		}
	})

	opts, err := d.options(ns, opt)
	if err != nil {
		return nil, err
	}

	as, err := d.Strategy.Select(ctx, opts, d.Env)
	return infochan(as), err
}

func (d *Discoverer) options(ns string, opt []discovery.Option) (*discovery.Options, error) {
	opts := newOptions()
	if err := d.Strategy.SetDefaultOptions(opts); err != nil {
		return nil, err
	}

	for _, option := range opt {
		if err := option(opts); err != nil {
			return nil, err
		}
	}

	return opts, d.validate(opts)
}

type SelectAll struct {
	nopOptionSetter
	sortedPeerLoader // ensure reproducibility in tests
}

func (s SelectAll) Select(_ context.Context, opts *discovery.Options, peers PeerListProvider) (AddrSlice, error) {
	return limit(opts, s.load(peers)), nil
}

type SelectRing struct {
	nopOptionSetter
	sortedPeerLoader // ensure reproducibility in tests
}

func (s SelectRing) Select(_ context.Context, opts *discovery.Options, peers PeerListProvider) (AddrSlice, error) {
	id, ok := peerID(opts)
	if !ok {
		return nil, errors.New("ring topology requires option 'WithPeerID'")
	}

	var (
		as       = s.load(peers)
		neighbor multiaddr.Multiaddr
	)

	for i, ma := range as {
		info, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			return nil, err
		}

		if id != info.ID {
			continue
		}

		// last peer?
		if i == len(as)-1 {
			neighbor = as[0] // wrap around to the beginning of the slice
			break
		}

		neighbor = as[i+1]
	}

	if neighbor == nil {
		return nil, errors.New("peer not in environment")
	}

	return AddrSlice{neighbor}, nil
}

type SelectRandom struct {
	init sync.Once
	Src  rand.Source

	loader
	nopOptionSetter
}

func (r *SelectRandom) Select(_ context.Context, opts *discovery.Options, peers PeerListProvider) (AddrSlice, error) {
	r.init.Do(func() {
		if r.loader = (globalShuffleLoader{}); r.Src != nil {
			r.loader = &shuffleLoader{r: rand.New(r.Src)}
		}
	})

	return limit(opts, r.load(peers)), nil
}

func WithPeerID(id peer.ID) discovery.Option {
	return func(opts *discovery.Options) error {
		opts.Other[keyPeerID] = id
		return nil
	}
}

func infochan(as AddrSlice) <-chan peer.AddrInfo {
	ch := make(chan peer.AddrInfo, len(as))
	defer close(ch)

	for _, ma := range as {
		info, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			panic(err) // as comes from Env. Guaranteed correct.
		}

		ch <- *info
	}

	return ch
}

type validator interface {
	Validate(*discovery.Options) error
}

func newOptions() *discovery.Options {
	return &discovery.Options{Other: make(map[interface{}]interface{})}
}

type key uint8

const (
	keyNamespace key = iota
	keyPeerID
)

func limit(opts *discovery.Options, as AddrSlice) AddrSlice {
	if opts.Limit == 0 || opts.Limit >= len(as) {
		return as
	}

	return as[:opts.Limit]
}

func peerID(opts *discovery.Options) (peer.ID, bool) {
	if v, ok := opts.Other[keyPeerID]; ok {
		return v.(peer.ID), true
	}

	return "", false
}

type nopOptionSetter struct{}

func (nopOptionSetter) SetDefaultOptions(*discovery.Options) error { return nil }

type loader interface {
	load(PeerListProvider) AddrSlice
}

type sortedPeerLoader struct{}

func (sortedPeerLoader) load(peers PeerListProvider) AddrSlice {
	as := peers.List()
	sort.Sort(as)
	return as
}

type globalShuffleLoader struct{}

func (globalShuffleLoader) load(peers PeerListProvider) AddrSlice {
	return loadAndShuffle(peers, rand.Shuffle)
}

type shuffleLoader struct {
	mu sync.Mutex
	r  *rand.Rand
}

func (loader *shuffleLoader) load(peers PeerListProvider) AddrSlice {
	loader.mu.Lock()
	defer loader.mu.Unlock()

	return loadAndShuffle(peers, loader.r.Shuffle)
}

func loadAndShuffle(peers PeerListProvider, shuffle func(int, func(i, j int))) AddrSlice {
	as := sortedPeerLoader{}.load(peers) // needed to make shuffle order reproducible
	shuffle(len(as), as.Swap)
	return as
}
