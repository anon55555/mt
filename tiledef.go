package mt

type AlignType uint8

const (
	NoAlign AlignType = iota
	WorldAlign
	UserAlign
)

//go:generate stringer -type AlignType

type TileFlags uint16

const (
	TileBackfaceCull TileFlags = 1 << iota
	TileAbleH
	TileAbleV
	TileColor
	TileScale
	TileAlign
)

//go:generate stringer -type TileFlags

type TileDef struct {
	//mt:const uint8(6)

	Texture
	Anim  TileAnim
	Flags TileFlags

	//mt:if %s.Flags&TileColor != 0
	R, G, B uint8
	//mt:end

	//mt:if %s.Flags&TileScale != 0
	Scale uint8
	//mt:end

	//mt:if %s.Flags&TileAlign != 0
	Align AlignType
	//mt:end
}
