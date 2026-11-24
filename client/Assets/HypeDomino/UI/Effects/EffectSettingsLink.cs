using System;
using HypeDomino.UI.Graphics;
using HypeDomino.UI.Motion;

namespace HypeDomino.UI.Effects
{
    public interface IEffectSettingsTarget
    {
        void Configure(EffectLevel level, bool reduceMotion);
    }

    public sealed class EffectSettingsLink : IDisposable
    {
        readonly IEffectSettingsTarget[] targets;
        readonly GraphicsQuality graphics;
        readonly MotionPreference motion;
        readonly Action apply;

        public EffectSettingsLink(GraphicsQuality graphics, MotionPreference motion, params IEffectSettingsTarget[] targets)
        {
            this.graphics = graphics ?? throw new ArgumentNullException(nameof(graphics));
            this.motion = motion ?? throw new ArgumentNullException(nameof(motion));
            this.targets = targets ?? throw new ArgumentNullException(nameof(targets));
            foreach (var target in targets)
                if (target == null) throw new ArgumentException("an effect settings target is null", nameof(targets));
            apply = Apply;
            graphics.Changed += apply;
            motion.Changed += apply;
            Apply();
        }

        public void Dispose()
        {
            graphics.Changed -= apply;
            motion.Changed -= apply;
        }

        void Apply()
        {
            var level = graphics.Effects;
            var reduceMotion = motion.ReduceMotion;
            foreach (var target in targets) target.Configure(level, reduceMotion);
        }
    }
}
