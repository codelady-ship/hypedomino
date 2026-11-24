using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class EffectCuesTests
    {
        MatchSnapshot snapshot;
        EffectCues cues;

        [SetUp]
        public void StartFresh()
        {
            snapshot = new MatchSnapshot();
            cues = new EffectCues();
        }

        static MatchView View(uint teamA, uint teamB, uint hands = 0, uint placements = 1) => new()
        {
            State = MatchState.Playing,
            Seat = Seat.South,
            Mode = Mode._2V2,
            Current = Seat.West,
            HandsPlayed = hands,
            TurnPlacements = placements,
            Scores = new TeamPoints { TeamA = teamA, TeamB = teamB },
        };

        static MatchView HandEnd(HandEnding ending, Team winner, uint teamA, uint teamB, uint hands)
        {
            var view = View(teamA, teamB, hands, 0);
            view.LastResult = new HandResult { Ending = ending, Winner = winner };
            return view;
        }

        EffectCue Read(MatchView view)
        {
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return cues.Read(snapshot);
        }

        [Test]
        public void TheFirstViewOfAConnectionPlaysNothing()
        {
            var cue = Read(HandEnd(HandEnding.Domino, Team.A, 120, 40, 3));

            Assert.AreEqual(default(EffectCue), cue);
        }

        [Test]
        public void BoardPointsGiveTheirStepToTheTeamThatScored()
        {
            Read(View(0, 0));

            var cue = Read(View(0, 15));

            Assert.AreEqual(0, cue.StepA);
            Assert.AreEqual(3, cue.StepB);
            Assert.IsFalse(cue.HandEnded);
        }

        [Test]
        public void ALaterPlacementOfAChainIsBoosted()
        {
            Read(View(0, 0));

            Assert.AreEqual(2 + 1, Read(View(10, 0, 0, 2)).StepA);
        }

        [Test]
        public void ADominoEndsTheHandWithItsWinnerAndUnboostedHandPoints()
        {
            Read(View(30, 0, 0, 3));

            var cue = Read(HandEnd(HandEnding.Domino, Team.A, 55, 0, 1));

            Assert.AreEqual(HandEnding.Domino, cue.HandEnding);
            Assert.AreEqual(Team.A, cue.HandWinner);
            Assert.AreEqual(5, cue.StepA);
            Assert.AreEqual(Team.Unspecified, cue.MatchWinner);
        }

        [Test]
        public void ABlockIsReadOncePerHand()
        {
            Read(View(0, 0));

            Assert.AreEqual(HandEnding.Block, Read(HandEnd(HandEnding.Block, Team.B, 0, 25, 1)).HandEnding);
            Assert.AreEqual(HandEnding.Unspecified, Read(HandEnd(HandEnding.Block, Team.B, 0, 25, 1)).HandEnding);
        }

        [Test]
        public void TheMatchWinIsReadWhenTheMatchFinishes()
        {
            Read(View(140, 90));
            var finished = HandEnd(HandEnding.Domino, Team.B, 140, 155, 1);
            finished.State = MatchState.Finished;
            finished.Winner = Team.B;

            var cue = Read(finished);

            Assert.AreEqual(Team.B, cue.MatchWinner);
            Assert.AreEqual(HandEnding.Domino, cue.HandEnding);
            Assert.AreEqual(Team.Unspecified, Read(finished).MatchWinner, "a later finished view does not replay it");
        }

        [Test]
        public void ANewConnectionDoesNotReplayEffects()
        {
            Read(View(0, 0));
            cues.StartConnection();

            Assert.AreEqual(default(EffectCue), Read(View(60, 0)));
        }
    }
}
