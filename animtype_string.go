// Code generated by "stringer -linecomment -type AnimType"; DO NOT EDIT.

package mt

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[NoAnim-0]
	_ = x[VerticalFrameAnim-1]
	_ = x[SpriteSheetAnim-2]
	_ = x[maxAnim-3]
}

const _AnimType_name = "nonevertical framesprite sheetmaxAnim"

var _AnimType_index = [...]uint8{0, 4, 18, 30, 37}

func (i AnimType) String() string {
	if i >= AnimType(len(_AnimType_index)-1) {
		return "AnimType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _AnimType_name[_AnimType_index[i]:_AnimType_index[i+1]]
}
