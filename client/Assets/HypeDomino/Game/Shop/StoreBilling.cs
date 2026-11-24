using System;

namespace HypeDomino.Game.Shop
{
    public enum StorePlatform
    {
        Google,
        Apple,
    }

    public enum StoreBillingOutcome
    {
        Purchased,
        Deferred,
        Cancelled,
        Unavailable,
        Failed,
    }

    public sealed class StoreReceipt
    {
        public StoreReceipt(StorePlatform platform, string token)
        {
            Platform = platform;
            Token = token;
        }

        public StorePlatform Platform { get; }
        public string Token { get; }

        public string PlatformWireName => Platform == StorePlatform.Apple ? "apple" : "google";
    }

    public readonly struct StoreBillingResult
    {
        public StoreBillingResult(StoreBillingOutcome outcome, StoreReceipt receipt = null)
        {
            Outcome = outcome;
            Receipt = receipt;
        }

        public StoreBillingOutcome Outcome { get; }
        public StoreReceipt Receipt { get; }
    }

    public interface IStoreBilling
    {
        bool CanPurchase { get; }

        void Purchase(string productId, string accountToken, Action<StoreBillingResult> done);

        void Finish(StoreReceipt receipt, bool finishedByServer);
    }

    public sealed class NoStoreBilling : IStoreBilling
    {
        public bool CanPurchase => false;

        public void Purchase(string productId, string accountToken, Action<StoreBillingResult> done) => done(new StoreBillingResult(StoreBillingOutcome.Unavailable));

        public void Finish(StoreReceipt receipt, bool finishedByServer)
        {
        }
    }
}
