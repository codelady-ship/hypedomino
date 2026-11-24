using System;
using UnityEngine;
using UnityEngine.Networking;

namespace HypeDomino.Core.Diagnostics
{
    public sealed class UnityEnvelopeTransport : IEnvelopeTransport
    {
        public const string ContentType = "application/x-sentry-envelope";

        public void Post(string url, string authHeader, byte[] envelope, Action<EnvelopeReply> done)
        {
            var request = new UnityWebRequest(url, UnityWebRequest.kHttpVerbPOST, new DownloadHandlerBuffer(), new UploadHandlerRaw(envelope));
            request.SetRequestHeader("Content-Type", ContentType);
            request.SetRequestHeader("X-Sentry-Auth", authHeader);
            request.SendWebRequest().completed += new PendingPost(request, done).Completed;
        }

        sealed class PendingPost
        {
            readonly UnityWebRequest request;
            readonly Action<EnvelopeReply> done;

            public PendingPost(UnityWebRequest request, Action<EnvelopeReply> done)
            {
                this.request = request;
                this.done = done;
            }

            public void Completed(AsyncOperation operation)
            {
                var answered = request.result is UnityWebRequest.Result.Success or UnityWebRequest.Result.ProtocolError;
                var reply = answered
                    ? new EnvelopeReply((int)request.responseCode, request.GetResponseHeader("Retry-After"), request.GetResponseHeader("X-Sentry-Rate-Limits"))
                    : new EnvelopeReply(EnvelopeReply.NoResponse, null, null);
                request.Dispose();
                done(reply);
            }
        }
    }
}
