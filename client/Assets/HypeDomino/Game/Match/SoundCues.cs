using System;
using HypeDomino.Net;
using HypeDomino.UI.Audio;
using Hypedomino.V1;

namespace HypeDomino.Game.Match
{
    [Flags]
    public enum MatchSounds
    {
        None = 0,
        TilePlace = 1,
        TileDraw = 2,
        Pass = 4,
        Score = 8,
        HandEnd = 16,
        MatchWin = 32,
        MatchLose = 64,
        Combo1 = 128,
        Combo2 = 256,
        Combo3 = 512,
        Combo4 = 1024,
        Combo5 = 2048,
        TurnWarning = 4096,
    }

    public sealed class SoundCues
    {
        public const int ComboSteps = 5;

        bool seen;
        bool warned;
        MatchState state;
        Seat current;
        uint hand;
        int boardTiles;
        uint boneyard;
        uint placements;
        uint teamScore;

        public void StartConnection() => seen = false;

        public MatchSounds Read(MatchSnapshot view)
        {
            var tiles = view.Line.Length + view.Top.Length + view.Bottom.Length;
            var seated = view.Seat != Seat.Unspecified;
            var team = seated ? SeatOrder.TeamOf(view.Seat) : Team.Unspecified;
            var score = team == Team.A ? view.Scores.TeamA : team == Team.B ? view.Scores.TeamB : 0;
            var sounds = seen ? SoundsOf(view, team, tiles, score) : MatchSounds.None;
            seen = true;
            state = view.State;
            current = view.Current;
            hand = view.HandsPlayed;
            boardTiles = tiles;
            boneyard = view.BoneyardSize;
            placements = view.TurnPlacements;
            teamScore = score;
            return sounds;
        }

        public bool ReadTurnWarning(bool localTurnInWarning)
        {
            var starts = localTurnInWarning && !warned;
            warned = localTurnInWarning;
            return starts;
        }

        public static void Play(MatchSounds sounds, SoundPlayer player)
        {
            if (sounds == MatchSounds.None || player == null) return;
            PlayIf(sounds, MatchSounds.MatchWin, Sound.MatchWin, player);
            PlayIf(sounds, MatchSounds.MatchLose, Sound.MatchLose, player);
            PlayIf(sounds, MatchSounds.HandEnd, Sound.HandEnd, player);
            PlayIf(sounds, MatchSounds.TilePlace, Sound.TilePlace, player);
            PlayIf(sounds, MatchSounds.TileDraw, Sound.TileDraw, player);
            PlayIf(sounds, MatchSounds.Pass, Sound.Pass, player);
            PlayIf(sounds, MatchSounds.Score, Sound.Score, player);
            PlayIf(sounds, MatchSounds.TurnWarning, Sound.TurnWarning, player);
            for (var step = 0; step < ComboSteps; step++)
                PlayIf(sounds, (MatchSounds)((int)MatchSounds.Combo1 << step), Sound.Combo1 + step, player);
        }

        public static MatchSounds ScoreSoundOf(int step) =>
            step <= 0 ? MatchSounds.Score : (MatchSounds)((int)MatchSounds.Combo1 << (Math.Min(step, ComboSteps) - 1));

        static void PlayIf(MatchSounds sounds, MatchSounds cue, Sound sound, SoundPlayer player)
        {
            if ((sounds & cue) != 0) player.Play(sound);
        }

        MatchSounds SoundsOf(MatchSnapshot view, Team team, int tiles, uint score)
        {
            if (team != Team.Unspecified && view.State == MatchState.Finished && state != MatchState.Finished && view.Winner != Team.Unspecified)
                return view.Winner == team ? MatchSounds.MatchWin : MatchSounds.MatchLose;
            if (view.HandsPlayed != hand && view.HasLastResult && view.LeftSeats.Length > 0) return MatchSounds.HandEnd;
            var sounds = MatchSounds.None;
            var sameHand = view.HandsPlayed == hand;
            if (sameHand && tiles > boardTiles) sounds |= MatchSounds.TilePlace;
            if (sameHand && view.State == MatchState.Playing && view.BoneyardSize < boneyard) sounds |= MatchSounds.TileDraw;
            if (sameHand && IsPass(view, tiles)) sounds |= MatchSounds.Pass;
            if (team != Team.Unspecified && score > teamScore)
                sounds |= ScoreSoundOf(EffectCues.StepOf(score, teamScore, view.HandsPlayed != hand && view.HasLastResult, (int)view.TurnPlacements));
            return sounds;
        }

        bool IsPass(MatchSnapshot view, int tiles) =>
            view.State == MatchState.Playing && state == MatchState.Playing && tiles == boardTiles && view.BoneyardSize == boneyard &&
            placements == 0 && view.Current != current && current != Seat.Unspecified;
    }
}
