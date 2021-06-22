package mt

// A Dir represents a direction parallel to an axis.
type Dir uint8

const (
	East  Dir = iota // +X
	Above            // +Y
	North            // +Z
	South            // -Z
	Below            // -Y
	West             // -X
	NoDir
)

//go:generate stringer -type Dir

// Opposite returns the Dir's opposite.
// NoDir is its own opposite.
func (d Dir) Opposite() Dir {
	if d >= NoDir {
		return NoDir
	}
	return 5 - d
}
