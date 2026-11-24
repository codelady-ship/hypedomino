using System;
using UnityEngine;

namespace HypeDomino.UI.Audio
{
    public enum AudioGroup
    {
        Music,
        Sfx,
        Ui,
    }

    public sealed class AudioVolumes
    {
        public const float Full = 1f;

        readonly float[] levels = new float[3];

        public AudioVolumes(float music, float sfx, float ui)
        {
            levels[(int)AudioGroup.Music] = Clamped(music);
            levels[(int)AudioGroup.Sfx] = Clamped(sfx);
            levels[(int)AudioGroup.Ui] = Clamped(ui);
        }

        public event Action Changed;

        public float Level(AudioGroup group) => levels[(int)group];

        public bool Audible(AudioGroup group) => levels[(int)group] > 0f;

        public bool Silent => !Audible(AudioGroup.Music) && !Audible(AudioGroup.Sfx) && !Audible(AudioGroup.Ui);

        public void Set(AudioGroup group, float level)
        {
            level = Clamped(level);
            if (levels[(int)group] == level) return;
            levels[(int)group] = level;
            Changed?.Invoke();
        }

        static float Clamped(float level) => level > 0f ? Mathf.Min(level, Full) : 0f;
    }
}
