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

h, _ := libp2p.New(context.Context(),
  libp2p.Transport(inproc.New()),
  libp2p.ListenAddrString("/inproc/foo"))

// host is reachable at /inproc/foo 
```
