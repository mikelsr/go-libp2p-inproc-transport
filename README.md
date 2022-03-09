# go-libp2p-inproc-transport

[![GoDoc](https://godoc.org/github.com/lthibault/go-libp2p-inproc-transport?status.svg)](https://godoc.org/github.com/lthibault/go-libp2p-inproc-transport)
[![](https://img.shields.io/badge/project-libp2p-yellow.svg?style=flat-square)](https://libp2p.io/)

An in-process transport for go-libp2p, suitable for testing.

## Installation

```bash
go get -u github.com/lthibault/go-libp2p-inproc-transport
```

## Usage

```go

h, _ := libp2p.New(
  libp2p.Transport(inproc.New()),
  libp2p.ListenAddrString("/inproc/foo"))

// host is reachable at /inproc/foo 
```

**Note:** Users may listen on `/inproc/~` to bind to the first available address.  This is equivalent to `/ip4/0.0.0.0`.

## Stability

As of `v0.1.0`, `go-libp2p-inproc-transport` is considered stable and production-ready.  We will tag a `v1.0` release when `go-libp2p` and `go-libp2p-core` have stable releases.

In the meantime, we reserve the right to make backwards-incompatible changes in order to keep up with changes to these APIs.  Such changes will be accompanied by a minor version increment.
