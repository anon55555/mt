package rudp

import (
	"encoding/binary"
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

	if id := binary.BigEndian.Uint32(pkt.Data[0:4]); id != protoID {
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

			sn := seqnum(binary.BigEndian.Uint16(pkt.Data[2:4]))

			if ack, ok := c.ackchans.LoadAndDelete(sn); ok {
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

			p.idOfPeer = PeerID(binary.BigEndian.Uint16(pkt.Data[2:4]))
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

		sn := seqnum(binary.BigEndian.Uint16(pkt.Data[1:3]))
		count := binary.BigEndian.Uint16(pkt.Data[3:5])
		i := binary.BigEndian.Uint16(pkt.Data[5:7])

		if i >= count {
			return nil
		}

		splitpkts := p.chans[pkt.ChNo].insplit

		// Delete old incomplete split packets
		// so new ones don't get corrupted.
		delete(splitpkts, sn-0x8000)

		if splitpkts[sn] == nil {
			splitpkts[sn] = make([][]byte, count)
		}

		chunks := splitpkts[sn]

		if int(count) != len(chunks) {
			return fmt.Errorf("chunk count changed on seqnum: %d", sn)
		}

		chunks[i] = pkt.Data[7:]

		for _, chunk := range chunks {
			if chunk == nil {
				return nil
			}
		}

		var data []byte
		for _, chunk := range chunks {
			data = append(data, chunk...)
		}

		p.pkts <- Pkt{
			Data:  data,
			ChNo:  pkt.ChNo,
			Unrel: pkt.Unrel,
		}

		delete(splitpkts, sn)
	case rawTypeRel:
		defer errWrap("rel: %w")

		if len(pkt.Data) < 1+2 {
			return io.ErrUnexpectedEOF
		}

		sn := seqnum(binary.BigEndian.Uint16(pkt.Data[1:3]))

		ackdata := make([]byte, 1+1+2)
		ackdata[0] = uint8(rawTypeCtl)
		ackdata[1] = uint8(ctlAck)
		binary.BigEndian.PutUint16(ackdata[2:4], uint16(sn))
		ack := rawPkt{
			Data:  ackdata,
			ChNo:  pkt.ChNo,
			Unrel: true,
		}
		if _, err := p.sendRaw(ack); err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("can't ack %d: %w", sn, err)
		}

		if sn-c.inrelsn >= 0x8000 {
			return nil // Already received.
		}

		c.inrel[sn] = pkt.Data[3:]

		for ; c.inrel[c.inrelsn] != nil; c.inrelsn++ {
			data := c.inrel[c.inrelsn]
			delete(c.inrel, c.inrelsn)

			rpkt := rawPkt{
				Data:  data,
				ChNo:  pkt.ChNo,
				Unrel: false,
			}
			if err := p.processRawPkt(rpkt); err != nil {
				p.errs <- PktError{"rel", rpkt.Data, err}
			}
		}
	default:
		return fmt.Errorf("unsupported pkt type: %d", t)
	}

	return nil
}
