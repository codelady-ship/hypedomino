package rules

type Config struct {
	HandSize                int
	RedealDoublesThreshold  int
	TargetScore             int
	ScoringMultiple         int
	HandPointsLockThreshold int
	FirstHandOpeningTile    Tile
	ScoringOpeningTile      Tile
	ScoringOpeningPoints    int
	TurnSeconds             int
	TurnWarningSeconds      int
	AfkConsecutiveTimeouts  int
	MatchLoadingSeconds     int
}

func DefaultConfig() Config {
	return Config{
		HandSize:                7,
		RedealDoublesThreshold:  6,
		TargetScore:             365,
		ScoringMultiple:         5,
		HandPointsLockThreshold: 300,
		FirstHandOpeningTile:    Tile{Low: 2, High: 3},
		ScoringOpeningTile:      Tile{Low: 5, High: 5},
		ScoringOpeningPoints:    10,
		TurnSeconds:             20,
		TurnWarningSeconds:      5,
		AfkConsecutiveTimeouts:  3,
		MatchLoadingSeconds:     10,
	}
}
