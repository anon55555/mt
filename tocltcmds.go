package mt

import (
	"crypto/sha1"
	"fmt"
	"image/color"
	"io"
	"math"
)

type ToCltCmd interface {
	Cmd
	toCltCmdNo() uint16
}

//go:generate ./cmdno.sh tocltcmds ToClt toClt uint16 Cmd newToCltCmd

// ToCltHello is sent as a response to ToSrvInit.
// The client responds to ToCltHello by authenticating.
type ToCltHello struct {
	SerializeVer uint8
	Compression  CompressionModes
	ProtoVer     uint16
	AuthMethods
	Username string
}

// ToCltAcceptAuth is sent after the client successfully authenticates.
// The client responds to ToCltAcceptAuth with ToSrvInit2.
type ToCltAcceptAuth struct {
	// The client does the equivalent of
	//	PlayerPos[1] -= 5
	// before using PlayerPos.
	PlayerPos Pos

	MapSeed         uint64
	SendInterval    float32
	SudoAuthMethods AuthMethods
}

type ToCltAcceptSudoMode struct{}

type ToCltDenySudoMode struct{}

// ToCltDisco tells that the client that it has been disconnected by the server.
type ToCltDisco struct {
	Reason DiscoReason
	//mt:assert %s.Reason < maxDiscoReason

	//mt:if dr := %s.Reason; dr == Custom || dr == Shutdown || dr == Crash
	Custom string
	//mt:end

	//mt:if dr := %s.Reason; dr == Shutdown || dr == Crash
	Reconnect bool
	//mt:end
}

type DiscoReason uint8

const (
	WrongPasswd DiscoReason = iota
	UnexpectedData
	SrvIsSingleplayer
	UnsupportedVer
	BadNameChars
	BadName
	TooManyClts
	EmptyPasswd
	AlreadyConnected
	SrvErr
	Custom
	Shutdown
	Crash
	maxDiscoReason
)

func (cmd ToCltDisco) String() (msg string) {
	switch cmd.Reason {
	case WrongPasswd:
		return "wrong password"
	case UnexpectedData:
		return "unexpected data"
	case SrvIsSingleplayer:
		return "server is singleplayer"
	case UnsupportedVer:
		return "unsupported client version"
	case BadNameChars:
		return "disallowed character(s) in player name"
	case BadName:
		return "disallowed player name"
	case TooManyClts:
		return "too many clients"
	case EmptyPasswd:
		return "empty password"
	case AlreadyConnected:
		return "another client is already connected with the same name"
	case SrvErr:
		return "server error"
	case Custom:
		return cmd.Custom
	case Shutdown:
		msg = "server shutdown"
	case Crash:
		msg = "server crash"
	default:
		msg = fmt.Sprintf("DiscoReason(%d)", cmd.Reason)
	}

	if cmd.Custom != "" {
		msg += ": " + cmd.Custom
	}

	return
}

// ToCltBlkData tells the client the contents of a nearby MapBlk.
type ToCltBlkData struct {
	Blkpos [3]int16
	Blk    MapBlk
}

// ToCltAddNode tells the client that a nearby node changed
// to something other than air.
type ToCltAddNode struct {
	Pos [3]int16
	Node
	KeepMeta bool
}

// ToCltRemoveNode tells the client that a nearby node changed to air.
type ToCltRemoveNode struct {
	Pos [3]int16
}

// ToCltInv updates the client's inventory.
type ToCltInv struct {
	//mt:raw
	Inv string
}

// ToCltTimeOfDay updates the client's in-game time of day.
type ToCltTimeOfDay struct {
	Time  uint16  // %24000
	Speed float32 // Speed times faster than real time
}

// ToCltCSMRestrictionFlags tells the client how use of CSMs should be restricted.
type ToCltCSMRestrictionFlags struct {
	Flags CSMRestrictionFlags

	// MapRange is the maximum distance from the player CSMs can read the map
	// if Flags&LimitMapRange != 0.
	MapRange uint32
}

type CSMRestrictionFlags uint64

const (
	NoCSMs CSMRestrictionFlags = 1 << iota
	NoChatMsgs
	NoItemDefs
	NoNodeDefs
	LimitMapRange
	NoPlayerList
)

// ToCltAddPlayerVel tells the client to add Vel to the player's velocity.
type ToCltAddPlayerVel struct {
	Vel Vec
}

// ToCltMediaPush is sent when a media file is dynamically added.
type ToCltMediaPush struct {
	//mt:const uint16(sha1.Size)
	SHA1        [sha1.Size]byte
	Filename    string
	ShouldCache bool

	//mt:len32
	Data []byte
}

// ToCltChatMsg tells the client that is has received a chat message.
type ToCltChatMsg struct {
	//mt:const uint8(1)

	Type ChatMsgType

	//mt:utf16
	Sender, Text string

	Timestamp int64 // Unix time.
}

type ChatMsgType uint8

const (
	RawMsg      ChatMsgType = iota // raw
	NormalMsg                      // normal
	AnnounceMsg                    // announce
	SysMsg                         // sys
	maxMsg
)

//go:generate stringer -linecomment -type ChatMsgType

// ToCltAORmAdd tells the client that AOs have been removed from and/or added to
// the AOs that it can see.
type ToCltAORmAdd struct {
	Remove []AOID
	Add    []struct {
		ID AOID
		//mt:const genericCAO
		//mt:lenhdr 32
		InitData AOInitData
		//mt:end
	}
}

// ToCltAOMsgs updates the client about nearby AOs.
type ToCltAOMsgs struct {
	//mt:raw
	Msgs []IDAOMsg
}

// ToCltHP updates the player's HP on the client.
type ToCltHP struct {
	HP uint16
}

// ToCltMovePlayer tells the client that the player has been moved server-side.
type ToCltMovePlayer struct {
	Pos
	Pitch, Yaw float32
}

type ToCltDiscoLegacy struct {
	//mt:utf16
	Reason string
}

// ToCltFOV tells the client to change its FOV.
type ToCltFOV struct {
	FOV            float32
	Multiplier     bool
	TransitionTime float32
}

// ToCltDeathScreen tells the client to show the death screen.
type ToCltDeathScreen struct {
	PointCam bool
	PointAt  Pos
}

// ToCltMedia responds to a ToSrvMedia packet with the requested media files.
type ToCltMedia struct {
	// N is the total number of ToCltMedia packets.
	// I is the index of this packet.
	N, I uint16

	//mt:len32
	Files []struct {
		Name string

		//mt:len32
		Data []byte
	}
}

// ToCltNodeDefs tells the client the definitions of nodes.
type ToCltNodeDefs struct {
	//mt:lenhdr 32
	//mt:zlib

	// Version.
	//mt:const uint8(1)

	// See (de)serialize.fmt.
	Defs []NodeDef

	//mt:end
	//mt:end
}

// ToCltAnnounceMedia tells the client what media is available on request.
// See ToSrvReqMedia.
type ToCltAnnounceMedia struct {
	Files []struct {
		Name       string
		Base64SHA1 string
	}
	URL string
}

// ToCltItemDefs tells the client the definitions of items.
type ToCltItemDefs struct {
	//mt:lenhdr 32
	//mt:zlib

	//mt:const uint8(0)

	Defs    []ItemDef
	Aliases []struct{ Alias, Orig string }

	//mt:end
	//mt:end
}

// ToCltPlaySound tells the client to play a sound.
type ToCltPlaySound struct {
	ID      SoundID
	Name    string
	Gain    float32
	SrcType SoundSrcType
	Pos
	SrcAOID   AOID
	Loop      bool
	Fade      float32
	Pitch     float32
	Ephemeral bool
}

// ToCltStopSound tells the client to stop playing a sound.
type ToCltStopSound struct {
	ID SoundID
}

// ToCltPrivs tells the client its privs.
type ToCltPrivs struct {
	Privs []string
}

// ToCltInvFormspec tells the client its inventory formspec.
type ToCltInvFormspec struct {
	//mt:len32
	Formspec string
}

// ToCltDetachedInv updates a detached inventory on the client.
type ToCltDetachedInv struct {
	Name string
	Keep bool
	Len  uint16 // deprecated

	//mt:raw
	Inv string
}

// ToCltShowFormspec tells the client to show a formspec.
type ToCltShowFormspec struct {
	//mt:len32
	Formspec string

	Formname string
}

// ToCltMovement tells the client how to move.
type ToCltMovement struct {
	DefaultAccel, AirAccel, FastAccel,
	WalkSpeed, CrouchSpeed, FastSpeed, ClimbSpeed, JumpSpeed,
	Fluidity, Smoothing, Sink, // liquids
	Gravity float32
}

// ToCltSpawnParticle tells the client to spawn a particle.
type ToCltSpawnParticle struct {
	Pos, Vel, Acc  [3]float32
	ExpirationTime float32 // in seconds.
	Size           float32
	Collide        bool

	//mt:len32
	Texture

	Vertical    bool
	CollisionRm bool
	AnimParams  TileAnim
	Glow        uint8
	AOCollision bool
	NodeParam0  Content
	NodeParam2  uint8
	NodeTile    uint8
}

type ParticleSpawnerID uint32

// ToCltAddParticleSpawner tells the client to add a particle spawner.
type ToCltAddParticleSpawner struct {
	Amount         uint16
	Duration       float32
	Pos, Vel, Acc  [2][3]float32
	ExpirationTime [2]float32 // in seconds.
	Size           [2]float32
	Collide        bool

	//mt:len32
	Texture

	ID           ParticleSpawnerID
	Vertical     bool
	CollisionRm  bool
	AttachedAOID AOID
	AnimParams   TileAnim
	Glow         uint8
	AOCollision  bool
	NodeParam0   Content
	NodeParam2   uint8
	NodeTile     uint8
}

type HUDID uint32

// ToCltHUDAdd tells the client to add a HUD.
type ToCltAddHUD struct {
	ID HUDID

	Type HUDType

	Pos      [2]float32
	Name     string
	Scale    [2]float32
	Text     string
	Number   uint32
	Item     uint32
	Dir      uint32
	Align    [2]float32
	Offset   [2]float32
	WorldPos Pos
	Size     [2]int32
	ZIndex   int16
	Text2    string
}

type HUDType uint8

const (
	ImgHUD HUDType = iota
	TextHUD
	StatbarHUD
	InvHUD
	WaypointHUD
	ImgWaypointHUD
)

//go:generate stringer -type HUDType

// ToCltRmHUD tells the client to remove a HUD.
type ToCltRmHUD struct {
	ID HUDID
}

// ToCltChangeHUD tells the client to change a field in a HUD.
type ToCltChangeHUD struct {
	ID HUDID

	Field HUDField

	//mt:assert %s.Field < hudMax

	//mt:if %s.Field == HUDPos
	Pos [2]float32
	//mt:end

	//mt:if %s.Field == HUDName
	Name string
	//mt:end

	//mt:if %s.Field == HUDScale
	Scale [2]float32
	//mt:end

	//mt:if %s.Field == HUDText
	Text string
	//mt:end

	//mt:if %s.Field == HUDNumber
	Number uint32
	//mt:end

	//mt:if %s.Field == HUDItem
	Item uint32
	//mt:end

	//mt:if %s.Field == HUDDir
	Dir uint32
	//mt:end

	//mt:if %s.Field == HUDAlign
	Align [2]float32
	//mt:end

	//mt:if %s.Field == HUDOffset
	Offset [2]float32
	//mt:end

	//mt:if %s.Field == HUDWorldPos
	WorldPos Pos
	//mt:end

	//mt:if %s.Field == HUDSize
	Size [2]int32
	//mt:end

	//mt:if %s.Field == HUDZIndex
	ZIndex uint32
	//mt:end

	//mt:if %s.Field == HUDText2
	Text2 string
	//mt:end
}

type HUDField uint8

const (
	HUDPos HUDField = iota
	HUDName
	HUDScale
	HUDText
	HUDNumber
	HUDItem
	HUDDir
	HUDAlign
	HUDOffset
	HUDWorldPos
	HUDSize
	HUDZIndex
	HUDText2
	hudMax
)

//go:generate stringer -trimprefix HUD -type HUDField

// ToCltHUDFlags tells the client to update its HUD flags.
type ToCltHUDFlags struct {
	// &^= Mask
	// |= Flags
	Flags, Mask HUDFlags
}

type HUDFlags uint32

const (
	ShowHotbar HUDFlags = 1 << iota
	ShowHealthBar
	ShowCrosshair
	ShowWieldedItem
	ShowBreathBar
	ShowMinimap
	ShowRadarMinimap
)

// ToCltSetHotbarParam tells the client to set a hotbar parameter.
type ToCltSetHotbarParam struct {
	Param HotbarParam

	//mt:if %s.Param == HotbarSize
	//mt:const uint16(4) // Size of Size field.
	Size int32
	//mt:end

	//mt:if %s.Param != HotbarSize
	Img Texture
	//mt:end
}

type HotbarParam uint16

const (
	HotbarSize HotbarParam = 1 + iota
	HotbarImg
	HotbarSelImg
)

//go:generate stringer -trimprefix Hotbar -type HotbarParam

// ToCltBreath tells the client how much breath it has.
type ToCltBreath struct {
	Breath uint16
}

// ToCltSkyParams tells the client how to render the sky.
type ToCltSkyParams struct {
	BgColor     color.NRGBA
	Type        string
	Clouds      bool
	SunFogTint  color.NRGBA
	MoonFogTint color.NRGBA
	FogTintType string

	//mt:if %s.Type == "skybox"
	Textures []Texture
	//mt:end

	//mt:if %s.Type == "regular"
	DaySky, DayHorizon,
	DawnSky, DawnHorizon,
	NightSky, NightHorizon,
	Indoor color.NRGBA
	//mt:end
}

// ToCltOverrideDayNightRatio overrides the client's day-night ratio
type ToCltOverrideDayNightRatio struct {
	Override bool
	Ratio    uint16
}

// ToCltLocalPlayerAnim tells the client how to animate the player.
type ToCltLocalPlayerAnim struct {
	Idle, Walk, Dig, WalkDig [2]int32
	Speed                    float32
}

// ToCltEyeOffset tells the client where to position the camera
// relative to the player.
type ToCltEyeOffset struct {
	First, Third Vec
}

// ToCltDelParticleSpawner tells the client to delete a particle spawner.
type ToCltDelParticleSpawner struct {
	ID ParticleSpawnerID
}

// ToCltCloudParams tells the client how to render the clouds.
type ToCltCloudParams struct {
	Density      float32
	DiffuseColor color.NRGBA
	AmbientColor color.NRGBA
	Height       float32
	Thickness    float32
	Speed        [2]float32
}

// ToCltFadeSound tells the client to fade a sound.
type ToCltFadeSound struct {
	ID   SoundID
	Step float32
	Gain float32
}

// ToCltUpdatePlayerList informs the client of players leaving or joining.
type ToCltUpdatePlayerList struct {
	Type    PlayerListUpdateType
	Players []string
}

type PlayerListUpdateType uint8

const (
	InitPlayers   PlayerListUpdateType = iota // init
	AddPlayers                                // add
	RemovePlayers                             // remove
)

//go:generate stringer -linecomment -type PlayerListUpdateType

// ToCltModChanMsg tells the client it has been sent a message on a mod channel.
type ToCltModChanMsg struct {
	Channel string
	Sender  string
	Msg     string
}

// ToCltModChanMsg tells the client it has received a signal on a mod channel.
type ToCltModChanSig struct {
	Signal  ModChanSig
	Channel string
}

type ModChanSig uint8

const (
	JoinOK ModChanSig = iota
	JoinFail
	LeaveOK
	LeaveFail
	NotRegistered
	SetState
)

//go:generate stringer -type ModChanSig

// ToCltModChanMsg is sent when node metadata near the client changes.
type ToCltNodeMetasChanged struct {
	//mt:lenhdr 32
	Changed map[[3]int16]*NodeMeta
	//mt:end
}

// ToCltSunParams tells the client how to render the sun.
type ToCltSunParams struct {
	Visible bool
	Texture
	ToneMap Texture
	Rise    Texture
	Rising  bool
	Size    float32
}

// ToCltMoonParams tells the client how to render the moon.
type ToCltMoonParams struct {
	Visible bool
	Texture
	ToneMap Texture
	Size    float32
}

// ToCltStarParams tells the client how to render the stars.
type ToCltStarParams struct {
	Visible bool
	Count   uint32
	Color   color.NRGBA
	Size    float32
}

type ToCltSRPBytesSaltB struct {
	Salt, B []byte
}

// ToCltFormspecPrepend tells the client to prepend a string to all formspecs.
type ToCltFormspecPrepend struct {
	Prepend string
}

// ToCltMinimapModes tells the client the set of available minimap modes.
type ToCltMinimapModes struct {
	Current uint16
	Modes   []MinimapMode
}

func (cmd *ToCltMinimapModes) serialize(w io.Writer) error {
	buf := make([]byte, 4)
	if len(cmd.Modes) > math.MaxUint16 {
		return ErrTooLong
	}
	be.PutUint16(buf[0:2], uint16(len(cmd.Modes)))
	be.PutUint16(buf[2:4], cmd.Current)
	if _, err := w.Write(buf); err != nil {
		return err
	}
	for i := range cmd.Modes {
		if err := serialize(w, &cmd.Modes[i]); err != nil {
			return err
		}
	}
	return nil
}

func (cmd *ToCltMinimapModes) deserialize(r io.Reader) error {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	cmd.Modes = make([]MinimapMode, be.Uint16(buf[0:2]))
	cmd.Current = be.Uint16(buf[2:4])
	for i := range cmd.Modes {
		if err := deserialize(r, &cmd.Modes[i]); err != nil {
			return err
		}
	}
	return nil
}
