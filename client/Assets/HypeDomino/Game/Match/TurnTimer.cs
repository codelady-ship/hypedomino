using System;
using HypeDomino.Net;
using Hypedomino.V1;
using UnityEngine;

namespace HypeDomino.Game.Match
{
    public sealed class TurnTimer
    {
        ulong deadlineUnixMs;
        uint turnSeconds;
        uint warningSeconds;

        public bool Running { get; private set; }
        public float Remaining { get; private set; }
        public float Fill { get; private set; }
        public bool IsWarning { get; private set; }

        public void Read(MatchSnapshot view)
        {
            var turnRuns = view.State == MatchState.Opening || view.State == MatchState.Playing;
            deadlineUnixMs = turnRuns ? view.TurnDeadlineUnixMs : 0;
            turnSeconds = view.TurnSeconds;
            warningSeconds = view.TurnWarningSeconds;
        }

        public void Tick(ServerClock clock, double nowSeconds)
        {
            Running = deadlineUnixMs != 0 && turnSeconds != 0 && clock.Synced;
            if (!Running)
            {
                Remaining = 0f;
                Fill = 0f;
                IsWarning = false;
                return;
            }
            var left = Math.Max(0d, clock.SecondsUntil(deadlineUnixMs, nowSeconds));
            Remaining = (float)left;
            Fill = Mathf.Clamp01((float)(left / turnSeconds));
            IsWarning = left <= warningSeconds;
        }
    }
}
