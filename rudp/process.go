package rudp

import (
	"errors"
	"fmt"
	"io"
	"net"
)

// A PktError is an error that occured while processing a packet.
type PktError struct {
	Type string // "net", "raw" or "rel".
	Data []byte
	Err  error
}

func (e PktError) Error() string {
	return fmt.Sprintf("error processing %s pkt: %x: %v", e.Type, e.Data, e.Err)
}

func (e PktError) Unwrap() error { return e.Err }

func (p *Peer) processNetPkts(pkts <-chan netPkt) {
	for pkt := range pkts {
		if err := p.processNetPkt(pkt); err != nil {
			p.errs <- PktError{"net", pkt.Data, err}
		}
	}

	close(p.pkts)
}

// A TrailingDataError reports a packet with trailing data,
// it doesn't stop a packet from being processed.
type TrailingDataError []byte

func (e TrailingDataError) Error() string {
	return fmt.Sprintf("trailing data: %x", []byte(e))
}

func (p *Peer) processNetPkt(pkt netPkt) (err error) {
	if pkt.SrcAddr.String() != p.Addr().String() {
		return fmt.Errorf("got pkt from wrong addr: %s", p.Addr().String())
	}

	if len(pkt.Data) < MtHdrSize {
		return io.ErrUnexpectedEOF
	}

	if id := be.Uint32(pkt.Data[0:4]); id != protoID {
		return fmt.Errorf("unsupported protocol id: 0x%08x", id)
	}

	// src PeerID at pkt.Data[4:6]

	chno := pkt.Data[6]
	if chno >= ChannelCount {
		return fmt.Errorf("invalid channel number: %d: >= ChannelCount", chno)
	}

	p.mu.RLock()
	if p.timeout != nil {
		p.timeout.Reset(ConnTimeout)
	}
	p.mu.RUnlock()

	rpkt := rawPkt{
		Data:  pkt.Data[MtHdrSize:],
		ChNo:  chno,
		Unrel: true,
	}
	if err := p.processRawPkt(rpkt); err != nil {
		p.errs <- PktError{"raw", rpkt.Data, err}
	}

	return nil
}

func (p *Peer) processRawPkt(pkt rawPkt) (err error) {
	errWrap := func(format string, a ...interface{}) {
		if err != nil {
			err = fmt.Errorf(format, append(a, err)...)
		}
	}

	c := &p.chans[pkt.ChNo]

	if len(pkt.Data) < 1 {
		return fmt.Errorf("can't read pkt type: %w", io.ErrUnexpectedEOF)
	}
	switch t := rawType(pkt.Data[0]); t {
	case rawTypeCtl:
		defer errWrap("ctl: %w")

		if len(pkt.Data) < 1+1 {
			return fmt.Errorf("can't read type: %w", io.ErrUnexpectedEOF)
		}
		switch ct := ctlType(pkt.Data[1]); ct {
		case ctlAck:
			defer errWrap("ack: %w")

			if len(pkt.Data) < 1+1+2 {
				return io.ErrUnexpectedEOF
			}

			sn := seqnum(be.Uint16(pkt.Data[2:4]))

			if ack, ok := c.ackChans.LoadAndDelete(sn); ok {
				close(ack.(chan struct{}))
			}

			if len(pkt.Data) > 1+1+2 {
				return TrailingDataError(pkt.Data[1+1+2:])
			}
		case ctlSetPeerID:
			defer errWrap("set peer id: %w")

			if len(pkt.Data) < 1+1+2 {
				return io.ErrUnexpectedEOF
			}

			// Ensure no concurrent senders while peer id changes.
			p.mu.Lock()
			if p.idOfPeer != PeerIDNil {
				return errors.New("peer id already set")
			}

			p.idOfPeer = PeerID(be.Uint16(pkt.Data[2:4]))
			p.mu.Unlock()

			if len(pkt.Data) > 1+1+2 {
				return TrailingDataError(pkt.Data[1+1+2:])
			}
		case ctlPing:
			defer errWrap("ping: %w")

			if len(pkt.Data) > 1+1 {
				return TrailingDataError(pkt.Data[1+1:])
			}
		case ctlDisco:
			defer errWrap("disco: %w")

			p.Close()

			if len(pkt.Data) > 1+1 {
				return TrailingDataError(pkt.Data[1+1:])
			}
		default:
			return fmt.Errorf("unsupported ctl type: %d", ct)
		}
	case rawTypeOrig:
		p.pkts <- Pkt{
			Data:  pkt.Data[1:],
			ChNo:  pkt.ChNo,
			Unrel: pkt.Unrel,
		}
	case rawTypeSplit:
		defer errWrap("split: %w")

		if len(pkt.Data) < 1+2+2+2 {
			return io.ErrUnexpectedEOF
		}

		sn := seqnum(be.Uint16(pkt.Data[1:3]))
		count := be.Uint16(pkt.Data[3:5])
		i := be.Uint16(pkt.Data[5:7])

		if i >= count {
			return nil
		}

		splits := p.chans[pkt.ChNo].inSplit

		// Delete old incomplete split packets
		// so new ones don't get corrupted.
		splits[sn-0x8000] = nil

		if splits[sn] == nil {
			splits[sn] = &inSplit{chunks: make([][]byte, count)}
		}

		s := splits[sn]

		if int(count) != len(s.chunks) {
			return fmt.Errorf("chunk count changed on split packet: %d", sn)
		}

		s.chunks[i] = pkt.Data[7:]
		s.size += len(s.chunks[i])
		s.got++

		if s.got == len(s.chunks) {
			data := make([]byte, 0, s.size)
			for _, chunk := range s.chunks {
				data = append(data, chunk...)
			}

			p.pkts <- Pkt{
				Data:  data,
				ChNo:  pkt.ChNo,
				Unrel: pkt.Unrel,
			}

			splits[sn] = nil
		}
	case rawTypeRel:
		defer errWrap("rel: %w")

		if len(pkt.Data) < 1+2 {
			return io.ErrUnexpectedEOF
		}

		sn := seqnum(be.Uint16(pkt.Data[1:3]))

		ack := make([]byte, 1+1+2)
		ack[0] = uint8(rawTypeCtl)
		ack[1] = uint8(ctlAck)
		be.PutUint16(ack[2:4], uint16(sn))
		if _, err := p.sendRaw(rawPkt{
			Data:  ack,
			ChNo:  pkt.ChNo,
			Unrel: true,
		}); err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("can't ack %d: %w", sn, err)
		}

		if sn-c.inRelSN >= 0x8000 {
			return nil // Already received.
		}

		c.inRel[sn] = pkt.Data[3:]

		for ; c.inRel[c.inRelSN] != nil; c.inRelSN++ {
			rpkt := rawPkt{
				Data:  c.inRel[c.inRelSN],
				ChNo:  pkt.ChNo,
				Unrel: false,
			}
			c.inRel[c.inRelSN] = nil

			if err := p.processRawPkt(rpkt); err != nil {
				p.errs <- PktError{"rel", rpkt.Data, err}
			}
		}
	default:
		return fmt.Errorf("unsupported pkt type: %d", t)
	}

	return nil
}
