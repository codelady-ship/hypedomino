using System;
using HypeDomino.Net.Api;

namespace HypeDomino.Game.Privacy
{
    public sealed class ConsentStartup
    {
        readonly ApiClient api;
        readonly ConsentGates gates;
        readonly Action<ApiReply> loaded;
        readonly Action<FirstLaunchConsentsResult> written;
        FirstLaunchConsents firstLaunch;
        string signingIn;

        public ConsentStartup(ApiClient api, ConsentGates gates)
        {
            this.api = api ?? throw new ArgumentNullException(nameof(api));
            this.gates = gates ?? throw new ArgumentNullException(nameof(gates));
            loaded = Loaded;
            written = Written;
        }

        public bool Busy { get; private set; }

        public string AppliedAccount { get; private set; }

        public void Hold(FirstLaunchConsents choices) => firstLaunch = choices ?? throw new ArgumentNullException(nameof(choices));

        public bool SignedIn(string accountId)
        {
            if (Busy || string.IsNullOrEmpty(accountId)) return false;
            if (firstLaunch != null && firstLaunch.Held > 0) return Write(accountId);
            if (accountId == AppliedAccount) return false;
            Busy = true;
            signingIn = accountId;
            api.Get(PrivacySettings.Path, loaded);
            return true;
        }

        bool Write(string accountId)
        {
            Busy = true;
            signingIn = accountId;
            if (firstLaunch.Write(api, written)) return true;
            Busy = false;
            return false;
        }

        void Written(FirstLaunchConsentsResult result)
        {
            Busy = false;
            if (result.Outcome != FirstLaunchConsentsOutcome.Written) return;
            if (firstLaunch.Held == 0) firstLaunch = null;
            Apply(result.Choices);
        }

        void Loaded(ApiReply reply)
        {
            Busy = false;
            var result = PrivacySettingsResult.From(reply);
            if (result.Outcome == PrivacySettingsOutcome.Done) Apply(result.Choices);
        }

        void Apply(ConsentChoicesReply choices)
        {
            gates.Apply(choices);
            AppliedAccount = signingIn;
        }
    }
}
