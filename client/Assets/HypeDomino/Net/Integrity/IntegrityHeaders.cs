using HypeDomino.Net.Api;
using Hypedomino.V1;

namespace HypeDomino.Net.Integrity
{
    public static class IntegrityHeaders
    {
        public const string PlayIntegrity = "X-HD-Play-Integrity";
        public const string AttestKey = "X-HD-Attest-Key";
        public const string AttestNonce = "X-HD-Attest-Nonce";
        public const string AttestAssertion = "X-HD-Attest-Assertion";

        public static ApiHeader[] Of(IntegrityProof proof)
        {
            if (proof == null) return null;
            if (proof.ProofCase == IntegrityProof.ProofOneofCase.PlayIntegrityToken) return OfPlay(proof.PlayIntegrityToken);
            if (proof.ProofCase == IntegrityProof.ProofOneofCase.AppAttest) return OfAppAttest(proof.AppAttest);
            return null;
        }

        static ApiHeader[] OfPlay(string token) => string.IsNullOrEmpty(token) ? null : new[] { new ApiHeader(PlayIntegrity, token) };

        static ApiHeader[] OfAppAttest(AppAttestAssertion assertion) => new[]
        {
            new ApiHeader(AttestKey, assertion.KeyId.ToBase64()),
            new ApiHeader(AttestNonce, assertion.Nonce),
            new ApiHeader(AttestAssertion, assertion.Assertion.ToBase64()),
        };
    }
}
