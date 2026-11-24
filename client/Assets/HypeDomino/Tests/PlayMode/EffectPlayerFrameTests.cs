using System.Collections;
using System.IO;
using HypeDomino.UI.Effects;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using NUnit.Framework;
using Unity.Profiling;
using UnityEngine;
using UnityEngine.TestTools;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class EffectPlayerFrameTests
    {
        const int Frames = 600;
        const int FramesPerTurn = 45;
        const int TurnsPerHand = 6;
        const float FrameMs = 1000f / 60f;
        const int RealFrames = 120;
        const int RecordedSamplesCap = 256;

        static readonly Vector3 Hand = new Vector3(0f, -600f, 0f);
        static readonly Vector3 Meter = new Vector3(400f, 700f, 0f);
        static readonly Vector2 TileReach = new Vector2(40f, 80f);
        static readonly Vector2 BoardReach = new Vector2(500f, 300f);

        GameObject root;
        EffectPlayer player;
        TestDelegate frame;
        float nowMs;
        int frameIndex;

        [SetUp]
        public void BuildPlayer()
        {
            root = new GameObject("Screen", typeof(RectTransform), typeof(Canvas));
            root.GetComponent<Canvas>().renderMode = RenderMode.WorldSpace;
            var layer = new GameObject("Effects", typeof(RectTransform));
            layer.transform.SetParent(root.transform, false);
            player = layer.AddComponent<EffectPlayer>();
            var library = EffectLibrary.Parse(File.ReadAllText(Path.Combine(Application.dataPath, "HypeDomino", "UI", "Effects", "effects.json")));
            var shared = Canvas.GetDefaultCanvasMaterial();
            player.Build(library, NamedSprites(library), shared, shared, EffectLevel.High, false);
            frame = PlayOneFrame;
            nowMs = 0f;
            frameIndex = 0;
        }

        [TearDown]
        public void DestroyPlayer() => Object.Destroy(root);

        static Sprite[] NamedSprites(EffectLibrary library)
        {
            var sprites = new Sprite[library.SpriteCount];
            for (var sprite = 0; sprite < sprites.Length; sprite++)
            {
                sprites[sprite] = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), new Vector2(0.5f, 0.5f));
                sprites[sprite].name = library.SpriteName(sprite);
            }
            return sprites;
        }

        void PlayOneFrame()
        {
            var turnFrame = frameIndex % FramesPerTurn;
            var turn = frameIndex / FramesPerTurn;
            if (turnFrame == 0) PlayTurn(turn);
            if (turnFrame == 0 && turn % TurnsPerHand == TurnsPerHand - 1) PlayHandEnd(turn);
            player.Tick(nowMs);
            Canvas.ForceUpdateCanvases();
            nowMs += FrameMs;
            frameIndex++;
        }

        void PlayTurn(int turn)
        {
            player.Play(EffectRequest.On(TokenEffectKind.TilePlace, Vector3.zero, TileReach, nowMs).TrailFrom(Hand, TokenDurationMs.Base));
            player.Play(EffectRequest.On(TokenEffectKind.ScoreCascade, Vector3.zero, TileReach, nowMs).AtStep(1 + turn % ScoreCascade.MaxStep).TrailTo(Meter));
            player.Play(EffectRequest.On(TokenEffectKind.Emoji, Meter, TileReach, nowMs));
            player.Play(EffectRequest.On(TokenEffectKind.SpinnerFormation, Vector3.zero, TileReach, nowMs));
        }

        void PlayHandEnd(int turn)
        {
            var kind = turn % 2 == 0 ? TokenEffectKind.Domino : TokenEffectKind.Block;
            player.Play(EffectRequest.On(kind, Vector3.zero, BoardReach, nowMs).ForTeam(TokenColor.RoleTeamAFill));
            player.Play(EffectRequest.On(TokenEffectKind.MatchWin, Vector3.zero, BoardReach, nowMs + TokenEffect.DominoDurationMs).ForTeam(TokenColor.RoleTeamBFill));
            player.Play(EffectRequest.On(TokenEffectKind.RankUp, Meter, BoardReach, nowMs));
        }

        void RunFrames(bool measured)
        {
            for (var at = 0; at < Frames; at++)
            {
                if (!measured)
                {
                    frame();
                    continue;
                }
                Assert.AreEqual(0, AllocatesNothingConstraint.CountOnCallingThread(frame), $"GC allocations in frame {at}");
                Assert.LessOrEqual(player.Budget.Live, player.Budget.Limit);
            }
        }

        [TestCase(EffectLevel.Low, false)]
        [TestCase(EffectLevel.Medium, false)]
        [TestCase(EffectLevel.High, false)]
        [TestCase(EffectLevel.High, true)]
        public void EveryFrameOfAPlayedMatchAllocatesNothing(EffectLevel level, bool reduceMotion)
        {
            player.Configure(level, reduceMotion);
            RunFrames(false);
            player.EndAll();
            nowMs = 0f;
            frameIndex = 0;

            RunFrames(true);

            Assert.Greater(player.LiveQuads, 0);
        }

        [Test]
        public void TheLiveParticlesStayWithinTheLevelsCapEveryFrame()
        {
            var peak = 0;
            for (var at = 0; at < Frames; at++)
            {
                frame();
                Assert.LessOrEqual(player.ParticleQuads, TokenEffectLevel.HighMaxParticles);
                Assert.AreEqual(player.Budget.Live, player.ParticleQuads);
                peak = Mathf.Max(peak, player.ParticleQuads);
            }
            Assert.Greater(peak, TokenEffectLevel.MediumMaxParticles, "the high level carries more than the medium cap at its peak");
        }

        [UnityTest]
        public IEnumerator TheUpdateLoopPlaysInsideItsProfilerMarker()
        {
            player.Play(EffectRequest.On(TokenEffectKind.MatchWin, Vector3.zero, BoardReach, Time.unscaledTime * 1000f).ForTeam(TokenColor.RoleTeamAFill));
            var recorder = new ProfilerRecorder(EffectPlayer.ProfilerMarkerName, RecordedSamplesCap, ProfilerRecorderOptions.CollectOnlyOnCurrentThread);
            try
            {
                Assert.IsTrue(recorder.Valid);
                recorder.Start();
                for (var at = 0; at < RealFrames; at++) yield return null;
                recorder.Stop();
                Assert.GreaterOrEqual(recorder.Count, RealFrames - 1);
            }
            finally
            {
                recorder.Dispose();
            }
        }
    }
}
