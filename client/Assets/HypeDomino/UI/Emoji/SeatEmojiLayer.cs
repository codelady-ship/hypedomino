using HypeDomino.Core.Pooling;
using HypeDomino.UI.Effects;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using Unity.Profiling;
using UnityEngine;

namespace HypeDomino.UI.Emoji
{
    public sealed class SeatEmojiLayer : MonoBehaviour, IEffectSettingsTarget
    {
        public const string ProfilerMarkerName = "Effect.emoji";

        const float MillisecondsPerSecond = 1000f;

        static readonly ProfilerMarker TickMarker = new(ProfilerMarkerName);

        PrefabPool<EmojiPopup> pool;
        RectTransform[] seats;
        EmojiPopup[] live;
        EffectPlayer effects;
        EffectLevel level = EffectLevel.Medium;
        bool reduceMotion;

        public int LiveCount => pool.CountActive;
        public int PooledCount => pool.CountInactive;

        public EmojiPopup LiveAt(int seat) => live[seat];

        public void Build(EmojiPopup prefab, RectTransform[] seatAnchors)
        {
            seats = seatAnchors;
            live = new EmojiPopup[seatAnchors.Length];
            pool = new PrefabPool<EmojiPopup>(prefab, transform, TokenEffect.EmojiPooled, TokenEffect.EmojiPooled);
        }

        public void Configure(EffectLevel effectLevel, bool reduce)
        {
            level = effectLevel;
            reduceMotion = reduce;
        }

        public void Attach(EffectPlayer player) => effects = player;

        public void Show(int seat, LoopArt art, float now)
        {
            if (effects != null) effects.Play(EffectRequest.On(TokenEffectKind.Emoji, seats[seat], now * MillisecondsPerSecond));
            var popup = live[seat];
            if (popup == null)
            {
                popup = pool.Get();
                live[seat] = popup;
            }
            popup.transform.position = seats[seat].position;
            popup.Begin(art, level, reduceMotion, now);
        }

        public void Tick(float now)
        {
            using (TickMarker.Auto())
            {
                for (var seat = 0; seat < live.Length; seat++)
                {
                    var popup = live[seat];
                    if (popup == null || popup.Advance(now)) continue;
                    pool.Release(popup);
                    live[seat] = null;
                }
            }
        }

        void Update()
        {
            if (live != null) Tick(Time.unscaledTime);
        }

        void OnDestroy() => pool?.Clear();
    }
}
