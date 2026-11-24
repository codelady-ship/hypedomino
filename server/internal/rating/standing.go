package rating

import (
	"context"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type SeasonClock struct {
	ID       string    `json:"id"`
	NameKey  string    `json:"name_key"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

type ModeStanding struct {
	Mode             string `json:"mode"`
	Placed           bool   `json:"placed"`
	PlacementPlayed  int    `json:"placement_played"`
	PlacementMatches int    `json:"placement_matches"`
	Rank             string `json:"rank"`
	Tier             string `json:"tier"`
	LP               int    `json:"lp"`
	DivisionLP       int    `json:"division_lp"`
	Wins             int    `json:"wins"`
}

type Standing struct {
	Season SeasonClock    `json:"season"`
	Modes  []ModeStanding `json:"modes"`
}

func (s *Store) Standing(ctx context.Context, account string, now time.Time) (Standing, error) {
	season, err := s.SeasonAt(ctx, now)
	if err != nil {
		return Standing{}, err
	}
	standing := Standing{Season: SeasonClock{ID: season.ID, NameKey: season.NameKey, StartsAt: season.Start, EndsAt: season.End}, Modes: make([]ModeStanding, 0, len(modes))}
	for _, mode := range modes {
		config, _, err := s.Config(mode)
		if err != nil {
			return Standing{}, err
		}
		r, ok, err := s.Get(ctx, account, mode)
		if err != nil {
			return Standing{}, err
		}
		if !ok || (r.Season != "" && r.Season != season.ID) {
			r = Rating{}
		}
		standing.Modes = append(standing.Modes, StandingOf(mode, config, r))
	}
	return standing, nil
}

func StandingOf(mode rules.Mode, config Config, r Rating) ModeStanding {
	standing := ModeStanding{Mode: mode.String(), Placed: r.Placed, PlacementPlayed: min(r.PlacementPlayed, config.PlacementMatches), PlacementMatches: config.PlacementMatches, DivisionLP: config.DivisionLp, Wins: r.Wins}
	if r.Placed {
		standing.PlacementPlayed = config.PlacementMatches
		standing.Rank, standing.Tier, standing.LP = r.Label(), r.Tier, r.Rank.LP
	}
	return standing
}
