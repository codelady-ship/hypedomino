namespace HypeDomino.Core.Analytics
{
    public readonly struct AnalyticsEvent
    {
        public readonly string Name;
        public readonly string Label;
        public readonly long Value;
        public readonly long ClientTimeMs;

        public AnalyticsEvent(string name, string label, long value, long clientTimeMs)
        {
            Name = name;
            Label = label;
            Value = value;
            ClientTimeMs = clientTimeMs;
        }
    }
}
