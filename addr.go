package inproc

import (
	"errors"
	"fmt"
	"net"
	"strings"

	syncutil "github.com/lthibault/util/sync"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

const (
	prefix   = "inproc"
	P_INPROC = 2020
)

var proto = multiaddr.Protocol{
	Name:       prefix,
	Code:       P_INPROC,
	VCode:      multiaddr.CodeToVarint(P_INPROC),
	Size:       multiaddr.LengthPrefixedVarSize,
	Transcoder: transcoder{},
}

func init() {
	if err := multiaddr.AddProtocol(proto); err != nil {
		panic(err)
	}

	manet.RegisterFromNetAddr(toInprocMultiaddr)
	manet.RegisterToNetAddr(toInprocNetAddr)
}

var c syncutil.Ctr

// Resolve expands a multiaddress in the form "/inproc/~" to a random
// free address.  It returns all other valid inproc addresses unchanged.
func Resolve(ma multiaddr.Multiaddr) (multiaddr.Multiaddr, error) {
	s, err := ma.ValueForProtocol(P_INPROC)
	if err != nil {
		return nil, err
	}

	if s == "~" {
		ma = multiaddr.StringCast(fmt.Sprintf("/inproc/%016x", c.Incr()))
	}

	return ma, nil
}

// ResolveString expands a multiaddress.  See 'Resolve'.
func ResolveString(addr string) (multiaddr.Multiaddr, error) {
	ma, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}

	return Resolve(ma)
}

func toInprocMultiaddr(na net.Addr) (multiaddr.Multiaddr, error) {
	if a, ok := na.(addr); ok {
		return a.Multiaddr, nil
	}

	return nil, errors.New("invalid address")
}

func toInprocNetAddr(ma multiaddr.Multiaddr) (net.Addr, error) { return addr{ma}, nil }

type addr struct{ multiaddr.Multiaddr }

func (addr) Network() string  { return prefix }
func (a addr) String() string { return strings.TrimPrefix(a.Multiaddr.String(), "/"+prefix) }

type transcoder struct{}

func (transcoder) StringToBytes(s string) ([]byte, error) { return []byte(s), nil }
func (transcoder) BytesToString(b []byte) (string, error) { return string(b), nil }
func (transcoder) ValidateBytes([]byte) error             { return nil }
