package inproc

import (
	"errors"
	"net"
	"strings"

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

	manet.RegisterNetCodec(&manet.NetCodec{
		NetAddrNetworks:  []string{prefix},
		ProtocolName:     prefix,
		ParseNetAddr:     toInprocMultiaddr,
		ConvertMultiaddr: toInprocNetAddr,
		Protocol:         proto,
	})
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
