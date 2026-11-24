using System;
using Google.Protobuf;
using Hypedomino.V1;

namespace HypeDomino.Net
{
    public enum ConnectionState
    {
        Offline,
        Connecting,
        Online,
        Waiting,
        Stopped,
    }

    public sealed class Connection
    {
        readonly Func<ISocketTransport> openTransport;
        readonly Func<ClientHello> nextHello;
        readonly Func<double> secondsNow;
        readonly Random random;
        readonly ReconnectSchedule schedule = new();
        ISocketTransport transport;
        Uri uri;
        double helloSentSeconds;
        double retryAtSeconds;

        public Connection(Func<ISocketTransport> openTransport, Func<ClientHello> nextHello, Func<double> secondsNow, Random random)
        {
            this.openTransport = openTransport;
            this.nextHello = nextHello;
            this.secondsNow = secondsNow;
            this.random = random;
            Session.HelloReceived += OnHello;
            Session.ErrorReceived += OnError;
        }

        public Session Session { get; } = new();
        public ServerClock Clock { get; } = new();
        public RoundTrip RoundTrip { get; } = new();
        public ConnectionState State { get; private set; } = ConnectionState.Offline;
        public int FailedAttempts { get; private set; }
        public bool WasOnline { get; private set; }
        public double RetryAtSeconds => retryAtSeconds;

        public event Action<ConnectionState> Changed;

        public void Start(Uri serverUri)
        {
            if (State != ConnectionState.Offline && State != ConnectionState.Stopped) throw new InvalidOperationException($"already {State}");
            uri = serverUri;
            schedule.Reset();
            FailedAttempts = 0;
            WasOnline = false;
            Dial();
        }

        public void Stop()
        {
            CloseTransport();
            Enter(ConnectionState.Stopped);
        }

        public bool Rejoin()
        {
            if (State != ConnectionState.Online) return false;
            CloseTransport();
            Dial();
            return true;
        }

        public ulong Send(Envelope.PayloadOneofCase intent, IMessage message)
        {
            var sequence = Session.Send(intent, message);
            if (sequence != 0) RoundTrip.Sent(sequence, secondsNow());
            return sequence;
        }

        public void Tick()
        {
            switch (State)
            {
                case ConnectionState.Waiting:
                    if (secondsNow() >= retryAtSeconds) Dial();
                    return;
                case ConnectionState.Connecting:
                case ConnectionState.Online:
                    Read();
                    return;
            }
        }

        void Read()
        {
            var closed = transport.State == TransportState.Closed;
            Session.Poll();
            if (State == ConnectionState.Stopped) return;
            RoundTrip.Acked(Session.AckedSequence, secondsNow());
            if (closed || Session.Broken) Retry();
        }

        void Dial()
        {
            transport = openTransport();
            transport.Connect(uri);
            RoundTrip.ForgetPending();
            helloSentSeconds = secondsNow();
            Session.Hello(transport, nextHello());
            Enter(ConnectionState.Connecting);
        }

        void Retry()
        {
            CloseTransport();
            if (State == ConnectionState.Connecting) FailedAttempts++;
            retryAtSeconds = secondsNow() + schedule.NextDelaySeconds(random.NextDouble());
            Enter(ConnectionState.Waiting);
        }

        void OnHello(ServerHello hello)
        {
            if (hello.Status != HelloStatus.Accepted)
            {
                Stop();
                return;
            }
            Clock.Sync(hello.ServerTimeUnixMs, helloSentSeconds, secondsNow());
            schedule.Reset();
            FailedAttempts = 0;
            WasOnline = true;
            Enter(ConnectionState.Online);
        }

        void OnError(Error error)
        {
            switch (error.Code)
            {
                case ErrorCode.Unauthenticated:
                case ErrorCode.IntegrityFailed:
                case ErrorCode.UnsupportedProtocol:
                    Stop();
                    return;
                default:
                    RoundTrip.Rejected(error.RejectedSequence);
                    return;
            }
        }

        void CloseTransport()
        {
            transport?.Dispose();
            transport = null;
        }

        void Enter(ConnectionState next)
        {
            if (next == State) return;
            State = next;
            Changed?.Invoke(next);
        }
    }
}
