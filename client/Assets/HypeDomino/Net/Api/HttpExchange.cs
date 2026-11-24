using System.Collections.Generic;
using System.Text;
using HypeDomino.Net.Auth;
using UnityEngine.Networking;

namespace HypeDomino.Net.Api
{
    public readonly struct HttpRequestPlan
    {
        public readonly string Url;
        public readonly string Method;
        public readonly byte[] Body;
        public readonly IReadOnlyList<ApiHeader> Headers;

        public HttpRequestPlan(string url, string method, byte[] body, IReadOnlyList<ApiHeader> headers)
        {
            Url = url;
            Method = method;
            Body = body;
            Headers = headers;
        }
    }

    public static class HttpExchange
    {
        public const string ContentType = "Content-Type";
        public const string JsonMediaType = "application/json";
        public const string Authorization = "Authorization";

        public static HttpRequestPlan Plan(string baseUrl, string method, string pathAndQuery, string body, string authorization, string mac, IReadOnlyList<ApiHeader> extraHeaders)
        {
            var headers = new List<ApiHeader>();
            if (body != null) headers.Add(new ApiHeader(ContentType, JsonMediaType));
            if (authorization != null) headers.Add(new ApiHeader(Authorization, authorization));
            if (mac != null) headers.Add(new ApiHeader(RequestMac.Header, mac));
            if (extraHeaders != null) headers.AddRange(extraHeaders);
            var bytes = body == null ? null : Encoding.UTF8.GetBytes(body);
            return new HttpRequestPlan(baseUrl.TrimEnd('/') + pathAndQuery, method, bytes, headers);
        }

        public static HttpReply Reply(UnityWebRequest.Result result, long responseCode, string text) =>
            IsAnswered(result) ? new HttpReply((int)responseCode, text) : new HttpReply(HttpReply.NoResponse, null);

        static bool IsAnswered(UnityWebRequest.Result result) =>
            result is UnityWebRequest.Result.Success or UnityWebRequest.Result.ProtocolError;
    }
}
