using System;
using System.Globalization;
using UnityEngine;

namespace HypeDomino.Net
{
    [Serializable]
    public sealed class ServerEndpoint
    {
        public const string ResourceName = "ServerEndpoint";
        public const string EditorServerUrl = "http://localhost:8080";
        public const string SocketPath = "/ws";
        public const int NoTimeout = 0;

        public string serverUrl;
        public int httpTimeoutSeconds;

        public static ServerEndpoint Editor => new() { serverUrl = EditorServerUrl, httpTimeoutSeconds = NoTimeout };

        public Uri SocketUri => SocketUriOf(serverUrl);

        public static ServerEndpoint Load(bool isEditor)
        {
            var asset = Resources.Load<TextAsset>(ResourceName);
            if (asset != null) return FromJson(asset.text);
            return isEditor ? Editor : null;
        }

        public static ServerEndpoint FromJson(string json)
        {
            var read = JsonUtility.FromJson<ServerEndpoint>(json) ?? throw new ArgumentException("the server endpoint is empty");
            return Parse(read.serverUrl, read.httpTimeoutSeconds.ToString(CultureInfo.InvariantCulture));
        }

        public static ServerEndpoint Parse(string serverUrl, string httpTimeoutSeconds)
        {
            if (!IsServerUrl(serverUrl)) throw new ArgumentException($"the server URL must be an absolute http or https URL without user, query or fragment, got '{serverUrl}'");
            if (!int.TryParse(httpTimeoutSeconds, NumberStyles.None, CultureInfo.InvariantCulture, out var timeout) || timeout < 1)
                throw new ArgumentException($"the HTTP timeout must be a whole number of seconds from 1, got '{httpTimeoutSeconds}'");
            return new ServerEndpoint { serverUrl = serverUrl.TrimEnd('/'), httpTimeoutSeconds = timeout };
        }

        public static Uri SocketUriOf(string serverUrl)
        {
            var trimmed = serverUrl.TrimEnd('/');
            var scheme = new Uri(trimmed).Scheme;
            return new Uri((scheme == Uri.UriSchemeHttps ? "wss" : "ws") + trimmed.Substring(scheme.Length) + SocketPath);
        }

        static bool IsServerUrl(string url) =>
            Uri.TryCreate(url, UriKind.Absolute, out var uri)
            && (uri.Scheme == Uri.UriSchemeHttp || uri.Scheme == Uri.UriSchemeHttps)
            && uri.Query.Length == 0 && uri.Fragment.Length == 0 && uri.UserInfo.Length == 0;
    }
}
