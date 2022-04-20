package inproc

import (
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
)

// pipeDeadline is an abstraction for handling timeouts.
type pipeDeadline struct {
	mu     sync.Mutex // Guards timer and cancel
	timer  *time.Timer
	cancel chan struct{} // Must be non-nil
}

func makePipeDeadline() pipeDeadline {
	return pipeDeadline{cancel: make(chan struct{})}
}

// set sets the point in time when the deadline will time out.
// A timeout event is signaled by closing the channel returned by waiter.
// Once a timeout has occurred, the deadline can be refreshed by specifying a
// t value in the future.
//
// A zero value for t prevents timeout.
func (d *pipeDeadline) set(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil && !d.timer.Stop() {
		<-d.cancel // Wait for the timer callback to finish and close cancel
	}
	d.timer = nil

	// Time is zero, then there is no deadline.
	closed := isClosedChan(d.cancel)
	if t.IsZero() {
		if closed {
			d.cancel = make(chan struct{})
		}
		return
	}

	// Time in the future, setup a timer to cancel in the future.
	if dur := time.Until(t); dur > 0 {
		if closed {
			d.cancel = make(chan struct{})
		}
		d.timer = time.AfterFunc(dur, func() {
			close(d.cancel)
		})
		return
	}

	// Time in the past, so close immediately.
	if !closed {
		close(d.cancel)
	}
}

// wait returns a channel that is closed when the deadline is exceeded.
func (d *pipeDeadline) wait() chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.cancel
}

func isClosedChan(cs ...<-chan struct{}) bool {
	for _, c := range cs {
		select {
		case <-c:
			return true
		default:
		}
	}

	return false
}

type pipe struct {
	wrMu sync.Mutex // Serialize Write operations

	// Used by local Read to interact with remote Write.
	// Successful receive on rdRx is always followed by send on rdTx.
	rdRx <-chan []byte
	rdTx chan<- int

	// Used by local Write to interact with remote Read.
	// Successful send on wrTx is always followed by receive on wrRx.
	wrTx chan<- []byte
	wrRx <-chan int

	once, ronce, wonce, resetOnce                            sync.Once // Protects closing localDone
	localDone, localReadDone, localWriteDone, localReset     chan struct{}
	remoteDone, remoteReadDone, remoteWriteDone, remoteReset <-chan struct{}

	readDeadline  pipeDeadline
	writeDeadline pipeDeadline
}

func newPipe() (*pipe, *pipe) {
	cb1 := make(chan []byte)
	cb2 := make(chan []byte)
	cn1 := make(chan int)
	cn2 := make(chan int)
	done1 := make(chan struct{})
	done2 := make(chan struct{})
	rdone1 := make(chan struct{})
	rdone2 := make(chan struct{})
	wdone1 := make(chan struct{})
	wdone2 := make(chan struct{})
	reset1 := make(chan struct{})
	reset2 := make(chan struct{})

	p1 := &pipe{
		rdRx: cb1, rdTx: cn1,
		wrTx: cb2, wrRx: cn2,
		localDone: done1, remoteDone: done2,
		localReadDone: rdone1, remoteReadDone: rdone2,
		localWriteDone: wdone1, remoteWriteDone: wdone2,
		localReset: reset1, remoteReset: reset2,
		readDeadline:  makePipeDeadline(),
		writeDeadline: makePipeDeadline(),
	}
	p2 := &pipe{
		rdRx: cb2, rdTx: cn2,
		wrTx: cb1, wrRx: cn1,
		localDone: done2, remoteDone: done1,
		localReadDone: rdone2, remoteReadDone: rdone1,
		localWriteDone: wdone2, remoteWriteDone: wdone1,
		localReset: reset2, remoteReset: reset1,
		readDeadline:  makePipeDeadline(),
		writeDeadline: makePipeDeadline(),
	}
	return p1, p2
}

func (p *pipe) Read(b []byte) (int, error) {
	n, err := p.read(b)
	if err != nil && err != io.EOF && err != io.ErrClosedPipe {
		err = &net.OpError{Op: "read", Net: "pipe", Err: err}
	}
	return n, err
}

func (p *pipe) read(b []byte) (n int, err error) {
	switch {
	case isClosedChan(p.localDone):
		return 0, io.ErrClosedPipe
	case isClosedChan(p.localReadDone, p.localReset, p.remoteReset):
		return 0, network.ErrReset
	case isClosedChan(p.remoteDone, p.remoteWriteDone):
		return 0, io.EOF
	case isClosedChan(p.readDeadline.wait()):
		return 0, os.ErrDeadlineExceeded
	}

	select {
	case bw := <-p.rdRx:
		nr := copy(b, bw)
		p.rdTx <- nr
		return nr, nil
	case <-p.localDone:
		return 0, io.ErrClosedPipe
	case <-p.remoteDone:
		return 0, io.EOF
	case <-p.remoteWriteDone:
		return 0, io.EOF
	case <-p.localReset:
		return 0, network.ErrReset
	case <-p.remoteReset:
		return 0, network.ErrReset
	case <-p.readDeadline.wait():
		return 0, os.ErrDeadlineExceeded
	}
}

func (p *pipe) Write(b []byte) (int, error) {
	n, err := p.write(b)
	if err != nil && err != io.ErrClosedPipe {
		err = &net.OpError{Op: "write", Net: "pipe", Err: err}
	}
	return n, err
}

func (p *pipe) write(b []byte) (n int, err error) {
	switch {
	case isClosedChan(p.localDone, p.remoteDone):
		return 0, io.ErrClosedPipe
	case isClosedChan(p.remoteReadDone, p.localReset, p.remoteReset):
		return 0, network.ErrReset
	case isClosedChan(p.writeDeadline.wait()):
		return 0, os.ErrDeadlineExceeded
	}

	p.wrMu.Lock() // Ensure entirety of b is written together
	defer p.wrMu.Unlock()
	for once := true; once || len(b) > 0; once = false {
		select {
		case p.wrTx <- b:
			nw := <-p.wrRx
			b = b[nw:]
			n += nw
		case <-p.localDone:
			return n, io.ErrClosedPipe
		case <-p.remoteDone:
			return n, io.ErrClosedPipe
		case <-p.localReset:
			return 0, network.ErrReset
		case <-p.remoteReset:
			return 0, network.ErrReset
		case <-p.remoteReadDone:
			return 0, network.ErrReset
		case <-p.writeDeadline.wait():
			return n, os.ErrDeadlineExceeded
		}
	}
	return n, nil
}

func (p *pipe) SetDeadline(t time.Time) error {
	if isClosedChan(p.localDone) || isClosedChan(p.remoteDone) {
		return io.ErrClosedPipe
	}
	p.readDeadline.set(t)
	p.writeDeadline.set(t)
	return nil
}

func (p *pipe) SetReadDeadline(t time.Time) error {
	if isClosedChan(p.localDone) || isClosedChan(p.remoteDone) {
		return io.ErrClosedPipe
	}
	p.readDeadline.set(t)
	return nil
}

func (p *pipe) SetWriteDeadline(t time.Time) error {
	if isClosedChan(p.localDone) || isClosedChan(p.remoteDone) {
		return io.ErrClosedPipe
	}
	p.writeDeadline.set(t)
	return nil
}

// Close closes the stream.
//
// * Any buffered data for writing will be flushed.
// * Future reads will fail.
// * Any in-progress reads/writes will be interrupted.
//
// Close may be asynchronous and _does not_ guarantee receipt of the
// data.
func (p *pipe) Close() error {
	p.once.Do(func() { close(p.localDone) })
	return nil
}

// CloseWrite closes the stream for writing but leaves it open for
// reading.
//
// CloseWrite does not free the stream, users must still call Close or
// Reset.
func (p *pipe) CloseWrite() error {
	p.wonce.Do(func() { close(p.localWriteDone) })
	return nil
}

// CloseRead closes the stream for writing but leaves it open for
// reading.
//
// CloseRead does not free the stream, users must still call Close or
// Reset.
func (p *pipe) CloseRead() error {
	p.ronce.Do(func() { close(p.localReadDone) })
	return nil
}

// Reset closes both ends of the stream. Use this to tell the remote
// side to hang up and go away.
func (p *pipe) Reset() error {
	p.resetOnce.Do(func() { close(p.localReset) })
	return nil
}
