package mt

type AuthMethods uint32

const (
	LegacyPasswd AuthMethods = 1 << iota
	SRP
	FirstSRP
)

//go:generate stringer -type AuthMethods
