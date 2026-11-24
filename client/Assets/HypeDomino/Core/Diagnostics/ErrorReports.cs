using UnityEngine;

namespace HypeDomino.Core.Diagnostics
{
    public sealed class ErrorReports
    {
        readonly IErrorSender sender;
        volatile bool consented;

        public ErrorReports(IErrorSender sender)
        {
            this.sender = sender;
        }

        public bool Consented => consented;

        public void Consent() => consented = true;

        public void Withdraw() => consented = false;

        public void OnLog(string message, string stackTrace, LogType type)
        {
            if (!consented || !IsReported(type)) return;
            sender.Send(new ErrorReport(type, message, stackTrace));
        }

        static bool IsReported(LogType type) => type == LogType.Exception || type == LogType.Error || type == LogType.Assert;
    }
}
