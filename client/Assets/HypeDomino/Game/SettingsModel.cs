using System;
using HypeDomino.Content;
using HypeDomino.UI.Audio;
using HypeDomino.UI.Graphics;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Motion;

namespace HypeDomino.Game
{
    public sealed class SettingsModel : IDisposable
    {
        readonly ContentCache cache;

        public SettingsModel(LanguageChoice language, AudioVolumes volumes, MotionPreference motion, GraphicsQuality graphics,
            FrameRatePolicy frameRate, HapticsPreference haptics, ContentCache cache)
        {
            Language = language;
            Volumes = volumes;
            Motion = motion;
            Graphics = graphics;
            FrameRate = frameRate;
            Haptics = haptics;
            this.cache = cache;
            Language.Changed += RaiseChanged;
            Volumes.Changed += RaiseChanged;
            Motion.Changed += RaiseChanged;
            FrameRate.Changed += RaiseChanged;
            Haptics.Changed += RaiseChanged;
        }

        public event Action Changed;

        public LanguageChoice Language { get; }

        public AudioVolumes Volumes { get; }

        public MotionPreference Motion { get; }

        public GraphicsQuality Graphics { get; }

        public FrameRatePolicy FrameRate { get; }

        public HapticsPreference Haptics { get; }

        public long CacheBytes => cache.OccupiedBytes;

        public CacheClearResult ClearCache()
        {
            var result = cache.Clear();
            if (result == CacheClearResult.Cleared) RaiseChanged();
            return result;
        }

        public void Dispose()
        {
            Language.Changed -= RaiseChanged;
            Volumes.Changed -= RaiseChanged;
            Motion.Changed -= RaiseChanged;
            FrameRate.Changed -= RaiseChanged;
            Haptics.Changed -= RaiseChanged;
        }

        void RaiseChanged() => Changed?.Invoke();
    }
}
