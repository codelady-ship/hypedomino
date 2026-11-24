using System;
using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.UI.Motion
{
    public readonly struct ScoreBurst
    {
        public readonly int Step;
        public readonly int ScoreSize;
        public readonly int Particles;
        public readonly int HoldMs;
        public readonly int ShakeAmplitude;
        public readonly int ShakeDurationMs;

        public ScoreBurst(int step, int scoreSize, int particles, int holdMs, int shakeAmplitude, int shakeDurationMs)
        {
            Step = step;
            ScoreSize = scoreSize;
            Particles = particles;
            HoldMs = holdMs;
            ShakeAmplitude = shakeAmplitude;
            ShakeDurationMs = shakeDurationMs;
        }

        public float LifetimeMs => Step == 0 ? 0f : ScorePopup.LifetimeMs(HoldMs);
    }

    public static class ScoreCascade
    {
        public static int MaxStep => TokenScoreCascade.StepMinPoints.Length;

        public static int StepOf(uint points)
        {
            var step = 0;
            var minPoints = TokenScoreCascade.StepMinPoints;
            for (var at = 0; at < minPoints.Length && points >= minPoints[at]; at++) step = at + 1;
            return step;
        }

        public static int ChainStepOf(uint points, int placement)
        {
            var step = StepOf(points);
            if (step == 0) return 0;
            var boosts = TokenScoreCascade.ChainBoostSteps;
            var boost = boosts[Math.Min(Math.Max(placement, 1), boosts.Length) - 1];
            return Math.Min(step + boost, MaxStep);
        }

        public static bool ShowsTurnTotal(int scoringPlacements) => scoringPlacements >= 2;

        public static float ChainStartMs(int placement) => Math.Max(placement - 1, 0) * TokenScoreCascade.ChainStepGapMs;

        public static float CarryOverStartMs => TokenScoreCascade.CarryOverGapMs;

        public static float RollMs(uint points) => Math.Min(points * TokenScoreCascade.RollMsPer5Points / 5f, TokenScoreCascade.RollMaxMs);

        public static ScoreBurst Burst(int step, EffectLevel level, bool reduceMotion)
        {
            if (step <= 0) return default;
            var at = Math.Min(step, MaxStep) - 1;
            var particleLevel = reduceMotion ? EffectLevel.Low : level;
            var particles = (TokenScoreCascade.ParticlesOfEffectLevel[(int)particleLevel] * TokenScoreCascade.StepParticlePercent[at] + 50) / 100;
            var scoreSize = RoundHalfUp(TokenTextSize.Score * TokenScoreCascade.StepTextScale[at]);
            var shakeAmplitude = reduceMotion ? 0 : TokenScoreCascade.StepShakeAmplitude[at];
            var shakeDurationMs = reduceMotion ? 0 : TokenScoreCascade.StepShakeDurationMs[at];
            return new ScoreBurst(at + 1, scoreSize, particles, TokenScoreCascade.StepHoldMs[at], shakeAmplitude, shakeDurationMs);
        }

        static int RoundHalfUp(float value) => Mathf.FloorToInt(value + 0.5f);
    }
}
