package penalties

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

var start = time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

func afk(byBot bool) rules.PresenceReport {
	return rules.PresenceReport{Timeouts: 3, MarkedAfk: true, ByBot: byBot}
}

func left(byBot bool) rules.PresenceReport {
	return rules.PresenceReport{Left: true, ByBot: byBot}
}

func TestDefaultConfigIsValid(t *testing.T) {
	if err := Validate(DefaultConfig()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsEveryBrokenField(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(*Config)
	}{
		{"zero weight", func(c *Config) { c.LeaveWeight = 0 }},
		{"zero window", func(c *Config) { c.OffenseWindowDays = 0 }},
		{"negative delay", func(c *Config) { c.QueueDelaySeconds = -1 }},
		{"ladder out of order", func(c *Config) { c.BanLevel = c.ReviewLevel + 1 }},
		{"escalation below one", func(c *Config) { c.BanEscalationFactor = 0.5 }},
		{"percent above 100", func(c *Config) { c.PartnerLossReductionPercent = 101 }},
		{"unequal casual steps", func(c *Config) { c.CasualQueueDelaySeconds = []int{60} }},
		{"casual levels not ascending", func(c *Config) { c.CasualQueueDelayLevels = []int{8, 4} }},
		{"casual delay heavier than ranked", func(c *Config) { c.CasualQueueDelaySeconds = []int{60, 301} }},
	}
	for _, tc := range cases {
		config := DefaultConfig()
		tc.break_(&config)
		if err := Validate(config); !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("%s: accepted (%v)", tc.name, err)
		}
	}
}

func TestDetectMapsTheReportToTheHeaviestOffense(t *testing.T) {
	config := DefaultConfig()
	cases := []struct {
		name   string
		report rules.PresenceReport
		want   Kind
	}{
		{"clean", rules.PresenceReport{}, KindNone},
		{"timeouts below the afk mark", rules.PresenceReport{Timeouts: 2}, KindNone},
		{"afk and back", afk(false), KindAfkReturned},
		{"afk until the end", afk(true), KindAfk},
		{"left and back", left(false), KindLeaveReturned},
		{"left for good", left(true), KindLeave},
		{"afk then left for good", rules.PresenceReport{MarkedAfk: true, Left: true, ByBot: true}, KindLeave},
		{"afk, left, came back", rules.PresenceReport{MarkedAfk: true, Left: true}, KindLeaveReturned},
	}
	for _, tc := range cases {
		if got := Detect(config, tc.report); got != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, got, tc.want)
		}
	}
	config.AfkWeight = 5
	if got := Detect(config, rules.PresenceReport{MarkedAfk: true, ByBot: true, Left: true}); got != KindAfk {
		t.Fatalf("the heavier configured weight wins: %s", got)
	}
	config.AfkWeight = 3
	if got := Detect(config, rules.PresenceReport{MarkedAfk: true, ByBot: true, Left: true}); got != KindLeave {
		t.Fatalf("equal weights fall to the leave: %s", got)
	}
}

func TestResultsPenaliseTheOffenderAndSoftenThePartnersLoss(t *testing.T) {
	config := DefaultConfig()
	var offenses [6]Kind
	offenses[rules.SeatNorth] = KindLeave
	results := Results(config, matchmaking.QueueRanked, rules.Mode2v2, rules.TeamB, true, offenses)
	north, south := results[rules.SeatNorth], results[rules.SeatSouth]
	if !north.CountsAsLoss || north.RatingPenalty != 10 || north.RewardsKept {
		t.Fatalf("offender: %+v", north)
	}
	if south.CountsAsLoss || south.RatingPenalty != 0 || !south.RewardsKept || south.LossReductionPercent != 50 {
		t.Fatalf("partner on the losing team: %+v", south)
	}
	for _, seat := range []rules.Seat{rules.SeatEast, rules.SeatWest} {
		if results[seat] != (SeatResult{RewardsKept: true}) {
			t.Fatalf("opponent %s: %+v", seat, results[seat])
		}
	}
	results = Results(config, matchmaking.QueueRanked, rules.Mode2v2, rules.TeamA, true, offenses)
	if !results[rules.SeatNorth].CountsAsLoss || results[rules.SeatSouth].LossReductionPercent != 0 {
		t.Fatalf("a win still counts as a loss for the offender only: %+v", results)
	}
	offenses[rules.SeatSouth] = KindAfkReturned
	results = Results(config, matchmaking.QueueRanked, rules.Mode2v2, rules.TeamB, true, offenses)
	if results[rules.SeatNorth].LossReductionPercent != 0 || results[rules.SeatSouth].LossReductionPercent != 0 {
		t.Fatalf("two offending partners get no reduction: %+v", results)
	}
	if !results[rules.SeatSouth].RewardsKept || results[rules.SeatSouth].RatingPenalty != 10 {
		t.Fatalf("a returned offender keeps rewards but takes the rating penalty: %+v", results[rules.SeatSouth])
	}
	results = Results(config, matchmaking.QueueCasual, rules.Mode2v2, rules.TeamB, true, offenses)
	if results[rules.SeatNorth].CountsAsLoss || results[rules.SeatNorth].RatingPenalty != 0 || results[rules.SeatNorth].RewardsKept {
		t.Fatalf("casual offender loses rewards only: %+v", results[rules.SeatNorth])
	}
	if results[rules.SeatSouth].LossReductionPercent != 0 || !results[rules.SeatEast].RewardsKept {
		t.Fatalf("casual partner and opponents are untouched: %+v", results)
	}
}

func TestLadderClimbsWithTheOffenseLevel(t *testing.T) {
	config := DefaultConfig()
	var record Record
	now := start
	ended := func(matchID string, report rules.PresenceReport) Notice {
		now = now.Add(time.Minute)
		return record.MatchEnd(config, matchmaking.QueueRanked, matchID, rules.SeatX, report, 7, now)
	}
	notice := ended("m1", afk(false))
	if !notice.Applied || notice.Kind != KindAfkReturned || notice.Level != 1 || notice.Step != StepWarning || notice.QueueDelay != 0 {
		t.Fatalf("first offense: %+v", notice)
	}
	notice = ended("m2", left(false))
	if notice.Level != 3 || notice.Step != StepQueueDelay || notice.QueueDelay != 300*time.Second {
		t.Fatalf("level 3: %+v", notice)
	}
	if record.QueueDelay(matchmaking.QueueRanked) != 300*time.Second || record.QueueDelay(matchmaking.QueueCasual) != 0 {
		t.Fatal("the queue delay applies to ranked entries only")
	}
	notice = ended("m3", left(true))
	if notice.Level != 6 || notice.Step != StepBan || !notice.BanUntil.Equal(now.Add(24*time.Hour)) || notice.UnderReview {
		t.Fatalf("level 6: %+v", notice)
	}
	if until, banned := record.Banned(now); !banned || !until.Equal(notice.BanUntil) {
		t.Fatal("not banned")
	}
	if _, banned := record.Banned(now.Add(24 * time.Hour)); banned {
		t.Fatal("still banned after the ban ends")
	}
	notice = ended("m4", left(true))
	if notice.Level != 9 || notice.Step != StepBan || !notice.BanUntil.Equal(now.Add(48*time.Hour)) {
		t.Fatalf("the second ban in the window doubles: %+v", notice)
	}
	notice = ended("m5", left(true))
	if notice.Level != 12 || notice.Step != StepReview || !notice.UnderReview || !notice.BanUntil.Equal(now.Add(96*time.Hour)) {
		t.Fatalf("level 12: %+v", notice)
	}
	if _, banned := record.Banned(now.Add(1000 * time.Hour)); !banned {
		t.Fatal("a review keeps the ban until an admin lifts it")
	}
	record.Lift()
	if _, banned := record.Banned(now); banned || record.QueueDelay(matchmaking.QueueRanked) != 0 {
		t.Fatal("lift did not clear the ban and the delay")
	}
	if record.Level(config, matchmaking.QueueRanked, now) != 12 {
		t.Fatal("lifting a penalty keeps the offenses")
	}
	offense := record.Ranked.Offenses[0]
	if offense.MatchID != "m1" || offense.Queue != matchmaking.QueueRanked || offense.Seat != rules.SeatX || offense.Evidence != afk(false) || offense.ConfigVersion != 7 || offense.Weight != 1 {
		t.Fatalf("stored offense: %+v", offense)
	}
}

func TestAMatchEndIsAppliedOnce(t *testing.T) {
	config := DefaultConfig()
	var record Record
	first := record.MatchEnd(config, matchmaking.QueueRanked, "m1", rules.SeatX, left(true), 1, start)
	again := record.MatchEnd(config, matchmaking.QueueRanked, "m1", rules.SeatX, left(true), 1, start.Add(time.Hour))
	if !first.Applied || again.Applied || again.Level != 3 || len(record.Ranked.Offenses) != 1 {
		t.Fatalf("replayed match end: first %+v again %+v", first, again)
	}
	record.MatchEnd(config, matchmaking.QueueRanked, "m2", rules.SeatX, rules.PresenceReport{}, 1, start.Add(2*time.Hour))
	if record.Ranked.CleanStreak != 1 {
		t.Fatalf("clean streak %d", record.Ranked.CleanStreak)
	}
	record.MatchEnd(config, matchmaking.QueueRanked, "m2", rules.SeatX, rules.PresenceReport{}, 1, start.Add(3*time.Hour))
	if record.Ranked.CleanStreak != 1 {
		t.Fatalf("a replayed clean match counted twice: streak %d", record.Ranked.CleanStreak)
	}
}

func TestOffensesExpireAndCleanMatchesForgive(t *testing.T) {
	config := DefaultConfig()
	config.CleanMatchesToForgive = 2
	var record Record
	record.MatchEnd(config, matchmaking.QueueRanked, "m1", rules.SeatX, left(true), 1, start)
	record.MatchEnd(config, matchmaking.QueueRanked, "m2", rules.SeatX, afk(true), 1, start.Add(time.Hour))
	if record.Level(config, matchmaking.QueueRanked, start.Add(time.Hour)) != 5 {
		t.Fatal("two offenses")
	}
	record.MatchEnd(config, matchmaking.QueueRanked, "m3", rules.SeatX, rules.PresenceReport{}, 1, start.Add(2*time.Hour))
	record.MatchEnd(config, matchmaking.QueueRanked, "m4", rules.SeatX, rules.PresenceReport{}, 1, start.Add(3*time.Hour))
	if level := record.Level(config, matchmaking.QueueRanked, start.Add(3*time.Hour)); level != 2 {
		t.Fatalf("two clean matches forgive the oldest offense: level %d", level)
	}
	if record.Ranked.CleanStreak != 0 {
		t.Fatal("the streak restarts after forgiving")
	}
	record.MatchEnd(config, matchmaking.QueueRanked, "m5", rules.SeatX, rules.PresenceReport{}, 1, start.Add(4*time.Hour))
	if record.Ranked.CleanStreak != 1 || len(record.Ranked.Offenses) != 1 {
		t.Fatal("one clean match forgives nothing")
	}
	later := start.Add(time.Hour + 30*24*time.Hour)
	if level := record.Level(config, matchmaking.QueueRanked, later); level != 0 {
		t.Fatalf("offenses expire after the window: level %d", level)
	}
	if record.Step(config, matchmaking.QueueRanked, later) != StepNone {
		t.Fatal("no step without active offenses")
	}
}

func TestQueueDelayLastsTheConfiguredMatches(t *testing.T) {
	config := DefaultConfig()
	config.QueueDelayMatches = 2
	var record Record
	now := start
	ended := func(matchID string, report rules.PresenceReport) {
		now = now.Add(time.Minute)
		record.MatchEnd(config, matchmaking.QueueRanked, matchID, rules.SeatX, report, 1, now)
	}
	ended("m1", left(true))
	if record.QueueDelay(matchmaking.QueueRanked) != 300*time.Second {
		t.Fatal("delay not applied")
	}
	ended("m2", rules.PresenceReport{})
	if record.QueueDelay(matchmaking.QueueRanked) != 300*time.Second {
		t.Fatal("delay ended after one match")
	}
	ended("m3", rules.PresenceReport{})
	if record.QueueDelay(matchmaking.QueueRanked) != 0 {
		t.Fatal("delay did not end after two matches")
	}
	ended("m4", left(true))
	ended("m5", left(true))
	if record.Ranked.DelayMatchesLeft != 2 {
		t.Fatalf("a new offense restarts the delay: %d left", record.Ranked.DelayMatchesLeft)
	}
}

func TestCasualLadderIsSeparateAndLighter(t *testing.T) {
	config := DefaultConfig()
	var record Record
	now := start
	ended := func(matchID string, report rules.PresenceReport) Notice {
		now = now.Add(time.Minute)
		return record.MatchEnd(config, matchmaking.QueueCasual, matchID, rules.SeatX, report, 1, now)
	}
	notice := ended("c1", afk(false))
	if notice.Step != StepNone || notice.Level != 1 {
		t.Fatalf("below the casual warning: %+v", notice)
	}
	notice = ended("c2", afk(false))
	if notice.Step != StepWarning {
		t.Fatalf("casual warning: %+v", notice)
	}
	notice = ended("c3", left(false))
	if notice.Step != StepQueueDelay || notice.QueueDelay != 60*time.Second {
		t.Fatalf("first casual delay step: %+v", notice)
	}
	ended("c4", left(true))
	notice = ended("c5", left(true))
	if notice.Level != 10 || notice.Step != StepQueueDelay || notice.QueueDelay != 180*time.Second || notice.UnderReview || !notice.BanUntil.IsZero() {
		t.Fatalf("second casual delay step without ban or review: %+v", notice)
	}
	for i := 0; i < 5; i++ {
		ended("c6", left(true))
	}
	if _, banned := record.Banned(now); banned || record.Level(config, matchmaking.QueueRanked, now) != 0 {
		t.Fatal("casual offenses never touch the ranked ladder")
	}
	if record.QueueDelay(matchmaking.QueueRanked) != 0 {
		t.Fatal("casual delay does not delay ranked entries")
	}
	if later := now.Add(14 * 24 * time.Hour); record.Level(config, matchmaking.QueueCasual, later) != 0 {
		t.Fatal("casual offenses expire after the casual window")
	}
}

func TestAdminRemovesAnOffense(t *testing.T) {
	config := DefaultConfig()
	var record Record
	record.MatchEnd(config, matchmaking.QueueRanked, "m1", rules.SeatX, left(true), 1, start)
	record.MatchEnd(config, matchmaking.QueueCasual, "c1", rules.SeatX, left(true), 1, start)
	if !record.Remove(matchmaking.QueueRanked, "m1") || record.Remove(matchmaking.QueueRanked, "m1") || record.Remove(matchmaking.QueueRanked, "c1") {
		t.Fatal("remove did not find the offense exactly once on its own ladder")
	}
	if record.Level(config, matchmaking.QueueRanked, start) != 0 || record.Level(config, matchmaking.QueueCasual, start) != 3 {
		t.Fatal("remove touched the wrong ladder")
	}
}

func TestNamesAreStable(t *testing.T) {
	kinds := []string{KindNone.String(), KindAfkReturned.String(), KindAfk.String(), KindLeaveReturned.String(), KindLeave.String(), Kind(9).String()}
	steps := []string{StepNone.String(), StepWarning.String(), StepQueueDelay.String(), StepBan.String(), StepReview.String(), Step(9).String()}
	if got := strings.Join(kinds, ","); got != "none,afk_returned,afk,leave_returned,leave,invalid" {
		t.Fatal(got)
	}
	if got := strings.Join(steps, ","); got != "none,warning,queue_delay,ban,review,invalid" {
		t.Fatal(got)
	}
}

func TestParseKeepsDefaultsForMissingKeysAndRefusesUnknownOrBrokenOnes(t *testing.T) {
	config, err := Parse([]byte(`{"banHours":48,"casualQueueDelayLevels":[3],"casualQueueDelaySeconds":[120]}`))
	if err != nil || config.BanHours != 48 || config.LeaveWeight != DefaultConfig().LeaveWeight || len(config.CasualQueueDelayLevels) != 1 {
		t.Fatalf("parse %+v, %v", config, err)
	}
	for _, raw := range []string{`{"banHour":48}`, `{"banLevel":2}`, `{"casualQueueDelaySeconds":[600,700]}`, `[`} {
		if _, err := Parse([]byte(raw)); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("%s: %v", raw, err)
		}
	}
}
