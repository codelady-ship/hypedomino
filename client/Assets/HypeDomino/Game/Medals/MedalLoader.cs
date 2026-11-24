using System;
using HypeDomino.Net.Api;
using HypeDomino.UI.Medals;

namespace HypeDomino.Game.Medals
{
    public enum MedalLoadOutcome
    {
        Loaded,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class MedalLoadKeys
    {
        public const string Offline = "medal.load.offline";
        public const string RateLimited = "medal.load.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(MedalLoadOutcome outcome) => outcome switch
        {
            MedalLoadOutcome.Loaded => null,
            MedalLoadOutcome.Offline => Offline,
            MedalLoadOutcome.RateLimited => RateLimited,
            MedalLoadOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct MedalLoadResult
    {
        MedalLoadResult(MedalLoadOutcome outcome, MedalCollectionReply collection = null)
        {
            Outcome = outcome;
            Collection = collection;
        }

        public MedalLoadOutcome Outcome { get; }
        public MedalCollectionReply Collection { get; }
        public string MessageKey => MedalLoadKeys.Of(Outcome);

        public static MedalLoadResult Loaded(MedalCollectionReply collection) => new(MedalLoadOutcome.Loaded, collection);

        public static MedalLoadResult Refused(ApiReply reply) => new(reply.Outcome switch
        {
            ApiOutcome.Offline => MedalLoadOutcome.Offline,
            ApiOutcome.SignInRequired => MedalLoadOutcome.SignInRequired,
            ApiOutcome.RateLimited => MedalLoadOutcome.RateLimited,
            _ => MedalLoadOutcome.Failed,
        });

        public static MedalLoadResult Unreadable() => new(MedalLoadOutcome.Failed);
    }

    public sealed class MedalLoader
    {
        readonly ApiClient api;

        public MedalLoader(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public bool Load(Action<MedalLoadResult> done)
        {
            if (Busy)
            {
                return false;
            }
            Busy = true;
            api.Get(MedalCollection.CollectionPath, reply => Answered(reply, done));
            return true;
        }

        void Answered(ApiReply reply, Action<MedalLoadResult> done)
        {
            Busy = false;
            done(Read(reply));
        }

        static MedalLoadResult Read(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                return MedalLoadResult.Refused(reply);
            }
            var collection = ParseReply(reply.Body);
            return collection == null ? MedalLoadResult.Unreadable() : MedalLoadResult.Loaded(collection);
        }

        static MedalCollectionReply ParseReply(string body)
        {
            if (string.IsNullOrEmpty(body))
            {
                return null;
            }
            try
            {
                return MedalCollectionReply.FromJson(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
