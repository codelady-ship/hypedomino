package rating_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func (f *fixture) removeFromBoard(season, account string, at time.Time) (rating.BoardRemoval, error) {
	var removal rating.BoardRemoval
	err := pgx.BeginFunc(context.Background(), f.pool, func(tx pgx.Tx) error {
		var err error
		removal, err = rating.RemoveFromBoardIn(context.Background(), tx, season, rules.Mode1v1, account, at)
		return err
	})
	return removal, err
}

func TestARemovedAccountLeavesTheBoardOfThatSeasonAndMode(t *testing.T) {
	f := newFixture(t)
	boards := &fakeBoards{}
	f.store.SetBoards(boards, slog.New(slog.NewTextHandler(io.Discard, nil)))
	kept, removed := f.account(t), f.account(t)
	f.set(t, kept, rules.Mode1v1, 1400, 50, rank(t, "Gold II", 50))
	f.set(t, removed, rules.Mode1v1, 1500, 50, rank(t, "Gold I", 50))
	f.set(t, removed, rules.Mode2v2, 1500, 50, rank(t, "Gold I", 50))
	if _, err := f.pool.Exec(context.Background(), `update ratings set season_id = 'season.test' where account_id = $1 and mode = '2v2'`, removed); err != nil {
		t.Fatal(err)
	}
	f.play(t, "66666666-6666-6666-6666-666666666666", removed, kept, now, false)

	removal, err := f.removeFromBoard("season.test", removed, now)
	if err != nil || removal.Season != "season.test" || removal.Mode != "1v1" || removal.Account != removed || !removal.RemovedAt.Equal(now) {
		t.Fatalf("removal %+v %v", removal, err)
	}
	if got := f.placed(t, "season.test"); slices.Contains(got, removed) || !slices.Contains(got, kept) {
		t.Fatalf("placed after the removal %v", got)
	}
	if err := f.store.DropFromBoards(context.Background(), removal); err != nil {
		t.Fatal(err)
	}
	if len(boards.removed) != 1 || boards.removed[0] != "season.test 1v1 "+removed {
		t.Fatalf("removed %v", boards.removed)
	}

	boards.updates = nil
	f.play(t, "77777777-7777-7777-7777-777777777777", removed, kept, now, false)
	for _, update := range boards.updates {
		if strings.Contains(update, removed) {
			t.Fatalf("a removed account reached the board again: %v", boards.updates)
		}
	}
	if len(boards.updates) == 0 {
		t.Fatal("the other player left the board too")
	}

	var other []string
	if err := f.store.Placed(context.Background(), "season.test", rules.Mode2v2, func(e rating.Entry) error {
		other = append(other, e.Account)
		return nil
	}); err != nil || !slices.Contains(other, removed) {
		t.Fatalf("the 2v2 board lost the account %v %v", other, err)
	}
}

func TestARemovalIsRefusedTwiceAndForAnAccountOffTheBoard(t *testing.T) {
	f := newFixture(t)
	listed, unplaced := f.account(t), f.account(t)
	f.set(t, listed, rules.Mode1v1, 1400, 50, rank(t, "Gold II", 50))
	if _, err := f.pool.Exec(context.Background(), `update ratings set season_id = 'season.test' where account_id = $1`, listed); err != nil {
		t.Fatal(err)
	}
	if _, err := f.removeFromBoard("season.test", listed, now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.removeFromBoard("season.test", listed, now); !errors.Is(err, rating.ErrRemovedFromBoard) {
		t.Fatalf("second removal %v", err)
	}
	if _, err := f.removeFromBoard("season.test", unplaced, now); !errors.Is(err, rating.ErrNotOnBoard) {
		t.Fatalf("account off the board %v", err)
	}
	if _, err := f.removeFromBoard("season.missing", listed, now); !errors.Is(err, rating.ErrNotOnBoard) {
		t.Fatalf("unknown season %v", err)
	}
}

func TestAnAccountRemovedFromAClosedBoardEarnsNoEmblemAndMovesNobodyDown(t *testing.T) {
	f := newRewardFixture(t)
	end := now.Add(60 * 24 * time.Hour)
	removed, kept := f.account(t), f.account(t)
	f.set(t, removed, rules.Mode1v1, 1500, 50, rank(t, "Gold I", 50))
	f.set(t, kept, rules.Mode1v1, 1400, 50, rank(t, "Gold II", 50))
	if _, err := f.store.CloseSeason(context.Background(), "season.test", rules.Mode1v1, end); err != nil {
		t.Fatal(err)
	}
	if _, err := f.removeFromBoard("season.test", removed, end); err != nil {
		t.Fatal(err)
	}
	if got := f.placed(t, "season.test"); len(got) != 1 || got[0] != kept {
		t.Fatalf("frozen board %v", got)
	}
	if _, err := f.store.DistributeRewards(context.Background(), "season.test", rules.Mode1v1, end); err != nil {
		t.Fatal(err)
	}
	if emblems, _ := f.store.Emblems(context.Background(), removed); len(emblems) != 0 {
		t.Fatalf("a removed account earned %s", emblemNames(emblems))
	}
	emblems, _ := f.store.Emblems(context.Background(), kept)
	if emblemNames(emblems) != "season.test/1v1/global:1" {
		t.Fatalf("kept emblems %s", emblemNames(emblems))
	}
}
