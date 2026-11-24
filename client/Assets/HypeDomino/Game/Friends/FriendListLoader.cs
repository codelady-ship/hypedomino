using System;
using System.Collections.Generic;
using HypeDomino.Net.Api;

namespace HypeDomino.Game.Friends
{
    public enum FriendListLoadOutcome
    {
        Loaded,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public sealed class FriendListLoader
    {
        readonly ApiClient api;
        readonly Action<ApiReply> answered;
        readonly List<Friend> friends = new();
        Action<FriendListLoadOutcome> done;

        public FriendListLoader(ApiClient api)
        {
            this.api = api ?? throw new ArgumentNullException(nameof(api));
            answered = Answered;
        }

        public bool Busy { get; private set; }
        public IReadOnlyList<Friend> Friends => friends;

        public bool Load(Action<FriendListLoadOutcome> done)
        {
            if (Busy) return false;
            Busy = true;
            this.done = done;
            api.Get(FriendList.Path, answered);
            return true;
        }

        void Answered(ApiReply reply)
        {
            Busy = false;
            var outcome = reply.Outcome == ApiOutcome.Ok ? Read(reply.Body) : Refused(reply.Outcome);
            var answer = done;
            done = null;
            answer?.Invoke(outcome);
        }

        FriendListLoadOutcome Read(string body) =>
            FriendList.TryRead(body, friends) ? FriendListLoadOutcome.Loaded : FriendListLoadOutcome.Failed;

        static FriendListLoadOutcome Refused(ApiOutcome refusal) => refusal switch
        {
            ApiOutcome.Offline => FriendListLoadOutcome.Offline,
            ApiOutcome.SignInRequired => FriendListLoadOutcome.SignInRequired,
            ApiOutcome.RateLimited => FriendListLoadOutcome.RateLimited,
            _ => FriendListLoadOutcome.Failed,
        };
    }
}
