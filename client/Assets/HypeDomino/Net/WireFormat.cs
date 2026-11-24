using System;
using System.Buffers.Binary;

namespace HypeDomino.Net
{
    static class WireFormat
    {
        public const int Varint = 0;
        public const int Fixed64 = 1;
        public const int LengthDelimited = 2;
        public const int Fixed32 = 5;
        public const int MaxVarintBytes = 10;

        public static int Tag(int field, int wireType) => (field << 3) | wireType;

        public static bool TryReadVarint(ReadOnlySpan<byte> source, ref int at, out ulong value)
        {
            value = 0;
            for (var shift = 0; shift < 7 * MaxVarintBytes; shift += 7)
            {
                if (at >= source.Length) return false;
                var current = source[at++];
                value |= (ulong)(current & 0x7F) << shift;
                if (current < 0x80) return true;
            }
            return false;
        }

        public static bool TryReadFixed64(ReadOnlySpan<byte> source, ref int at, out ulong value)
        {
            value = 0;
            if (source.Length - at < sizeof(ulong)) return false;
            value = BinaryPrimitives.ReadUInt64LittleEndian(source.Slice(at));
            at += sizeof(ulong);
            return true;
        }

        public static bool TryReadBytes(ReadOnlySpan<byte> source, ref int at, out ReadOnlySpan<byte> bytes)
        {
            bytes = default;
            if (!TryReadVarint(source, ref at, out var length) || length > (ulong)(source.Length - at)) return false;
            bytes = source.Slice(at, (int)length);
            at += (int)length;
            return true;
        }

        public static bool TrySkip(ReadOnlySpan<byte> source, ref int at, int wireType)
        {
            switch (wireType)
            {
                case Varint:
                    return TryReadVarint(source, ref at, out _);
                case Fixed64:
                    return Advance(source, ref at, sizeof(ulong));
                case Fixed32:
                    return Advance(source, ref at, sizeof(uint));
                case LengthDelimited:
                    return TryReadVarint(source, ref at, out var length) && length <= int.MaxValue && Advance(source, ref at, (int)length);
                default:
                    return false;
            }
        }

        public static bool Advance(ReadOnlySpan<byte> source, ref int at, int count)
        {
            if (source.Length - at < count) return false;
            at += count;
            return true;
        }

        public static void WriteVarint(Span<byte> target, ref int at, ulong value)
        {
            while (value >= 0x80)
            {
                target[at++] = (byte)(value | 0x80);
                value >>= 7;
            }
            target[at++] = (byte)value;
        }

        public static void WriteFixed64(Span<byte> target, ref int at, ulong value)
        {
            BinaryPrimitives.WriteUInt64LittleEndian(target.Slice(at), value);
            at += sizeof(ulong);
        }
    }
}
