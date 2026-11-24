using System;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;

namespace HypeDomino.UI.Graphics
{
    public enum ProbeResult
    {
        NoData,
        Passed,
        Dropped,
        SlowAtLowest
    }

    public enum ThermalStatus
    {
        Normal,
        Moderate,
        Severe
    }

    public sealed class GraphicsQuality
    {
        ThermalStatus coolingTo;
        float coolingSince;

        public GraphicsQuality(DeviceTier tier, GraphicsLevel? savedChoice, int probeDrops, bool probeSettled)
        {
            Tier = tier;
            ProbeDrops = Math.Max(0, probeDrops);
            ProbeSettled = probeSettled;
            Choice = savedChoice > Highest ? Highest : savedChoice;
        }

        public event Action Changed;

        public DeviceTier Tier { get; }

        public GraphicsLevel? Choice { get; private set; }

        public int ProbeDrops { get; private set; }

        public bool ProbeSettled { get; private set; }

        public ThermalStatus Heat { get; private set; }

        public bool ProbeWanted => !Choice.HasValue && !ProbeSettled && Heat == ThermalStatus.Normal;

        public GraphicsLevel Auto => (GraphicsLevel)Math.Max((int)GraphicsLevel.Low, TokenGraphics.AutoLevelOfTier[(int)Tier] - ProbeDrops);

        public GraphicsLevel Highest => ProbeDrops > 0 ? Auto : (GraphicsLevel)TokenGraphics.HighestLevelOfTier[(int)Tier];

        public GraphicsLevel Level => Cooled(Choice ?? Auto);

        public EffectLevel Effects => (EffectLevel)TokenGraphics.EffectLevelOfLevel[(int)Level];

        public float WorkTimeGateMs
        {
            get
            {
                switch (Effects)
                {
                    case EffectLevel.High: return TokenEffectLevel.HighWorkTimeP95Ms;
                    case EffectLevel.Medium: return TokenEffectLevel.MediumWorkTimeP95Ms;
                    default: return TokenEffectLevel.LowWorkTimeP95Ms;
                }
            }
        }

        public bool CanChoose(GraphicsLevel level) => level <= Highest;

        public bool Choose(GraphicsLevel? level)
        {
            if (level.HasValue && !CanChoose(level.Value)) return false;
            if (level == Choice) return true;
            Choice = level;
            Changed?.Invoke();
            return true;
        }

        public ProbeResult SettleProbe(double workTimeP95Ms)
        {
            if (workTimeP95Ms <= 0 || Heat != ThermalStatus.Normal) return ProbeResult.NoData;
            if (workTimeP95Ms <= WorkTimeGateMs)
            {
                ProbeSettled = true;
                return ProbeResult.Passed;
            }
            if (Auto == GraphicsLevel.Low)
            {
                ProbeSettled = true;
                return ProbeResult.SlowAtLowest;
            }
            ProbeDrops++;
            Changed?.Invoke();
            return ProbeResult.Dropped;
        }

        public void UpdateHeat(ThermalStatus status, float nowSeconds)
        {
            if (status >= Heat)
            {
                coolingTo = status;
                if (status == Heat) return;
                Heat = status;
                Changed?.Invoke();
                return;
            }
            if (status != coolingTo)
            {
                coolingTo = status;
                coolingSince = nowSeconds;
                return;
            }
            if (nowSeconds - coolingSince < TokenGraphics.ThermalRecoverAfterSeconds) return;
            Heat = status;
            Changed?.Invoke();
        }

        GraphicsLevel Cooled(GraphicsLevel level)
        {
            switch (Heat)
            {
                case ThermalStatus.Severe: return GraphicsLevel.Low;
                case ThermalStatus.Moderate: return level == GraphicsLevel.Low ? level : level - 1;
                default: return level;
            }
        }

        public float RenderScale(int nativeShortSide)
        {
            if (nativeShortSide <= 0) return 1f;
            return Math.Min(1f, (float)TokenGraphics.MaxRenderShortSideOfLevel[(int)Level] / nativeShortSide);
        }
    }
}
