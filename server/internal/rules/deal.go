package rules

import (
	"errors"
	"fmt"
)

var ErrInvalidDeal = errors.New("rules: invalid deal")

type Random interface {
	IntN(n int) int
}

type Deal struct {
	Hands    map[Seat]Hand
	Boneyard Boneyard
}

func Shuffle(tiles []Tile, random Random) {
	for i := len(tiles) - 1; i > 0; i-- {
		j := random.IntN(i + 1)
		tiles[i], tiles[j] = tiles[j], tiles[i]
	}
}

func NewDeal(mode Mode, config Config, random Random) Deal {
	set := FullSet()
	Shuffle(set[:], random)
	seats := mode.Seats()
	deal := Deal{Hands: make(map[Seat]Hand, len(seats))}
	next := 0
	for _, seat := range seats {
		deal.Hands[seat] = Hand(set[next : next+config.HandSize : next+config.HandSize])
		next += config.HandSize
	}
	deal.Boneyard = Boneyard(set[next:])
	return deal
}

func NeedsDoublesRedeal(mode Mode, config Config, hands map[Seat]Hand) bool {
	for _, seat := range mode.Seats() {
		if hands[seat].Doubles() >= config.RedealDoublesThreshold {
			return true
		}
	}
	return false
}

func DoublesRedealCanEnd(mode Mode, config Config) bool {
	doublesInSet := MaxPip + 1
	boneyardSize := SetSize - mode.Players()*config.HandSize
	return config.RedealDoublesThreshold >= 1 && (config.RedealDoublesThreshold-1)*mode.Players()+boneyardSize >= doublesInSet
}

func ValidateDeal(mode Mode, config Config, deal Deal) error {
	seats := mode.Seats()
	if len(seats) == 0 {
		return ErrInvalidMode
	}
	if len(deal.Hands) != len(seats) {
		return fmt.Errorf("%w: %d hands for %d seats", ErrInvalidDeal, len(deal.Hands), len(seats))
	}
	var seen [SetSize]bool
	dealt := 0
	for _, seat := range seats {
		hand, ok := deal.Hands[seat]
		if !ok {
			return fmt.Errorf("%w: seat %s has no hand", ErrInvalidDeal, seat)
		}
		if len(hand) != config.HandSize {
			return fmt.Errorf("%w: seat %s holds %d tiles, want %d", ErrInvalidDeal, seat, len(hand), config.HandSize)
		}
		if err := markTiles(&seen, hand); err != nil {
			return err
		}
		dealt += len(hand)
	}
	if len(deal.Boneyard) != SetSize-dealt {
		return fmt.Errorf("%w: boneyard holds %d tiles, want %d", ErrInvalidDeal, len(deal.Boneyard), SetSize-dealt)
	}
	return markTiles(&seen, deal.Boneyard)
}

func markTiles(seen *[SetSize]bool, tiles []Tile) error {
	for _, tile := range tiles {
		if !tile.Valid() {
			return fmt.Errorf("%w: %s", ErrInvalidTile, tile)
		}
		index := tileIndex(tile)
		if seen[index] {
			return fmt.Errorf("%w: %s appears twice", ErrInvalidDeal, tile)
		}
		seen[index] = true
	}
	return nil
}

func tileIndex(tile Tile) int {
	low := int(tile.Low)
	return low*(MaxPip+1) - low*(low-1)/2 + int(tile.High) - low
}
