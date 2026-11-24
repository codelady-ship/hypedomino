using HypeDomino.Content;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using UnityEngine;

namespace HypeDomino.Game
{
    public static class MatchBoot
    {
        public static MatchHost Host { get; private set; }

        public static void Start(AppFlow flow, Connection connection)
        {
            Host = new MatchHost(flow, connection, ContentBoot.Cosmetics);
            var driver = new GameObject(nameof(MatchBoot)) { hideFlags = HideFlags.HideInHierarchy };
            Object.DontDestroyOnLoad(driver);
            var probe = driver.AddComponent<GraphicsProbe>();
            probe.enabled = false;
            driver.AddComponent<Driver>().Probe = probe;
        }

        sealed class Driver : MonoBehaviour
        {
            public GraphicsProbe Probe;
            bool live;

            void Update()
            {
                Host.Tick(Time.realtimeSinceStartupAsDouble, MotionBoot.Motion.ReduceMotion);
                if (Host.Live == live) return;
                live = Host.Live;
                Probe.enabled = live;
            }
        }
    }
}
