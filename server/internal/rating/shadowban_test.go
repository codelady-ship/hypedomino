package rating_test

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func (f *fixture) shadowban(t *testing.T, account string) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), `update accounts set shadowbanned_at = now() where id = $1`, account); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) placed(t *testing.T, season string) []string {
	t.Helper()
	var accounts []string
	if err := f.store.Placed(context.Background(), season, rules.Mode1v1, func(e rating.Entry) error {
		accounts = append(accounts, e.Account)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return accounts
}

func TestAShadowbannedAccountLeavesTheBoards(t *testing.T) {
	f := newFixture(t)
	boards := &fakeBoards{}
	f.store.SetBoards(boards, slog.New(slog.NewTextHandler(io.Discard, nil)))
	honest, cheater := f.account(t), f.account(t)
	f.set(t, honest, rules.Mode1v1, 1400, 50, rank(t, "Gold II", 50))
	f.set(t, cheater, rules.Mode1v1, 1500, 50, rank(t, "Gold I", 50))

	f.play(t, "66666666-6666-6666-6666-666666666666", cheater, honest, now, false)
	if got := f.placed(t, "season.test"); !slices.Contains(got, cheater) {
		t.Fatalf("placed before the shadowban %v", got)
	}
	f.shadowban(t, cheater)
	if got := f.placed(t, "season.test"); slices.Contains(got, cheater) || !slices.Contains(got, honest) {
		t.Fatalf("placed after the shadowban %v", got)
	}
	if err := f.store.HideFromBoards(context.Background(), cheater); err != nil {
		t.Fatal(err)
	}
	if len(boards.removed) != 1 || boards.removed[0] != "season.test 1v1 "+cheater {
		t.Fatalf("removed %v", boards.removed)
	}

	boards.updates = nil
	f.play(t, "77777777-7777-7777-7777-777777777777", cheater, honest, now, false)
	for _, update := range boards.updates {
		if strings.Contains(update, cheater) {
			t.Fatalf("a shadowbanned account reached the boards: %v", boards.updates)
		}
	}
	if len(boards.updates) == 0 {
		t.Fatal("the honest player left the boards too")
	}
}

func TestAShadowbannedAccountQueuesIntoTheCheaterPool(t *testing.T) {
	f := newFixture(t)
	honest, cheater, fresh := f.account(t), f.account(t), f.account(t)
	f.set(t, honest, rules.Mode1v1, 1400, 50, rank(t, "Gold II", 50))
	f.set(t, cheater, rules.Mode1v1, 1400, 50, rank(t, "Gold II", 50))
	f.shadowban(t, cheater)
	f.shadowban(t, fresh)
	for account, want := range map[string]bool{honest: false, cheater: true, fresh: true} {
		q, err := f.store.Queue(context.Background(), account, rules.Mode1v1, now)
		if err != nil || q.Shadowbanned != want {
			t.Fatalf("queue entry of %s %+v %v, want shadowbanned %v", account, q, err, want)
		}
	}
}

func TestAShadowbannedAccountEarnsNoEmblemAndMovesNobodyDown(t *testing.T) {
	f := newRewardFixture(t)
	end := now.Add(60 * 24 * time.Hour)
	cheater, honest := f.account(t), f.account(t)
	f.set(t, cheater, rules.Mode1v1, 1500, 50, rank(t, "Gold I", 50))
	f.set(t, honest, rules.Mode1v1, 1400, 50, rank(t, "Gold II", 50))
	f.shadowban(t, cheater)
	if _, err := f.store.CloseSeason(context.Background(), "season.test", rules.Mode1v1, end); err != nil {
		t.Fatal(err)
	}
	if got := f.placed(t, "season.test"); len(got) != 1 || got[0] != honest {
		t.Fatalf("frozen board %v", got)
	}
	if _, err := f.store.DistributeRewards(context.Background(), "season.test", rules.Mode1v1, end); err != nil {
		t.Fatal(err)
	}
	if emblems, _ := f.store.Emblems(context.Background(), cheater); len(emblems) != 0 {
		t.Fatalf("a shadowbanned account earned %s", emblemNames(emblems))
	}
	emblems, _ := f.store.Emblems(context.Background(), honest)
	if emblemNames(emblems) != "season.test/1v1/global:1" {
		t.Fatalf("honest emblems %s", emblemNames(emblems))
	}
}
