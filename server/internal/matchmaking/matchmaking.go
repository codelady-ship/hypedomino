package matchmaking

import (
	"math/rand/v2"
	"sort"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type Queue uint8

const (
	QueueRanked Queue = iota + 1
	QueueCasual
)

func (q Queue) String() string {
	switch q {
	case QueueRanked:
		return "ranked"
	case QueueCasual:
		return "casual"
	}
	return "invalid"
}

type Config struct {
	MmrWindowStart       int
	MmrWindowStep        int
	MmrWindowStepSeconds int
	MmrWindowMax         int
	RankWindowDivisions  int
	CasualBotFillSeconds int
}

func DefaultConfig() Config {
	return Config{
		MmrWindowStart:       100,
		MmrWindowStep:        50,
		MmrWindowStepSeconds: 10,
		MmrWindowMax:         400,
		RankWindowDivisions:  8,
		CasualBotFillSeconds: 30,
	}
}

type Entry struct {
	Account    string
	Mode       rules.Mode
	Queue      Queue
	Region     string
	Mmr        int
	Rank       int
	Placed     bool
	Avoid      []string
	EnqueuedAt time.Time
	SearchFrom time.Time
}

type Seat struct {
	Seat    rules.Seat
	Account string
	Bot     bool
}

type Match struct {
	Mode   rules.Mode
	Queue  Queue
	Region string
	Seats  []Seat
}

func (m Match) Ranked() bool {
	return m.Queue == QueueRanked
}

func (m Match) Teams() [2][]string {
	var teams [2][]string
	for _, seat := range m.Seats {
		if !seat.Bot {
			teams[seat.Seat.Team()] = append(teams[seat.Seat.Team()], seat.Account)
		}
	}
	return teams
}

func Window(config Config, waited time.Duration) int {
	if waited < 0 {
		waited = 0
	}
	steps := 0
	if config.MmrWindowStepSeconds > 0 {
		steps = int(waited / (time.Duration(config.MmrWindowStepSeconds) * time.Second))
	}
	window := config.MmrWindowStart + steps*config.MmrWindowStep
	if window > config.MmrWindowMax {
		return config.MmrWindowMax
	}
	return window
}

type bucket struct {
	mode   rules.Mode
	queue  Queue
	region string
}

func Form(config Config, entries []Entry, now time.Time, random *rand.Rand) ([]Match, []Entry) {
	buckets := map[bucket][]Entry{}
	var order []bucket
	for _, entry := range entries {
		key := bucket{entry.Mode, entry.Queue, entry.Region}
		if _, seen := buckets[key]; !seen {
			order = append(order, key)
		}
		buckets[key] = append(buckets[key], entry)
	}
	var matches []Match
	var remaining []Entry
	for _, key := range order {
		formed, left := formBucket(config, key, buckets[key], now, random)
		matches = append(matches, formed...)
		remaining = append(remaining, left...)
	}
	return matches, remaining
}

func formBucket(config Config, key bucket, entries []Entry, now time.Time, random *rand.Rand) ([]Match, []Entry) {
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].EnqueuedAt.Before(entries[j].EnqueuedAt) })
	players := key.mode.Players()
	taken := make([]bool, len(entries))
	var matches []Match
	for index, anchor := range entries {
		if taken[index] || anchor.SearchFrom.After(now) {
			continue
		}
		group := []int{index}
		window := Window(config, now.Sub(anchor.EnqueuedAt))
		for other := range entries {
			if len(group) == players {
				break
			}
			if other == index || taken[other] || entries[other].SearchFrom.After(now) || avoids(anchor, entries[other]) {
				continue
			}
			tolerance := min(window, Window(config, now.Sub(entries[other].EnqueuedAt)))
			if abs(anchor.Mmr-entries[other].Mmr) <= tolerance && rankClose(config, key.queue, anchor, entries[other]) {
				group = append(group, other)
			}
		}
		if len(group) < players && !botFill(config, key.queue, now.Sub(anchor.EnqueuedAt)) {
			continue
		}
		for _, member := range group {
			taken[member] = true
		}
		accounts := make([]string, 0, len(group))
		for _, member := range group {
			accounts = append(accounts, entries[member].Account)
		}
		matches = append(matches, seat(key, accounts, random))
	}
	var remaining []Entry
	for index, entry := range entries {
		if !taken[index] {
			remaining = append(remaining, entry)
		}
	}
	return matches, remaining
}

func avoids(a, b Entry) bool {
	for _, account := range a.Avoid {
		if account == b.Account {
			return true
		}
	}
	for _, account := range b.Avoid {
		if account == a.Account {
			return true
		}
	}
	return false
}

func rankClose(config Config, queue Queue, a, b Entry) bool {
	if queue != QueueRanked || config.RankWindowDivisions <= 0 || !a.Placed || !b.Placed {
		return true
	}
	return abs(a.Rank-b.Rank) <= config.RankWindowDivisions
}

func botFill(config Config, queue Queue, waited time.Duration) bool {
	if queue != QueueCasual || config.CasualBotFillSeconds <= 0 {
		return false
	}
	return waited >= time.Duration(config.CasualBotFillSeconds)*time.Second
}

func seat(key bucket, accounts []string, random *rand.Rand) Match {
	random.Shuffle(len(accounts), func(i, j int) { accounts[i], accounts[j] = accounts[j], accounts[i] })
	seats := key.mode.Seats()
	match := Match{Mode: key.mode, Queue: key.queue, Region: key.region, Seats: make([]Seat, len(seats))}
	for index, s := range seats {
		match.Seats[index] = Seat{Seat: s, Bot: true}
		if index < len(accounts) {
			match.Seats[index] = Seat{Seat: s, Account: accounts[index]}
		}
	}
	return match
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
