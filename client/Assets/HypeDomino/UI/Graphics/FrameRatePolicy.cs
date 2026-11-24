using System;
using HypeDomino.UI.Tokens;

namespace HypeDomino.UI.Graphics
{
    public sealed class FrameRatePolicy
    {
        readonly GraphicsQuality quality;
        readonly int[] refreshRatesHz;

        public FrameRatePolicy(GraphicsQuality quality, int[] refreshRatesHz, int? savedChoice, bool savedBatterySaver)
        {
            this.quality = quality;
            this.refreshRatesHz = refreshRatesHz;
            Choice = savedChoice;
            BatterySaver = savedBatterySaver;
            quality.Changed += OnQualityChanged;
        }

        public event Action Changed;

        public int? Choice { get; private set; }

        public bool BatterySaver { get; private set; }

        public bool SystemSaver { get; private set; }

        public bool InMatch { get; private set; }

        public int Highest => HighestOfferedUpTo(int.MaxValue);

        public int Chosen => HighestOfferedUpTo(Choice ?? TokenGraphics.AutoMaxFrameRate);

        public int Target
        {
            get
            {
                if (BatterySaver || SystemSaver) return TokenGraphics.BatterySaverFrameRate;
                return InMatch ? Chosen : Math.Min(Chosen, TokenGraphics.MenuMaxFrameRate);
            }
        }

        public bool IsOffered(int rate) =>
            Array.IndexOf(TokenGraphics.FrameRateOptions, rate) >= 0 && rate <= LevelCap && FitsDisplay(rate);

        public bool Choose(int? rate)
        {
            if (rate.HasValue && !IsOffered(rate.Value)) return false;
            if (rate == Choice) return true;
            Choice = rate;
            Changed?.Invoke();
            return true;
        }

        public void SetBatterySaver(bool on)
        {
            if (on == BatterySaver) return;
            BatterySaver = on;
            Changed?.Invoke();
        }

        public void SetSystemSaver(bool on)
        {
            if (on == SystemSaver) return;
            SystemSaver = on;
            Changed?.Invoke();
        }

        public void SetInMatch(bool inMatch)
        {
            if (inMatch == InMatch) return;
            InMatch = inMatch;
            Changed?.Invoke();
        }

        int LevelCap => TokenGraphics.TargetFpsOfEffectLevel[(int)quality.Effects];

        int HighestOfferedUpTo(int limit)
        {
            var options = TokenGraphics.FrameRateOptions;
            for (var at = options.Length - 1; at >= 0; at--)
                if (options[at] <= limit && IsOffered(options[at])) return options[at];
            return options[0];
        }

        bool FitsDisplay(int rate)
        {
            if (refreshRatesHz.Length == 0) return true;
            foreach (var refreshRate in refreshRatesHz)
                if (refreshRate % rate == 0) return true;
            return false;
        }

        void OnQualityChanged() => Changed?.Invoke();
    }
}
