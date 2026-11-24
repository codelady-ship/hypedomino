using System;
using Hypedomino.V1;

namespace HypeDomino.Net
{
    public readonly ref struct Frame
    {
        public readonly FrameHeader Header;
        public readonly Envelope.PayloadOneofCase Payload;
        public readonly ReadOnlySpan<byte> PayloadBytes;

        public Frame(FrameHeader header, Envelope.PayloadOneofCase payload, ReadOnlySpan<byte> payloadBytes)
        {
            Header = header;
            Payload = payload;
            PayloadBytes = payloadBytes;
        }
    }

    public static class FrameReader
    {
        const int ProtocolVersionField = 1;
        const int SequenceField = 2;
        const int AckedSequenceField = 3;
        const int SessionNonceField = 4;

        public static bool TryRead(ReadOnlySpan<byte> source, out Frame frame)
        {
            frame = default;
            uint protocolVersion = 0;
            ulong sequence = 0, ackedSequence = 0, sessionNonce = 0;
            var payload = Envelope.PayloadOneofCase.None;
            var payloadStart = 0;
            var payloadLength = 0;
            var at = 0;
            while (at < source.Length)
            {
                if (!WireFormat.TryReadVarint(source, ref at, out var tag) || tag > int.MaxValue) return false;
                var field = (int)(tag >> 3);
                var wireType = (int)(tag & 7);
                if (field == 0) return false;
                switch (field)
                {
                    case ProtocolVersionField when wireType == WireFormat.Varint:
                        if (!WireFormat.TryReadVarint(source, ref at, out var version)) return false;
                        protocolVersion = (uint)version;
                        break;
                    case SequenceField when wireType == WireFormat.Varint:
                        if (!WireFormat.TryReadVarint(source, ref at, out sequence)) return false;
                        break;
                    case AckedSequenceField when wireType == WireFormat.Varint:
                        if (!WireFormat.TryReadVarint(source, ref at, out ackedSequence)) return false;
                        break;
                    case SessionNonceField when wireType == WireFormat.Fixed64:
                        if (!WireFormat.TryReadFixed64(source, ref at, out sessionNonce)) return false;
                        break;
                    default:
                        if (IsPayload(field) && wireType == WireFormat.LengthDelimited)
                        {
                            if (!WireFormat.TryReadVarint(source, ref at, out var length) || length > (ulong)(source.Length - at)) return false;
                            payload = (Envelope.PayloadOneofCase)field;
                            payloadStart = at;
                            payloadLength = (int)length;
                            at += payloadLength;
                            break;
                        }
                        if (!WireFormat.TrySkip(source, ref at, wireType)) return false;
                        break;
                }
            }
            frame = new Frame(new FrameHeader(protocolVersion, sequence, ackedSequence, sessionNonce), payload, source.Slice(payloadStart, payloadLength));
            return true;
        }

        static bool IsPayload(int field) =>
            (field >= (int)Envelope.PayloadOneofCase.ClientHello && field <= (int)Envelope.PayloadOneofCase.LobbyInvite) ||
            (field >= (int)Envelope.PayloadOneofCase.PlaceTileIntent && field <= (int)Envelope.PayloadOneofCase.LobbyIntent);
    }
}
