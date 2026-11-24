using System;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using UnityEngine;

namespace HypeDomino.Game.Lobby
{
    [Serializable]
    public sealed class AppealRequest
    {
        public long sanction_id;
        public string text;
    }

    public sealed class AppealSender
    {
        public const string Path = "/appeals";

        readonly IApiTransport transport;
        readonly Action<HttpReply> answered;
        Action<AppealResult> done;

        public AppealSender(IApiTransport transport)
        {
            this.transport = transport ?? throw new ArgumentNullException(nameof(transport));
            answered = Answered;
        }

        public bool Busy => done != null;

        public bool Send(SanctionAppeal appeal, string text, Action<AppealResult> done)
        {
            if (done == null) throw new ArgumentNullException(nameof(done));
            if (Busy || !appeal.Offered) return false;
            this.done = done;
            var body = JsonUtility.ToJson(new AppealRequest { sanction_id = appeal.SanctionId, text = text ?? string.Empty });
            transport.Send("POST", Path, body, "Bearer " + appeal.Token, null, answered);
            return true;
        }

        void Answered(HttpReply reply)
        {
            var callback = done;
            done = null;
            callback(AppealResult.Of(reply));
        }
    }
}
