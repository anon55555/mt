package rudp

import (
	"errors"
	"net"
)

// ErrClosed is deprecated, use net.ErrClosed instead.
var ErrClosed = net.ErrClosed

/*
netPkt.Data format (big endian):

	ProtoID
	Src PeerID
	ChNo uint8 // Must be < ChannelCount.
	RawPkt.Data
*/
type netPkt struct {
	SrcAddr net.Addr
	Data    []byte
}

func readNetPkts(conn net.PacketConn, pkts chan<- netPkt, errs chan<- error) {
	for {
		buf := make([]byte, MaxNetPktSize)
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			errs <- err
			continue
		}

		pkts <- netPkt{addr, buf[:n]}
	}

	close(pkts)
}
