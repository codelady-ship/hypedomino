using System;

namespace HypeDomino.Core.Analytics
{
    public static class AnalyticsEvents
    {
        public const string AppOpen = "app_open";
        public const string SignIn = "sign_in";
        public const string NicknameSet = "nickname_set";
        public const string ScreenView = "screen_view";
        public const string QueueJoin = "queue_join";
        public const string PurchaseStart = "purchase_start";
        public const string PurchaseCancel = "purchase_cancel";

        static readonly string[] Client = { AppOpen, SignIn, NicknameSet, ScreenView, QueueJoin, PurchaseStart, PurchaseCancel };

        public static bool IsClientEvent(string name) => Array.IndexOf(Client, name) >= 0;
    }
}
