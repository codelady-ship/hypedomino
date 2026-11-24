using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MatchPrefetchTests
    {
        const string NeonTable = "cosmetic/table/neon/static";
        const string FoxAvatar = "cosmetic/avatar/fox/static";
        const string GoldFrame = "cosmetic/frame/gold/static";
        const string JadeTile = "cosmetic/tile/jade/static";
        const string StarterAvatar = "cosmetic/avatar/starter/static";
        const string OwlAvatar = "cosmetic/avatar/owl/static";
        const string Catalog = @"{""schemaVersion"":1,""manifestVersion"":1,""items"":[
{""id"":""avatar.starter"",""type"":""avatar"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/avatar/starter/static"",""sources"":[{""type"":""default""}]},
{""id"":""avatar.fox"",""type"":""avatar"",""rarityId"":""epic"",""availability"":""live"",""staticAddress"":""cosmetic/avatar/fox/static""},
{""id"":""avatar.owl"",""type"":""avatar"",""rarityId"":""common"",""availability"":""retired"",""staticAddress"":""cosmetic/avatar/owl/static""},
{""id"":""avatar.gone"",""type"":""avatar"",""rarityId"":""common"",""availability"":""disabled"",""staticAddress"":""cosmetic/avatar/gone/static""},
{""id"":""frame.gold"",""type"":""frame"",""rarityId"":""epic"",""availability"":""live"",""staticAddress"":""cosmetic/frame/gold/static""},
{""id"":""table.neon"",""type"":""table"",""rarityId"":""epic"",""availability"":""live"",""staticAddress"":""cosmetic/table/neon/static""},
{""id"":""tile.jade"",""type"":""tile"",""rarityId"":""epic"",""availability"":""live"",""staticAddress"":""cosmetic/tile/jade/static""}]}";

        sealed class Source : ICosmeticArtSource<Sprite>
        {
            readonly Dictionary<string, Action<Sprite>> pending = new();

            public readonly List<string> Loads = new();
            public readonly List<string> Releases = new();

            public void Load(string address, Action<Sprite> done)
            {
                Loads.Add(address);
                pending[address] = done;
            }

            public void Release(string address) => Releases.Add(address);

            public void Finish(string address, Sprite art)
            {
                var done = pending[address];
                pending.Remove(address);
                done(art);
            }
        }

        Source source;
        MatchPrefetch prefetch;
        CosmeticManifest manifest;
        Texture2D texture;
        Sprite art;

        [SetUp]
        public void CreatePrefetch()
        {
            source = new Source();
            Assert.IsTrue(CosmeticManifest.TryParse(Catalog, out manifest));
            prefetch = new MatchPrefetch(new CosmeticArtCache<Sprite>(source));
            texture = new Texture2D(4, 4);
            art = Sprite.Create(texture, new Rect(0, 0, 4, 4), Vector2.zero);
        }

        [TearDown]
        public void DestroyArt()
        {
            Object.DestroyImmediate(art);
            Object.DestroyImmediate(texture);
        }

        static MatchSnapshot Snapshot(string table, params RosterEntry[] roster)
        {
            var view = new MatchView { State = MatchState.Loading, Mode = Mode._1V1, Seat = Seat.X, Table = table };
            view.Roster.Add(roster);
            var snapshot = new MatchSnapshot();
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return snapshot;
        }

        static RosterEntry Player(Seat seat, string avatar, string frame, string tile) => new() { Seat = seat, Avatar = avatar, Frame = frame, Tile = tile };

        void Start(MatchSnapshot view) => prefetch.Start(view, manifest);

        [TestCase("avatar.gone")]
        [TestCase("avatar.ghost")]
        [TestCase("frame.gold")]
        public void AnAvatarTheSeatDoesNotShowLoadsTheDefaultInstead(string avatarId)
        {
            Start(Snapshot(string.Empty,
                Player(Seat.X, avatarId, string.Empty, string.Empty),
                Player(Seat.Y, "avatar.starter", string.Empty, string.Empty)));

            CollectionAssert.AreEqual(new[] { StarterAvatar }, source.Loads);
        }

        [Test]
        public void ARetiredAvatarLoadsItsOwnStatic()
        {
            Start(Snapshot(string.Empty, Player(Seat.X, "avatar.owl", string.Empty, string.Empty)));

            CollectionAssert.AreEqual(new[] { OwlAvatar }, source.Loads);
        }

        [Test]
        public void TheTableAndEachPlayersAvatarFrameAndTileAreLoadedOnce()
        {
            Start(Snapshot("table.neon",
                Player(Seat.X, "avatar.fox", "frame.gold", "tile.jade"),
                Player(Seat.Y, "avatar.fox", "frame.gold", "tile.jade")));

            CollectionAssert.AreEqual(new[] { NeonTable, FoxAvatar, GoldFrame, JadeTile }, source.Loads);
            Assert.AreEqual(4, prefetch.Count);
            Assert.IsFalse(prefetch.Done);
        }

        [Test]
        public void EmptyAndMalformedIdsAreSkipped()
        {
            Start(Snapshot(string.Empty, Player(Seat.X, "avatar.fox", string.Empty, "frame.gold")));

            CollectionAssert.AreEqual(new[] { FoxAvatar }, source.Loads);
        }

        [Test]
        public void LoadedRisesAsEachStaticSettles()
        {
            Start(Snapshot("table.neon", Player(Seat.X, "avatar.fox", "frame.gold", "tile.jade")));
            Assert.AreEqual(0f, prefetch.Loaded);

            source.Finish(NeonTable, art);
            Assert.AreEqual(0.25f, prefetch.Loaded, 1e-6f);

            source.Finish(FoxAvatar, art);
            source.Finish(GoldFrame, art);
            source.Finish(JadeTile, art);
            Assert.AreEqual(1f, prefetch.Loaded);
            Assert.IsTrue(prefetch.Done);
        }

        [Test]
        public void AFailedLoadCountsAsSettledSoTheMatchNeverWaitsForIt()
        {
            Start(Snapshot("table.neon"));

            source.Finish(NeonTable, null);

            Assert.IsTrue(prefetch.Done);
        }

        [Test]
        public void NothingToLoadIsDoneAtOnce()
        {
            Start(Snapshot(string.Empty));

            Assert.IsTrue(prefetch.Done);
            Assert.AreEqual(1f, prefetch.Loaded);
        }

        [Test]
        public void ReleaseFreesTheLoadedArt()
        {
            Start(Snapshot("table.neon"));
            source.Finish(NeonTable, art);

            prefetch.Release();

            CollectionAssert.AreEqual(new[] { NeonTable }, source.Releases);
            Assert.AreEqual(0, prefetch.Count);
        }

        [Test]
        public void ALoadFinishingAfterReleaseIsFreedAndNotCounted()
        {
            Start(Snapshot("table.neon", Player(Seat.X, "avatar.fox", string.Empty, string.Empty)));
            prefetch.Release();
            Start(Snapshot(string.Empty, Player(Seat.X, "avatar.fox", string.Empty, string.Empty)));

            source.Finish(NeonTable, art);

            CollectionAssert.AreEqual(new[] { NeonTable }, source.Releases);
            Assert.AreEqual(0, prefetch.Settled);
        }

        [Test]
        public void StartingAgainReleasesThePreviousMatch()
        {
            Start(Snapshot("table.neon"));
            source.Finish(NeonTable, art);

            Start(Snapshot(string.Empty, Player(Seat.X, "avatar.fox", string.Empty, string.Empty)));

            CollectionAssert.AreEqual(new[] { NeonTable }, source.Releases);
            CollectionAssert.AreEqual(new[] { NeonTable, FoxAvatar }, source.Loads);
            Assert.IsFalse(prefetch.Done);
        }
    }
}
