using System;
using System.Collections.Generic;
using HypeDomino.Core.Analytics;
using HypeDomino.Net.Api;
using UnityEngine;

namespace HypeDomino.Game.Analytics
{
    [Serializable]
    public sealed class AnalyticsEventBody
    {
        public string name;
        public string label;
        public long value;
        public long clientTimeMs;
    }

    [Serializable]
    public sealed class AnalyticsBatchRequest
    {
        public string analyticsId;
        public List<AnalyticsEventBody> events = new List<AnalyticsEventBody>();
    }

    public sealed class ApiAnalyticsSender : IAnalyticsSender
    {
        public const string Path = "/analytics/events";
        public const string ConsentRequired = "analytics.consent_required";
        public const string IdTaken = "analytics.id_taken";

        readonly ApiClient api;
        readonly AnalyticsBatchRequest request = new AnalyticsBatchRequest();
        readonly List<AnalyticsEventBody> bodies = new List<AnalyticsEventBody>();
        readonly Action<ApiReply> answered;
        Action<bool> pending;

        public ApiAnalyticsSender(ApiClient api)
        {
            this.api = api;
            answered = Answered;
        }

        public void Send(string analyticsId, IReadOnlyList<AnalyticsEvent> batch, Action<bool> done)
        {
            pending = done;
            api.Post(Path, Body(analyticsId, batch), answered);
        }

        public string Body(string analyticsId, IReadOnlyList<AnalyticsEvent> batch)
        {
            request.analyticsId = analyticsId;
            request.events.Clear();
            for (var i = 0; i < batch.Count; i++) request.events.Add(Fill(i, batch[i]));
            return JsonUtility.ToJson(request);
        }

        public static bool Settled(ApiReply reply) => reply.Outcome == ApiOutcome.Ok || RefusedForGood(reply);

        static bool RefusedForGood(ApiReply reply) => reply.Status switch
        {
            400 => true,
            403 => reply.Error == ConsentRequired,
            409 => reply.Error == IdTaken,
            _ => false,
        };

        void Answered(ApiReply reply)
        {
            var done = pending;
            pending = null;
            done(Settled(reply));
        }

        AnalyticsEventBody Fill(int index, AnalyticsEvent analyticsEvent)
        {
            if (index == bodies.Count) bodies.Add(new AnalyticsEventBody());
            var body = bodies[index];
            body.name = analyticsEvent.Name;
            body.label = analyticsEvent.Label ?? string.Empty;
            body.value = analyticsEvent.Value;
            body.clientTimeMs = analyticsEvent.ClientTimeMs;
            return body;
        }
    }
}
