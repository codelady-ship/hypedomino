namespace HypeDomino.Content
{
    public interface ICacheStore
    {
        long OccupiedBytes { get; }

        bool ClearAtStart { get; set; }

        void ApplyLimits(long limitBytes, int unusedForSeconds);

        bool TryClear();
    }
}
