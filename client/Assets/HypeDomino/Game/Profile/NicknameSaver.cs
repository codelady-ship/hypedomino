using System;
using HypeDomino.Core.Analytics;
using HypeDomino.Net.Api;
using HypeDomino.UI.Profile;

namespace HypeDomino.Game.Profile
{
    public enum NicknameSaveOutcome
    {
        Saved,
        InvalidLength,
        InvalidCharacters,
        InvalidSeparators,
        MixedScripts,
        DigitsOnly,
        Taken,
        Unchanged,
        NotAllowed,
        Cooldown,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class NicknameSaveKeys
    {
        public const string InvalidLength = "nickname.rename.invalid_length";
        public const string InvalidCharacters = "nickname.rename.invalid_characters";
        public const string InvalidSeparators = "nickname.rename.invalid_separators";
        public const string MixedScripts = "nickname.rename.mixed_scripts";
        public const string DigitsOnly = "nickname.rename.digits_only";
        public const string Taken = "nickname.rename.taken";
        public const string Unchanged = "nickname.rename.unchanged";
        public const string NotAllowed = "nickname.rename.not_allowed";
        public const string Cooldown = "nickname.rename.cooldown";
        public const string Offline = "nickname.rename.offline";
        public const string RateLimited = "nickname.rename.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(NicknameSaveOutcome outcome) => outcome switch
        {
            NicknameSaveOutcome.Saved => null,
            NicknameSaveOutcome.InvalidLength => InvalidLength,
            NicknameSaveOutcome.InvalidCharacters => InvalidCharacters,
            NicknameSaveOutcome.InvalidSeparators => InvalidSeparators,
            NicknameSaveOutcome.MixedScripts => MixedScripts,
            NicknameSaveOutcome.DigitsOnly => DigitsOnly,
            NicknameSaveOutcome.Taken => Taken,
            NicknameSaveOutcome.Unchanged => Unchanged,
            NicknameSaveOutcome.NotAllowed => NotAllowed,
            NicknameSaveOutcome.Cooldown => Cooldown,
            NicknameSaveOutcome.Offline => Offline,
            NicknameSaveOutcome.RateLimited => RateLimited,
            NicknameSaveOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct NicknameSaveResult
    {
        const string CooldownError = "nickname.cooldown";

        NicknameSaveResult(NicknameSaveOutcome outcome, PlayerProfile profile = null, ulong retryAtUnixMs = 0)
        {
            Outcome = outcome;
            Profile = profile;
            RetryAtUnixMs = retryAtUnixMs;
        }

        public NicknameSaveOutcome Outcome { get; }
        public PlayerProfile Profile { get; }
        public ulong RetryAtUnixMs { get; }
        public string MessageKey => NicknameSaveKeys.Of(Outcome);

        public static NicknameSaveResult Saved(PlayerProfile profile) => new NicknameSaveResult(NicknameSaveOutcome.Saved, profile);

        public static NicknameSaveResult Unreadable() => new NicknameSaveResult(NicknameSaveOutcome.Failed);

        public static NicknameSaveResult Refused(ApiReply reply) => reply.Outcome switch
        {
            ApiOutcome.Offline => new NicknameSaveResult(NicknameSaveOutcome.Offline),
            ApiOutcome.SignInRequired => new NicknameSaveResult(NicknameSaveOutcome.SignInRequired),
            ApiOutcome.RateLimited when reply.Error == CooldownError => new NicknameSaveResult(NicknameSaveOutcome.Cooldown, retryAtUnixMs: NicknameRenameOutcome.Read(reply.Body).RetryAtUnixMs),
            ApiOutcome.RateLimited => new NicknameSaveResult(NicknameSaveOutcome.RateLimited),
            ApiOutcome.Rejected => new NicknameSaveResult(ReasonOf(reply.Error)),
            _ => new NicknameSaveResult(NicknameSaveOutcome.Failed),
        };

        static NicknameSaveOutcome ReasonOf(string error) => error switch
        {
            "nickname.invalid_length" => NicknameSaveOutcome.InvalidLength,
            "nickname.invalid_characters" => NicknameSaveOutcome.InvalidCharacters,
            "nickname.invalid_separators" => NicknameSaveOutcome.InvalidSeparators,
            "nickname.mixed_scripts" => NicknameSaveOutcome.MixedScripts,
            "nickname.digits_only" => NicknameSaveOutcome.DigitsOnly,
            "nickname.taken" => NicknameSaveOutcome.Taken,
            "nickname.unchanged" => NicknameSaveOutcome.Unchanged,
            "nickname.not_allowed" => NicknameSaveOutcome.NotAllowed,
            _ => NicknameSaveOutcome.Failed,
        };
    }

    public sealed class NicknameSaver
    {
        readonly ApiClient api;
        readonly AnalyticsQueue analytics;

        public NicknameSaver(ApiClient api, AnalyticsQueue analytics)
        {
            this.api = api;
            this.analytics = analytics ?? throw new ArgumentNullException(nameof(analytics));
        }

        public bool Busy { get; private set; }

        public bool Save(string nickname, Action<NicknameSaveResult> done)
        {
            if (Busy || !NicknameRename.TryRequest(nickname, out var request))
            {
                return false;
            }
            Busy = true;
            api.Put(NicknameRename.Path, request.ToJson(), reply => Answered(reply, done));
            return true;
        }

        void Answered(ApiReply reply, Action<NicknameSaveResult> done)
        {
            Busy = false;
            var result = Read(reply);
            if (result.Outcome == NicknameSaveOutcome.Saved) analytics.Track(AnalyticsEvents.NicknameSet);
            done(result);
        }

        static NicknameSaveResult Read(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                return NicknameSaveResult.Refused(reply);
            }
            var outcome = NicknameRenameOutcome.Read(reply.Body);
            return outcome.Status == NicknameRenameStatus.Accepted ? NicknameSaveResult.Saved(outcome.Profile) : NicknameSaveResult.Unreadable();
        }
    }
}
