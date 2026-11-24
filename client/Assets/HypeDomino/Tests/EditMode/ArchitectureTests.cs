using System;
using System.Collections.Generic;
using System.IO;
using System.Reflection;
using System.Text.RegularExpressions;
using HypeDomino.Game;
using HypeDomino.Game.FriendLobby;
using HypeDomino.Game.Lobby;
using HypeDomino.Net;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ArchitectureTests
    {
        const string SessionFile = "Session.cs";
        const string BootSuffix = "Boot.cs";
        static readonly string[] LobbyNamespaces = { "HypeDomino.Game.Lobby", "HypeDomino.Game.FriendLobby" };
        static readonly string[] LobbyPayloadPrefixes = { "Lobby", "Queue" };
        static readonly string[] CoroutineUsers = { "CanvasBench.cs" };
        static readonly Regex Coroutine = new Regex(@"\bStartCoroutine\s*\(|\bIEnumerator\s+\w+\s*\(|\bnew\s+WaitFor\w*\s*\(");
        static readonly Regex FrameRead = new Regex(@"\.TryReceive\s*\(|\bParser\.ParseFrom\s*\(|\bFrameReader\.TryRead\s*\(");
        static readonly Regex StartupHook = new Regex(@"\[\s*RuntimeInitializeOnLoadMethod\b");
        static readonly Regex BeforeSceneHook = new Regex(@"\[\s*RuntimeInitializeOnLoadMethod\s*\(\s*RuntimeInitializeLoadType\.BeforeSceneLoad\s*\)\s*\]");

        [Test]
        public void MatchCodeNamesNothingFromTheLobby()
        {
            var words = LobbyWords();
            var offenders = new List<string>();
            foreach (var file in MatchSources())
            {
                var source = File.ReadAllText(file);
                foreach (var word in words)
                    if (Names(source, word)) offenders.Add($"{Path.GetFileName(file)}: {word}");
            }
            Assert.IsEmpty(offenders);
        }

        [Test]
        public void TheLobbyWordsCoverTheLobbyTypesHandlesAndEvents()
        {
            var words = LobbyWords();
            CollectionAssert.Contains(words, "HypeDomino.Game.Lobby");
            CollectionAssert.Contains(words, nameof(MatchQueue));
            CollectionAssert.Contains(words, nameof(FriendRoom));
            CollectionAssert.Contains(words, $"{nameof(NetBoot)}.{nameof(NetBoot.Queue)}");
            CollectionAssert.Contains(words, $"{nameof(NetBoot)}.{nameof(NetBoot.Room)}");
            CollectionAssert.Contains(words, nameof(Session.LobbyStateReceived));
            CollectionAssert.Contains(words, nameof(Session.QueueStateReceived));
            CollectionAssert.DoesNotContain(words, nameof(Session.ViewReceived));
        }

        [TestCase("using HypeDomino.Game.Lobby;", "HypeDomino.Game.Lobby")]
        [TestCase("readonly MatchQueue queue;", "MatchQueue")]
        [TestCase("NetBoot.Room.Read(state, id);", "NetBoot.Room")]
        [TestCase("session.LobbyStateReceived += OnLobby;", "LobbyStateReceived")]
        public void LobbyWordsAreRecognised(string line, string word) => Assert.IsTrue(Names(line, word));

        [TestCase("public const string BackToLobby = \"match.result.back_to_lobby\";", "HypeDomino.Game.Lobby")]
        [TestCase("readonly MatchQueueView view;", "MatchQueue")]
        [TestCase("NetBoot.Rooms.Count", "NetBoot.Room")]
        public void LookAlikeWordsAreNotFlagged(string line, string word) => Assert.IsFalse(Names(line, word));

        [Test]
        public void OnlySessionReadsFramesAndDecodesServerPayloads()
        {
            var readers = new List<string>();
            foreach (var file in RuntimeCodeRulesTests.RuntimeSources())
                if (FrameRead.IsMatch(File.ReadAllText(file))) readers.Add(Path.GetFileName(file));
            CollectionAssert.AreEqual(new[] { SessionFile }, readers);
        }

        [TestCase("while (transport.TryReceive(out var bytes))")]
        [TestCase("var hello = ServerHello.Parser.ParseFrom(frame.PayloadBytes);")]
        [TestCase("if (!FrameReader.TryRead(bytes, out var frame)) return;")]
        public void FrameReadsAreRecognised(string line) => Assert.IsTrue(FrameRead.IsMatch(line));

        [TestCase("public bool TryReceive(out ReadOnlySpan<byte> frame)")]
        [TestCase("bool TryReceive(out ReadOnlySpan<byte> frame);")]
        [TestCase("if (!View.TryRead(frame.PayloadBytes)) return false;")]
        public void FrameReadDeclarationsAreNotFlagged(string line) => Assert.IsFalse(FrameRead.IsMatch(line));

        [Test]
        public void OnlyTheListedBenchesRunCoroutines()
        {
            var offenders = new List<string>();
            foreach (var file in RuntimeCodeRulesTests.RuntimeSources())
                if (Array.IndexOf(CoroutineUsers, Path.GetFileName(file)) < 0 && Coroutine.IsMatch(File.ReadAllText(file))) offenders.Add(file);
            Assert.IsEmpty(offenders);
        }

        [TestCase("StartCoroutine(Fade());")]
        [TestCase("IEnumerator Start()")]
        [TestCase("yield return new WaitForSeconds(1f);")]
        public void CoroutinesAreRecognised(string line) => Assert.IsTrue(Coroutine.IsMatch(line));

        [TestCase("static IEnumerable<string> Keys()")]
        [TestCase("public IEnumerator<int> GetEnumerator()")]
        [TestCase("IEnumerator IEnumerable.GetEnumerator() => GetEnumerator();")]
        [TestCase("foreach (var item in items) yield return item;")]
        public void IteratorsAreNotFlagged(string line) => Assert.IsFalse(Coroutine.IsMatch(line));

        [Test]
        public void OnlyBootClassesComposeTheAppAtStartup()
        {
            var roots = new List<string>();
            var offenders = new List<string>();
            foreach (var file in RuntimeCodeRulesTests.RuntimeSources())
            {
                if (!StartupHook.IsMatch(File.ReadAllText(file))) continue;
                roots.Add(file);
                if (!Path.GetFileName(file).EndsWith(BootSuffix, StringComparison.Ordinal)) offenders.Add(file);
            }
            Assert.IsNotEmpty(roots);
            Assert.IsEmpty(offenders);
        }

        [TestCase("[RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.BeforeSceneLoad)]")]
        [TestCase("[ RuntimeInitializeOnLoadMethod]")]
        public void StartupHooksAreRecognised(string line) => Assert.IsTrue(StartupHook.IsMatch(line));

        [Test]
        public void EveryBootBeforeTheFirstSceneMarksItsStartupPhase()
        {
            var boots = new List<string>();
            var unmarked = new List<string>();
            foreach (var file in RuntimeCodeRulesTests.RuntimeSources())
            {
                var source = File.ReadAllText(file);
                if (!BeforeSceneHook.IsMatch(source)) continue;
                var boot = Path.GetFileNameWithoutExtension(file);
                boots.Add(boot);
                if (!source.Contains($"StartupPhases.App.Mark(nameof({boot}))")) unmarked.Add(boot);
            }
            CollectionAssert.Contains(boots, nameof(AppBoot));
            Assert.IsEmpty(unmarked);
        }

        [TestCase("[RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.BeforeSceneLoad)]", true)]
        [TestCase("[ RuntimeInitializeOnLoadMethod( RuntimeInitializeLoadType.BeforeSceneLoad ) ]", true)]
        [TestCase("[RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.AfterSceneLoad)]", false)]
        [TestCase("[RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.AfterAssembliesLoaded)]", false)]
        public void BeforeSceneHooksAreRecognised(string line, bool beforeScene) => Assert.AreEqual(beforeScene, BeforeSceneHook.IsMatch(line));

        static bool Names(string source, string word) => Regex.IsMatch(source, $@"(?<!\w){Regex.Escape(word)}(?!\w)");

        static HashSet<string> LobbyWords()
        {
            var words = new HashSet<string>(LobbyNamespaces);
            foreach (var type in typeof(AppFlow).Assembly.GetTypes())
                if (type.DeclaringType == null && IsLobbyType(type)) words.Add(TypeName(type));
            foreach (var property in typeof(NetBoot).GetProperties(BindingFlags.Public | BindingFlags.Static))
                if (IsLobbyType(property.PropertyType)) words.Add($"{nameof(NetBoot)}.{property.Name}");
            foreach (var serverEvent in typeof(Session).GetEvents())
            {
                var payload = serverEvent.EventHandlerType.GetGenericArguments();
                if (payload.Length != 1 || !IsLobbyPayload(payload[0])) continue;
                words.Add(serverEvent.Name);
                words.Add(payload[0].Name);
            }
            return words;
        }

        static bool IsLobbyType(Type type) => Array.IndexOf(LobbyNamespaces, type.Namespace) >= 0;

        static bool IsLobbyPayload(Type payload)
        {
            foreach (var prefix in LobbyPayloadPrefixes)
                if (payload.Name.StartsWith(prefix, StringComparison.Ordinal)) return true;
            return false;
        }

        static string TypeName(Type type)
        {
            var tick = type.Name.IndexOf('`');
            return tick < 0 ? type.Name : type.Name.Substring(0, tick);
        }

        static IEnumerable<string> MatchSources()
        {
            var folder = Path.Combine(Application.dataPath, "HypeDomino", "Game", "Match");
            var files = Directory.GetFiles(folder, "*.cs", SearchOption.AllDirectories);
            Assert.IsNotEmpty(files);
            return files;
        }
    }
}
