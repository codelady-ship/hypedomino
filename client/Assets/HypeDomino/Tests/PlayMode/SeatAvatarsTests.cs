using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Match;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Rarity;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class SeatAvatarsTests
    {
        const string FoxStatic = "cosmetic/avatar/fox/static";
        const string FoxAnim = "cosmetic/avatar/fox/anim";
        const string OwlStatic = "cosmetic/avatar/owl/static";
        const string StarterAvatar = "cosmetic/avatar/starter/static";
        const string GoldStatic = "cosmetic/frame/gold/static";
        const string StarterFrame = "cosmetic/frame/starter/static";
        const string Rarities = @"{""diamondMinSize"":72,""flowPeriodMs"":2000,""tiers"":[
{""id"":""common"",""ordinal"":1,""nameKey"":""rarity.common.name"",""fill"":""#0000FF"",""on"":""#000000"",""text"":""#FFFFFF"",""pips"":1,""glowAlpha"":0.5,""glowScale"":1.2,""flow"":false},
{""id"":""epic"",""ordinal"":3,""nameKey"":""rarity.epic.name"",""fill"":""#0000FF"",""on"":""#000000"",""text"":""#FFFFFF"",""pips"":3,""glowAlpha"":0.75,""glowScale"":1.5,""flow"":false}]}";
        const string Catalog = @"{""schemaVersion"":1,""manifestVersion"":1,""items"":[
{""id"":""avatar.starter"",""type"":""avatar"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/avatar/starter/static"",""sources"":[{""type"":""default""}]},
{""id"":""frame.starter"",""type"":""frame"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/frame/starter/static"",""sources"":[{""type"":""default""}]},
{""id"":""avatar.fox"",""type"":""avatar"",""rarityId"":""epic"",""availability"":""live"",""animated"":true,""staticAddress"":""cosmetic/avatar/fox/static"",""animationAddress"":""cosmetic/avatar/fox/anim""},
{""id"":""avatar.owl"",""type"":""avatar"",""rarityId"":""common"",""availability"":""retired"",""staticAddress"":""cosmetic/avatar/owl/static""},
{""id"":""avatar.gone"",""type"":""avatar"",""rarityId"":""common"",""availability"":""disabled"",""staticAddress"":""cosmetic/avatar/gone/static""},
{""id"":""frame.gold"",""type"":""frame"",""rarityId"":""epic"",""availability"":""live"",""staticAddress"":""cosmetic/frame/gold/static""}]}";

        GameObject root;
        FakeArtSource<Sprite> stills;
        FakeArtSource<LoopArt> loops;
        CosmeticManifest manifest;
        CosmeticStage stage;
        AvatarView[] views;
        SeatAvatars seats;
        Sprite fox;
        LoopArt foxLoop;

        [SetUp]
        public void Build()
        {
            root = new GameObject("Match", typeof(RectTransform), typeof(Canvas));
            stills = new FakeArtSource<Sprite>();
            loops = new FakeArtSource<LoopArt>();
            Assert.IsTrue(CosmeticManifest.TryParse(Catalog, out manifest));
            stage = new CosmeticStage(stills, loops, RarityTable.Parse(Rarities), NewSprite(), NewSprite(), EffectLevel.Medium, false);
            views = new AvatarView[TableSides.Count];
            for (var side = 0; side < views.Length; side++) views[side] = NewView(((TableSide)side).ToString());
            seats = new SeatAvatars(stage, views);
            fox = NewSprite();
            foxLoop = new LoopArt(fox, new[] { NewSprite(), NewSprite() }, 10f);
        }

        [TearDown]
        public void DestroyMatch() => Object.Destroy(root);

        static Sprite NewSprite() => Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);

        static CosmeticImage NewImage(string name, Transform parent)
        {
            var image = new GameObject(name, typeof(RectTransform), typeof(Image)).AddComponent<AnimatedImage>().gameObject.AddComponent<CosmeticImage>();
            image.transform.SetParent(parent, false);
            return image;
        }

        AvatarView NewView(string name)
        {
            var view = new GameObject(name, typeof(RectTransform)).AddComponent<AvatarView>();
            view.transform.SetParent(root.transform, false);
            var halo = new GameObject("Halo", typeof(RectTransform), typeof(Image)).AddComponent<RarityHalo>();
            halo.transform.SetParent(view.transform, false);
            halo.Configure(NewSprite(), NewSprite());
            view.Configure(NewImage("Art", view.transform), NewImage("Frame", view.transform), halo, null);
            return view;
        }

        static MatchSnapshot Snapshot(Mode mode, Seat seat, params RosterEntry[] roster)
        {
            var view = new MatchView { State = MatchState.Playing, Mode = mode, Seat = seat };
            view.Roster.Add(roster);
            var snapshot = new MatchSnapshot();
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return snapshot;
        }

        static RosterEntry Player(Seat seat, string avatar, string frame) => new() { Seat = seat, Avatar = avatar, Frame = frame };

        AvatarView At(TableSide side) => views[(int)side];

        void FinishStill(string address)
        {
            while (stills.Loads.Remove(address)) stills.Finish(address, address == FoxStatic ? fox : NewSprite());
        }

        [Test]
        public void EachSeatShowsItsRosterAvatarAndFrameAtItsTableSide()
        {
            seats.Show(Snapshot(Mode._2V2, Seat.South,
                Player(Seat.South, "avatar.fox", "frame.gold"),
                Player(Seat.West, "avatar.owl", "frame.starter"),
                Player(Seat.North, "avatar.starter", "frame.gold"),
                Player(Seat.East, "avatar.fox", "frame.starter")), manifest);

            FinishStill(FoxStatic);
            FinishStill(OwlStatic);

            Assert.AreSame(fox, At(TableSide.Bottom).Avatar.Shown);
            Assert.AreSame(fox, At(TableSide.Right).Avatar.Shown);
            Assert.AreNotSame(fox, At(TableSide.Left).Avatar.Shown);
            Assert.AreNotSame(stage.BuiltInAvatar, At(TableSide.Left).Avatar.Shown);
            Assert.AreSame(stage.BuiltInAvatar, At(TableSide.Top).Avatar.Shown);
        }

        [Test]
        public void InOneVersusOneTheOpponentSitsAtTheTopAndTheSidesStayEmpty()
        {
            seats.Show(Snapshot(Mode._1V1, Seat.Y,
                Player(Seat.X, "avatar.fox", "frame.gold"),
                Player(Seat.Y, "avatar.owl", "frame.gold")), manifest);

            FinishStill(FoxStatic);

            Assert.AreSame(fox, At(TableSide.Top).Avatar.Shown);
            Assert.AreNotSame(fox, At(TableSide.Bottom).Avatar.Shown);
            Assert.IsNull(At(TableSide.Left).Avatar.Shown);
            Assert.IsNull(At(TableSide.Right).Avatar.Shown);
        }

        [Test]
        public void AViewerWithoutASeatSeesTheTableFromSouth()
        {
            seats.Show(Snapshot(Mode._2V2, Seat.Unspecified, Player(Seat.South, "avatar.fox", "frame.gold")), manifest);

            FinishStill(FoxStatic);

            Assert.AreSame(fox, At(TableSide.Bottom).Avatar.Shown);
        }

        [Test]
        public void ASeatOfAnotherModeIsSkipped()
        {
            seats.Show(Snapshot(Mode._1V1, Seat.X, Player(Seat.North, "avatar.fox", "frame.gold")), manifest);

            Assert.IsEmpty(stills.Loads);
        }

        [Test]
        public void ADisabledUnknownOrEmptyItemShowsTheDefaultOfItsType()
        {
            seats.Show(Snapshot(Mode._1V1, Seat.X,
                Player(Seat.X, "avatar.gone", "frame.gone"),
                Player(Seat.Y, "", "frame.gold")), manifest);

            Assert.AreEqual(new[] { StarterAvatar, StarterFrame, GoldStatic }, stills.Loads);
        }

        [Test]
        public void AnAnimatedAvatarPlaysWhereTheLevelAllowsAndRestsOnTheLowLevel()
        {
            seats.Show(Snapshot(Mode._1V1, Seat.X, Player(Seat.X, "avatar.fox", "frame.gold")), manifest);
            FinishStill(FoxStatic);
            loops.Finish(FoxAnim, foxLoop);
            Assert.IsTrue(At(TableSide.Bottom).Avatar.IsPlaying);

            stage.SetEffects(EffectLevel.Low, false);
            seats.RefreshEffects();

            Assert.IsFalse(At(TableSide.Bottom).Avatar.IsPlaying);
            Assert.AreSame(fox, At(TableSide.Bottom).Avatar.Shown);
            Assert.AreEqual(new[] { FoxAnim }, loops.Releases);
        }

        [Test]
        public void TheSameRosterAgainNeitherReloadsNorRestartsAndAllocatesNothing()
        {
            var view = Snapshot(Mode._2V2, Seat.South,
                Player(Seat.South, "avatar.fox", "frame.gold"),
                Player(Seat.West, "avatar.owl", "frame.starter"),
                Player(Seat.North, "avatar.gone", "frame.gold"),
                Player(Seat.East, "avatar.fox", "frame.starter"));
            seats.Show(view, manifest);
            FinishStill(FoxStatic);
            loops.Finish(FoxAnim, foxLoop);
            var playing = stage.Budget.Active;

            seats.Show(view, manifest);
            Assert.That(() => seats.Show(view, manifest), Allocates.Nothing);

            Assert.IsEmpty(stills.Releases);
            Assert.IsEmpty(loops.Releases);
            Assert.AreEqual(1, loops.Loads.Count);
            Assert.AreEqual(playing, stage.Budget.Active);
        }

        [Test]
        public void AChangedSeatShowsItsNewLookAndFreesTheOldArt()
        {
            seats.Show(Snapshot(Mode._1V1, Seat.X,
                Player(Seat.X, "avatar.fox", "frame.gold"),
                Player(Seat.Y, "avatar.owl", "frame.gold")), manifest);
            FinishStill(FoxStatic);
            FinishStill(OwlStatic);

            seats.Show(Snapshot(Mode._1V1, Seat.X,
                Player(Seat.X, "avatar.owl", "frame.gold"),
                Player(Seat.Y, "avatar.owl", "frame.gold")), manifest);

            Assert.AreEqual(new[] { FoxStatic }, stills.Releases);
            Assert.AreSame(At(TableSide.Top).Avatar.Shown, At(TableSide.Bottom).Avatar.Shown);
        }

        [Test]
        public void TheSeatsTakeOverThePrefetchedArtWithoutLoadingItAgain()
        {
            var view = Snapshot(Mode._1V1, Seat.X,
                Player(Seat.X, "avatar.fox", "frame.gold"),
                Player(Seat.Y, "avatar.owl", "frame.gold"));
            var prefetch = new MatchPrefetch(stage.Stills);
            prefetch.Start(view, manifest);
            FinishStill(FoxStatic);
            FinishStill(OwlStatic);
            FinishStill(GoldStatic);
            Assert.IsTrue(prefetch.Done);

            seats.Show(view, manifest);
            prefetch.Release();

            Assert.IsEmpty(stills.Loads);
            Assert.IsEmpty(stills.Releases);
            Assert.AreSame(fox, At(TableSide.Bottom).Avatar.Shown);
        }
    }
}
