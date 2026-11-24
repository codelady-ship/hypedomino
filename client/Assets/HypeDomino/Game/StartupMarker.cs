using System;
using HypeDomino.Core.Diagnostics;

namespace HypeDomino.Game
{
    public sealed class StartupMarker
    {
        const double MillisecondsPerSecond = 1000d;

        readonly AppFlow flow;
        readonly Func<double> secondsSinceStartup;
        readonly Action<string> log;
        readonly StartupPhases phases;

        public StartupMarker(AppFlow flow, Func<double> secondsSinceStartup, Action<string> log, StartupPhases phases)
        {
            this.flow = flow;
            this.secondsSinceStartup = secondsSinceStartup;
            this.log = log;
            this.phases = phases;
            flow.Changed += OnFirstScreen;
        }

        void OnFirstScreen(AppState state)
        {
            flow.Changed -= OnFirstScreen;
            var name = state.ToString();
            log($"Startup state={name} ms={(long)(secondsSinceStartup() * MillisecondsPerSecond)}");
            var trace = phases.Close(name);
            if (trace != null) log(trace);
        }
    }
}
