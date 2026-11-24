using System;
using HypeDomino.Net.Api;
using HypeDomino.UI.Cosmetics;

namespace HypeDomino.Game.Inventory
{
    public enum EquipOutcome
    {
        Equipped,
        NotOwned,
        Unavailable,
        SlotsFull,
        ProfileRequired,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class EquipKeys
    {
        public const string NotOwned = "inventory.equip.not_owned";
        public const string Unavailable = "inventory.equip.unavailable";
        public const string SlotsFull = "inventory.equip.slots_full";
        public const string Offline = "inventory.equip.offline";
        public const string RateLimited = "inventory.equip.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(EquipOutcome outcome) => outcome switch
        {
            EquipOutcome.Equipped => null,
            EquipOutcome.NotOwned => NotOwned,
            EquipOutcome.Unavailable => Unavailable,
            EquipOutcome.SlotsFull => SlotsFull,
            EquipOutcome.Offline => Offline,
            EquipOutcome.RateLimited => RateLimited,
            EquipOutcome.ProfileRequired => null,
            EquipOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct EquipResult
    {
        EquipResult(EquipOutcome outcome, EquipReply equipped = null)
        {
            Outcome = outcome;
            Equipped = equipped;
        }

        public EquipOutcome Outcome { get; }
        public EquipReply Equipped { get; }
        public string MessageKey => EquipKeys.Of(Outcome);
        public bool InventoryStale => Outcome == EquipOutcome.NotOwned || Outcome == EquipOutcome.Unavailable;

        public static EquipResult Accepted(EquipReply equipped) => new(EquipOutcome.Equipped, equipped);

        public static EquipResult Refused(ApiReply reply) => new(reply.Outcome switch
        {
            ApiOutcome.Offline => EquipOutcome.Offline,
            ApiOutcome.SignInRequired => EquipOutcome.SignInRequired,
            ApiOutcome.RateLimited => EquipOutcome.RateLimited,
            ApiOutcome.Rejected when reply.Error == "item.not_owned" => EquipOutcome.NotOwned,
            ApiOutcome.Rejected when reply.Error == "item.disabled" => EquipOutcome.Unavailable,
            ApiOutcome.Rejected when reply.Error == "item.unknown" => EquipOutcome.Unavailable,
            ApiOutcome.Rejected when reply.Error == "slot.too_many" => EquipOutcome.SlotsFull,
            ApiOutcome.Rejected when reply.Error == "profile.setup_required" => EquipOutcome.ProfileRequired,
            _ => EquipOutcome.Failed,
        });

        public static EquipResult Unreadable() => new(EquipOutcome.Failed);
    }

    public sealed class LoadoutEquipper
    {
        readonly ApiClient api;

        public LoadoutEquipper(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public bool Equip(EquipRequest request, Action<EquipResult> done)
        {
            if (Busy || string.IsNullOrEmpty(request?.slot) || request.items == null)
            {
                return false;
            }
            Busy = true;
            api.Put(CosmeticCollection.EquipPath, request.ToJson(), reply => Answered(reply, done));
            return true;
        }

        void Answered(ApiReply reply, Action<EquipResult> done)
        {
            Busy = false;
            done(Read(reply));
        }

        static EquipResult Read(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                return EquipResult.Refused(reply);
            }
            var equipped = ParseReply(reply.Body);
            return equipped?.loadout == null ? EquipResult.Unreadable() : EquipResult.Accepted(equipped);
        }

        static EquipReply ParseReply(string body)
        {
            if (string.IsNullOrEmpty(body))
            {
                return null;
            }
            try
            {
                return EquipReply.FromJson(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
