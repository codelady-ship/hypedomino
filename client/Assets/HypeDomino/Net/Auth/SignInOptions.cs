using System;
using System.Collections.Generic;
using UnityEngine;

namespace HypeDomino.Net.Auth
{
    public static class SignInOptions
    {
        static readonly AuthProvider[] None = Array.Empty<AuthProvider>();
        static readonly AuthProvider[] GoogleOnly = { AuthProvider.Google };
        static readonly AuthProvider[] GoogleAndApple = { AuthProvider.Google, AuthProvider.Apple };

        public static IReadOnlyList<AuthProvider> Current => For(Application.platform);

        public static IReadOnlyList<AuthProvider> For(RuntimePlatform platform) => platform switch
        {
            RuntimePlatform.Android => GoogleOnly,
            RuntimePlatform.IPhonePlayer => GoogleAndApple,
            _ => None,
        };
    }
}
