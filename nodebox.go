package mt

type Box [2]Vec

type NodeBoxType uint8

const (
	CubeBox NodeBoxType = iota
	FixedBox
	MountedBox
	LeveledBox
	ConnectedBox
	maxBox
)

type DirBoxes struct {
	Top, Bot                 []Box
	Front, Left, Back, Right []Box
}

type NodeBox struct {
	//mt:const uint8(6)

	Type NodeBoxType
	//mt:assert %s.Type < maxBox

	//mt:if %s.Type == MountedBox
	WallTop, WallBot, WallSides Box
	//mt:end

	//mt:if t := %s.Type; t == FixedBox || t == LeveledBox || t == ConnectedBox
	Fixed []Box
	//mt:end

	//mt:if %s.Type == ConnectedBox
	ConnDirs, DiscoDirs  DirBoxes
	DiscoAll, DiscoSides []Box
	//mt:end
}
