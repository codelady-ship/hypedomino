using HypeDomino.UI.Motion;
using HypeDomino.UI.Rarity;
using UnityEngine;

namespace HypeDomino.UI.Cosmetics
{
    public readonly struct CosmeticArt
    {
        public CosmeticArt(string staticAddress, string animationAddress)
        {
            StaticAddress = staticAddress;
            AnimationAddress = animationAddress;
        }

        public string StaticAddress { get; }

        public string AnimationAddress { get; }

        public bool IsAnimated => !string.IsNullOrEmpty(AnimationAddress);

        public bool IsSameAs(CosmeticArt other) => StaticAddress == other.StaticAddress && AnimationAddress == other.AnimationAddress;
    }

    public readonly struct AvatarLook
    {
        public AvatarLook(CosmeticArt avatar, CosmeticArt frame, string markRarityId)
        {
            Avatar = avatar;
            Frame = frame;
            MarkRarityId = markRarityId;
        }

        public CosmeticArt Avatar { get; }

        public CosmeticArt Frame { get; }

        public string MarkRarityId { get; }

        public bool IsSameAs(AvatarLook other) => Avatar.IsSameAs(other.Avatar) && Frame.IsSameAs(other.Frame) && MarkRarityId == other.MarkRarityId;

        public static AvatarLook Of(CosmeticManifest manifest, string avatarId, string frameId)
        {
            var avatar = manifest.ShownOf(CosmeticAddresses.Avatar, avatarId);
            var frame = manifest.ShownOf(CosmeticAddresses.Frame, frameId);
            return new AvatarLook(avatar?.Art ?? default(CosmeticArt), frame?.Art ?? default(CosmeticArt), frame?.RarityId);
        }
    }

    public sealed class CosmeticStage
    {
        public CosmeticStage(ICosmeticArtSource<Sprite> stills, ICosmeticArtSource<LoopArt> loops, RarityTable rarities, Sprite builtInAvatar, Sprite builtInFrame, EffectLevel level, bool reduceMotion)
        {
            Stills = new CosmeticArtCache<Sprite>(stills);
            Loops = new CosmeticArtCache<LoopArt>(loops);
            Rarities = rarities;
            BuiltInAvatar = builtInAvatar;
            BuiltInFrame = builtInFrame;
            SetEffects(level, reduceMotion);
        }

        public CosmeticArtCache<Sprite> Stills { get; }

        public CosmeticArtCache<LoopArt> Loops { get; }

        public RarityTable Rarities { get; }

        public Sprite BuiltInAvatar { get; }

        public Sprite BuiltInFrame { get; }

        public EffectLevel Level { get; private set; }

        public bool ReduceMotion { get; private set; }

        public AnimationBudget Budget { get; private set; }

        public bool AllowsAnimation => Level != EffectLevel.Low && !ReduceMotion;

        public void SetEffects(EffectLevel level, bool reduceMotion)
        {
            Level = level;
            ReduceMotion = reduceMotion;
            Budget = AnimationBudget.For(level);
        }
    }
}
