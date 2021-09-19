package mt

import "image/color"

type Param1Type uint8

const (
	P1Nothing Param1Type = iota
	P1Light
)

//go:generate stringer -trimprefix P1 -type Param1Type

type Param2Type uint8

const (
	P2Nibble Param2Type = iota
	P2Byte
	P2Flowing
	P2FaceDir
	P2Mounted
	P2Leveled
	P2Rotation
	P2Mesh
	P2Color
	P2ColorFaceDir
	P2ColorMounted
	P2GlassLikeLevel
)

//go:generate stringer -trimprefix P2 -type Param2Type

// A DrawType specifies how a node is drawn.
type DrawType uint8

const (
	DrawCube DrawType = iota
	DrawNothing
	DrawLiquid
	DrawFlowing
	DrawLikeGlass
	DrawAllFaces
	DrawAllFacesOpt
	DrawTorch
	DrawSign
	DrawPlant
	DrawFence
	DrawRail
	DrawNodeBox
	DrawGlassFrame
	DrawFire
	DrawGlassFrameOpt
	DrawMesh
	DrawRootedPlant
)

//go:generate stringer -trimprefix Draw -type DrawType

type WaveType uint8

const (
	NotWaving    WaveType = iota
	PlantWaving           // Only top waves from side to side.
	LeafWaving            // Wave side to side.
	LiquidWaving          // Wave up and down.
)

//go:generate stringer -type WaveType

type LiquidType uint8

const (
	NotALiquid LiquidType = iota
	FlowingLiquid
	LiquidSrc
)

//go:generate stringer -type LiquidType

// AlphaUse specifies how the alpha channel of a texture is used.
type AlphaUse uint8

const (
	Blend AlphaUse = iota
	Mask           // "Rounded" to either fully opaque or transparent.
	Opaque
	Legacy
)

//go:generate stringer -type AlphaUse

type NodeDef struct {
	Param0 Content

	//mt:lenhdr 16

	//mt:const uint8(13)

	Name   string
	Groups []Group

	P1Type   Param1Type
	P2Type   Param2Type
	DrawType DrawType

	Mesh  string
	Scale float32
	//mt:const uint8(6)
	Tiles        [6]TileDef
	OverlayTiles [6]TileDef
	//mt:const uint8(6)
	SpecialTiles [6]TileDef

	Color   color.NRGBA
	Palette Texture

	Waving       WaveType
	ConnectSides uint8
	ConnectTo    []Content
	InsideTint   color.NRGBA
	Level        uint8 // Must be < 128.

	Translucent bool // Sunlight is scattered and becomes normal light.
	Transparent bool // Sunlight isn't scattered.
	LightSrc    uint8

	GndContent   bool
	Collides     bool
	Pointable    bool
	Diggable     bool
	Climbable    bool
	Replaceable  bool
	OnRightClick bool

	DmgPerSec int32

	LiquidType   LiquidType
	FlowingAlt   string
	SrcAlt       string
	Viscosity    uint8 // 0-7
	LiqRenewable bool
	FlowRange    uint8
	DrownDmg     uint8
	Floodable    bool

	DrawBox, ColBox, SelBox NodeBox

	FootstepSnd, DiggingSnd, DugSnd SoundDef

	LegacyFaceDir bool
	LegacyMounted bool

	DigPredict string

	MaxLvl uint8

	AlphaUse

	//mt:end
}

func BuiltinNodeDefs(n int) map[Content]NodeDef {
	defs := make(map[Content]NodeDef, 3+n)
	defs[Unknown] = NodeDef{
		Name: "unknown",
	}
	defs[Air] = NodeDef{
		Name:        "air",
		DrawType:    DrawNothing,
		P1Type:      P1Light,
		Translucent: true,
		Transparent: true,
		Replaceable: true,
		Floodable:   true,
		GndContent:  true,
	}
	defs[Ignore] = NodeDef{
		Name:        "ignore",
		DrawType:    DrawNothing,
		Replaceable: true,
		GndContent:  true,
	}
	return defs
}
