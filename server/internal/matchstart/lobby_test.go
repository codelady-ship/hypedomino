package matchstart_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/custom"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstart"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestALobbyMemberCarriesTheEquippedEmojiAndTheMute(t *testing.T) {
	failure := errors.New("store down")
	for _, tc := range []struct {
		name      string
		muted     bool
		freezeErr error
		mutedErr  error
		err       error
	}{
		{name: "speaking"},
		{name: "muted", muted: true},
		{name: "loadout fails", freezeErr: failure, err: failure},
		{name: "mute check fails", mutedErr: failure, err: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			freeze := func(ctx context.Context, account string) (profile.Loadout, error) {
				return profile.Loadout{Avatar: "avatar.fox", Emojis: []string{"emoji.laugh"}}, tc.freezeErr
			}
			muted := func(ctx context.Context, account string) (bool, error) { return tc.muted, tc.mutedErr }
			member, err := matchstart.LobbyMember(freeze, muted)(context.Background(), "ana")
			if !errors.Is(err, tc.err) {
				t.Fatalf("error %v, want %v", err, tc.err)
			}
			if err != nil {
				return
			}
			want := custom.Member{Account: "ana", Emoji: []string{"emoji.laugh"}, Muted: tc.muted}
			if member.Account != want.Account || !slices.Equal(member.Emoji, want.Emoji) || member.Muted != want.Muted {
				t.Fatalf("member %+v, want %+v", member, want)
			}
		})
	}
}

type seed uint64

func (s seed) Uint64() uint64 { return uint64(s) }

func TestTheLobbyOpenerBuildsTheRosterFromTheSeatsAndUsesTheCasualBotLevel(t *testing.T) {
	f := &fakes{}
	var asked matchmaking.Match
	roster := func(ctx context.Context, match matchmaking.Match) ([]room.Player, error) {
		asked = match
		return []room.Player{{Seat: rules.SeatX, Nickname: "Ana"}, {Seat: rules.SeatY, Nickname: "Ben"}}, nil
	}
	config := matchmaking.DefaultConfig()
	config.CasualBotLevel = bot.Easy
	open := matchstart.LobbyOpener(f, roster, func() matchmaking.Config { return config }, clock{}, seed(42))
	seats := []custom.SeatState{{Seat: rules.SeatX, Account: "ana"}, {Seat: rules.SeatY, Account: "ben"}}
	if _, err := open(room.Options{Kind: room.KindCustom, Mode: rules.Mode1v1}, seats); err != nil {
		t.Fatal(err)
	}
	if asked.Mode != rules.Mode1v1 || len(asked.Seats) != 2 || asked.Seats[1] != (matchmaking.Seat{Seat: rules.SeatY, Account: "ben"}) {
		t.Fatalf("roster asked for %+v", asked)
	}
	options := f.options
	if options.Kind != room.KindCustom || options.Seed != 42 || options.BotLevel != bot.Easy || len(options.Roster) != 2 || options.Clock == nil {
		t.Fatalf("opened with %+v", options)
	}
}
