// Code generated by "stringer -type AuthMethods"; DO NOT EDIT.

package mt

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[LegacyPasswd-1]
	_ = x[SRP-2]
	_ = x[FirstSRP-4]
}

const (
	_AuthMethods_name_0 = "LegacyPasswdSRP"
	_AuthMethods_name_1 = "FirstSRP"
)

var (
	_AuthMethods_index_0 = [...]uint8{0, 12, 15}
)

func (i AuthMethods) String() string {
	switch {
	case 1 <= i && i <= 2:
		i -= 1
		return _AuthMethods_name_0[_AuthMethods_index_0[i]:_AuthMethods_index_0[i+1]]
	case i == 4:
		return _AuthMethods_name_1
	default:
		return "AuthMethods(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
