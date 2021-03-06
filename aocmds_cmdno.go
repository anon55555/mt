// Code generated by cmdno.sh; DO NOT EDIT.

package mt

func (*AOCmdProps) aoCmdNo() uint8        { return 0 }
func (*AOCmdPos) aoCmdNo() uint8          { return 1 }
func (*AOCmdTextureMod) aoCmdNo() uint8   { return 2 }
func (*AOCmdSprite) aoCmdNo() uint8       { return 3 }
func (*AOCmdHP) aoCmdNo() uint8           { return 4 }
func (*AOCmdArmorGroups) aoCmdNo() uint8  { return 5 }
func (*AOCmdAnim) aoCmdNo() uint8         { return 6 }
func (*AOCmdBonePos) aoCmdNo() uint8      { return 7 }
func (*AOCmdAttach) aoCmdNo() uint8       { return 8 }
func (*AOCmdPhysOverride) aoCmdNo() uint8 { return 9 }
func (*AOCmdSpawnInfant) aoCmdNo() uint8  { return 11 }
func (*AOCmdAnimSpeed) aoCmdNo() uint8    { return 12 }

var newAOMsg = map[uint8]func() AOMsg{
	0:  func() AOMsg { return new(AOCmdProps) },
	1:  func() AOMsg { return new(AOCmdPos) },
	2:  func() AOMsg { return new(AOCmdTextureMod) },
	3:  func() AOMsg { return new(AOCmdSprite) },
	4:  func() AOMsg { return new(AOCmdHP) },
	5:  func() AOMsg { return new(AOCmdArmorGroups) },
	6:  func() AOMsg { return new(AOCmdAnim) },
	7:  func() AOMsg { return new(AOCmdBonePos) },
	8:  func() AOMsg { return new(AOCmdAttach) },
	9:  func() AOMsg { return new(AOCmdPhysOverride) },
	11: func() AOMsg { return new(AOCmdSpawnInfant) },
	12: func() AOMsg { return new(AOCmdAnimSpeed) },
}
