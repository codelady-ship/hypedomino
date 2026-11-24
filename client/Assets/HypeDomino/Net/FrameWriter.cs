using System;
using Google.Protobuf;
using Hypedomino.V1;

namespace HypeDomino.Net
{
    public sealed class FrameWriter
    {
        const int ProtocolVersionTag = (1 << 3) | WireFormat.Varint;
        const int SequenceTag = (2 << 3) | WireFormat.Varint;
        const int AckedSequenceTag = (3 << 3) | WireFormat.Varint;
        const int SessionNonceTag = (4 << 3) | WireFormat.Fixed64;
        const int HeaderBytes = 4 * (1 + WireFormat.MaxVarintBytes);
        const int PayloadPrefixBytes = 2 + 5;

        readonly byte[] buffer;

        public FrameWriter(int maxFrameBytes) => buffer = new byte[maxFrameBytes];

        public ReadOnlySpan<byte> Write(in FrameHeader header, Envelope.PayloadOneofCase payload, IMessage message)
        {
            if (payload == Envelope.PayloadOneofCase.None) throw new ArgumentException("a frame needs a payload", nameof(payload));
            var size = message.CalculateSize();
            if (HeaderBytes + PayloadPrefixBytes + size > buffer.Length) throw new ArgumentException($"{payload} of {size} bytes does not fit {buffer.Length}", nameof(message));

            var target = buffer.AsSpan();
            var at = 0;
            if (header.ProtocolVersion != 0) WriteVarintField(target, ref at, ProtocolVersionTag, header.ProtocolVersion);
            if (header.Sequence != 0) WriteVarintField(target, ref at, SequenceTag, header.Sequence);
            if (header.AckedSequence != 0) WriteVarintField(target, ref at, AckedSequenceTag, header.AckedSequence);
            if (header.SessionNonce != 0)
            {
                WireFormat.WriteVarint(target, ref at, SessionNonceTag);
                WireFormat.WriteFixed64(target, ref at, header.SessionNonce);
            }
            WireFormat.WriteVarint(target, ref at, (ulong)WireFormat.Tag((int)payload, WireFormat.LengthDelimited));
            WireFormat.WriteVarint(target, ref at, (ulong)size);
            message.WriteTo(target.Slice(at, size));
            return target.Slice(0, at + size);
        }

        static void WriteVarintField(Span<byte> target, ref int at, int tag, ulong value)
        {
            WireFormat.WriteVarint(target, ref at, (ulong)tag);
            WireFormat.WriteVarint(target, ref at, value);
        }
    }
}
