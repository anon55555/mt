package mt

import "math"

// A Pos is a world space position,
// represented as a Vec from the origin.
type Pos Vec

// Add returns p+v.
func (p Pos) Add(v Vec) Pos {
	return Pos(Vec(p).Add(v))
}

// Sub returns p-v.
func (p Pos) Sub(v Vec) Pos {
	return Pos(Vec(p).Sub(v))
}

// From returns the Vec which moves to p from q.
func (p Pos) From(q Pos) Vec {
	return Vec(p).Sub(Vec(q))
}

// Int returns the position of the node which the Pos is inside.
func (p Pos) Int() (ip [3]int16) {
	for i := range ip {
		ip[i] = int16(math.Round(float64(p[i]) / 10))
	}
	return
}

// IntPos returns the Pos of the node at ip.
func IntPos(ip [3]int16) (p Pos) {
	for i := range p {
		p[i] = 10 * float32(ip[i])
	}
	return
}
