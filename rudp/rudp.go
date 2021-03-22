/*
Package rudp implements the low-level Minetest protocol described at
https://dev.minetest.net/Network_Protocol#Low-level_protocol.
*/
package rudp

import (
	"encoding/binary"
	"errors"
	"io"
	"time"
)

var be = binary.BigEndian

/*
UDP packet format:

	protoID
	src PeerID
	channel uint8
	rawType...
*/

var ErrTimedOut = errors.New("timed out")

const (
	ConnTimeout = 30 * time.Second
	PingTimeout = 5 * time.Second
)

const (
	MaxRelPktSize   = 32439825
	MaxUnrelPktSize = 32636430
)

// protoID must be at the start of every UDP packet.
const protoID uint32 = 0x4f457403

// PeerIDs aren't actually used to identify peers, IP addresses and ports are,
// these just exist for backward compatibility.
type PeerID uint16

const (
	// Used by clients before the server sets their ID.
	PeerIDNil PeerID = iota

	// The server always has this ID.
	PeerIDSrv

	// Lowest ID the server can assign to a client.
	PeerIDCltMin
)

type rawType uint8

const (
	rawCtl rawType = iota
	// ctlType...

	rawOrig
	// data...

	rawSplit
	// seqnum
	// n, i uint16
	// data...

	rawRel
	// seqnum
	// rawType...
)

type ctlType uint8

const (
	ctlAck ctlType = iota
	// seqnum

	ctlSetPeerID
	// PeerID

	ctlPing // Sent to prevent timeout.

	ctlDisco
)

type Pkt struct {
	io.Reader
	PktInfo
}

// Reliable packets in a channel are be received in the order they are sent in.
// A Channel must be less than ChannelCount.
type Channel uint8

const ChannelCount Channel = 3

type PktInfo struct {
	Channel

	// Unrel (unreliable) packets may be dropped, duplicated or reordered.
	Unrel bool
}

// seqnums are sequence numbers used to maintain reliable packet order
// and identify split packets.
type seqnum uint16

const initSeqnum seqnum = 65500
