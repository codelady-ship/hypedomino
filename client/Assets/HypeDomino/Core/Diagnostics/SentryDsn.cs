using System;

namespace HypeDomino.Core.Diagnostics
{
    public sealed class SentryDsn
    {
        public const string EuIngestHostSuffix = ".ingest.de.sentry.io";
        public const string ProtocolVersion = "7";

        public readonly string EnvelopeUrl;
        public readonly string AuthHeader;

        SentryDsn(string envelopeUrl, string publicKey)
        {
            EnvelopeUrl = envelopeUrl;
            AuthHeader = $"Sentry sentry_version={ProtocolVersion}, sentry_key={publicKey}";
        }

        public static SentryDsn Parse(string raw)
        {
            if (!Uri.TryCreate(raw, UriKind.Absolute, out var uri) || uri.Scheme != Uri.UriSchemeHttps || uri.Query.Length > 0 || uri.Fragment.Length > 0)
                throw new ArgumentException("the Sentry DSN is not an https DSN");
            var publicKey = PublicKeyOf(uri.UserInfo);
            if (publicKey.Length == 0) throw new ArgumentException("the Sentry DSN has no public key");
            var path = uri.AbsolutePath.TrimEnd('/');
            var slash = path.LastIndexOf('/');
            var projectId = path.Substring(slash + 1);
            if (!IsProjectId(projectId)) throw new ArgumentException("the Sentry DSN has no project id");
            if (!uri.Host.EndsWith(EuIngestHostSuffix, StringComparison.OrdinalIgnoreCase)) throw new ArgumentException("the Sentry DSN is not in the EU data region");
            return new SentryDsn($"https://{uri.Authority}{path.Substring(0, slash)}/api/{projectId}/envelope/", publicKey);
        }

        static string PublicKeyOf(string userInfo)
        {
            var colon = userInfo.IndexOf(':');
            return colon < 0 ? userInfo : userInfo.Substring(0, colon);
        }

        static bool IsProjectId(string text)
        {
            if (text.Length == 0) return false;
            foreach (var character in text)
                if (character < '0' || character > '9') return false;
            return true;
        }
    }
}
