using Hypedomino.V1;

namespace HypeDomino.Game.Match
{
    public sealed class LoadingReport
    {
        public const uint ProgressStep = 10;

        readonly LoadingIntent intent = new();
        bool sentReady;
        uint sentProgress;
        bool sentAny;

        public void Reset()
        {
            sentReady = false;
            sentProgress = 0;
            sentAny = false;
        }

        public bool TryReport(float loaded, bool ready, out LoadingIntent report)
        {
            report = null;
            if (sentReady) return false;
            var progress = ready ? LoadingCards.FullProgress : Step(loaded);
            if (sentAny && progress == sentProgress && !ready) return false;
            sentAny = true;
            sentProgress = progress;
            sentReady = ready;
            intent.Progress = progress;
            intent.Ready = ready;
            report = intent;
            return true;
        }

        static uint Step(float loaded)
        {
            if (!(loaded > 0f)) return 0;
            if (loaded >= 1f) return LoadingCards.FullProgress - ProgressStep;
            var percent = (uint)(loaded * LoadingCards.FullProgress);
            return percent / ProgressStep * ProgressStep;
        }
    }
}
