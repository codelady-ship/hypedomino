using System;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.UI.Profile;

namespace HypeDomino.Game.Profile
{
    public enum ProfileLoadOutcome
    {
        Loaded,
        SetupRequired,
        Unknown,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class ProfileLoadKeys
    {
        public const string Unknown = "profile.load.unknown";
        public const string Offline = "profile.load.offline";
        public const string RateLimited = "profile.load.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(ProfileLoadOutcome outcome) => outcome switch
        {
            ProfileLoadOutcome.Loaded => null,
            ProfileLoadOutcome.SetupRequired => null,
            ProfileLoadOutcome.Unknown => Unknown,
            ProfileLoadOutcome.Offline => Offline,
            ProfileLoadOutcome.RateLimited => RateLimited,
            ProfileLoadOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct ProfileLoadResult
    {
        const string UnknownProfileError = "profile.unknown";

        ProfileLoadResult(ProfileLoadOutcome outcome, PlayerProfile profile = null)
        {
            Outcome = outcome;
            Profile = profile;
        }

        public ProfileLoadOutcome Outcome { get; }
        public PlayerProfile Profile { get; }
        public string MessageKey => ProfileLoadKeys.Of(Outcome);

        public static ProfileLoadResult Loaded(PlayerProfile profile) => new(ProfileLoadOutcome.Loaded, profile);

        public static ProfileLoadResult Refused(ApiReply reply) => new(reply.Outcome switch
        {
            ApiOutcome.Offline => ProfileLoadOutcome.Offline,
            ApiOutcome.SignInRequired => ProfileLoadOutcome.SignInRequired,
            ApiOutcome.RateLimited => ProfileLoadOutcome.RateLimited,
            ApiOutcome.Rejected when reply.Error == UnknownProfileError => ProfileLoadOutcome.Unknown,
            _ => ProfileLoadOutcome.Failed,
        });

        public static ProfileLoadResult Unread(string body) => new(SetupRequired(body) ? ProfileLoadOutcome.SetupRequired : ProfileLoadOutcome.Failed);

        static bool SetupRequired(string body) => JsonMap.TryParse(body, out _) && OwnProfileReply.FromJson(body).setup_required;
    }

    public sealed class ProfileLoader
    {
        delegate bool ProfileReader(string json, out PlayerProfile profile);

        readonly ApiClient api;

        public ProfileLoader(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public bool Load(Action<ProfileLoadResult> done) => Start(PlayerProfile.OwnPath, PlayerProfile.TryFromOwn, done);

        public bool LoadPublic(string accountId, Action<ProfileLoadResult> done) =>
            !string.IsNullOrEmpty(accountId) && Start(PlayerProfile.PublicPath(accountId), PlayerProfile.TryFromPublic, done);

        bool Start(string path, ProfileReader read, Action<ProfileLoadResult> done)
        {
            if (Busy)
            {
                return false;
            }
            Busy = true;
            api.Get(path, reply => Answered(Read(reply, read), done));
            return true;
        }

        void Answered(ProfileLoadResult result, Action<ProfileLoadResult> done)
        {
            Busy = false;
            done(result);
        }

        static ProfileLoadResult Read(ApiReply reply, ProfileReader read)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                return ProfileLoadResult.Refused(reply);
            }
            return read(reply.Body, out var profile) ? ProfileLoadResult.Loaded(profile) : ProfileLoadResult.Unread(reply.Body);
        }
    }
}
