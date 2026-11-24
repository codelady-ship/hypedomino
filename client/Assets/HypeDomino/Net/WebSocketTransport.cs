using System;
using System.Net.WebSockets;
using System.Threading;

namespace HypeDomino.Net
{
    public enum TransportState
    {
        Idle,
        Connecting,
        Open,
        Closed,
    }

    public sealed class WebSocketTransport : ISocketTransport
    {
        public const int MaxReceivedFrameBytes = 16 * 1024;
        public const int ReceiveSlots = 8;
        public const int SendSlots = 8;

        readonly FrameRing received = new(ReceiveSlots, MaxReceivedFrameBytes);
        readonly FrameRing outgoing = new(SendSlots, Session.MaxSentFrameBytes);
        readonly SemaphoreSlim freeReceiveSlots = new(ReceiveSlots, ReceiveSlots);
        readonly SemaphoreSlim queuedSends = new(0, SendSlots);
        readonly CancellationTokenSource stop = new();
        ClientWebSocket socket;
        bool holdingReceived;
        int state;
        int failed;

        public TransportState State => (TransportState)Volatile.Read(ref state);
        public WebSocketCloseStatus CloseStatus { get; private set; }
        public string CloseDescription { get; private set; } = string.Empty;

        public void Connect(Uri uri)
        {
            if (Interlocked.CompareExchange(ref state, (int)TransportState.Connecting, (int)TransportState.Idle) != (int)TransportState.Idle)
                throw new InvalidOperationException("a transport connects once");
            socket = new ClientWebSocket();
            StartThread(() => ReceiveLoop(uri), "HypeDomino.Net.Receive");
        }

        public void Send(ReadOnlySpan<byte> frame)
        {
            if (frame.Length > outgoing.SlotBytes) throw new ArgumentException($"a frame of {frame.Length} bytes does not fit {outgoing.SlotBytes}", nameof(frame));
            if (State == TransportState.Closed) return;
            if (!outgoing.TryWrite(frame))
            {
                Fail(WebSocketCloseStatus.PolicyViolation, "send queue full");
                return;
            }
            queuedSends.Release();
        }

        public bool TryReceive(out ReadOnlySpan<byte> frame)
        {
            if (holdingReceived)
            {
                received.Release();
                freeReceiveSlots.Release();
            }
            holdingReceived = received.TryPeek(out frame);
            return holdingReceived;
        }

        public void Dispose() => Fail(WebSocketCloseStatus.NormalClosure, string.Empty);

        static void StartThread(ThreadStart loop, string name) => new Thread(loop) { IsBackground = true, Name = name }.Start();

        void ReceiveLoop(Uri uri)
        {
            try
            {
                socket.ConnectAsync(uri, stop.Token).GetAwaiter().GetResult();
                if (Interlocked.CompareExchange(ref state, (int)TransportState.Open, (int)TransportState.Connecting) != (int)TransportState.Connecting) return;
                StartThread(SendLoop, "HypeDomino.Net.Send");
                while (true)
                {
                    freeReceiveSlots.Wait(stop.Token);
                    if (!ReceiveFrame()) return;
                }
            }
            catch (Exception exception)
            {
                Fail(WebSocketCloseStatus.EndpointUnavailable, exception.Message);
            }
        }

        bool ReceiveFrame()
        {
            var buffer = received.WriteSlot;
            var length = 0;
            while (true)
            {
                if (length == buffer.Length)
                {
                    Fail(WebSocketCloseStatus.MessageTooBig, "frame too big");
                    return false;
                }
                var result = socket.ReceiveAsync(new ArraySegment<byte>(buffer, length, buffer.Length - length), stop.Token).GetAwaiter().GetResult();
                if (result.MessageType == WebSocketMessageType.Close)
                {
                    Fail(result.CloseStatus ?? WebSocketCloseStatus.Empty, result.CloseStatusDescription ?? string.Empty);
                    return false;
                }
                if (result.MessageType != WebSocketMessageType.Binary)
                {
                    Fail(WebSocketCloseStatus.InvalidMessageType, "binary frames only");
                    return false;
                }
                length += result.Count;
                if (result.EndOfMessage) break;
            }
            received.Commit(length);
            return true;
        }

        void SendLoop()
        {
            try
            {
                while (true)
                {
                    queuedSends.Wait(stop.Token);
                    socket.SendAsync(outgoing.PeekSegment(), WebSocketMessageType.Binary, true, stop.Token).GetAwaiter().GetResult();
                    outgoing.Release();
                }
            }
            catch (Exception exception)
            {
                Fail(WebSocketCloseStatus.EndpointUnavailable, exception.Message);
            }
        }

        void Fail(WebSocketCloseStatus status, string description)
        {
            if (Interlocked.Exchange(ref failed, 1) == 1) return;
            CloseStatus = status;
            CloseDescription = description;
            Volatile.Write(ref state, (int)TransportState.Closed);
            stop.Cancel();
            socket?.Abort();
        }
    }
}
