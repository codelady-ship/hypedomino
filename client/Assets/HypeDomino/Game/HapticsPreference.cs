using System;

namespace HypeDomino.Game
{
    public sealed class HapticsPreference
    {
        public HapticsPreference(bool enabled)
        {
            Enabled = enabled;
        }

        public event Action Changed;

        public bool Enabled { get; private set; }

        public void Set(bool on)
        {
            if (Enabled == on) return;
            Enabled = on;
            Changed?.Invoke();
        }
    }
}
