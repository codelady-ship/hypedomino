using HypeDomino.UI.Tokens;

namespace HypeDomino.UI.Motion
{
    public sealed class AnimationBudget
    {
        readonly int limit;

        public AnimationBudget(int limit) => this.limit = limit;

        public int Active { get; private set; }

        public bool HasRoom => Active < limit;

        public bool HasRoomFor(int count) => Active + count <= limit;

        public static AnimationBudget For(EffectLevel level) => new AnimationBudget(LimitOf(level));

        public bool TryAcquire()
        {
            if (Active >= limit) return false;
            Active++;
            return true;
        }

        public void Release()
        {
            if (Active > 0) Active--;
        }

        static int LimitOf(EffectLevel level)
        {
            switch (level)
            {
                case EffectLevel.High: return TokenEffectLevel.HighMaxAnimatedCosmetics;
                case EffectLevel.Medium: return TokenEffectLevel.MediumMaxAnimatedCosmetics;
                default: return TokenEffectLevel.LowMaxAnimatedCosmetics;
            }
        }
    }
}
