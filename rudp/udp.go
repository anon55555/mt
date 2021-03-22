package rudp

import "net"

const maxUDPPktSize = 512

type udpConn interface {
	recvUDP() ([]byte, error)
	Write([]byte) (int, error)
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}
