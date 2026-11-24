package pass

type Curve struct {
	Levels int `json:"levels"`
	XpBase int `json:"xpBase"`
	XpStep int `json:"xpStep"`
	XpMax  int `json:"xpMax"`
}

func (c Curve) XpToNext(level int) int {
	if level < 1 || level >= c.Levels {
		return 0
	}
	return min(c.XpBase+c.XpStep*(level-1), c.XpMax)
}

func (c Curve) XpToReach(level int) int64 {
	var total int64
	for l := 1; l < min(level, c.Levels+1); l++ {
		total += int64(c.XpToNext(l))
	}
	return total
}

func (c Curve) LevelFor(xp int64) int {
	level := 1
	for level < c.Levels {
		need := int64(c.XpToNext(level))
		if xp < need {
			return level
		}
		xp -= need
		level++
	}
	return level
}
