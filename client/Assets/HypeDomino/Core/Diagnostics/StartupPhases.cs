using System;
using System.Text;
using UnityEngine;

namespace HypeDomino.Core.Diagnostics
{
    public sealed class StartupPhases
    {
        public const string Assemblies = "assemblies";
        public const string Scene = "scene";

        const double MillisecondsPerSecond = 1000d;
        const string Header = "Startup phases";

        readonly Func<double> secondsSinceStartup;
        readonly StringBuilder line = new StringBuilder(Header);

        public StartupPhases(Func<double> secondsSinceStartup) => this.secondsSinceStartup = secondsSinceStartup;

        public static StartupPhases App { get; } = new StartupPhases(RealtimeSinceStartup);

        public bool Closed { get; private set; }

        public void Mark(string phase)
        {
            if (Closed) return;
            line.Append(' ').Append(phase).Append('=').Append((long)(secondsSinceStartup() * MillisecondsPerSecond));
        }

        public string Close(string firstScreen)
        {
            if (Closed) return null;
            Mark(firstScreen);
            Closed = true;
            return line.ToString();
        }

        static double RealtimeSinceStartup() => Time.realtimeSinceStartupAsDouble;
    }
}
