package matchstore

import (
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type Outcome struct {
	Match     string          `json:"match"`
	Mode      rules.Mode      `json:"mode"`
	Kind      string          `json:"kind"`
	BotSeat   bool            `json:"bot_seat"`
	Cancelled bool            `json:"cancelled"`
	EndedAt   time.Time       `json:"ended_at"`
	Players   []PlayerOutcome `json:"players"`
}

type PlayerOutcome struct {
	Account      string `json:"account"`
	Team         int    `json:"team"`
	Won          bool   `json:"won"`
	Offender     bool   `json:"offender"`
	Returned     bool   `json:"returned"`
	HandsWon     int64  `json:"hands_won"`
	Dominoes     int64  `json:"dominoes"`
	BoardPoints  int64  `json:"board_points"`
	DoubleChains int64  `json:"double_chains"`
}
