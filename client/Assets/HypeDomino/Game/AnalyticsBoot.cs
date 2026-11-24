using System;
using HypeDomino.Core.Analytics;
using HypeDomino.Game.Analytics;
using HypeDomino.Game.Lobby;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Game
{
    public static class AnalyticsBoot
    {
        static AnalyticsFlushTimer timer;
        static AppAnalytics app;
        static AnalyticsTuning tuning;

        public static AnalyticsQueue Queue { get; private set; }

        public static void Start(ApiClient api)
        {
            var settings = AnalyticsSettings.Draft;
            Queue = new AnalyticsQueue(new ApiAnalyticsSender(api), DeviceTimeMs, settings.BatchSize, settings.QueueLimit);
            timer = new AnalyticsFlushTimer(Queue, settings.FlushSeconds);
            app = new AppAnalytics(Queue);
            tuning = new AnalyticsTuning(Queue, timer);
            var driver = new GameObject(nameof(AnalyticsBoot)) { hideFlags = HideFlags.HideInHierarchy };
            Object.DontDestroyOnLoad(driver);
            driver.AddComponent<Driver>();
        }

        public static void Follow(AuthSession auth, MatchQueue matches, Session session)
        {
            app.Follow(auth, matches);
            session.HelloReceived += tuning.Read;
        }

        static long DeviceTimeMs() => DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();

        sealed class Driver : MonoBehaviour
        {
            void Update()
            {
                app.Tick();
                timer.Tick(Time.realtimeSinceStartupAsDouble);
            }

            void OnApplicationPause(bool pause)
            {
                if (pause) timer.AppPaused();
            }
        }
    }
}
