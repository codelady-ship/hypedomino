namespace HypeDomino.Core.Diagnostics
{
    public interface IErrorSender
    {
        void Send(ErrorReport report);
    }
}
