using System;
using System.Collections.Generic;
using System.IO;
using Google.Protobuf;
using Google.Protobuf.Reflection;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class FrameCodecTests
    {
        const int MaxFrameBytes = 8192;

        static string FramesFolder => Path.GetFullPath(Path.Combine(Application.dataPath, "..", "..", "proto", "testdata", "frames"));

        static IEnumerable<(string Name, byte[] Bytes, Envelope Expected)> GoldenFrames()
        {
            var files = Directory.GetFiles(FramesFolder, "*.bin");
            Assert.IsNotEmpty(files, FramesFolder);
            foreach (var file in files)
            {
                var bytes = File.ReadAllBytes(file);
                yield return (Path.GetFileNameWithoutExtension(file), bytes, Envelope.Parser.ParseFrom(bytes));
            }
        }

        static IMessage PayloadOf(Envelope envelope)
        {
            var field = Envelope.Descriptor.Oneofs[0].Accessor.GetCaseFieldDescriptor(envelope);
            return (IMessage)field.Accessor.GetValue(envelope);
        }

        static FrameHeader HeaderOf(Envelope envelope) =>
            new(envelope.ProtocolVersion, envelope.Sequence, envelope.AckedSequence, envelope.SessionNonce);

        static bool IsClientFrame(Envelope.PayloadOneofCase payload) =>
            payload == Envelope.PayloadOneofCase.ClientHello || payload >= Envelope.PayloadOneofCase.PlaceTileIntent;

        [Test]
        public void EveryGoldenFrameReadsLikeTheGeneratedParser()
        {
            foreach (var (name, bytes, expected) in GoldenFrames())
            {
                Assert.IsTrue(FrameReader.TryRead(bytes, out var frame), name);
                Assert.AreEqual(expected.ProtocolVersion, frame.Header.ProtocolVersion, name);
                Assert.AreEqual(expected.Sequence, frame.Header.Sequence, name);
                Assert.AreEqual(expected.AckedSequence, frame.Header.AckedSequence, name);
                Assert.AreEqual(expected.SessionNonce, frame.Header.SessionNonce, name);
                Assert.AreEqual(expected.PayloadCase, frame.Payload, name);

                var payload = PayloadOf(expected);
                var decoded = (IMessage)Activator.CreateInstance(payload.GetType());
                decoded.MergeFrom(frame.PayloadBytes);
                Assert.AreEqual(payload, decoded, name);
            }
        }

        [Test]
        public void EveryClientGoldenFrameIsWrittenByteForByte()
        {
            var writer = new FrameWriter(MaxFrameBytes);
            var written = 0;
            foreach (var (name, bytes, expected) in GoldenFrames())
            {
                if (!IsClientFrame(expected.PayloadCase)) continue;
                var frame = writer.Write(HeaderOf(expected), expected.PayloadCase, PayloadOf(expected));
                CollectionAssert.AreEqual(bytes, frame.ToArray(), name);
                written++;
            }
            Assert.Greater(written, 0);
        }

        [Test]
        public void AnEmptyIntentIsStillSentAsItsPayload()
        {
            var frame = new FrameWriter(MaxFrameBytes).Write(new FrameHeader(1, 7, 0, 0), Envelope.PayloadOneofCase.PassIntent, new PassIntent());

            Assert.IsTrue(FrameReader.TryRead(frame, out var read));
            Assert.AreEqual(Envelope.PayloadOneofCase.PassIntent, read.Payload);
            Assert.AreEqual(0, read.PayloadBytes.Length);
            Assert.AreEqual(7UL, read.Header.Sequence);
        }

        [Test]
        public void AnUnknownFieldIsSkippedAndAnUnknownPayloadReadsAsNone()
        {
            var frame = new byte[] { 0x08, 0x01, 0x10, 0x05, 0x9A, 0x06, 0x02, 0x08, 0x01, 0x98, 0x07, 0x2A };

            Assert.IsTrue(FrameReader.TryRead(frame, out var read));
            Assert.AreEqual(1U, read.Header.ProtocolVersion);
            Assert.AreEqual(5UL, read.Header.Sequence);
            Assert.AreEqual(Envelope.PayloadOneofCase.None, read.Payload);
        }

        [TestCase(new byte[] { 0x10 }, TestName = "a field without its value")]
        [TestCase(new byte[] { 0x10, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01 }, TestName = "a varint longer than ten bytes")]
        [TestCase(new byte[] { 0x6A, 0x05, 0x08 }, TestName = "a payload longer than the frame")]
        [TestCase(new byte[] { 0x21, 0x01, 0x02 }, TestName = "a short session nonce")]
        [TestCase(new byte[] { 0x0B }, TestName = "a group wire type")]
        [TestCase(new byte[] { 0x00 }, TestName = "field number zero")]
        public void AMalformedFrameIsRefused(byte[] frame) => Assert.IsFalse(FrameReader.TryRead(frame, out _));

        [Test]
        public void AFrameThatDoesNotFitTheBufferIsRefused()
        {
            var writer = new FrameWriter(64);
            var intent = new EmojiIntent { ItemId = new string('e', 64) };

            Assert.Throws<ArgumentException>(() => writer.Write(new FrameHeader(1, 1, 0, 0), Envelope.PayloadOneofCase.EmojiIntent, intent));
        }

        [Test]
        public void ReadingAMatchViewFrameAllocatesNothing()
        {
            var bytes = File.ReadAllBytes(Path.Combine(FramesFolder, "match_view.bin"));
            FrameReader.TryRead(bytes, out _);

            TestDelegate step = () =>
            {
                FrameReader.TryRead(bytes, out var frame);
                if (frame.Payload != Envelope.PayloadOneofCase.MatchView) throw new InvalidOperationException();
            };
            step();

            Assert.That(step, Allocates.Nothing);
        }

        [Test]
        public void WritingAReusedIntentAllocatesNothing()
        {
            var writer = new FrameWriter(MaxFrameBytes);
            var intent = new PlaceTileIntent { Tile = new Tile { Low = 3, High = 5 }, End = BoardEnd.Left };
            writer.Write(new FrameHeader(1, 1, 0, 42), Envelope.PayloadOneofCase.PlaceTileIntent, intent);

            TestDelegate step = () =>
            {
                intent.Tile.High = 6;
                writer.Write(new FrameHeader(1, 2, 0, 42), Envelope.PayloadOneofCase.PlaceTileIntent, intent);
            };
            step();

            Assert.That(step, Allocates.Nothing);
        }
    }
}
