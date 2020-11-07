package rudp

import (
	"errors"
	"net"
	"strings"
)

// TODO: Use net.ErrClosed when Go 1.16 is released.
var ErrClosed = errors.New("use of closed peer")

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
			// TODO: Change to this when Go 1.16 is released:
			// if errors.Is(err, net.ErrClosed) {
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}

			errs <- err
			continue
		}

		pkts <- netPkt{addr, buf[:n]}
	}

	close(pkts)
}
