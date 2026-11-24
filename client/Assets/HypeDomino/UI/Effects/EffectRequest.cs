using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.UI.Effects
{
    public readonly struct EffectRequest
    {
        const float Half = 0.5f;

        public readonly TokenEffectKind Kind;
        public readonly Vector3 At;
        public readonly Vector2 Reach;
        public readonly float StartMs;
        public readonly Color32 Team;
        public readonly int Step;
        public readonly bool HasTrail;
        public readonly Vector3 Trail;
        public readonly bool TrailArrives;
        public readonly float TravelMs;

        EffectRequest(TokenEffectKind kind, Vector3 at, Vector2 reach, float startMs, Color32 team, int step, bool hasTrail, Vector3 trail, bool trailArrives, float travelMs)
        {
            Kind = kind;
            At = at;
            Reach = reach;
            StartMs = startMs;
            Team = team;
            Step = step;
            HasTrail = hasTrail;
            Trail = trail;
            TrailArrives = trailArrives;
            TravelMs = travelMs;
        }

        public static EffectRequest On(TokenEffectKind kind, Vector3 at, Vector2 reach, float startMs) =>
            new(kind, at, reach, startMs, default, 0, false, default, false, TokenDurationMs.Base);

        public static EffectRequest On(TokenEffectKind kind, RectTransform area, float startMs) => On(kind, area.position, ReachOf(area), startMs);

        public static Vector2 ReachOf(RectTransform area)
        {
            var scale = area.lossyScale;
            var size = area.rect.size;
            return new Vector2(size.x * scale.x, size.y * scale.y) * Half;
        }

        public EffectRequest ForTeam(Color32 team) => new(Kind, At, Reach, StartMs, team, Step, HasTrail, Trail, TrailArrives, TravelMs);

        public EffectRequest AtStep(int step) => new(Kind, At, Reach, StartMs, Team, step, HasTrail, Trail, TrailArrives, TravelMs);

        public EffectRequest TrailFrom(Vector3 from, float travelMs) => new(Kind, At, Reach, StartMs, Team, Step, true, from, true, travelMs);

        public EffectRequest TrailTo(Vector3 to) => new(Kind, At, Reach, StartMs, Team, Step, true, to, false, TravelMs);
    }
}
