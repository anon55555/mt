package mt

type AnimType uint8

const (
	NoAnim AnimType = iota
	VerticalFrameAnim
	SpritesheetAnim
	maxAnim
)

type TileAnim struct {
	Type AnimType
	//mt:assert %s.Type < maxAnim

	//mt:if %s.Type == SpritesheetAnim
	AspectRatio [2]uint8
	//mt:end

	//mt:if %s.Type == VerticalFrameAnim
	NFrames [2]uint16
	//mt:end

	//mt:if %s.Type != NoAnim
	Duration float32 // in seconds
	//mt:end
}
