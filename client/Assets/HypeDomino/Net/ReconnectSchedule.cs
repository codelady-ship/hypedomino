namespace HypeDomino.Net
{
    public sealed class ReconnectSchedule
    {
        public const double SteadyDelaySeconds = 5;
        public const double Jitter = 0.2;

        static readonly double[] FirstDelaysSeconds = { 0, 0.5, 1, 2, 4 };

        int attempt;

        public double NextDelaySeconds(double random01)
        {
            var delay = attempt < FirstDelaysSeconds.Length ? FirstDelaysSeconds[attempt] : SteadyDelaySeconds;
            attempt++;
            return delay * (1 + Jitter * (2 * random01 - 1));
        }

        public void Reset() => attempt = 0;
    }
}
