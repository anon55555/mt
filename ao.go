package mt

import (
	"fmt"
	"image/color"
	"io"
)

type AOID uint16

type aoType uint8

const genericCAO aoType = 101

type AOInitData struct {
	// Version.
	//mt:const uint8(1)

	// For players.
	Name     string
	IsPlayer bool

	ID AOID

	Pos
	Rot [3]float32

	HP uint16

	// See (de)serialize.fmt.
	Msgs []AOMsg
}

type AOProps struct {
	// Version.
	//mt:const uint8(4)

	MaxHP            uint16 // Player only.
	CollideWithNodes bool
	Weight           float32 // deprecated
	ColBox, SelBox   Box
	Pointable        bool
	Visual           string
	VisualSize       [3]float32
	Textures         []Texture
	SpritesheetSize  [2]int16 // in sprites.
	SpritePos        [2]int16 // in sprite sheet.
	Visible          bool
	MakeFootstepSnds bool
	RotateSpeed      float32 // in radians per second.
	Mesh             string
	Colors           []color.NRGBA
	CollideWithAOs   bool
	StepHeight       float32
	FaceRotateDir    bool
	FaceRotateDirOff float32 // in degrees.
	BackfaceCull     bool
	Nametag          string
	NametagColor     color.NRGBA
	FaceRotateSpeed  float32 // in degrees per second.
	Infotext         string
	Itemstring       string
	Glow             int8
	MaxBreath        uint16  // Player only.
	EyeHeight        float32 // Player only.
	ZoomFOV          float32 // in degrees. Player only.
	UseTextureAlpha  bool
	DmgTextureMod    Texture // suffix
	Shaded           bool
	ShowOnMinimap    bool
	NametagBG        color.NRGBA
}

type AOPos struct {
	Pos
	Vel, Acc Vec
	Rot      [3]float32

	Interpolate    bool
	End            bool
	UpdateInterval float32
}

type AOSprite struct {
	Frame0          [2]int16
	Frames          uint16
	FrameDuration   float32
	ViewAngleFrames bool
}

type AOAnim struct {
	Frames [2]int32
	Speed  float32
	Blend  float32
	NoLoop bool
}

type AOBonePos struct {
	Pos Vec
	Rot [3]float32
}

type AOAttach struct {
	ParentID     AOID
	Bone         string
	Pos          Vec
	Rot          [3]float32
	ForceVisible bool
}

type AOPhysOverride struct {
	Walk, Jump, Gravity float32

	// Player only.
	NoSneak, NoSneakGlitch, OldSneak bool
}

type AOCmdProps struct {
	Props AOProps
}

type AOCmdPos struct {
	Pos AOPos
}

type AOCmdTextureMod struct {
	Mod Texture // suffix
}

type AOCmdSprite struct {
	Sprite AOSprite
}

type AOCmdHP struct {
	HP uint16
}

type AOCmdArmorGroups struct {
	Armor []Group
}

type AOCmdAnim struct {
	Anim AOAnim
}

type AOCmdBonePos struct {
	Bone string
	Pos  AOBonePos
}

type AOCmdAttach struct {
	Attach AOAttach
}

type AOCmdPhysOverride struct {
	Phys AOPhysOverride
}

type AOCmdSpawnInfant struct {
	ID AOID

	// Type.
	//mt:const genericCAO
}

type AOCmdAnimSpeed struct {
	Speed float32
}

//go:generate ./cmdno.sh aocmds AOCmd ao uint8 AOMsg newAOMsg

type AOMsg interface {
	aoCmdNo() uint8
}

func writeAOMsg(w io.Writer, msg AOMsg) error {
	if _, err := w.Write([]byte{msg.aoCmdNo()}); err != nil {
		return err
	}
	return serialize(w, msg)
}

func readAOMsg(r io.Reader) (AOMsg, error) {
	buf := make([]byte, 1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	newCmd, ok := newAOMsg[buf[0]]
	if !ok {
		return nil, fmt.Errorf("unsupported ao msg: %d", buf[0])
	}
	msg := newCmd()
	return msg, deserialize(r, msg)
}

type IDAOMsg struct {
	ID AOID
	//mt:lenhdr 16
	Msg AOMsg
	//mt:end
}
