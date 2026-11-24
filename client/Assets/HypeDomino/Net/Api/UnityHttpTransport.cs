using System;
using System.Collections.Generic;
using HypeDomino.Net.Auth;
using UnityEngine.Networking;

namespace HypeDomino.Net.Api
{
    public sealed class UnityHttpTransport : IApiHeaderTransport, IAuthTransport
    {
        readonly string baseUrl;
        readonly int timeoutSeconds;

        public UnityHttpTransport(string baseUrl, int timeoutSeconds)
        {
            this.baseUrl = baseUrl;
            this.timeoutSeconds = timeoutSeconds;
        }

        public void Post(string path, string json, Action<HttpReply> done) => Send("POST", path, json, null, null, null, done);

        public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
            Send(method, pathAndQuery, body, authorization, mac, null, done);

        public void Send(string method, string pathAndQuery, string body, string authorization, string mac, IReadOnlyList<ApiHeader> headers, Action<HttpReply> done)
        {
            var plan = HttpExchange.Plan(baseUrl, method, pathAndQuery, body, authorization, mac, headers);
            var request = new UnityWebRequest(plan.Url, plan.Method, new DownloadHandlerBuffer(), null) { timeout = timeoutSeconds };
            if (plan.Body != null) request.uploadHandler = new UploadHandlerRaw(plan.Body);
            foreach (var header in plan.Headers) request.SetRequestHeader(header.Name, header.Value);
            request.SendWebRequest().completed += _ =>
            {
                var reply = HttpExchange.Reply(request.result, request.responseCode, request.downloadHandler.text);
                request.Dispose();
                done(reply);
            };
        }
    }
}
