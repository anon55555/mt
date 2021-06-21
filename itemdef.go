package mt

import "image/color"

type ItemType uint8

const (
	_ ItemType = iota
	NodeItem
	CraftItem
	ToolItem
)

// An ItemDef defines the properties of an item.
type ItemDef struct {
	//mt:lenhdr 16

	// Version.
	//mt:const uint8(6)

	Type ItemType

	Name, Desc string

	InvImg, WieldImg Texture
	WieldScale       [3]float32

	StackMax uint16

	Usable          bool
	CanPointLiquids bool

	ToolCaps ToolCaps

	Groups []Group

	PlacePredict string

	PlaceSnd, PlaceFailSnd SoundDef

	PointRange float32

	// Set index in Palette with "palette_index" item meta field,
	// this overrides Color.
	Palette Texture
	Color   color.NRGBA

	// Texture overlays.
	InvOverlay, WieldOverlay Texture

	ShortDesc string

	PlaceParam2 byte

	//mt:end
}
