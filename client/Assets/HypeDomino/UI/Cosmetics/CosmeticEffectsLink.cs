using System;
using HypeDomino.UI.Graphics;
using HypeDomino.UI.Motion;

namespace HypeDomino.UI.Cosmetics
{
    public sealed class CosmeticEffectsLink : IDisposable
    {
        readonly CosmeticStage stage;
        readonly GraphicsQuality graphics;
        readonly MotionPreference motion;
        readonly Action refresh;
        readonly Action apply;

        public CosmeticEffectsLink(CosmeticStage stage, GraphicsQuality graphics, MotionPreference motion, Action refresh)
        {
            this.stage = stage ?? throw new ArgumentNullException(nameof(stage));
            this.graphics = graphics ?? throw new ArgumentNullException(nameof(graphics));
            this.motion = motion ?? throw new ArgumentNullException(nameof(motion));
            this.refresh = refresh ?? throw new ArgumentNullException(nameof(refresh));
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
            if (level == stage.Level && reduceMotion == stage.ReduceMotion) return;
            stage.SetEffects(level, reduceMotion);
            refresh();
        }
    }
}
