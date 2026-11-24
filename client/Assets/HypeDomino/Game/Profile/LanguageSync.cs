using System;
using HypeDomino.Net.Api;
using HypeDomino.UI.Localization;
using UnityEngine;

namespace HypeDomino.Game.Profile
{
    [Serializable]
    public sealed class AccountLanguage
    {
        public string language;

        public string ToJson() => JsonUtility.ToJson(this);

        public static string Read(string body)
        {
            if (string.IsNullOrEmpty(body)) return null;
            try
            {
                return JsonUtility.FromJson<AccountLanguage>(body)?.language ?? string.Empty;
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }

    public sealed class LanguageSync
    {
        public const string Path = "/profile/language";

        readonly ApiClient api;
        readonly LanguageChoice choice;
        readonly AccountLanguage request = new();
        readonly Action<ApiReply> onRead;
        readonly Action<ApiReply> onSaved;
        string account;
        string requestAccount;
        bool accountRead;
        bool readNeeded;
        bool unsaved;

        public LanguageSync(ApiClient api, LanguageChoice choice)
        {
            this.api = api;
            this.choice = choice;
            onRead = Read;
            onSaved = Saved;
            choice.Changed += ChoiceChanged;
        }

        public bool Busy { get; private set; }

        public void SignedIn(string accountId)
        {
            if (string.IsNullOrEmpty(accountId)) return;
            if (accountId != account)
            {
                account = accountId;
                accountRead = false;
                readNeeded = true;
                unsaved = false;
            }
            Pump();
        }

        void ChoiceChanged()
        {
            if (account == null) return;
            unsaved = true;
            Pump();
        }

        void Pump()
        {
            if (Busy) return;
            if (readNeeded)
            {
                Busy = true;
                readNeeded = false;
                requestAccount = account;
                api.Get(Path, onRead);
                return;
            }
            if (!accountRead || !unsaved) return;
            Busy = true;
            unsaved = false;
            requestAccount = account;
            request.language = choice.Language;
            api.Put(Path, request.ToJson(), onSaved);
        }

        void Read(ApiReply reply)
        {
            Busy = false;
            if (requestAccount != account)
            {
                Pump();
                return;
            }
            var stored = reply.Outcome == ApiOutcome.Ok ? AccountLanguage.Read(reply.Body) : null;
            if (stored == null)
            {
                readNeeded = true;
                return;
            }
            accountRead = true;
            if (!GameLanguage.IsLaunch(stored)) unsaved = true;
            Pump();
        }

        void Saved(ApiReply reply)
        {
            Busy = false;
            var failed = reply.Outcome != ApiOutcome.Ok && reply.Outcome != ApiOutcome.Rejected;
            if (failed && requestAccount == account)
            {
                unsaved = true;
                return;
            }
            Pump();
        }
    }
}
