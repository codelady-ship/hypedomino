using HypeDomino.Core.Diagnostics;
using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.Game
{
    public sealed class GraphicsProbe : MonoBehaviour
    {
        readonly FrameTiming[] latestTiming = new FrameTiming[1];
        FrameWindow window;
        int timedFrames;

        void OnEnable()
        {
            window = new FrameWindow(TokenGraphics.ProbeFrames, double.MaxValue);
            timedFrames = 0;
        }

        void Update()
        {
            var quality = GraphicsBoot.Quality;
            if (quality == null || !quality.ProbeWanted)
            {
                enabled = false;
                return;
            }
            FrameTimingManager.CaptureFrameTimings();
            if (FrameTimingManager.GetLatestTimings(1, latestTiming) == 0) return;
            if (++timedFrames <= TokenGraphics.ProbeWarmupFrames) return;

            window.Add(FrameWindow.WorkMs(latestTiming[0]), 0, 0, 0);
            if (!window.IsFull) return;

            GraphicsBoot.SettleProbe(window.Close().WorkP95Ms);
            enabled = false;
        }
    }
}
