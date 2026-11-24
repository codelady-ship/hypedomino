using System;
using System.Collections.Generic;
using HypeDomino.Net.Auth;
using UnityEngine;

namespace HypeDomino.Net.Api
{
    public enum ApiOutcome
    {
        Ok,
        Offline,
        SignInRequired,
        Rejected,
        RateLimited,
        Unavailable,
    }

    public readonly struct ApiReply
    {
        public readonly ApiOutcome Outcome;
        public readonly int Status;
        public readonly string Body;
        public readonly string Error;

        public ApiReply(ApiOutcome outcome, int status = HttpReply.NoResponse, string body = null, string error = null)
        {
            Outcome = outcome;
            Status = status;
            Body = body;
            Error = error;
        }
    }

    public sealed class ApiClient
    {
        public const string AccessExpired = "access_expired";
        public const string AccessRejected = "access_rejected";

        sealed class Request
        {
            public string Method;
            public string PathAndQuery;
            public string Body;
            public IReadOnlyList<ApiHeader> Headers;
            public Action<ApiReply> Done;
        }

        [Serializable]
        sealed class ErrorBody
        {
            public string error;
        }

        readonly IApiTransport transport;
        readonly AuthSession session;
        readonly List<Request> waitingForRefresh = new();
        bool refreshing;

        public ApiClient(IApiTransport transport, AuthSession session)
        {
            this.transport = transport;
            this.session = session;
        }

        public void Get(string pathAndQuery, Action<ApiReply> done) => Send("GET", pathAndQuery, null, null, done);

        public void Post(string pathAndQuery, string json, Action<ApiReply> done) => Send("POST", pathAndQuery, json, null, done);

        public void Post(string pathAndQuery, string json, IReadOnlyList<ApiHeader> headers, Action<ApiReply> done) => Send("POST", pathAndQuery, json, headers, done);

        public void Put(string pathAndQuery, string json, Action<ApiReply> done) => Send("PUT", pathAndQuery, json, null, done);

        public void Delete(string pathAndQuery, string json, Action<ApiReply> done) => Send("DELETE", pathAndQuery, json, null, done);

        void Send(string method, string pathAndQuery, string body, IReadOnlyList<ApiHeader> headers, Action<ApiReply> done)
        {
            var request = new Request { Method = method, PathAndQuery = pathAndQuery, Body = body, Headers = headers, Done = done };
            if (refreshing)
            {
                waitingForRefresh.Add(request);
                return;
            }
            Attempt(request, false);
        }

        void Attempt(Request request, bool afterRefresh)
        {
            if (!session.SignedIn)
            {
                request.Done(new ApiReply(ApiOutcome.SignInRequired));
                return;
            }
            var access = session.AccessToken;
            var mac = request.Method == "GET" ? null : RequestMac.Sign(session.RequestKey, request.Method, request.PathAndQuery, access, request.Body ?? string.Empty);
            Action<HttpReply> answered = reply => Answer(request, reply, afterRefresh);
            if (request.Headers != null && request.Headers.Count > 0 && transport is IApiHeaderTransport withHeaders)
            {
                withHeaders.Send(request.Method, request.PathAndQuery, request.Body, "Bearer " + access, mac, request.Headers, answered);
                return;
            }
            transport.Send(request.Method, request.PathAndQuery, request.Body, "Bearer " + access, mac, answered);
        }

        void Answer(Request request, HttpReply reply, bool afterRefresh)
        {
            var error = ErrorOf(reply);
            if (reply.Status == 401 && error == AccessExpired && !afterRefresh)
            {
                Refresh(request);
                return;
            }
            request.Done(new ApiReply(OutcomeOf(reply.Status, error), reply.Status, reply.Body, error));
        }

        void Refresh(Request request)
        {
            waitingForRefresh.Add(request);
            if (refreshing) return;
            refreshing = true;
            session.Resume(Refreshed);
        }

        void Refreshed(AuthResult result)
        {
            refreshing = false;
            var waiting = waitingForRefresh.ToArray();
            waitingForRefresh.Clear();
            foreach (var request in waiting)
            {
                if (result.Outcome == AuthOutcome.SignedIn) Attempt(request, true);
                else request.Done(new ApiReply(OutcomeOf(result.Outcome), error: result.Error));
            }
        }

        static ApiOutcome OutcomeOf(int status, string error)
        {
            if (status == HttpReply.NoResponse) return ApiOutcome.Offline;
            if (status >= 200 && status < 300) return ApiOutcome.Ok;
            if (status == 401 && (error == AccessExpired || error == AccessRejected)) return ApiOutcome.SignInRequired;
            if (status == 429) return ApiOutcome.RateLimited;
            return status >= 500 ? ApiOutcome.Unavailable : ApiOutcome.Rejected;
        }

        static ApiOutcome OutcomeOf(AuthOutcome outcome) => outcome switch
        {
            AuthOutcome.Offline => ApiOutcome.Offline,
            AuthOutcome.RateLimited => ApiOutcome.RateLimited,
            AuthOutcome.Unavailable or AuthOutcome.Busy => ApiOutcome.Unavailable,
            _ => ApiOutcome.SignInRequired,
        };

        static string ErrorOf(HttpReply reply)
        {
            if (reply.Status < 400 || string.IsNullOrEmpty(reply.Body)) return null;
            try
            {
                return JsonUtility.FromJson<ErrorBody>(reply.Body)?.error;
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
