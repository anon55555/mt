package mt

type SoundID int32

type SoundSrcType uint8

const (
	NoSrc SoundSrcType = iota
	PosSrc
	AOSrc
)

type SoundDef struct {
	Name              string
	Gain, Pitch, Fade float32
}
