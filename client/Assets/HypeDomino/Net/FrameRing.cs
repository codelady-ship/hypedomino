using System;
using System.Threading;

namespace HypeDomino.Net
{
    sealed class FrameRing
    {
        readonly byte[][] slots;
        readonly int[] lengths;
        int head;
        int tail;

        public FrameRing(int slotCount, int slotBytes)
        {
            slots = new byte[slotCount][];
            for (var i = 0; i < slotCount; i++) slots[i] = new byte[slotBytes];
            lengths = new int[slotCount];
        }

        public int SlotBytes => slots[0].Length;

        public byte[] WriteSlot => slots[Index(tail)];

        public void Commit(int length)
        {
            lengths[Index(tail)] = length;
            Volatile.Write(ref tail, tail + 1);
        }

        public bool TryWrite(ReadOnlySpan<byte> frame)
        {
            if (tail - Volatile.Read(ref head) == slots.Length) return false;
            frame.CopyTo(WriteSlot);
            Commit(frame.Length);
            return true;
        }

        public bool TryPeek(out ReadOnlySpan<byte> frame)
        {
            frame = default;
            if (head == Volatile.Read(ref tail)) return false;
            var index = Index(head);
            frame = new ReadOnlySpan<byte>(slots[index], 0, lengths[index]);
            return true;
        }

        public ArraySegment<byte> PeekSegment()
        {
            var index = Index(head);
            return new ArraySegment<byte>(slots[index], 0, lengths[index]);
        }

        public void Release() => Volatile.Write(ref head, head + 1);

        int Index(int position) => (int)((uint)position % (uint)slots.Length);
    }
}
