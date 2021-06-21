package mt

type ToolCaps struct {
	//mt:if _ = %s; false
	NonNil bool `json:"-"`
	//mt:end

	//mt:lenhdr 16

	//mt:ifde
	//mt:if r.N > 0 { %s.NonNil = true}; /**/
	//mt:if %s.NonNil

	//mt:const uint8(5)

	AttackCooldown float32 `json:"full_punch_interval"`
	MaxDropLvl     int16   `json:"max_drop_level"`

	//mt:len32
	GroupCaps []ToolGroupCaps `json:"groupcaps"`

	//mt:len32
	DmgGroups []Group `json:"damage_groups"`

	AttackUses uint16 `json:"punch_attack_uses"`

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
