using System;
using HypeDomino.Net.Api;
using Hypedomino.V1;
using UnityEngine;

namespace HypeDomino.Net.Integrity
{
    public sealed class IntegrityProofs
    {
        public const string NoncePath = "/integrity/nonce";

        [Serializable]
        sealed class NonceRequest
        {
            public string purpose;
        }

        [Serializable]
        sealed class NonceReply
        {
            public string nonce;
        }

        readonly ApiClient api;
        readonly IDeviceAttestor attestor;

        public IntegrityProofs(ApiClient api, IDeviceAttestor attestor)
        {
            this.api = api;
            this.attestor = attestor;
        }

        public void Request(IntegrityPurpose purpose, Action<IntegrityProof> done)
        {
            if (!attestor.CanAttest)
            {
                done(null);
                return;
            }
            var body = JsonUtility.ToJson(new NonceRequest { purpose = IntegrityPurposes.WireName(purpose) });
            api.Post(NoncePath, body, reply => Attest(NonceOf(reply), done));
        }

        void Attest(string nonce, Action<IntegrityProof> done)
        {
            if (string.IsNullOrEmpty(nonce))
            {
                done(null);
                return;
            }
            attestor.Attest(nonce, done);
        }

        static string NonceOf(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok || string.IsNullOrEmpty(reply.Body)) return null;
            try
            {
                return JsonUtility.FromJson<NonceReply>(reply.Body)?.nonce;
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
