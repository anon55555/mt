package mt

const (
	MaxLight = 14 // Maximum artificial light.
	SunLight = 15
)

type LightBank uint8

const (
	Day LightBank = iota
	Night
)

//go:generate stringer -type LightBank
