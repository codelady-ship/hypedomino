using Hypedomino.V1;

namespace HypeDomino.Game.Lobby
{
    public static class BanReasonKeys
    {
        public const string Cheating = "ban.reason.cheating";
        public const string Collusion = "ban.reason.collusion";
        public const string AfkLeaving = "ban.reason.afk_leaving";
        public const string OffensiveName = "ban.reason.offensive_name";
        public const string PaymentFraud = "ban.reason.payment_fraud";
        public const string Other = "ban.reason.other";

        public static string Of(BanReason reason) => reason switch
        {
            BanReason.Unspecified => null,
            BanReason.Cheating => Cheating,
            BanReason.Collusion => Collusion,
            BanReason.AfkLeaving => AfkLeaving,
            BanReason.OffensiveName => OffensiveName,
            BanReason.PaymentFraud => PaymentFraud,
            _ => Other,
        };

        public static string Of(string category) => category switch
        {
            "cheating" => Cheating,
            "collusion" => Collusion,
            "afk_leaving" => AfkLeaving,
            "offensive_name" => OffensiveName,
            "payment_fraud" => PaymentFraud,
            _ => Other,
        };
    }
}
