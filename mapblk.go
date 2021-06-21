package mt

type MapBlkFlags uint8

const (
	BlkIsUnderground MapBlkFlags = 1 << iota
	BlkDayNightDiff
	BlkLightExpired
	BlkNotGenerated
)

type LitFromBlks uint16

const AlwaysLitFrom LitFromBlks = 0xf000

func LitFrom(d Dir, b LightBank) LitFromBlks {
	return 1 << (uint8(d) + uint8(6*b))
}

type MapBlk struct {
	Flags   MapBlkFlags
	LitFrom LitFromBlks

	//mt:const uint8(2)     // Size of param0 in bytes.
	//mt:const uint8(1 + 1) // Size of param1 and param2 combined, in bytes.

	//mt:zlib
	Param0 [4096]Content
	Param1 [4096]uint8
	Param2 [4096]uint8
	//mt:end

	NodeMetas map[uint16]*NodeMeta

	// net info
	//mt:const uint8(2) // version
}

// Pos2BlkPos converts a node position to a MapBlk position and index.
func Pos2Blkpos(pos [3]int16) (blkpos [3]int16, i uint16) {
	for j := range pos {
		blkpos[j] = pos[j] >> 4
		i |= uint16(pos[j]&0xf) << (4 * j)
	}

	return
}

// BlkPos2Pos converts a MapBlk position and index to a node position.
func Blkpos2Pos(blkpos [3]int16, i uint16) (pos [3]int16) {
	for j := range pos {
		pos[j] = blkpos[j]<<4 | int16(i>>(4*j)&0xf)
	}

	return
}
