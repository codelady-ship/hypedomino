using System;
using System.Collections.Generic;

namespace HypeDomino.Core.Analytics
{
    public interface IAnalyticsSender
    {
        void Send(string analyticsId, IReadOnlyList<AnalyticsEvent> batch, Action<bool> done);
    }
}
