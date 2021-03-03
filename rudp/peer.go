package rudp

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	// ConnTimeout is the amount of time after no packets being received
	// from a Peer that it is automatically disconnected.
	ConnTimeout = 30 * time.Second

	// ConnTimeout is the amount of time after no packets being sent
	// to a Peer that a CtlPing is automatically sent to prevent timeout.
	PingTimeout = 5 * time.Second
)

// A Peer is a connection to a client or server.
type Peer struct {
	pc   net.PacketConn
	addr net.Addr
	conn net.Conn

	disco chan struct{} // close-only

	id PeerID

	pkts     chan Pkt
	errs     chan error    // don't close
	timedOut chan struct{} // close-only

	chans [ChannelCount]pktchan // read/write

	mu       sync.RWMutex
	idOfPeer PeerID
	timeout  *time.Timer
	ping     *time.Ticker
}

// Conn returns the net.PacketConn used to communicate with the Peer.
func (p *Peer) Conn() net.PacketConn { return p.pc }

// Addr returns the address of the Peer.
func (p *Peer) Addr() net.Addr { return p.addr }

// Disco returns a channel that is closed when the Peer is closed.
func (p *Peer) Disco() <-chan struct{} { return p.disco }

// ID returns the ID of the Peer.
func (p *Peer) ID() PeerID { return p.id }

// IsSrv reports whether the Peer is a server.
func (p *Peer) IsSrv() bool {
	return p.ID() == PeerIDSrv
}

// TimedOut reports whether the Peer has timed out.
func (p *Peer) TimedOut() bool {
	select {
	case <-p.timedOut:
		return true
	default:
		return false
	}
}

type inSplit struct {
	chunks    [][]byte
	size, got int
}

type pktchan struct {
	// Only accessed by Peer.processRawPkt.
	inSplit *[65536]*inSplit
	inRel   *[65536][]byte
	inRelSN seqnum

	ackChans sync.Map // map[seqnum]chan struct{}

	outSplitMu sync.Mutex
	outSplitSN seqnum

	outRelMu  sync.Mutex
	outRelSN  seqnum
	outRelWin seqnum
}

// Recv recieves a packet from the Peer.
// You should keep calling this until it returns net.ErrClosed
// so it doesn't leak a goroutine.
func (p *Peer) Recv() (Pkt, error) {
	select {
	case pkt, ok := <-p.pkts:
		if !ok {
			select {
			case err := <-p.errs:
				return Pkt{}, err
			default:
				return Pkt{}, net.ErrClosed
			}
		}
		return pkt, nil
	case err := <-p.errs:
		return Pkt{}, err
	}
}

// Close closes the Peer but does not send a disconnect packet.
func (p *Peer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	select {
	case <-p.Disco():
		return net.ErrClosed
	default:
	}

	p.timeout.Stop()
	p.timeout = nil
	p.ping.Stop()
	p.ping = nil

	close(p.disco)

	return nil
}

func newPeer(pc net.PacketConn, addr net.Addr, id, idOfPeer PeerID) *Peer {
	p := &Peer{
		pc:       pc,
		addr:     addr,
		id:       id,
		idOfPeer: idOfPeer,

		pkts:  make(chan Pkt),
		disco: make(chan struct{}),
		errs:  make(chan error),
	}

	if conn, ok := pc.(net.Conn); ok && conn.RemoteAddr() != nil {
		p.conn = conn
	}

	for i := range p.chans {
		p.chans[i] = pktchan{
			inSplit: new([65536]*inSplit),
			inRel:   new([65536][]byte),
			inRelSN: seqnumInit,

			outSplitSN: seqnumInit,
			outRelSN:   seqnumInit,
			outRelWin:  seqnumInit,
		}
	}

	p.timedOut = make(chan struct{})
	p.timeout = time.AfterFunc(ConnTimeout, func() {
		close(p.timedOut)

		p.SendDisco(0, true)
		p.Close()
	})

	p.ping = time.NewTicker(PingTimeout)
	go p.sendPings(p.ping.C)

	return p
}

func (p *Peer) sendPings(ping <-chan time.Time) {
	pkt := rawPkt{Data: []byte{uint8(rawTypeCtl), uint8(ctlPing)}}

	for {
		select {
		case <-ping:
			if _, err := p.sendRaw(pkt); err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				p.errs <- fmt.Errorf("can't send ping: %w", err)
			}
		case <-p.Disco():
			return
		}
	}
}

// Connect connects to addr using pc
// and closes pc when the returned *Peer disconnects.
func Connect(pc net.PacketConn, addr net.Addr) *Peer {
	srv := newPeer(pc, addr, PeerIDSrv, PeerIDNil)

	pkts := make(chan netPkt)
	go readNetPkts(pc, pkts, srv.errs)
	go srv.processNetPkts(pkts)

	go func() {
		<-srv.Disco()
		pc.Close()
	}()

	return srv
}
