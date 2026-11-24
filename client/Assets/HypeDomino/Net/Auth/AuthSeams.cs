using System;

namespace HypeDomino.Net.Auth
{
    public enum AuthProvider
    {
        Google,
        Apple,
    }

    public enum IdTokenOutcome
    {
        Token,
        Cancelled,
        Failed,
    }

    public readonly struct IdTokenResult
    {
        public readonly IdTokenOutcome Outcome;
        public readonly string Token;

        public IdTokenResult(IdTokenOutcome outcome, string token)
        {
            Outcome = outcome;
            Token = token;
        }
    }

    public interface IIdentityProvider
    {
        bool Supports(AuthProvider provider);

        void RequestIdToken(AuthProvider provider, string nonce, Action<IdTokenResult> done);
    }

    public readonly struct HttpReply
    {
        public const int NoResponse = 0;

        public readonly int Status;
        public readonly string Body;

        public HttpReply(int status, string body)
        {
            Status = status;
            Body = body;
        }
    }

    public interface IAuthTransport
    {
        void Post(string path, string json, Action<HttpReply> done);
    }

    public interface ISecureStore
    {
        string Read(string key);

        void Write(string key, string value);

        void Delete(string key);
    }
}
