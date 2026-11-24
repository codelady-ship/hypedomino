using System;

namespace HypeDomino.Net
{
    public interface ISocketTransport : IFrameTransport, IDisposable
    {
        TransportState State { get; }

        void Connect(Uri uri);
    }
}
