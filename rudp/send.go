package rudp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var ErrPktTooBig = errors.New("can't send pkt: too big")

// A TooBigChError reports a Channel greater than or equal to ChannelCount.
type TooBigChError Channel

func (e TooBigChError) Error() string {
	return fmt.Sprintf("channel >= ChannelCount (%d): %d", ChannelCount, e)
}

// Send sends a Pkt to the Conn.
// Ack is closed when the packet is acknowledged.
// Ack is nil if pkt.Unrel is true or err != nil.
func (c *Conn) Send(pkt Pkt) (ack <-chan struct{}, err error) {
	if pkt.Channel >= ChannelCount {
		return nil, TooBigChError(pkt.Channel)
	}

	var e error
	send := c.sendRaw(func(buf []byte) int {
		buf[0] = uint8(rawOrig)

		nn := 1
		for nn < len(buf) {
			n, err := pkt.Read(buf[nn:])
			nn += n
			if err != nil {
				e = err
				return nn
			}
		}

		if _, e = pkt.Read(nil); e != nil {
			return nn
		}

		pkt.Reader = io.MultiReader(
			bytes.NewReader([]byte(buf[1:nn])),
			pkt.Reader,
		)
		return nn
	}, pkt.PktInfo)
	if e != nil {
		if e == io.EOF {
			return send()
		}
		return nil, e
	}

	var (
		sn seqnum
		i  uint16

		sends []func() (<-chan struct{}, error)
	)

	for {
		var (
			b []byte
			e error
		)
		send := c.sendRaw(func(buf []byte) int {
			buf[0] = uint8(rawSplit)

			n, err := io.ReadFull(pkt, buf[7:])
			if err != nil && err != io.ErrUnexpectedEOF {
				e = err
				return 0
			}

			be.PutUint16(buf[5:7], i)
			if i++; i == 0 {
				e = ErrPktTooBig
				return 0
			}

			b = buf
			return 7 + n
		}, pkt.PktInfo)
		if e != nil {
			if e == io.EOF {
				break
			}
			return nil, e
		}

		sends = append(sends, func() (<-chan struct{}, error) {
			be.PutUint16(b[1:3], uint16(sn))
			be.PutUint16(b[3:5], i)
			return send()
		})
	}

	ch := &c.chans[pkt.Channel]

	ch.outSplitMu.Lock()
	sn = ch.outSplitSN
	ch.outSplitSN++
	ch.outSplitMu.Unlock()

	var wg sync.WaitGroup

	for _, send := range sends {
		ack, err := send()
		if err != nil {
			return nil, err
		}
		if !pkt.Unrel {
			wg.Add(1)
			go func() {
				<-ack
				wg.Done()
			}()
		}
	}

	if !pkt.Unrel {
		ack := make(chan struct{})
		go func() {
			wg.Wait()
			close(ack)
		}()
		return ack, nil
	}

	return nil, nil
}

func (c *Conn) sendRaw(read func([]byte) int, pi PktInfo) func() (<-chan struct{}, error) {
	if pi.Unrel {
		buf := make([]byte, maxUDPPktSize)
		be.PutUint32(buf[0:4], protoID)
		c.mu.RLock()
		be.PutUint16(buf[4:6], uint16(c.remoteID))
		c.mu.RUnlock()
		buf[6] = uint8(pi.Channel)
		buf = buf[:7+read(buf[7:])]

		return func() (<-chan struct{}, error) {
			if _, err := c.udpConn.Write(buf); err != nil {
				c.close(err)
				return nil, net.ErrClosed
			}

			c.ping.Reset(PingTimeout)
			if atomic.LoadUint32(&c.closing) == 1 {
				c.ping.Stop()
			}

			return nil, nil
		}
	}

	pi.Unrel = true
	var snBuf []byte
	send := c.sendRaw(func(buf []byte) int {
		buf[0] = uint8(rawRel)
		snBuf = buf[1:3]
		return 3 + read(buf[3:])
	}, pi)

	return func() (<-chan struct{}, error) {
		ch := &c.chans[pi.Channel]

		ch.outRelMu.Lock()
		defer ch.outRelMu.Unlock()

		sn := ch.outRelSN
		be.PutUint16(snBuf, uint16(sn))
		for ; sn-ch.outRelWin >= 0x8000; ch.outRelWin++ {
			if ack, ok := ch.ackChans.Load(ch.outRelWin); ok {
				select {
				case <-ack.(chan struct{}):
				case <-c.Closed():
				}
			}
		}

		ack := make(chan struct{})
		ch.ackChans.Store(sn, ack)

		if _, err := send(); err != nil {
			if ack, ok := ch.ackChans.LoadAndDelete(sn); ok {
				close(ack.(chan struct{}))
			}
			return nil, err
		}
		ch.outRelSN++

		go func() {
			t := time.NewTimer(500 * time.Millisecond)
			defer t.Stop()

			for {
				select {
				case <-ack:
					return
				case <-t.C:
					send()
					t.Reset(500 * time.Millisecond)
				case <-c.Closed():
					return
				}
			}
		}()

		return ack, nil
	}
}
