using HypeDomino.Game;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class AppBootTests
    {
        [Test]
        public void TheRunningAppFlowTellsTheFrameRateWhenAMatchStartsAndEnds()
        {
            var flow = AppBoot.Flow;
            Assert.IsNotNull(flow, "AppBoot did not create the app flow");

            flow.Hello(new ServerHello { Status = HelloStatus.Accepted, InMatch = true });
            Assert.AreEqual(AppState.Match, flow.State);
            Assert.IsTrue(GraphicsBoot.FrameRate.InMatch);
            Assert.AreEqual(GraphicsBoot.FrameRate.Target, Application.targetFrameRate);

            flow.View(MatchState.Finished);
            Assert.IsTrue(flow.LeaveMatch());
            Assert.AreEqual(AppState.Lobby, flow.State);
            Assert.IsFalse(GraphicsBoot.FrameRate.InMatch);
            Assert.AreEqual(GraphicsBoot.FrameRate.Target, Application.targetFrameRate);
        }
    }
}
