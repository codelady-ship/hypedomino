using System;
using HypeDomino.Core.Diagnostics;
using HypeDomino.Editor;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SentryDsnTests
    {
        const string EuDsn = "https://publickey@o42.ingest.de.sentry.io/4507";

        [Test]
        public void AnEuDsnPostsEnvelopesToItsProjectWithThePublicKey()
        {
            var dsn = SentryDsn.Parse(EuDsn);

            Assert.AreEqual("https://o42.ingest.de.sentry.io/api/4507/envelope/", dsn.EnvelopeUrl);
            Assert.AreEqual("Sentry sentry_version=7, sentry_key=publickey", dsn.AuthHeader);
        }

        [Test]
        public void ASecretKeyAndAPathPrefixAreHandled()
        {
            var dsn = SentryDsn.Parse("https://public:secret@o42.ingest.de.sentry.io/relay/4507");

            Assert.AreEqual("https://o42.ingest.de.sentry.io/relay/api/4507/envelope/", dsn.EnvelopeUrl);
            StringAssert.DoesNotContain("secret", dsn.AuthHeader);
        }

        [TestCase("https://publickey@o42.ingest.us.sentry.io/4507")]
        [TestCase("https://publickey@o42.ingest.sentry.io/4507")]
        [TestCase("https://publickey@ingest.de.sentry.io.example.com/4507")]
        [TestCase("http://publickey@o42.ingest.de.sentry.io/4507")]
        [TestCase("https://o42.ingest.de.sentry.io/4507")]
        [TestCase("https://publickey@o42.ingest.de.sentry.io/")]
        [TestCase("https://publickey@o42.ingest.de.sentry.io/project")]
        [TestCase("https://publickey@o42.ingest.de.sentry.io/4507?x=1")]
        [TestCase("not a dsn publickey")]
        [TestCase("")]
        [TestCase(null)]
        public void AnythingButAnHttpsEuDsnIsRefusedWithoutEchoingIt(string raw)
        {
            var error = Assert.Throws<ArgumentException>(() => SentryDsn.Parse(raw));
            StringAssert.DoesNotContain("publickey", error.Message);
        }

        [TestCase("{}")]
        [TestCase("{\"dsn\":\"\",\"environment\":\"staging\"}")]
        public void SettingsWithoutADsnTurnCrashReportsOff(string json)
        {
            Assert.IsNull(CrashReportSettings.FromJson(json));
        }

        [Test]
        public void SettingsWithAnEuDsnKeepTheDsnAndEnvironment()
        {
            var settings = CrashReportSettings.FromJson("{\"dsn\":\"" + EuDsn + "\",\"environment\":\"staging\"}");

            Assert.AreEqual(EuDsn, settings.dsn);
            Assert.AreEqual("staging", settings.environment);
            Assert.AreEqual("https://o42.ingest.de.sentry.io/api/4507/envelope/", settings.Dsn.EnvelopeUrl);
        }

        [Test]
        public void SettingsWithADsnOutsideTheEuAreRefused()
        {
            Assert.Throws<ArgumentException>(() => CrashReportSettings.FromJson("{\"dsn\":\"https://k@o42.ingest.us.sentry.io/4507\"}"));
        }

        [TestCase("staging prod")]
        [TestCase("a/b")]
        [TestCase("line\\nbreak")]
        public void SettingsWithAnEnvironmentSentryRefusesAreRefused(string environment)
        {
            Assert.Throws<ArgumentException>(() => CrashReportSettings.FromJson("{\"dsn\":\"" + EuDsn + "\",\"environment\":\"" + environment + "\"}"));
        }

        [Test]
        public void TheReleaseBuildRefusesACrashReportingFileWithoutADsn()
        {
            Assert.Throws<ArgumentException>(() => ReleaseBuild.ParseCrashReporting("{\"dsn\":\"\"}"));
            Assert.AreEqual(EuDsn, ReleaseBuild.ParseCrashReporting("{\"dsn\":\"" + EuDsn + "\"}").dsn);
        }
    }
}
