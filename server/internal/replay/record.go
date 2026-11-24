package replay

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

const Version = 3

var (
	ErrBadRecord     = errors.New("replay: malformed record")
	ErrIntentRefused = errors.New("replay: the engine refused a recorded intent")
	ErrResultDiffers = errors.New("replay: the replayed match differs from the recorded result")
)

var magic = [3]byte{'H', 'D', 'R'}

type Kind uint8

const (
	KindOpen Kind = iota + 1
	KindPlace
	KindEndTurn
	KindDraw
	KindPass
	kindEnd
)

type Intent struct {
	Kind Kind
	Seat rules.Seat
	Tile rules.Tile
	End  rules.End
}

type Result struct {
	Scores      rules.TeamPoints
	HandsPlayed int
	Winner      rules.Team
	HasWinner   bool
}

type Record struct {
	Mode      rules.Mode
	Seed      uint64
	TableDraw [32]byte
	Config    rules.Config
	Intents   []Intent
	Result    Result
	HasResult bool
}

func (r *Record) MarshalBinary() ([]byte, error) {
	data := make([]byte, 0, 64+3*len(r.Intents))
	data = append(data, magic[:]...)
	data = append(data, Version, byte(r.Mode))
	data = binary.LittleEndian.AppendUint64(data, r.Seed)
	data = append(data, r.TableDraw[:]...)
	data = appendConfig(data, r.Config)
	data = binary.AppendUvarint(data, uint64(len(r.Intents)))
	for _, intent := range r.Intents {
		data = append(data, byte(intent.Kind), byte(intent.Seat))
		switch intent.Kind {
		case KindOpen:
			data = append(data, packTile(intent.Tile))
		case KindPlace:
			data = append(data, packTile(intent.Tile), byte(intent.End))
		}
	}
	if !r.HasResult {
		return append(data, 0), nil
	}
	data = append(data, 1)
	data = binary.AppendUvarint(data, uint64(r.Result.Scores[rules.TeamA]))
	data = binary.AppendUvarint(data, uint64(r.Result.Scores[rules.TeamB]))
	data = binary.AppendUvarint(data, uint64(r.Result.HandsPlayed))
	winner := byte(0)
	if r.Result.HasWinner {
		winner = byte(r.Result.Winner) + 1
	}
	return append(data, winner), nil
}

func appendConfig(data []byte, c rules.Config) []byte {
	for _, value := range []int{c.HandSize, c.RedealDoublesThreshold, c.TargetScore, c.ScoringMultiple, c.HandPointsLockThreshold, c.ScoringOpeningPoints, c.TurnSeconds, c.AfkConsecutiveTimeouts} {
		data = binary.AppendUvarint(data, uint64(value))
	}
	return append(data, packTile(c.FirstHandOpeningTile), packTile(c.ScoringOpeningTile))
}

func packTile(tile rules.Tile) byte {
	return tile.Low*(rules.MaxPip+1) + tile.High
}

type reader struct {
	data []byte
	err  error
}

func (r *reader) byte() byte {
	if r.err != nil || len(r.data) == 0 {
		r.fail("truncated")
		return 0
	}
	value := r.data[0]
	r.data = r.data[1:]
	return value
}

func (r *reader) uvarint() int {
	if r.err != nil {
		return 0
	}
	value, n := binary.Uvarint(r.data)
	if n <= 0 || value > 1<<31 || len(binary.AppendUvarint(nil, value)) != n {
		r.fail("bad varint")
		return 0
	}
	r.data = r.data[n:]
	return int(value)
}

func (r *reader) tile() rules.Tile {
	packed := r.byte()
	low, high := packed/(rules.MaxPip+1), packed%(rules.MaxPip+1)
	tile, err := rules.NewTile(low, high)
	if err != nil || low > high {
		r.fail("bad tile")
	}
	return tile
}

func (r *reader) fail(what string) {
	if r.err == nil {
		r.err = fmt.Errorf("%w: %s", ErrBadRecord, what)
	}
}

func (r *Record) UnmarshalBinary(data []byte) error {
	in := reader{data: data}
	for _, expected := range magic {
		if in.byte() != expected {
			return fmt.Errorf("%w: bad magic", ErrBadRecord)
		}
	}
	if version := in.byte(); version != Version {
		return fmt.Errorf("%w: version %d", ErrBadRecord, version)
	}
	mode := rules.Mode(in.byte())
	if in.err != nil || mode.Players() == 0 {
		return fmt.Errorf("%w: bad mode", ErrBadRecord)
	}
	if len(in.data) < 40 {
		return fmt.Errorf("%w: truncated seed", ErrBadRecord)
	}
	seed := binary.LittleEndian.Uint64(in.data)
	var tableDraw [32]byte
	copy(tableDraw[:], in.data[8:40])
	in.data = in.data[40:]
	config := rules.Config{
		HandSize:                in.uvarint(),
		RedealDoublesThreshold:  in.uvarint(),
		TargetScore:             in.uvarint(),
		ScoringMultiple:         in.uvarint(),
		HandPointsLockThreshold: in.uvarint(),
		ScoringOpeningPoints:    in.uvarint(),
		TurnSeconds:             in.uvarint(),
		AfkConsecutiveTimeouts:  in.uvarint(),
		FirstHandOpeningTile:    in.tile(),
		ScoringOpeningTile:      in.tile(),
	}
	if in.err != nil {
		return in.err
	}
	if config.HandSize < 1 || config.HandSize*mode.Players() > rules.SetSize || config.ScoringMultiple < 1 || !rules.DoublesRedealCanEnd(mode, config) {
		return fmt.Errorf("%w: bad config", ErrBadRecord)
	}
	count := in.uvarint()
	if in.err != nil || count > len(in.data) {
		return fmt.Errorf("%w: bad intent count", ErrBadRecord)
	}
	intents := make([]Intent, 0, count)
	for range count {
		intent := Intent{Kind: Kind(in.byte()), Seat: rules.Seat(in.byte())}
		if in.err != nil || intent.Kind == 0 || intent.Kind >= kindEnd || !seatOf(mode, intent.Seat) {
			return fmt.Errorf("%w: bad intent", ErrBadRecord)
		}
		switch intent.Kind {
		case KindOpen:
			intent.Tile = in.tile()
		case KindPlace:
			intent.Tile = in.tile()
			intent.End = rules.End(in.byte())
			if intent.End > rules.EndBottom {
				in.fail("bad end")
			}
		}
		if in.err != nil {
			return in.err
		}
		intents = append(intents, intent)
	}
	record := Record{Mode: mode, Seed: seed, TableDraw: tableDraw, Config: config, Intents: intents}
	switch in.byte() {
	case 0:
	case 1:
		record.HasResult = true
		record.Result.Scores[rules.TeamA] = in.uvarint()
		record.Result.Scores[rules.TeamB] = in.uvarint()
		record.Result.HandsPlayed = in.uvarint()
		switch winner := in.byte(); winner {
		case 0:
		case 1, 2:
			record.Result.Winner = rules.Team(winner - 1)
			record.Result.HasWinner = true
		default:
			in.fail("bad winner")
		}
	default:
		in.fail("bad result flag")
	}
	if in.err != nil {
		return in.err
	}
	if len(in.data) != 0 {
		return fmt.Errorf("%w: %d trailing bytes", ErrBadRecord, len(in.data))
	}
	*r = record
	return nil
}

func seatOf(mode rules.Mode, seat rules.Seat) bool {
	for _, candidate := range mode.Seats() {
		if candidate == seat {
			return true
		}
	}
	return false
}
