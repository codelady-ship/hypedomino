using System;
using HypeDomino.Net;
using Hypedomino.V1;

namespace HypeDomino.Game
{
    public sealed class MaintenanceScreen
    {
        readonly Action dial;
        readonly Func<double> secondsNow;
        readonly ServerClock clock = new();
        ulong autoRetriedEndsUnixMs;

        public MaintenanceScreen(Action dial, Func<double> secondsNow)
        {
            this.dial = dial;
            this.secondsNow = secondsNow;
        }

        public bool Active { get; private set; }
        public ulong EndsUnixMs { get; private set; }
        public bool HasEnd => EndsUnixMs != 0;
        public bool Checking { get; private set; }

        public event Action Changed;

        public void Read(ServerHello hello)
        {
            switch (hello.Status)
            {
                case HelloStatus.Maintenance:
                    clock.Sync(hello.ServerTimeUnixMs, secondsNow(), secondsNow());
                    Show(true, hello.MaintenanceEndsUnixMs);
                    return;
                case HelloStatus.Accepted:
                case HelloStatus.UpdateRequired:
                    autoRetriedEndsUnixMs = 0;
                    Show(false, 0);
                    return;
            }
        }

        public bool Retry()
        {
            if (!Active || Checking) return false;
            Checking = true;
            Changed?.Invoke();
            dial();
            return true;
        }

        public void Tick()
        {
            if (!Active || Checking || !HasEnd || autoRetriedEndsUnixMs == EndsUnixMs) return;
            if (clock.SecondsUntil(EndsUnixMs, secondsNow()) > 0) return;
            autoRetriedEndsUnixMs = EndsUnixMs;
            Retry();
        }

        void Show(bool active, ulong endsUnixMs)
        {
            if (Active == active && EndsUnixMs == endsUnixMs && !Checking) return;
            Active = active;
            EndsUnixMs = endsUnixMs;
            Checking = false;
            Changed?.Invoke();
        }
    }
}
