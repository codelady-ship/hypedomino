using HypeDomino.Net.Integrity;
using Hypedomino.V1;

namespace HypeDomino.Game.Lobby
{
    public readonly struct PlayMode
    {
        public PlayMode(QueueKind kind, Mode mode)
        {
            Kind = kind;
            Mode = mode;
        }

        public QueueKind Kind { get; }
        public Mode Mode { get; }
        public IntegrityPurpose Purpose => IntegrityPurposes.ForQueue(Kind);

        public bool Is(Mode mode, QueueKind kind) => Mode == mode && Kind == kind;
    }

    public static class PlayModes
    {
        static readonly PlayMode[] offered =
        {
            new(QueueKind.Ranked, Mode._1V1),
            new(QueueKind.Ranked, Mode._2V2),
            new(QueueKind.Casual, Mode._1V1),
            new(QueueKind.Casual, Mode._2V2),
        };

        public static int Count => offered.Length;

        public static PlayMode At(int card) => offered[card];

        public static bool IsCard(int card) => card >= 0 && card < offered.Length;
    }
}
