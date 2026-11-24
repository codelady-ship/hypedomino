using System;
using UnityEngine;

namespace HypeDomino.Net.Auth
{
    [Serializable]
    public sealed class NonceReply
    {
        public string nonce;
    }

    [Serializable]
    public sealed class TokenReply
    {
        public string access_token;
        public string refresh_token;
        public int expires_in;
        public string request_key;
        public string account_id;
        public bool created;
    }

    [Serializable]
    public sealed class ErrorReply
    {
        public string error;
        public string delete_at;
        public string category;
        public string banned_until;
        public AppealGrantReply appeal;
    }

    [Serializable]
    public sealed class SignInRequest
    {
        public string provider;
        public string id_token;
        public string nonce;
        public string device_id;
    }

    [Serializable]
    public sealed class RefreshRequest
    {
        public string refresh_token;
        public string device_id;
    }

    [Serializable]
    public sealed class LogoutRequest
    {
        public string refresh_token;
    }

    static class AuthJson
    {
        public static T Read<T>(string json) where T : class
        {
            if (string.IsNullOrEmpty(json)) return null;
            try
            {
                return JsonUtility.FromJson<T>(json);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
