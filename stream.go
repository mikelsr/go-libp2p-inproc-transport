package inproc

import (
	"errors"
	"net"
	"sync"

	"github.com/libp2p/go-libp2p-core/mux"
)

var _ mux.MuxedStream = (*stream)(nil)

type stream struct {
	c *conn
	net.Conn

	r, w   sync.Mutex
	rc, wc bool
}

func (c *conn) newStream(nc net.Conn) *stream {
	return &stream{
		c:    c,
		Conn: nc,
	}
}

func (s *stream) Read(b []byte) (int, error) {
	s.r.Lock()
	defer s.r.Unlock()

	if s.rc {
		return 0, errors.New("closed")
	}

	return s.Conn.Read(b)
}

func (s *stream) Write(b []byte) (int, error) {
	s.w.Lock()
	defer s.w.Unlock()

	if s.wc {
		return 0, errors.New("closed")
	}

	return s.Conn.Write(b)
}

// CloseWrite closes the stream for writing but leaves it open for
// reading.
//
// CloseWrite does not free the stream, users must still call Close or
// Reset.
func (s *stream) CloseWrite() error {
	s.r.Lock()
	defer s.r.Unlock()

	if s.rc {
		return errors.New("closed")
	}

	s.rc = true
	return nil
}

// CloseRead closes the stream for writing but leaves it open for
// reading.
//
// CloseRead does not free the stream, users must still call Close or
// Reset.
func (s *stream) CloseRead() error {
	s.w.Lock()
	defer s.w.Unlock()

	if s.wc {
		return errors.New("closed")
	}

	s.wc = true
	return nil
}

// Reset closes both ends of the stream. Use this to tell the remote
// side to hang up and go away.
func (s *stream) Reset() error {
	s.CloseRead()
	s.CloseWrite()
	return s.Close()
}
