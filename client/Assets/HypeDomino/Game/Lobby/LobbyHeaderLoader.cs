using System;
using HypeDomino.Game.Profile;
using HypeDomino.Game.Shop;
using HypeDomino.Net.Api;
using HypeDomino.UI.Profile;
using HypeDomino.UI.Shop;

namespace HypeDomino.Game.Lobby
{
    public enum LobbyHeaderLoadOutcome
    {
        Loaded,
        SetupRequired,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public sealed class LobbyHeaderLoader
    {
        readonly ApiClient api;
        readonly LobbyHeader header;
        readonly Action<ApiReply> walletAnswered;
        readonly Action<ApiReply> profileAnswered;
        int answersDue;
        LobbyHeaderLoadOutcome outcome;
        Action<LobbyHeaderLoadOutcome> done;

        public LobbyHeaderLoader(ApiClient api, LobbyHeader header)
        {
            this.api = api;
            this.header = header;
            walletAnswered = WalletAnswered;
            profileAnswered = ProfileAnswered;
        }

        public bool Busy => answersDue > 0;

        public bool Load(Action<LobbyHeaderLoadOutcome> done)
        {
            if (Busy) return false;
            answersDue = 2;
            outcome = LobbyHeaderLoadOutcome.Loaded;
            this.done = done;
            api.Get(ShopLoader.WalletPath, walletAnswered);
            api.Get(PlayerProfile.OwnPath, profileAnswered);
            return true;
        }

        void WalletAnswered(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok) Fail(Refused(reply.Outcome));
            else if (Balances.TryRead(reply.Body, out var balances)) header.Show(balances);
            else Fail(LobbyHeaderLoadOutcome.Failed);
            Answered();
        }

        void ProfileAnswered(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok) Fail(Refused(reply.Outcome));
            else if (PlayerProfile.TryFromOwn(reply.Body, out var profile)) header.ShowLevel(profile.Level);
            else Fail(Unread(reply.Body));
            Answered();
        }

        void Fail(LobbyHeaderLoadOutcome failure)
        {
            if (outcome == LobbyHeaderLoadOutcome.Loaded) outcome = failure;
        }

        void Answered()
        {
            if (--answersDue > 0) return;
            var answer = done;
            done = null;
            answer?.Invoke(outcome);
        }

        static LobbyHeaderLoadOutcome Unread(string body) =>
            ProfileLoadResult.Unread(body).Outcome == ProfileLoadOutcome.SetupRequired ? LobbyHeaderLoadOutcome.SetupRequired : LobbyHeaderLoadOutcome.Failed;

        static LobbyHeaderLoadOutcome Refused(ApiOutcome refusal) => refusal switch
        {
            ApiOutcome.Offline => LobbyHeaderLoadOutcome.Offline,
            ApiOutcome.SignInRequired => LobbyHeaderLoadOutcome.SignInRequired,
            ApiOutcome.RateLimited => LobbyHeaderLoadOutcome.RateLimited,
            _ => LobbyHeaderLoadOutcome.Failed,
        };
    }
}
