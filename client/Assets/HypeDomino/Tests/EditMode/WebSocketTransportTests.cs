using System;
using System.Diagnostics;
using System.IO;
using System.Net;
using System.Net.Sockets;
using System.Net.WebSockets;
using System.Security.Cryptography;
using System.Text;
using System.Text.RegularExpressions;
using System.Threading;
using HypeDomino.Net;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class WebSocketTransportTests
    {
        const int TimeoutMs = 5000;
        const int Binary = 2;
        const int Text = 1;
        const int Close = 8;

        sealed class LoopbackServer : IDisposable
        {
            readonly TcpListener listener = new(IPAddress.Loopback, 0);
            TcpClient client;
            NetworkStream stream;

            public LoopbackServer()
            {
                listener.Start();
                Uri = new Uri($"ws://127.0.0.1:{((IPEndPoint)listener.LocalEndpoint).Port}/ws");
            }

            public Uri Uri { get; }

            public void Accept()
            {
                var pending = listener.AcceptTcpClientAsync();
                Assert.IsTrue(pending.Wait(TimeoutMs), "no connection");
                client = pending.Result;
                stream = client.GetStream();
                stream.ReadTimeout = TimeoutMs;
                var key = Regex.Match(ReadRequest(), @"Sec-WebSocket-Key:\s*(\S+)", RegexOptions.IgnoreCase).Groups[1].Value;
                using var sha1 = SHA1.Create();
                var accept = Convert.ToBase64String(sha1.ComputeHash(Encoding.ASCII.GetBytes(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11")));
                var response = Encoding.ASCII.GetBytes($"HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: {accept}\r\n\r\n");
                stream.Write(response, 0, response.Length);
            }

            public byte[] ReadFrame()
            {
                var first = ReadExactly(2);
                Assert.AreEqual(0x80 | Binary, first[0], "a final binary frame");
                var length = (long)(first[1] & 0x7F);
                if (length == 126) length = (ReadExactly(1)[0] << 8) | ReadExactly(1)[0];
                if (length == 127) throw new InvalidDataException("frame too long for a test");
                var mask = (first[1] & 0x80) != 0 ? ReadExactly(4) : new byte[4];
                var payload = ReadExactly((int)length);
                for (var i = 0; i < payload.Length; i++) payload[i] ^= mask[i % 4];
                return payload;
            }

            public void WriteFrame(byte[] payload, int opcode = Binary)
            {
                var header = payload.Length < 126
                    ? new[] { (byte)(0x80 | opcode), (byte)payload.Length }
                    : new[] { (byte)(0x80 | opcode), (byte)126, (byte)(payload.Length >> 8), (byte)payload.Length };
                stream.Write(header, 0, header.Length);
                stream.Write(payload, 0, payload.Length);
            }

            public void WriteClose(WebSocketCloseStatus status, string reason)
            {
                var text = Encoding.UTF8.GetBytes(reason);
                var payload = new byte[2 + text.Length];
                payload[0] = (byte)((int)status >> 8);
                payload[1] = (byte)status;
                text.CopyTo(payload, 2);
                WriteFrame(payload, Close);
            }

            public void Dispose()
            {
                client?.Close();
                listener.Stop();
            }

            string ReadRequest()
            {
                var request = new StringBuilder();
                while (!request.ToString().EndsWith("\r\n\r\n")) request.Append((char)ReadExactly(1)[0]);
                return request.ToString();
            }

            byte[] ReadExactly(int count)
            {
                var bytes = new byte[count];
                for (var read = 0; read < count;)
                {
                    var chunk = stream.Read(bytes, read, count - read);
                    if (chunk == 0) throw new EndOfStreamException();
                    read += chunk;
                }
                return bytes;
            }
        }

        LoopbackServer server;
        WebSocketTransport transport;

        [SetUp]
        public void StartServer()
        {
            server = new LoopbackServer();
            transport = new WebSocketTransport();
        }

        [TearDown]
        public void StopServer()
        {
            transport.Dispose();
            server.Dispose();
        }

        static byte[] Golden(string name) =>
            File.ReadAllBytes(Path.GetFullPath(Path.Combine(Application.dataPath, "..", "..", "proto", "testdata", "frames", name + ".bin")));

        static void WaitUntil(Func<bool> condition, string what)
        {
            var clock = Stopwatch.StartNew();
            while (!condition())
            {
                if (clock.ElapsedMilliseconds > TimeoutMs) Assert.Fail($"timed out waiting for {what}");
                Thread.Sleep(5);
            }
        }

        byte[] NextFrame()
        {
            var clock = Stopwatch.StartNew();
            while (clock.ElapsedMilliseconds < TimeoutMs)
            {
                if (transport.TryReceive(out var frame)) return frame.ToArray();
                Thread.Sleep(5);
            }
            Assert.Fail("timed out waiting for a frame");
            return null;
        }

        void Open()
        {
            transport.Connect(server.Uri);
            server.Accept();
            WaitUntil(() => transport.State == TransportState.Open, "the socket to open");
        }

        [Test]
        public void AHelloSentWhileConnectingGoesOutAndTheAnswerComesBack()
        {
            var hello = Golden("client_hello");
            var answer = Golden("server_hello_accepted");

            transport.Connect(server.Uri);
            transport.Send(hello);
            server.Accept();

            CollectionAssert.AreEqual(hello, server.ReadFrame());
            server.WriteFrame(answer);
            CollectionAssert.AreEqual(answer, NextFrame());
        }

        [Test]
        public void FramesArriveInOrderPastEveryReceiveSlot()
        {
            Open();
            const int count = WebSocketTransport.ReceiveSlots * 3;
            for (var i = 0; i < count; i++) server.WriteFrame(new[] { (byte)i, (byte)(i * 7) });

            for (var i = 0; i < count; i++) CollectionAssert.AreEqual(new[] { (byte)i, (byte)(i * 7) }, NextFrame(), $"frame {i}");
        }

        [Test]
        public void FramesSentInARowArriveInOrder()
        {
            Open();
            const int count = WebSocketTransport.SendSlots;
            for (var i = 0; i < count; i++) transport.Send(new[] { (byte)i });

            for (var i = 0; i < count; i++) CollectionAssert.AreEqual(new[] { (byte)i }, server.ReadFrame(), $"frame {i}");
        }

        [Test]
        public void AFrameBiggerThanASlotClosesTheTransport()
        {
            Open();
            server.WriteFrame(new byte[WebSocketTransport.MaxReceivedFrameBytes + 1]);

            WaitUntil(() => transport.State == TransportState.Closed, "the close");
            Assert.AreEqual(WebSocketCloseStatus.MessageTooBig, transport.CloseStatus);
        }

        [Test]
        public void ATextFrameClosesTheTransport()
        {
            Open();
            server.WriteFrame(Encoding.UTF8.GetBytes("hello"), Text);

            WaitUntil(() => transport.State == TransportState.Closed, "the close");
            Assert.AreEqual(WebSocketCloseStatus.InvalidMessageType, transport.CloseStatus);
        }

        [Test]
        public void TheServersCloseIsReportedAfterTheFramesBeforeIt()
        {
            Open();
            var answer = Golden("server_hello_update_required");
            server.WriteFrame(answer);
            server.WriteClose(WebSocketCloseStatus.PolicyViolation, "session replaced");

            WaitUntil(() => transport.State == TransportState.Closed, "the close");
            CollectionAssert.AreEqual(answer, NextFrame());
            Assert.AreEqual(WebSocketCloseStatus.PolicyViolation, transport.CloseStatus);
            Assert.AreEqual("session replaced", transport.CloseDescription);
        }

        [Test]
        public void ARefusedConnectionClosesTheTransport()
        {
            var unused = server.Uri;
            server.Dispose();

            transport.Connect(unused);

            WaitUntil(() => transport.State == TransportState.Closed, "the close");
            Assert.AreEqual(WebSocketCloseStatus.EndpointUnavailable, transport.CloseStatus);
        }

        [Test]
        public void ADisposedTransportIsClosedAndDropsSends()
        {
            Open();

            transport.Dispose();
            transport.Send(new byte[] { 1 });

            Assert.AreEqual(TransportState.Closed, transport.State);
            Assert.AreEqual(WebSocketCloseStatus.NormalClosure, transport.CloseStatus);
        }

        [Test]
        public void ATransportConnectsOnce()
        {
            Open();

            Assert.Throws<InvalidOperationException>(() => transport.Connect(server.Uri));
        }

        [Test]
        public void SendingAndReceivingAllocateNothingOnTheCallingThread()
        {
            Open();
            var frame = Golden("place_tile_intent");
            TestDelegate step = () =>
            {
                transport.Send(frame);
                transport.TryReceive(out _);
            };
            step();
            server.ReadFrame();

            Assert.That(step, Allocates.Nothing);
            CollectionAssert.AreEqual(frame, server.ReadFrame());
        }
    }
}
