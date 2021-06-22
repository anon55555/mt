package mt

type MinimapType uint16

const (
	NoMinimap      MinimapType = iota // none
	SurfaceMinimap                    // surface
	RadarMinimap                      // radar
	TextureMinimap                    // texture
)

//go:generate stringer -linecomment -type MinimapType

type MinimapMode struct {
	Type  MinimapType
	Label string
	Size  uint16
	Texture
	Scale uint16
}

// DefaultMinimap is the initial set of MinimapModes used by the client.
var DefaultMinimap = []MinimapMode{
	{Type: NoMinimap},
	{Type: SurfaceMinimap, Size: 256},
	{Type: SurfaceMinimap, Size: 128},
	{Type: SurfaceMinimap, Size: 64},
	{Type: RadarMinimap, Size: 512},
	{Type: RadarMinimap, Size: 256},
	{Type: RadarMinimap, Size: 128},
}
