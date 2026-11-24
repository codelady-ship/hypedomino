using UnityEngine;

namespace HypeDomino.Core.Diagnostics
{
    public readonly struct ErrorReport
    {
        public readonly LogType Type;
        public readonly string Message;
        public readonly string StackTrace;

        public ErrorReport(LogType type, string message, string stackTrace)
        {
            Type = type;
            Message = message;
            StackTrace = stackTrace;
        }
    }
}
