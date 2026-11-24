using System;
using HypeDomino.Core.Analytics;
using HypeDomino.Core.Diagnostics;
using HypeDomino.Net.Api;

namespace HypeDomino.Game.Privacy
{
    public sealed class PrivacySettings
    {
        public const string Path = "/consents";

        readonly ApiClient api;
        readonly ConsentGates gates;
        readonly string policyVersion;
        readonly string appVersion;

        public PrivacySettings(ApiClient api, AnalyticsQueue analytics, ErrorReports diagnostics, Func<string> newAnalyticsId, string policyVersion, string appVersion)
        {
            if (string.IsNullOrEmpty(policyVersion)) throw new ArgumentException("policy version required", nameof(policyVersion));
            if (string.IsNullOrEmpty(appVersion)) throw new ArgumentException("app version required", nameof(appVersion));
            this.api = api;
            gates = new ConsentGates(analytics, diagnostics, newAnalyticsId);
            this.policyVersion = policyVersion;
            this.appVersion = appVersion;
        }

        public event Action Changed;

        public bool Busy { get; private set; }

        public bool Loaded { get; private set; }

        public bool Analytics { get; private set; }

        public bool Diagnostics { get; private set; }

        public static string RandomAnalyticsId() => Guid.NewGuid().ToString("D");

        public bool Load(Action<PrivacySettingsResult> done)
        {
            if (Busy) return false;
            Busy = true;
            api.Get(Path, reply => Answer(reply, done));
            return true;
        }

        public bool SetAnalytics(bool on, Action<PrivacySettingsResult> done) => Save(on, Diagnostics, done);

        public bool SetDiagnostics(bool on, Action<PrivacySettingsResult> done) => Save(Analytics, on, done);

        bool Save(bool analyticsOn, bool diagnosticsOn, Action<PrivacySettingsResult> done)
        {
            if (Busy || !Loaded) return false;
            Busy = true;
            gates.Withdraw(analyticsOn, diagnosticsOn);
            var request = new ConsentChangeRequest { analytics = analyticsOn, diagnostics = diagnosticsOn, policyVersion = policyVersion, appVersion = appVersion };
            api.Put(Path, request.ToJson(), reply => Answer(reply, done));
            return true;
        }

        void Answer(ApiReply reply, Action<PrivacySettingsResult> done)
        {
            Busy = false;
            var result = PrivacySettingsResult.From(reply);
            if (result.Outcome == PrivacySettingsOutcome.Done) Apply(result.Choices);
            done(result);
        }

        void Apply(ConsentChoicesReply choices)
        {
            var changed = !Loaded || choices.analytics != Analytics || choices.diagnostics != Diagnostics;
            Loaded = true;
            Analytics = choices.analytics;
            Diagnostics = choices.diagnostics;
            gates.Apply(choices);
            if (changed) Changed?.Invoke();
        }
    }
}
