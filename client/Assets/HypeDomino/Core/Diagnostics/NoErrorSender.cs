namespace HypeDomino.Core.Diagnostics
{
    public sealed class NoErrorSender : IErrorSender
    {
        public static readonly NoErrorSender Instance = new NoErrorSender();

        NoErrorSender()
        {
        }

        public void Send(ErrorReport report)
        {
        }
    }
}
