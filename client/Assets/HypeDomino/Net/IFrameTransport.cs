using System;

namespace HypeDomino.Net
{
    public interface IFrameTransport
    {
        void Send(ReadOnlySpan<byte> frame);

        bool TryReceive(out ReadOnlySpan<byte> frame);
    }
}
