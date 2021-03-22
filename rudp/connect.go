package rudp

import "net"

type udpSrv struct {
	net.Conn
}

func (us udpSrv) recvUDP() ([]byte, error) {
	buf := make([]byte, maxUDPPktSize)
	n, err := us.Read(buf)
	return buf[:n], err
}

// Connect returns a Conn connected to conn.
func Connect(conn net.Conn) *Conn {
	return newConn(udpSrv{conn}, PeerIDSrv, PeerIDNil)
}
