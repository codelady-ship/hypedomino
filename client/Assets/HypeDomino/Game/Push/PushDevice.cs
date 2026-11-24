using System;
using HypeDomino.Net.Api;
using UnityEngine;

namespace HypeDomino.Game.Push
{
    [Serializable]
    public sealed class PushDeviceRequest
    {
        public string token;
        public string platform;

        public string ToJson() => JsonUtility.ToJson(this);
    }

    public sealed class PushDevice
    {
        public const string Path = "/push/devices";
        public const string Android = "android";
        public const string Ios = "ios";

        readonly ApiClient api;
        readonly string platform;
        string latest;
        string sending;

        public PushDevice(ApiClient api, string platform)
        {
            if (platform != Android && platform != Ios) throw new ArgumentException("platform must be android or ios", nameof(platform));
            this.api = api;
            this.platform = platform;
        }

        public string Registered { get; private set; }

        public bool Register(string token)
        {
            if (string.IsNullOrEmpty(token) || token == Registered) return false;
            latest = token;
            if (sending == null) Send();
            return true;
        }

        public void Unregister(Action done)
        {
            var token = latest;
            latest = null;
            Registered = null;
            if (token == null)
            {
                done();
                return;
            }
            api.Delete(Path, new PushDeviceRequest { token = token }.ToJson(), _ => done());
        }

        void Send()
        {
            sending = latest;
            api.Put(Path, new PushDeviceRequest { token = sending, platform = platform }.ToJson(), Answer);
        }

        void Answer(ApiReply reply)
        {
            var sent = sending;
            sending = null;
            if (latest == null) return;
            if (reply.Outcome == ApiOutcome.Ok) Registered = sent;
            if (latest != sent && latest != Registered) Send();
        }
    }
}
