namespace HypeDomino.Game.Match
{
    public static class MatchResultKeys
    {
        public const string Won = "match.result.won";
        public const string Lost = "match.result.lost";
        public const string Over = "match.result.over";
        public const string PlayAgain = "match.result.play_again";
        public const string BackToLobby = "match.result.back_to_lobby";

        public static string Title(MatchOutcome outcome) => outcome switch
        {
            MatchOutcome.Won => Won,
            MatchOutcome.Lost => Lost,
            _ => Over,
        };
    }
}
