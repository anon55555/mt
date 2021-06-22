package mt

type SoundID int32

type SoundSrcType uint8

const (
	NoSrc SoundSrcType = iota // nowhere
	PosSrc                    // pos
	AOSrc                     // ao
)

//go:generate stringer -linecomment -type SoundSrcType

type SoundDef struct {
	Name              string
	Gain, Pitch, Fade float32
}
