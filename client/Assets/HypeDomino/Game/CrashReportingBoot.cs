using System;
using HypeDomino.Core.Diagnostics;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Game
{
    public static class CrashReportingBoot
    {
        public static ErrorReports Reports { get; private set; } = new ErrorReports(NoErrorSender.Instance);

        static SentryErrorSender sender;

        [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.BeforeSceneLoad)]
        static void Start()
        {
            Build();
            StartupPhases.App.Mark(nameof(CrashReportingBoot));
        }

        static void Build()
        {
            var settings = CrashReportSettings.Load();
            if (settings == null) return;
            var envelope = new SentryEnvelope(Application.version, settings.environment, SystemInfo.operatingSystem, SystemInfo.deviceModel);
            sender = new SentryErrorSender(settings.Dsn, envelope, new UnityEnvelopeTransport(), UtcNow);
            Reports = new ErrorReports(sender);
            Application.logMessageReceivedThreaded += Reports.OnLog;
            var driver = new GameObject(nameof(CrashReportingBoot)) { hideFlags = HideFlags.HideInHierarchy };
            Object.DontDestroyOnLoad(driver);
            driver.AddComponent<Driver>();
        }

        static DateTime UtcNow() => DateTime.UtcNow;

        sealed class Driver : MonoBehaviour
        {
            void Update() => sender.Pump();
        }
    }
}
