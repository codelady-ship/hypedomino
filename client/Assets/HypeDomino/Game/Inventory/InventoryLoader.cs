using System;
using HypeDomino.Net.Api;
using HypeDomino.UI.Cosmetics;

namespace HypeDomino.Game.Inventory
{
    public enum InventoryLoadOutcome
    {
        Loaded,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class InventoryLoadKeys
    {
        public const string Offline = "inventory.load.offline";
        public const string RateLimited = "inventory.load.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(InventoryLoadOutcome outcome) => outcome switch
        {
            InventoryLoadOutcome.Loaded => null,
            InventoryLoadOutcome.Offline => Offline,
            InventoryLoadOutcome.RateLimited => RateLimited,
            InventoryLoadOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct InventoryLoadResult
    {
        InventoryLoadResult(InventoryLoadOutcome outcome, InventoryReply inventory = null)
        {
            Outcome = outcome;
            Inventory = inventory;
        }

        public InventoryLoadOutcome Outcome { get; }
        public InventoryReply Inventory { get; }
        public string MessageKey => InventoryLoadKeys.Of(Outcome);

        public static InventoryLoadResult Loaded(InventoryReply inventory) => new(InventoryLoadOutcome.Loaded, inventory);

        public static InventoryLoadResult Refused(ApiReply reply) => new(reply.Outcome switch
        {
            ApiOutcome.Offline => InventoryLoadOutcome.Offline,
            ApiOutcome.SignInRequired => InventoryLoadOutcome.SignInRequired,
            ApiOutcome.RateLimited => InventoryLoadOutcome.RateLimited,
            _ => InventoryLoadOutcome.Failed,
        });

        public static InventoryLoadResult Unreadable() => new(InventoryLoadOutcome.Failed);
    }

    public sealed class InventoryLoader
    {
        readonly ApiClient api;

        public InventoryLoader(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public bool Load(Action<InventoryLoadResult> done)
        {
            if (Busy)
            {
                return false;
            }
            Busy = true;
            api.Get(CosmeticCollection.InventoryPath, reply => Answered(reply, done));
            return true;
        }

        void Answered(ApiReply reply, Action<InventoryLoadResult> done)
        {
            Busy = false;
            done(Read(reply));
        }

        static InventoryLoadResult Read(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                return InventoryLoadResult.Refused(reply);
            }
            var inventory = ParseReply(reply.Body);
            return inventory == null ? InventoryLoadResult.Unreadable() : InventoryLoadResult.Loaded(inventory);
        }

        static InventoryReply ParseReply(string body)
        {
            if (string.IsNullOrEmpty(body))
            {
                return null;
            }
            try
            {
                return InventoryReply.FromJson(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
