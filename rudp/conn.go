package rudp

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// A Conn is a connection to a client or server.
// All Conn's methods are safe for concurrent use.
type Conn struct {
	udpConn udpConn

	id PeerID

	pkts chan Pkt
	errs chan error

	timeout *time.Timer
	ping    *time.Ticker

	closing uint32
	closed  chan struct{}
	err     error

	mu       sync.RWMutex
	remoteID PeerID

	chans [ChannelCount]pktChan // read/write
}

// ID returns the PeerID of the Conn.
func (c *Conn) ID() PeerID { return c.id }

// IsSrv reports whether the Conn is a connection to a server.
func (c *Conn) IsSrv() bool { return c.ID() == PeerIDSrv }

// Closed returns a channel which is closed when the Conn is closed.
func (c *Conn) Closed() <-chan struct{} { return c.closed }

// WhyClosed returns the error that caused the Conn to be closed or nil
// if the Conn was closed using the Close method or by the peer.
// WhyClosed returns nil if the Conn is not closed.
func (c *Conn) WhyClosed() error {
	select {
	case <-c.Closed():
		return c.err
	default:
		return nil
	}
}

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr { return c.udpConn.LocalAddr() }

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr { return c.udpConn.RemoteAddr() }

type pktChan struct {
	// Only accessed by Conn.recvUDPPkts goroutine.
	inRels  *[0x8000][]byte
	inRelSN seqnum
	sendAck func() (<-chan struct{}, error)
	ackBuf  []byte

	inSplitsMu sync.RWMutex
	inSplits   map[seqnum]*inSplit

	ackChans sync.Map // map[seqnum]chan struct{}

	outSplitMu sync.Mutex
	outSplitSN seqnum

	outRelMu  sync.Mutex
	outRelSN  seqnum
	outRelWin seqnum
}

type inSplit struct {
	chunks  [][]byte
	got     int
	timeout *time.Timer
}

// Close closes the Conn.
// Any blocked Send or Recv calls will return net.ErrClosed.
func (c *Conn) Close() error {
	return c.closeDisco(nil)
}

func (c *Conn) closeDisco(err error) error {
	c.sendRaw(func(buf []byte) int {
		buf[0] = uint8(rawCtl)
		buf[1] = uint8(ctlDisco)
		return 2
	}, PktInfo{Unrel: true})()

	return c.close(err)
}

func (c *Conn) close(err error) error {
	if atomic.SwapUint32(&c.closing, 1) == 1 {
		return net.ErrClosed
	}

	c.timeout.Stop()
	c.ping.Stop()

	c.err = err
	defer close(c.closed)

	return c.udpConn.Close()
}

func newConn(uc udpConn, id, remoteID PeerID) *Conn {
	var c *Conn
	c = &Conn{
		udpConn: uc,

		id: id,

		pkts: make(chan Pkt),
		errs: make(chan error),

		timeout: time.AfterFunc(ConnTimeout, func() {
			c.closeDisco(ErrTimedOut)
		}),
		ping: time.NewTicker(PingTimeout),

		closed: make(chan struct{}),

		remoteID: remoteID,
	}

	for i := range c.chans {
		c.chans[i] = pktChan{
			inRels:  new([0x8000][]byte),
			inRelSN: initSeqnum,

			inSplits: make(map[seqnum]*inSplit),

			outSplitSN: initSeqnum,

			outRelSN:  initSeqnum,
			outRelWin: initSeqnum,
		}
	}

	c.newAckBuf()

	go c.sendPings(c.ping.C)
	go c.recvUDPPkts()

	return c
}

func (c *Conn) sendPings(ping <-chan time.Time) {
	send := c.sendRaw(func(buf []byte) int {
		buf[0] = uint8(rawCtl)
		buf[1] = uint8(ctlPing)
		return 2
	}, PktInfo{})

	for {
		select {
		case <-ping:
			send()
		case <-c.Closed():
			return
		}
	}
}
