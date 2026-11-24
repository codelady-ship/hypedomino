package penalties

import (
	"slices"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
)

type OffenseView struct {
	MatchID   string    `json:"match_id"`
	Kind      string    `json:"kind"`
	Weight    int       `json:"weight"`
	At        time.Time `json:"at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type LadderView struct {
	Level                 int           `json:"level"`
	Step                  string        `json:"step"`
	NextStepLevel         int           `json:"next_step_level"`
	QueueDelaySeconds     int           `json:"queue_delay_seconds"`
	DelayMatchesLeft      int           `json:"delay_matches_left"`
	BanUntil              *time.Time    `json:"ban_until"`
	UnderReview           bool          `json:"under_review"`
	CleanStreak           int           `json:"clean_streak"`
	CleanMatchesToForgive int           `json:"clean_matches_to_forgive"`
	Offenses              []OffenseView `json:"offenses"`
}

type View struct {
	Ranked LadderView `json:"ranked"`
	Casual LadderView `json:"casual"`
}

func (r Record) View(c Config, now time.Time) View {
	return View{Ranked: r.ladderView(c, matchmaking.QueueRanked, now), Casual: r.ladderView(c, matchmaking.QueueCasual, now)}
}

func (r *Record) ladderView(c Config, queue matchmaking.Queue, now time.Time) LadderView {
	policy := rulesFor(c, queue)
	ladder := *r.ladder(queue)
	ladder.Offenses, ladder.Completed, ladder.Bans = slices.Clone(ladder.Offenses), slices.Clone(ladder.Completed), slices.Clone(ladder.Bans)
	ladder.prune(policy.window, now)
	step, _ := ladder.step(policy)
	view := LadderView{
		Level:                 ladder.level(),
		Step:                  step.String(),
		NextStepLevel:         nextStepLevel(policy, ladder.level()),
		DelayMatchesLeft:      ladder.DelayMatchesLeft,
		UnderReview:           ladder.UnderReview,
		CleanStreak:           ladder.CleanStreak,
		CleanMatchesToForgive: policy.forgiveAfter,
		Offenses:              []OffenseView{},
	}
	if ladder.DelayMatchesLeft > 0 {
		view.QueueDelaySeconds = ladder.DelaySeconds
	}
	if now.Before(ladder.BanUntil) {
		until := ladder.BanUntil
		view.BanUntil = &until
	}
	for _, o := range ladder.Offenses {
		view.Offenses = append(view.Offenses, OffenseView{MatchID: o.MatchID, Kind: o.Kind.String(), Weight: o.Weight, At: o.At, ExpiresAt: o.At.Add(policy.window)})
	}
	return view
}

func nextStepLevel(policy ladderRules, level int) int {
	next := 0
	for _, from := range append([]int{policy.warningLevel, policy.banLevel, policy.reviewLevel}, policy.delayLevels...) {
		if from > level && (next == 0 || from < next) {
			next = from
		}
	}
	return next
}
