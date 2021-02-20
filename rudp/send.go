package rudp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"
)

const (
	// protoID + src PeerID + channel number
	MtHdrSize = 4 + 2 + 1

	// rawTypeOrig
	OrigHdrSize = 1

	// rawTypeSpilt + seqnum + chunk count + chunk number
	SplitHdrSize = 1 + 2 + 2 + 2

	// rawTypeRel + seqnum
	RelHdrSize = 1 + 2
)

const (
	MaxNetPktSize = 512

	MaxUnrelRawPktSize = MaxNetPktSize - MtHdrSize
	MaxRelRawPktSize   = MaxUnrelRawPktSize - RelHdrSize

	MaxRelPktSize   = (MaxRelRawPktSize - SplitHdrSize) * math.MaxUint16
	MaxUnrelPktSize = (MaxUnrelRawPktSize - SplitHdrSize) * math.MaxUint16
)

var ErrPktTooBig = errors.New("can't send pkt: too big")
var ErrChNoTooBig = errors.New("can't send pkt: channel number >= ChannelCount")

// Send sends a packet to the Peer.
// It returns a channel that's closed when all chunks are acked or an error.
// The ack channel is nil if pkt.Unrel is true.
func (p *Peer) Send(pkt Pkt) (ack <-chan struct{}, err error) {
	if pkt.ChNo >= ChannelCount {
		return nil, ErrChNoTooBig
	}

	hdrsize := MtHdrSize
	if !pkt.Unrel {
		hdrsize += RelHdrSize
	}

	if hdrsize+OrigHdrSize+len(pkt.Data) > MaxNetPktSize {
		c := &p.chans[pkt.ChNo]

		c.outsplitmu.Lock()
		sn := c.outsplitsn
		c.outsplitsn++
		c.outsplitmu.Unlock()

		chunks := split(pkt.Data, MaxNetPktSize-(hdrsize+SplitHdrSize))

		if len(chunks) > math.MaxUint16 {
			return nil, ErrPktTooBig
		}

		var wg sync.WaitGroup

		for i, chunk := range chunks {
			data := make([]byte, SplitHdrSize+len(chunk))
			data[0] = uint8(rawTypeSplit)
			binary.BigEndian.PutUint16(data[1:3], uint16(sn))
			binary.BigEndian.PutUint16(data[3:5], uint16(len(chunks)))
			binary.BigEndian.PutUint16(data[5:7], uint16(i))
			copy(data[SplitHdrSize:], chunk)

			wg.Add(1)
			ack, err := p.sendRaw(rawPkt{
				Data:  data,
				ChNo:  pkt.ChNo,
				Unrel: pkt.Unrel,
			})
			if err != nil {
				return nil, err
			}
			if !pkt.Unrel {
				if ack == nil {
					panic("ack is nil")
				}
				go func() {
					<-ack
					wg.Done()
				}()
			}
		}

		if pkt.Unrel {
			return nil, nil
		} else {
			ack := make(chan struct{})

			go func() {
				wg.Wait()
				close(ack)
			}()

			return ack, nil
		}
	}

	return p.sendRaw(rawPkt{
		Data:  append([]byte{uint8(rawTypeOrig)}, pkt.Data...),
		ChNo:  pkt.ChNo,
		Unrel: pkt.Unrel,
	})
}

// sendRaw sends a raw packet to the Peer.
func (p *Peer) sendRaw(pkt rawPkt) (ack <-chan struct{}, err error) {
	if pkt.ChNo >= ChannelCount {
		return nil, ErrChNoTooBig
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	select {
	case <-p.Disco():
		return nil, net.ErrClosed
	default:
	}

	if !pkt.Unrel {
		return p.sendRel(pkt)
	}

	data := make([]byte, MtHdrSize+len(pkt.Data))
	binary.BigEndian.PutUint32(data[0:4], protoID)
	binary.BigEndian.PutUint16(data[4:6], uint16(p.idOfPeer))
	data[6] = pkt.ChNo
	copy(data[MtHdrSize:], pkt.Data)

	if len(data) > MaxNetPktSize {
		return nil, ErrPktTooBig
	}

	_, err = p.Conn().WriteTo(data, p.Addr())
	if errors.Is(err, net.ErrWriteToConnected) {
		conn, ok := p.Conn().(net.Conn)
		if !ok {
			return nil, err
		}
		_, err = conn.Write(data)
	}
	if err != nil {
		return nil, err
	}

	p.ping.Reset(PingTimeout)

	return nil, nil
}

// sendRel sends a reliable raw packet to the Peer.
func (p *Peer) sendRel(pkt rawPkt) (ack <-chan struct{}, err error) {
	if pkt.Unrel {
		panic("mt/rudp: sendRel: pkt.Unrel is true")
	}

	c := &p.chans[pkt.ChNo]

	c.outrelmu.Lock()
	defer c.outrelmu.Unlock()

	sn := c.outrelsn
	for ; sn-c.outrelwin >= 0x8000; c.outrelwin++ {
		if ack, ok := c.ackchans.Load(c.outrelwin); ok {
			<-ack.(chan struct{})
		}
	}
	c.outrelsn++

	rwack := make(chan struct{}) // close-only
	c.ackchans.Store(sn, rwack)
	ack = rwack

	reldata := make([]byte, RelHdrSize+len(pkt.Data))
	reldata[0] = uint8(rawTypeRel)
	binary.BigEndian.PutUint16(reldata[1:3], uint16(sn))
	copy(reldata[RelHdrSize:], pkt.Data)
	relpkt := rawPkt{
		Data:  reldata,
		ChNo:  pkt.ChNo,
		Unrel: true,
	}

	if _, err := p.sendRaw(relpkt); err != nil {
		c.ackchans.Delete(sn)

		return nil, err
	}

	go func() {
		for {
			select {
			case <-time.After(500 * time.Millisecond):
				if _, err := p.sendRaw(relpkt); err != nil {
					p.errs <- fmt.Errorf("failed to re-send timed out reliable seqnum: %d: %w", sn, err)
				}
			case <-ack:
				return
			case <-p.Disco():
				return
			}
		}
	}()

	return ack, nil
}

// SendDisco sends a disconnect packet to the Peer but does not close it.
// It returns a channel that's closed when it's acked or an error.
// The ack channel is nil if unrel is true.
func (p *Peer) SendDisco(chno uint8, unrel bool) (ack <-chan struct{}, err error) {
	return p.sendRaw(rawPkt{
		Data:  []byte{uint8(rawTypeCtl), uint8(ctlDisco)},
		ChNo:  chno,
		Unrel: unrel,
	})
}

func split(data []byte, chunksize int) [][]byte {
	chunks := make([][]byte, 0, (len(data)+chunksize-1)/chunksize)

	for i := 0; i < len(data); i += chunksize {
		end := i + chunksize
		if end > len(data) {
			end = len(data)
		}

		chunks = append(chunks, data[i:end])
	}

	return chunks
}
