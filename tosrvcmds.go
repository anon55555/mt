package mt

import "github.com/anon55555/mt/rudp"

type ToSrvCmd interface {
	Cmd
	toSrvCmdNo() uint16
}

//go:generate ./cmdno.sh tosrvcmds ToSrv toSrv uint16 Cmd newToSrvCmd

// ToSrvNil is the first packet sent in a connection.
type ToSrvNil struct{}

// ToSrvInit is sent as unreliable after ToSrvNil and is re-sent repeatedly
// until either the server replies with ToCltHello or 10 seconds pass and
// the connection times out.
type ToSrvInit struct {
	SerializeVer             uint8
	SupportedCompression     CompressionModes
	MinProtoVer, MaxProtoVer uint16
	PlayerName               string

	//mt:opt
	SendFullItemMeta bool
}

// ToSrvInit2 is sent after ToCltAcceptAuth is received.
// The server responds to ToSrvInit2 by sending ToCltItemDefs, ToCltNodeDefs,
// ToCltAnnounceMedia, ToCltMovement and ToCltCSMRestrictionFlags.
type ToSrvInit2 struct {
	Lang string
}

// ToSrvJoinModChan attempts to join a mod channel.
type ToSrvJoinModChan struct {
	Channel string
}

// ToSrvLeaveModChan attempts to leave a mod channel.
type ToSrvLeaveModChan struct {
	Channel string
}

// ToSrvMsgModChan sends a message on a mod channel.
type ToSrvMsgModChan struct {
	Channel string
	Msg     string
}

// ToSrvPlayerPos tells the server that the client's PlayerPos has changed.
type ToSrvPlayerPos struct {
	Pos PlayerPos
}

// ToSrvGotBlks tells the server that the client has received Blks.
type ToSrvGotBlks struct {
	//mt:len8
	Blks [][3]int16
}

// ToSrvDeletedBlks tells the server that the client has deleted Blks.
type ToSrvDeletedBlks struct {
	//mt:len8
	Blks [][3]int16
}

// ToSrvInvAction tells the server that the client has performed an inventory action.
type ToSrvInvAction struct {
	//mt:raw
	Action string
}

// ToSrvChatMsg tells the server that the client has sent a chat message.
type ToSrvChatMsg struct {
	//mt:utf16
	Msg string
}

// ToSrvFallDmg tells the server that the client has taken fall damage.
type ToSrvFallDmg struct {
	Amount uint16
}

// ToSrvSelectItem tells the server the selected item in the client's hotbar.
type ToSrvSelectItem struct {
	Slot uint16
}

// ToSrvRespawn tells the server that the player has respawned.
type ToSrvRespawn struct{}

// ToSrvInteract tells the server that a node or AO has been interacted with.
type ToSrvInteract struct {
	Action   Interaction
	ItemSlot uint16
	//mt:lenhdr 32
	Pointed PointedThing
	//mt:end
	Pos PlayerPos
}

type Interaction uint8

const (
	Dig Interaction = iota
	StopDigging
	Dug
	Place
	Use      // Left click snowball-like.
	Activate // Right click air.
)

//go:generate stringer -type Interaction

// ToSrvRemovedSounds tells the server that the client has finished playing
// the sounds with the given IDs.
type ToSrvRemovedSounds struct {
	IDs []SoundID
}

type ToSrvNodeMetaFields struct {
	Pos      [3]int16
	Formname string
	Fields   []Field
}

type ToSrvInvFields struct {
	Formname string
	Fields   []Field
}

// ToSrvReqMedia requests media files from the server.
type ToSrvReqMedia struct {
	Filenames []string
}

type ToSrvCltReady struct {
	// Version information.
	Major, Minor, Patch uint8
	Reserved            uint8
	Version             string
	Formspec            uint16
}

type ToSrvFirstSRP struct {
	Salt        []byte
	Verifier    []byte
	EmptyPasswd bool
}

type ToSrvSRPBytesA struct {
	A      []byte
	NoSHA1 bool
}

type ToSrvSRPBytesM struct {
	M []byte
}

type ToSrvDisco struct{}

func (*ToSrvDisco) cmd()                         {}
func (*ToSrvDisco) toSrvCmdNo() uint16           { return 0xffff }
func (*ToSrvDisco) DefaultPktInfo() rudp.PktInfo { return rudp.PktInfo{} }
