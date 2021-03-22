package rudp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

// Recv receives a Pkt from the Conn.
func (c *Conn) Recv() (Pkt, error) {
	select {
	case pkt := <-c.pkts:
		return pkt, nil
	case err := <-c.errs:
		return Pkt{}, err
	case <-c.Closed():
		return Pkt{}, net.ErrClosed
	}
}

func (c *Conn) gotPkt(pkt Pkt) {
	select {
	case c.pkts <- pkt:
	case <-c.Closed():
	}
}

func (c *Conn) gotErr(kind string, data []byte, err error) {
	select {
	case c.errs <- fmt.Errorf("%s: %x: %w", kind, data, err):
	case <-c.Closed():
	}
}

func (c *Conn) recvUDPPkts() {
	for {
		pkt, err := c.udpConn.recvUDP()
		if err != nil {
			c.closeDisco(err)
			break
		}

		if err := c.processUDPPkt(pkt); err != nil {
			c.gotErr("udp", pkt, err)
		}
	}
}

func (c *Conn) processUDPPkt(pkt []byte) error {
	if c.timeout.Stop() {
		c.timeout.Reset(ConnTimeout)
	}

	if len(pkt) < 6 {
		return io.ErrUnexpectedEOF
	}

	if id := be.Uint32(pkt[0:4]); id != protoID {
		return fmt.Errorf("unsupported protocol id: 0x%08x", id)
	}

	ch := Channel(pkt[6])
	if ch >= ChannelCount {
		return TooBigChError(ch)
	}

	if err := c.processRawPkt(pkt[7:], PktInfo{Channel: ch, Unrel: true}); err != nil {
		c.gotErr("raw", pkt, err)
	}

	return nil
}

// A TrailingDataError reports trailing data after a packet.
type TrailingDataError []byte

func (e TrailingDataError) Error() string {
	return fmt.Sprintf("trailing data: %x", []byte(e))
}

func (c *Conn) processRawPkt(data []byte, pi PktInfo) (err error) {
	errWrap := func(format string, a ...interface{}) {
		if err != nil {
			err = fmt.Errorf(format+": %w", append(a, err)...)
		}
	}

	eof := new(byte)
	defer func() {
		switch r := recover(); r {
		case nil:
		case eof:
			err = io.ErrUnexpectedEOF
		default:
			panic(r)
		}
	}()

	off := 0
	eat := func(n int) []byte {
		i := off
		off += n
		if i > len(data) {
			panic(eof)
		}
		return data[i:off]
	}

	ch := &c.chans[pi.Channel]

	switch t := rawType(eat(1)[0]); t {
	case rawCtl:
		defer errWrap("ctl")

		switch ct := ctlType(eat(1)[0]); ct {
		case ctlAck:
			defer errWrap("ack")

			sn := seqnum(be.Uint16(eat(2)))

			if ack, ok := ch.ackChans.LoadAndDelete(sn); ok {
				close(ack.(chan struct{}))
			}
		case ctlSetPeerID:
			defer errWrap("set peer id")

			c.mu.Lock()
			if c.remoteID != PeerIDNil {
				return errors.New("peer id already set")
			}

			c.remoteID = PeerID(be.Uint16(eat(2)))
			c.mu.Unlock()

			c.newAckBuf()
		case ctlPing:
			defer errWrap("ping")
		case ctlDisco:
			defer errWrap("disco")

			c.close(nil)
		default:
			return fmt.Errorf("unsupported ctl type: %d", ct)
		}

		if off < len(data) {
			return TrailingDataError(data[off:])
		}
	case rawOrig:
		c.gotPkt(Pkt{
			Reader:  bytes.NewReader(data[off:]),
			PktInfo: pi,
		})
	case rawSplit:
		defer errWrap("split")

		sn := seqnum(be.Uint16(eat(2)))
		n := be.Uint16(eat(2))
		i := be.Uint16(eat(2))

		defer errWrap("%d", sn)

		if i >= n {
			return fmt.Errorf("chunk number (%d) > chunk count (%d)", i, n)
		}

		ch.inSplitsMu.RLock()
		s := ch.inSplits[sn]
		ch.inSplitsMu.RUnlock()

		if s == nil {
			s = &inSplit{chunks: make([][]byte, n)}
			if pi.Unrel {
				s.timeout = time.AfterFunc(ConnTimeout, func() {
					ch.inSplitsMu.Lock()
					delete(ch.inSplits, sn)
					ch.inSplitsMu.Unlock()
				})
			}

			ch.inSplitsMu.Lock()
			ch.inSplits[sn] = s
			ch.inSplitsMu.Unlock()
		}

		if int(n) != len(s.chunks) {
			return fmt.Errorf("chunk count changed from %d to %d", len(s.chunks), n)
		}

		if s.chunks[i] == nil {
			s.chunks[i] = data[off:]
			s.got++
		}

		if s.got < len(s.chunks) {
			if s.timeout != nil && s.timeout.Stop() {
				s.timeout.Reset(ConnTimeout)
			}
			return
		}

		if s.timeout != nil {
			s.timeout.Stop()
		}

		ch.inSplitsMu.Lock()
		delete(ch.inSplits, sn)
		ch.inSplitsMu.Unlock()

		c.gotPkt(Pkt{
			Reader:  (*net.Buffers)(&s.chunks),
			PktInfo: pi,
		})
	case rawRel:
		defer errWrap("rel")

		sn := seqnum(be.Uint16(eat(2)))

		defer errWrap("%d", sn)

		be.PutUint16(ch.ackBuf, uint16(sn))
		ch.sendAck()

		if sn-ch.inRelSN >= 0x8000 {
			// Already received.
			return nil
		}

		ch.inRels[sn&0x7fff] = data[off:]

		i := func() seqnum { return ch.inRelSN & 0x7fff }
		for ; ch.inRels[i()] != nil; ch.inRelSN++ {
			data := ch.inRels[i()]
			ch.inRels[i()] = nil
			if err := c.processRawPkt(data, PktInfo{Channel: pi.Channel}); err != nil {
				c.gotErr("rel", data, err)
			}
		}
	default:
		return fmt.Errorf("unsupported pkt type: %d", t)
	}

	return nil
}

func (c *Conn) newAckBuf() {
	for i := range c.chans {
		ch := &c.chans[i]
		ch.sendAck = c.sendRaw(func(buf []byte) int {
			buf[0] = uint8(rawCtl)
			buf[1] = uint8(ctlAck)
			ch.ackBuf = buf[2:4]
			return 4
		}, PktInfo{Channel: Channel(i), Unrel: true})
	}
}
