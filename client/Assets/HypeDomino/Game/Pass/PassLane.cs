using System;
using System.Collections.Generic;
using HypeDomino.UI.Cosmetics;

namespace HypeDomino.Game.Pass
{
    public readonly struct PassCell
    {
        readonly CosmeticItem[] cosmetics;

        public PassCell(int level, PassRewardState state, PassReward reward, CosmeticItem[] cosmetics)
        {
            Level = level;
            State = state;
            Reward = reward;
            this.cosmetics = cosmetics;
        }

        public int Level { get; }

        public PassRewardState State { get; }

        public PassReward Reward { get; }

        public bool HasReward => Reward != null;

        public bool IsLocked => State == PassRewardState.Locked;

        public int CosmeticCount => cosmetics.Length;

        public CosmeticItem CosmeticAt(int index) => cosmetics[index];
    }

    public sealed class PassLane
    {
        readonly PassCell[] free;
        readonly PassCell[] premium;

        PassLane(PassCell[] free, PassCell[] premium)
        {
            this.free = free;
            this.premium = premium;
        }

        public int Count => free.Length;

        public PassCell CellAt(PassTrack track, int row) => (track == PassTrack.Premium ? premium : free)[row];

        public static PassLane Of(PassStanding standing, PassTracks tracks, CosmeticManifest manifest)
        {
            if (standing == null) throw new ArgumentNullException(nameof(standing));
            if (tracks == null) throw new ArgumentNullException(nameof(tracks));
            if (manifest == null) throw new ArgumentNullException(nameof(manifest));
            return new PassLane(Cells(PassTrack.Free, standing, tracks, manifest), Cells(PassTrack.Premium, standing, tracks, manifest));
        }

        static PassCell[] Cells(PassTrack track, PassStanding standing, PassTracks tracks, CosmeticManifest manifest)
        {
            var cells = new PassCell[standing.Levels];
            for (var row = 0; row < cells.Length; row++)
            {
                var level = row + 1;
                tracks.TryRewardAt(track, level, out var reward);
                cells[row] = new PassCell(level, standing.StateAt(track, level), reward, Shown(reward, manifest));
            }
            return cells;
        }

        static CosmeticItem[] Shown(PassReward reward, CosmeticManifest manifest)
        {
            if (reward == null || reward.Cosmetics.Count == 0) return Array.Empty<CosmeticItem>();
            var shown = new List<CosmeticItem>(reward.Cosmetics.Count);
            foreach (var id in reward.Cosmetics)
                if (manifest.TryGet(id, out var item) && item.IsRendered) shown.Add(item);
            return shown.ToArray();
        }
    }
}
