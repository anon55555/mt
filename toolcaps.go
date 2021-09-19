package mt

import (
	"math"
	"time"
)

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
	GroupCaps []ToolGroupCap

	//mt:len32
	DmgGroups []Group

	//mt:32tou16
	PunchUses int32

	//mt:end
	//mt:end

	//mt:end
}

type ToolGroupCap struct {
	Name string

	//mt:32to16
	Uses int32

	MaxLvl int16

	//mt:len32
	Times []DigTime
}

type DigTime struct {
	Rating int16
	Time   float32
}

func (tc ToolCaps) DigTime(groups map[string]int16) (time.Duration, bool) {
	immDig := groups["dig_immediate"]

	minTime := float32(math.Inf(1))

	lvl := groups["level"]
	for _, gc := range tc.GroupCaps {
		if gc.Name == "dig_immediate" {
			immDig = 0
		}

		if lvl > gc.MaxLvl {
			continue
		}

		r := groups[gc.Name]
		for _, dt := range gc.Times {
			t := dt.Time
			if lvl < gc.MaxLvl {
				t /= float32(gc.MaxLvl - lvl)
			}
			if dt.Rating == r && t < minTime {
				minTime = t
			}
		}
	}

	switch immDig {
	case 2:
		return time.Second / 2, true
	case 3:
		return 0, true
	}

	if math.IsInf(float64(minTime), 1) {
		return 0, false
	}

	return time.Duration(math.Ceil(float64(minTime) * float64(time.Second))), true
}
