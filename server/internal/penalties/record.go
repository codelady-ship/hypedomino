package penalties

import (
	"math"
	"slices"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type Step uint8

const (
	StepNone Step = iota
	StepWarning
	StepQueueDelay
	StepBan
	StepReview
)

var stepNames = [...]string{"none", "warning", "queue_delay", "ban", "review"}

func (s Step) String() string {
	if int(s) < len(stepNames) {
		return stepNames[s]
	}
	return "invalid"
}

type Offense struct {
	MatchID       string
	Queue         matchmaking.Queue
	Seat          rules.Seat
	Kind          Kind
	Weight        int
	Evidence      rules.PresenceReport
	At            time.Time
	ConfigVersion int
}

type Completion struct {
	MatchID string
	At      time.Time
}

type Ladder struct {
	Offenses         []Offense
	Completed        []Completion
	CleanStreak      int
	DelayMatchesLeft int
	DelaySeconds     int
	Bans             []time.Time
	BanUntil         time.Time
	BanSanction      int64
	UnderReview      bool
}

type Record struct {
	Ranked Ladder
	Casual Ladder
}

type Notice struct {
	Applied     bool
	Kind        Kind
	Level       int
	Step        Step
	QueueDelay  time.Duration
	BanUntil    time.Time
	UnderReview bool
}

type ladderRules struct {
	window        time.Duration
	forgiveAfter  int
	delayMatches  int
	warningLevel  int
	delayLevels   []int
	delaySeconds  []int
	banLevel      int
	reviewLevel   int
	banHours      int
	banEscalation float64
}

func rulesFor(c Config, queue matchmaking.Queue) ladderRules {
	if queue == matchmaking.QueueCasual {
		return ladderRules{
			window:       time.Duration(c.CasualOffenseWindowDays) * 24 * time.Hour,
			forgiveAfter: c.CasualCleanMatchesToForgive,
			delayMatches: c.CasualQueueDelayMatches,
			warningLevel: c.CasualWarningLevel,
			delayLevels:  c.CasualQueueDelayLevels,
			delaySeconds: c.CasualQueueDelaySeconds,
		}
	}
	return ladderRules{
		window:        time.Duration(c.OffenseWindowDays) * 24 * time.Hour,
		forgiveAfter:  c.CleanMatchesToForgive,
		delayMatches:  c.QueueDelayMatches,
		warningLevel:  c.WarningLevel,
		delayLevels:   []int{c.QueueDelayLevel},
		delaySeconds:  []int{c.QueueDelaySeconds},
		banLevel:      c.BanLevel,
		reviewLevel:   c.ReviewLevel,
		banHours:      c.BanHours,
		banEscalation: c.BanEscalationFactor,
	}
}

func (r *Record) ladder(queue matchmaking.Queue) *Ladder {
	if queue == matchmaking.QueueCasual {
		return &r.Casual
	}
	return &r.Ranked
}

func (l *Ladder) prune(window time.Duration, now time.Time) {
	expired := func(at time.Time) bool { return !at.Add(window).After(now) }
	l.Offenses = slices.DeleteFunc(l.Offenses, func(o Offense) bool { return expired(o.At) })
	l.Completed = slices.DeleteFunc(l.Completed, func(c Completion) bool { return expired(c.At) })
	l.Bans = slices.DeleteFunc(l.Bans, expired)
}

func (l *Ladder) level() int {
	level := 0
	for _, offense := range l.Offenses {
		level += offense.Weight
	}
	return level
}

func (l *Ladder) seen(matchID string) bool {
	return slices.ContainsFunc(l.Completed, func(c Completion) bool { return c.MatchID == matchID })
}

func (l *Ladder) step(policy ladderRules) (Step, int) {
	level := l.level()
	step := StepNone
	delay := 0
	if level >= policy.warningLevel {
		step = StepWarning
	}
	for index, from := range policy.delayLevels {
		if level >= from {
			step, delay = StepQueueDelay, policy.delaySeconds[index]
		}
	}
	if policy.banLevel > 0 && level >= policy.banLevel {
		step = StepBan
	}
	if policy.reviewLevel > 0 && level >= policy.reviewLevel {
		step = StepReview
	}
	return step, delay
}

func (l *Ladder) notice(policy ladderRules, kind Kind, applied bool) Notice {
	step, _ := l.step(policy)
	return Notice{
		Applied:     applied,
		Kind:        kind,
		Level:       l.level(),
		Step:        step,
		QueueDelay:  l.delay(),
		BanUntil:    l.BanUntil,
		UnderReview: l.UnderReview,
	}
}

func (l *Ladder) delay() time.Duration {
	if l.DelayMatchesLeft <= 0 {
		return 0
	}
	return time.Duration(l.DelaySeconds) * time.Second
}

func (r *Record) MatchEnd(c Config, queue matchmaking.Queue, matchID string, seat rules.Seat, report rules.PresenceReport, configVersion int, now time.Time) Notice {
	policy := rulesFor(c, queue)
	ladder := r.ladder(queue)
	ladder.prune(policy.window, now)
	if ladder.seen(matchID) {
		return ladder.notice(policy, KindNone, false)
	}
	ladder.Completed = append(ladder.Completed, Completion{MatchID: matchID, At: now})
	if ladder.DelayMatchesLeft > 0 {
		ladder.DelayMatchesLeft--
	}
	kind := Detect(c, report)
	if kind == KindNone {
		ladder.forgive(policy.forgiveAfter)
		return ladder.notice(policy, kind, true)
	}
	ladder.CleanStreak = 0
	ladder.Offenses = append(ladder.Offenses, Offense{
		MatchID: matchID, Queue: queue, Seat: seat, Kind: kind, Weight: kind.Weight(c),
		Evidence: report, At: now, ConfigVersion: configVersion,
	})
	ladder.escalate(policy, now)
	return ladder.notice(policy, kind, true)
}

func (l *Ladder) forgive(after int) {
	l.CleanStreak++
	if after <= 0 || l.CleanStreak < after || len(l.Offenses) == 0 {
		return
	}
	l.Offenses = l.Offenses[1:]
	l.CleanStreak = 0
}

func (l *Ladder) escalate(policy ladderRules, now time.Time) {
	step, delaySeconds := l.step(policy)
	if step < StepQueueDelay {
		return
	}
	l.DelayMatchesLeft = policy.delayMatches
	l.DelaySeconds = max(l.DelaySeconds, delaySeconds)
	if step < StepBan {
		return
	}
	hours := float64(policy.banHours) * math.Pow(policy.banEscalation, float64(len(l.Bans)))
	l.BanUntil = now.Add(time.Duration(hours * float64(time.Hour)))
	l.Bans = append(l.Bans, now)
	if step == StepReview {
		l.UnderReview = true
	}
}

func (r *Record) Level(c Config, queue matchmaking.Queue, now time.Time) int {
	ladder := r.ladder(queue)
	ladder.prune(rulesFor(c, queue).window, now)
	return ladder.level()
}

func (r *Record) Step(c Config, queue matchmaking.Queue, now time.Time) Step {
	ladder := r.ladder(queue)
	ladder.prune(rulesFor(c, queue).window, now)
	step, _ := ladder.step(rulesFor(c, queue))
	return step
}

func (r *Record) QueueDelay(queue matchmaking.Queue) time.Duration {
	return r.ladder(queue).delay()
}

func (r *Record) Banned(now time.Time) (time.Time, bool) {
	if r.Ranked.UnderReview {
		return time.Time{}, true
	}
	if now.Before(r.Ranked.BanUntil) {
		return r.Ranked.BanUntil, true
	}
	return time.Time{}, false
}

func (r *Record) Lift() {
	r.Ranked.UnderReview = false
	r.Ranked.BanUntil = time.Time{}
	r.Ranked.BanSanction = 0
	r.Ranked.DelayMatchesLeft = 0
	r.Casual.DelayMatchesLeft = 0
}

func (r *Record) Remove(queue matchmaking.Queue, matchID string) bool {
	ladder := r.ladder(queue)
	before := len(ladder.Offenses)
	ladder.Offenses = slices.DeleteFunc(ladder.Offenses, func(o Offense) bool { return o.MatchID == matchID })
	return len(ladder.Offenses) != before
}
