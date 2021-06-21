package mt

import (
	"fmt"
	"io"
)

type PointedThing interface {
	pt()
}

func (*PointedNode) pt() {}
func (*PointedAO) pt()   {}

type PointedNode struct {
	Under, Above [3]int16
}

func PointedSameNode(pos [3]int16) PointedThing {
	return &PointedNode{pos, pos}
}

type PointedAO struct {
	ID AOID
}

func writePointedThing(w io.Writer, pt PointedThing) error {
	buf := make([]byte, 2)
	buf[0] = 0
	switch pt.(type) {
	case nil:
		buf[1] = 0
	case *PointedNode:
		buf[1] = 1
	case *PointedAO:
		buf[1] = 2
	default:
		panic(pt)
	}
	if _, err := w.Write(buf); err != nil {
		return err
	}
	if pt == nil {
		return nil
	}
	return serialize(w, pt)
}

func readPointedThing(r io.Reader) (PointedThing, error) {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	if buf[0] != 0 {
		return nil, fmt.Errorf("unsupported PointedThing version: %d", buf[0])
	}
	var pt PointedThing
	switch buf[1] {
	case 0:
		return nil, nil
	case 1:
		pt = new(PointedNode)
	case 2:
		pt = new(PointedAO)
	case 3:
		return nil, fmt.Errorf("invalid PointedThing type: %d", buf[1])
	}
	return pt, deserialize(r, pt)
}
