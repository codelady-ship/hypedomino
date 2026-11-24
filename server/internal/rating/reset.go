package rating

import (
	"fmt"
)

type ResetTable [ApexIndex + 1]int

func (c Config) ResetTable() (ResetTable, error) {
	floor, err := ParseRankIndex(c.ResetFloor)
	if err != nil {
		return ResetTable{}, fmt.Errorf("resetFloor: %w", err)
	}
	ceiling, err := ParseRankIndex(c.ResetCeiling)
	if err != nil {
		return ResetTable{}, fmt.Errorf("resetCeiling: %w", err)
	}
	if floor > ceiling || ceiling > MaxDivisionIndex {
		return ResetTable{}, fmt.Errorf("%w: resetFloor must not exceed resetCeiling and both must be division ranks", ErrInvalidConfig)
	}
	var table ResetTable
	for old := range table {
		table[old] = min(ceiling, max(old-4*c.ResetTierDrop, min(old, floor)))
	}
	for name, target := range c.SeasonResetTable {
		old, err := ParseRankIndex(name)
		if err != nil {
			return ResetTable{}, fmt.Errorf("seasonResetTable: %w", err)
		}
		reset, err := ParseRankIndex(target)
		if err != nil {
			return ResetTable{}, fmt.Errorf("seasonResetTable %s: %w", name, err)
		}
		table[old] = reset
	}
	for old, reset := range table {
		switch {
		case reset > MaxDivisionIndex:
			return ResetTable{}, fmt.Errorf("%w: %s resets into the apex tiers", ErrInvalidConfig, Rank{Index: old})
		case reset > old:
			return ResetTable{}, fmt.Errorf("%w: %s resets above itself", ErrInvalidConfig, Rank{Index: old})
		case old > 0 && reset < table[old-1]:
			return ResetTable{}, fmt.Errorf("%w: %s resets below %s", ErrInvalidConfig, Rank{Index: old}, Rank{Index: old - 1})
		}
	}
	return table, nil
}

func (c Config) ResetIndex(old int) int {
	table, err := c.ResetTable()
	if err != nil {
		return min(old, MaxDivisionIndex)
	}
	return table[min(max(old, 0), ApexIndex)]
}
