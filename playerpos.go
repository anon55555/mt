package mt

type Keys uint32

const (
	ForwardKey Keys = 1 << iota
	BackwardKey
	LeftKey
	RightKey
	JumpKey
	SpecialKey
	SneakKey
	DigKey
	PlaceKey
	ZoomKey
)

type PlayerPos struct {
	Pos100, Vel100   [3]int32
	Pitch100, Yaw100 int32
	Keys             Keys
	FOV80            uint8
	WantedRange      uint8 // in MapBlks.
}

func (p PlayerPos) Pos() (pos Pos) {
	for i := range pos {
		pos[i] = float32(p.Pos100[i]) / 100
	}
	return
}

func (p *PlayerPos) SetPos(pos Pos) {
	for i, x := range pos {
		p.Pos100[i] = int32(x * 100)
	}
}

func (p PlayerPos) Vel() (vel Vec) {
	for i := range vel {
		vel[i] = float32(p.Vel100[i]) / 100
	}
	return
}

func (p *PlayerPos) SetVel(vel Vec) {
	for i, x := range vel {
		p.Vel100[i] = int32(x * 100)
	}
}

func (p PlayerPos) Pitch() float32 {
	return float32(p.Pitch100) / 100
}

func (p *PlayerPos) SetPitch(pitch float32) {
	p.Pitch100 = int32(pitch * 100)
}

func (p PlayerPos) Yaw() float32 {
	return float32(p.Yaw100) / 100
}

func (p *PlayerPos) SetYaw(yaw float32) {
	p.Yaw100 = int32(yaw * 100)
}

func (p PlayerPos) FOV() float32 {
	return float32(p.FOV80) / 80
}

func (p *PlayerPos) SetFOV(fov float32) {
	p.FOV80 = uint8(fov * 80)
}
