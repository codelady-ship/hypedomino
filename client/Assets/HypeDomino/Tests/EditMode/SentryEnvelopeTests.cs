using System;
using HypeDomino.Core.Diagnostics;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SentryEnvelopeTests
    {
        const string EventId = "0123456789abcdef0123456789abcdef";
        static readonly DateTime At = new DateTime(2026, 11, 21, 9, 30, 15, 250, DateTimeKind.Utc);

        static readonly string ExceptionTrace =
            "HypeDomino.Game.Match.MatchView.Play (System.Int32 tile) (at Assets/HypeDomino/Game/Match/MatchView.cs:42)\n" +
            "HypeDomino.UI.TileButton.OnClick () (at Assets/HypeDomino/UI/TileButton.cs:7)\n" +
            "UnityEngine.Events.InvokableCall.Invoke () (at <00000000000000000000000000000000>:0)\n";

        static SentryEnvelope Envelope(string environment = "staging") => new SentryEnvelope("1.4.2", environment, "Android OS 14 / API-34", "samsung SM-A366B");

        static string[] Lines(string envelope) => envelope.Split('\n');

        [Test]
        public void TheEnvelopeHasAHeaderAnEventItemHeaderAndTheEvent()
        {
            var lines = Lines(Envelope().Build(new ErrorReport(LogType.Error, "boom", ""), EventId, At));

            Assert.AreEqual(4, lines.Length);
            Assert.AreEqual("{\"event_id\":\"" + EventId + "\",\"sent_at\":\"2026-11-21T09:30:15.250Z\"}", lines[0]);
            Assert.AreEqual("{\"type\":\"event\"}", lines[1]);
            Assert.AreEqual(string.Empty, lines[3]);
            var item = lines[2];
            StringAssert.Contains("\"event_id\":\"" + EventId + "\"", item);
            StringAssert.Contains("\"timestamp\":\"2026-11-21T09:30:15.250Z\"", item);
            StringAssert.Contains("\"platform\":\"csharp\"", item);
            StringAssert.Contains("\"level\":\"error\"", item);
            StringAssert.Contains("\"release\":\"1.4.2\"", item);
            StringAssert.Contains("\"environment\":\"staging\"", item);
            StringAssert.Contains("\"os\":{\"raw_description\":\"Android OS 14 / API-34\"}", item);
            StringAssert.Contains("\"device\":{\"model\":\"samsung SM-A366B\"}", item);
            StringAssert.Contains("\"tags\":{\"log_type\":\"Error\"}", item);
        }

        [Test]
        public void TheEventCarriesNoUserAndTellsSentryNotToInferTheIp()
        {
            var item = Lines(Envelope().Build(new ErrorReport(LogType.Exception, "boom", ExceptionTrace), EventId, At))[2];

            StringAssert.DoesNotContain("\"user\"", item);
            StringAssert.DoesNotContain("ip_address", item);
            StringAssert.DoesNotContain("server_name", item);
            StringAssert.Contains("\"settings\":{\"infer_ip\":\"never\"}", item);
        }

        [Test]
        public void AnEmptyEnvironmentIsLeftOut()
        {
            var item = Lines(Envelope(string.Empty).Build(new ErrorReport(LogType.Error, "boom", ""), EventId, At))[2];

            StringAssert.DoesNotContain("\"environment\"", item);
        }

        [Test]
        public void AnExceptionIsSplitIntoTypeAndValue()
        {
            var item = Lines(Envelope().Build(new ErrorReport(LogType.Exception, "NullReferenceException: Object reference not set", ExceptionTrace), EventId, At))[2];

            StringAssert.Contains("\"type\":\"NullReferenceException\",\"value\":\"Object reference not set\"", item);
        }

        [TestCase(LogType.Error, "Error")]
        [TestCase(LogType.Assert, "Assert")]
        public void ALoggedErrorKeepsItsMessageUnderItsLogType(LogType type, string name)
        {
            var item = Lines(Envelope().Build(new ErrorReport(type, "Content: manifest failed", ""), EventId, At))[2];

            StringAssert.Contains("\"type\":\"" + name + "\",\"value\":\"Content: manifest failed\"", item);
        }

        [Test]
        public void FramesRunFromTheOldestCallToTheThrowingOneAndMarkOurCodeInApp()
        {
            var item = Lines(Envelope().Build(new ErrorReport(LogType.Exception, "boom", ExceptionTrace), EventId, At))[2];

            var frames = "\"frames\":[" +
                "{\"function\":\"UnityEngine.Events.InvokableCall.Invoke\",\"in_app\":false}," +
                "{\"function\":\"HypeDomino.UI.TileButton.OnClick\",\"in_app\":true,\"filename\":\"Assets/HypeDomino/UI/TileButton.cs\",\"lineno\":7}," +
                "{\"function\":\"HypeDomino.Game.Match.MatchView.Play\",\"in_app\":true,\"filename\":\"Assets/HypeDomino/Game/Match/MatchView.cs\",\"lineno\":42}" +
                "]";
            StringAssert.Contains(frames, item);
        }

        [Test]
        public void ALoggedErrorFrameWithoutALocationKeepsItsFunction()
        {
            var item = Lines(Envelope().Build(new ErrorReport(LogType.Error, "boom", "UnityEngine.Debug:LogError (object)\nHypeDomino.Game.Shop.ShopLoader:Fail ()\n"), EventId, At))[2];

            StringAssert.Contains("\"frames\":[{\"function\":\"HypeDomino.Game.Shop.ShopLoader:Fail\",\"in_app\":true},{\"function\":\"UnityEngine.Debug:LogError\",\"in_app\":false}]", item);
        }

        [Test]
        public void TextIsEscapedSoEachItemStaysOnOneLine()
        {
            var lines = Lines(Envelope().Build(new ErrorReport(LogType.Error, "say \"hi\"\n\tC:\\path\u0001", ""), EventId, At));

            Assert.AreEqual(4, lines.Length);
            StringAssert.Contains("\"value\":\"say \\\"hi\\\"\\n\\tC:\\\\path\\u0001\"", lines[2]);
        }

        [Test]
        public void TheMessageAndStackTraceAreScrubbed()
        {
            var report = new ErrorReport(LogType.Exception,
                "InvalidOperationException: account 3f2504e0-4f89-11d3-9a0c-0305e82c3301 mail a.b@example.com",
                "HypeDomino.Net.Auth.AuthSession.Refresh (System.String id=3f2504e0-4f89-11d3-9a0c-0305e82c3301) (at Assets/HypeDomino/Net/Auth/AuthSession.cs:9)\n");
            var item = Lines(Envelope().Build(report, EventId, At))[2];

            StringAssert.DoesNotContain("3f2504e0", item);
            StringAssert.DoesNotContain("a.b@example.com", item);
            StringAssert.Contains("\"value\":\"account [id] mail [email]\"", item);
        }

        [Test]
        public void AReleaseIsRequired()
        {
            Assert.Throws<ArgumentException>(() => new SentryEnvelope("", "staging", "os", "model"));
        }
    }
}
