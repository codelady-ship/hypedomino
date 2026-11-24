using System;
using System.Collections.Generic;
using HypeDomino.Net.Auth;

namespace HypeDomino.Net.Api
{
    public interface IApiTransport
    {
        void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done);
    }

    public interface IApiHeaderTransport : IApiTransport
    {
        void Send(string method, string pathAndQuery, string body, string authorization, string mac, IReadOnlyList<ApiHeader> headers, Action<HttpReply> done);
    }

    public readonly struct ApiHeader
    {
        public readonly string Name;
        public readonly string Value;

        public ApiHeader(string name, string value)
        {
            Name = name;
            Value = value;
        }
    }
}
