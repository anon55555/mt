package mt

type ToolCaps struct {
	//mt:if _ = %s; false
	NonNil bool
	//mt:end

	//mt:lenhdr 16

	//mt:ifde
	//mt:if r.N > 0 { %s.NonNil = true}; /**/
	//mt:if %s.NonNil

	// Version.
	//mt:const uint8(5)

	AttackCooldown float32
	MaxDropLvl     int16

	//mt:len32
	GroupCaps []ToolGroupCaps

	//mt:len32
	DmgGroups []Group

	AttackUses uint16

	//mt:end
	//mt:end

	//mt:end
}

type ToolGroupCaps struct {
	Name   string
	Uses   int16
	MaxLvl int16

	//mt:len32
	Times []DigTime
}

type DigTime struct {
	Rating int16
	Time   float32
}
