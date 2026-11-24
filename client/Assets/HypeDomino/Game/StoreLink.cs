using System;
using UnityEngine;

namespace HypeDomino.Game
{
    public static class StoreLink
    {
        public const string PlayStorePrefix = "https://play.google.com/store/apps/details?id=";

        public static string Of(RuntimePlatform platform, string applicationId) =>
            platform == RuntimePlatform.Android && !string.IsNullOrEmpty(applicationId) ? PlayStorePrefix + Uri.EscapeDataString(applicationId) : string.Empty;
    }
}
