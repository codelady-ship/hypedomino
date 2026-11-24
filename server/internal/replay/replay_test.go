package replay

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func recordMatch(tb testing.TB, mode rules.Mode, seed uint64) Record {
	tb.Helper()
	recorder := NewRecorder(mode, rules.DefaultConfig(), seed, [32]byte{1, 2, 3})
	m := recorder.Match
	random := rules.NewSeededRandom(seed + 1000)
	players := map[rules.Seat]*bot.Player{}
	for i, seat := range mode.Seats() {
		players[seat] = bot.New(bot.Level(i%3), bot.DefaultConfig(), random)
	}
	var buffer [128]rules.Move
	for step := 0; m.Phase() != rules.PhaseFinished; step++ {
		if step > 20000 {
			tb.Fatalf("seed %d: match does not end", seed)
		}
		if err := act(recorder, players, random, buffer[:0]); err != nil {
			tb.Fatalf("seed %d step %d: %v", seed, step, err)
		}
	}
	return recorder.Record()
}

func act(r *Recorder, players map[rules.Seat]*bot.Player, random rules.Random, buffer []rules.Move) error {
	m := r.Match
	if m.Phase() == rules.PhaseOpening {
		opening, forced := m.ForcedOpening()
		seat := opening.Seat
		if !forced {
			seats := m.Mode().TeamSeats(m.OpeningTeam(), make([]rules.Seat, 0, 2))
			seat = seats[random.IntN(len(seats))]
			hand := m.Hand(seat)
			opening.Tile = hand[random.IntN(len(hand))]
		}
		_, err := r.Open(seat, opening.Tile)
		return err
	}
	seat := m.Current()
	moves := m.LegalMoves(seat, buffer)
	switch {
	case len(moves) > 0 && m.Turn().Placements() > 0 && random.IntN(2) == 0:
		return r.EndTurn(seat)
	case len(moves) > 0:
		move := moves[random.IntN(len(moves))]
		_, err := r.Place(seat, move.Tile, move.End)
		return err
	}
	if _, err := r.Draw(seat); err == nil {
		return nil
	} else if !errors.Is(err, rules.ErrCannotDraw) {
		return err
	}
	return r.Pass(seat)
}

func TestRecordRoundTripsAndReplaysToTheSameResult(t *testing.T) {
	for _, mode := range []rules.Mode{rules.Mode1v1, rules.Mode2v2} {
		for seed := uint64(1); seed <= 25; seed++ {
			record := recordMatch(t, mode, seed)
			if !record.HasResult || !record.Result.HasWinner {
				t.Fatalf("%s seed %d: no result", mode, seed)
			}
			data, err := record.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if len(data) > 72+4*len(record.Intents) {
				t.Fatalf("%s seed %d: %d bytes for %d intents", mode, seed, len(data), len(record.Intents))
			}
			var decoded Record
			if err := decoded.UnmarshalBinary(data); err != nil {
				t.Fatalf("%s seed %d: %v", mode, seed, err)
			}
			again, err := decoded.MarshalBinary()
			if err != nil || !bytes.Equal(data, again) {
				t.Fatalf("%s seed %d: re-encoding differs", mode, seed)
			}
			if err := Verify(decoded); err != nil {
				t.Fatalf("%s seed %d: %v", mode, seed, err)
			}
			replayed, _ := Replay(decoded)
			if replayed.Scores() != record.Result.Scores || replayed.HandsPlayed() != record.Result.HandsPlayed {
				t.Fatalf("%s seed %d: replay differs", mode, seed)
			}
		}
	}
}

func TestRecorderKeepsOnlyAcceptedIntents(t *testing.T) {
	recorder := NewRecorder(rules.Mode2v2, rules.DefaultConfig(), 3, [32]byte{})
	opening, _ := recorder.Match.ForcedOpening()
	if _, err := recorder.Open(opening.Seat.Next(), opening.Tile); err == nil {
		t.Fatal("wrong opener accepted")
	}
	if err := recorder.Pass(opening.Seat); err == nil {
		t.Fatal("pass before the opening accepted")
	}
	if _, err := recorder.Open(opening.Seat, opening.Tile); err != nil {
		t.Fatal(err)
	}
	record := recorder.Record()
	if len(record.Intents) != 1 || record.Intents[0] != (Intent{Kind: KindOpen, Seat: opening.Seat, Tile: opening.Tile}) || record.HasResult {
		t.Fatalf("record = %+v", record)
	}
	data, _ := record.MarshalBinary()
	var decoded Record
	if err := decoded.UnmarshalBinary(data); err != nil || decoded.HasResult || len(decoded.Intents) != 1 {
		t.Fatalf("unfinished record: %+v %v", decoded, err)
	}
	if err := Verify(decoded); err != nil {
		t.Fatalf("unfinished record does not verify: %v", err)
	}
}

func TestTamperedRecordsAreDetected(t *testing.T) {
	record := recordMatch(t, rules.Mode2v2, 7)
	tampered := record
	tampered.Intents = append([]Intent(nil), record.Intents...)
	tampered.Intents[len(tampered.Intents)/2].Seat = tampered.Intents[len(tampered.Intents)/2].Seat.Next()
	if err := Verify(tampered); !errors.Is(err, ErrIntentRefused) {
		t.Fatalf("a changed seat verifies: %v", err)
	}
	tampered = record
	tampered.Result.Scores[rules.TeamA] += 5
	if err := Verify(tampered); !errors.Is(err, ErrResultDiffers) {
		t.Fatalf("a changed score verifies: %v", err)
	}
	tampered = record
	tampered.Seed++
	if err := Verify(tampered); err == nil {
		t.Fatal("a changed seed verifies")
	}
	tampered = record
	tampered.Config.ScoringMultiple = 1
	if err := Verify(tampered); err == nil {
		t.Fatal("a changed config verifies")
	}
	tampered = record
	tampered.Intents = record.Intents[:len(record.Intents)-1]
	if err := Verify(tampered); !errors.Is(err, ErrResultDiffers) {
		t.Fatalf("a truncated record verifies: %v", err)
	}
	if err := Apply(rules.NewMatch(rules.Mode1v1, rules.DefaultConfig(), rules.NewSeededRandom(1)), Intent{Kind: 9}); !errors.Is(err, ErrBadRecord) {
		t.Fatalf("unknown kind: %v", err)
	}
}

func TestMalformedBytesAreRejected(t *testing.T) {
	record := recordMatch(t, rules.Mode1v1, 2)
	data, _ := record.MarshalBinary()
	cases := map[string][]byte{
		"empty":          {},
		"magic":          append([]byte("XXX"), data[3:]...),
		"version":        append(append([]byte{}, data[:3]...), append([]byte{9}, data[4:]...)...),
		"mode":           append(append([]byte{}, data[:4]...), append([]byte{7}, data[5:]...)...),
		"truncated":      data[:len(data)/2],
		"trailing":       append(append([]byte{}, data...), 0),
		"header only":    data[:5],
		"seed cut":       data[:9],
		"bad tile":       tileTampered(data),
		"bad result":     append(append([]byte{}, data[:len(data)-1]...), 9),
		"bad end":        endTampered(data),
		"endless redeal": endlessRedeal(record),
	}
	for name, bytes := range cases {
		var decoded Record
		if err := decoded.UnmarshalBinary(bytes); !errors.Is(err, ErrBadRecord) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func endlessRedeal(record Record) []byte {
	record.Config.RedealDoublesThreshold = 0
	out, _ := record.MarshalBinary()
	return out
}

func tileTampered(data []byte) []byte {
	out := append([]byte{}, data...)
	out[len(out)-1] = 1
	out[45] = 60
	return out
}

func endTampered(data []byte) []byte {
	var record Record
	if err := record.UnmarshalBinary(data); err != nil {
		panic(err)
	}
	for i := range record.Intents {
		if record.Intents[i].Kind == KindPlace {
			record.Intents[i].End = rules.End(9)
			break
		}
	}
	out, _ := record.MarshalBinary()
	return out
}

func FuzzUnmarshalBinary(f *testing.F) {
	record := recordMatch(f, rules.Mode2v2, 5)
	data, _ := record.MarshalBinary()
	f.Add(data)
	f.Add(data[:20])
	f.Add([]byte("HDR"))
	f.Fuzz(func(t *testing.T, data []byte) {
		var record Record
		if err := record.UnmarshalBinary(data); err != nil {
			return
		}
		again, err := record.MarshalBinary()
		if err != nil || !bytes.Equal(again, data) {
			t.Fatalf("decoded record does not re-encode to its bytes")
		}
	})
}

func BenchmarkVerify(b *testing.B) {
	record := recordMatch(b, rules.Mode2v2, 11)
	data, _ := record.MarshalBinary()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var decoded Record
		if err := decoded.UnmarshalBinary(data); err != nil {
			b.Fatal(err)
		}
		if err := Verify(decoded); err != nil {
			b.Fatal(err)
		}
	}
}
