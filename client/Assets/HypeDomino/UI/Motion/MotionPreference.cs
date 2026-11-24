using System;

namespace HypeDomino.UI.Motion
{
    public sealed class MotionPreference
    {
        public MotionPreference(bool? savedChoice, bool systemReduce)
        {
            Choice = savedChoice;
            SystemReduce = systemReduce;
        }

        public event Action Changed;

        public bool? Choice { get; private set; }

        public bool SystemReduce { get; private set; }

        public bool ReduceMotion => Choice ?? SystemReduce;

        public void Choose(bool? reduce)
        {
            if (Choice == reduce) return;
            Choice = reduce;
            Changed?.Invoke();
        }

        public void SetSystemReduce(bool on)
        {
            if (SystemReduce == on) return;
            SystemReduce = on;
            if (!Choice.HasValue) Changed?.Invoke();
        }
    }
}
