package rudp

import (
	"errors"
	"net"
	"sync"
)

func tryClose(ch chan struct{}) (ok bool) {
	defer func() { recover() }()
	close(ch)
	return true
}

type udpClt struct {
	l      *Listener
	id     PeerID
	addr   net.Addr
	pkts   chan []byte
	closed chan struct{}
}

func (c *udpClt) mkConn() {
	conn := newConn(c, c.id, PeerIDSrv)
	go func() {
		<-conn.Closed()
		c.l.wg.Done()
	}()
	conn.sendRaw(func(buf []byte) int {
		buf[0] = uint8(rawCtl)
		buf[1] = uint8(ctlSetPeerID)
		be.PutUint16(buf[2:4], uint16(conn.ID()))
		return 4
	}, PktInfo{})()
	select {
	case c.l.conns <- conn:
	case <-c.l.closed:
		conn.Close()
	}
}

func (c *udpClt) Write(pkt []byte) (int, error) {
	select {
	case <-c.closed:
		return 0, net.ErrClosed
	default:
	}

	return c.l.pc.WriteTo(pkt, c.addr)
}

func (c *udpClt) recvUDP() ([]byte, error) {
	select {
	case pkt := <-c.pkts:
		return pkt, nil
	case <-c.closed:
		return nil, net.ErrClosed
	}
}

func (c *udpClt) Close() error {
	if !tryClose(c.closed) {
		return net.ErrClosed
	}

	c.l.mu.Lock()
	defer c.l.mu.Unlock()

	delete(c.l.ids, c.id)
	delete(c.l.clts, c.addr.String())

	return nil
}

func (c *udpClt) LocalAddr() net.Addr  { return c.l.pc.LocalAddr() }
func (c *udpClt) RemoteAddr() net.Addr { return c.addr }

// All Listener's methods are safe for concurrent use.
type Listener struct {
	pc net.PacketConn

	peerID PeerID
	conns  chan *Conn
	errs   chan error
	closed chan struct{}
	wg     sync.WaitGroup

	mu   sync.RWMutex
	ids  map[PeerID]bool
	clts map[string]*udpClt
}

// Listen listens for connections on pc, pc is closed once the returned Listener
// and all Conns connected through it are closed.
func Listen(pc net.PacketConn) *Listener {
	l := &Listener{
		pc: pc,

		conns:  make(chan *Conn),
		closed: make(chan struct{}),

		ids:  make(map[PeerID]bool),
		clts: make(map[string]*udpClt),
	}

	go func() {
		for {
			if err := l.processNetPkt(); err != nil {
				if errors.Is(err, net.ErrClosed) {
					break
				}
				select {
				case l.errs <- err:
				case <-l.closed:
				}
			}
		}
	}()

	return l
}

// Accept waits for and returns the next incoming Conn or an error.
func (l *Listener) Accept() (*Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case err := <-l.errs:
		return nil, err
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

// Close makes the Listener stop listening for new Conns.
// Blocked Accept calls will return net.ErrClosed.
// Already Accepted Conns are not closed.
func (l *Listener) Close() error {
	if !tryClose(l.closed) {
		return net.ErrClosed
	}

	go func() {
		l.wg.Wait()
		l.pc.Close()
	}()

	return nil
}

// Addr returns the Listener's network address.
func (l *Listener) Addr() net.Addr { return l.pc.LocalAddr() }

var ErrOutOfPeerIDs = errors.New("out of peer ids")

func (l *Listener) processNetPkt() error {
	buf := make([]byte, maxUDPPktSize)
	n, addr, err := l.pc.ReadFrom(buf)
	if err != nil {
		return err
	}

	l.mu.RLock()
	clt, ok := l.clts[addr.String()]
	l.mu.RUnlock()
	if !ok {
		select {
		case <-l.closed:
			return nil
		default:
		}

		clt, err = l.add(addr)
		if err != nil {
			return err
		}
	}

	select {
	case clt.pkts <- buf[:n]:
	case <-clt.closed:
	}

	return nil
}

func (l *Listener) add(addr net.Addr) (*udpClt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	start := l.peerID
	l.peerID++
	for l.peerID < PeerIDCltMin || l.ids[l.peerID] {
		if l.peerID == start {
			return nil, ErrOutOfPeerIDs
		}
		l.peerID++
	}
	l.ids[l.peerID] = true

	clt := &udpClt{
		l:      l,
		id:     l.peerID,
		addr:   addr,
		pkts:   make(chan []byte),
		closed: make(chan struct{}),
	}
	l.clts[addr.String()] = clt

	l.wg.Add(1)
	go clt.mkConn()

	return clt, nil
}
