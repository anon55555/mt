/*
Package rudp implements the low-level Minetest protocol described at
https://dev.minetest.net/Network_Protocol#Low-level_protocol.

All exported functions and methods in this package are safe for concurrent use
by multiple goroutines.
*/
package rudp

import "encoding/binary"

var be = binary.BigEndian

// protoID must be at the start of every network packet.
const protoID uint32 = 0x4f457403

// PeerIDs aren't actually used to identify peers, network addresses are,
// these just exist for backward compatability.
type PeerID uint16

const (
	// Used by clients before the server sets their ID.
	PeerIDNil PeerID = iota

	// The server always has this ID.
	PeerIDSrv

	// Lowest ID the server can assign to a client.
	PeerIDCltMin
)

// ChannelCount is the maximum channel number + 1.
const ChannelCount = 3

/*
rawPkt.Data format (big endian):

	rawType
	switch rawType {
	case rawTypeCtl:
		ctlType
		switch ctlType {
		case ctlAck:
			// Tells peer you received a rawTypeRel
			// and it doesn't need to resend it.
			seqnum
		case ctlSetPeerId:
			// Tells peer to send packets with this Src PeerID.
			PeerId
		case ctlPing:
			// Sent to prevent timeout.
		case ctlDisco:
			// Tells peer that you disconnected.
		}
	case rawTypeOrig:
		Pkt.(Data)
	case rawTypeSplit:
		// Packet larger than MaxNetPktSize split into smaller packets.
		// Packets with I >= Count should be ignored.
		// Once all Count chunks are recieved, they are sorted by I and
		// concatenated to make a Pkt.(Data).
		seqnum // Identifies split packet.
		Count, I uint16
		Chunk...
	case rawTypeRel:
		// Resent until a ctlAck with same seqnum is recieved.
		// seqnums are sequencial and start at seqnumInit,
		// These should be processed in seqnum order.
		seqnum
		rawPkt.Data
	}
*/
type rawPkt struct {
	Data  []byte
	ChNo  uint8
	Unrel bool
}

type rawType uint8

const (
	rawTypeCtl rawType = iota
	rawTypeOrig
	rawTypeSplit
	rawTypeRel
)

type ctlType uint8

const (
	ctlAck ctlType = iota
	ctlSetPeerID
	ctlPing
	ctlDisco
)

type Pkt struct {
	Data  []byte
	ChNo  uint8
	Unrel bool
}

// seqnums are sequence numbers used to maintain reliable packet order
// and to identify split packets.
type seqnum uint16

const seqnumInit seqnum = 65500
