using System;

namespace HypeDomino.Game.Tutorial
{
    public sealed class TutorialOffer
    {
        public const string Title = "tutorial.offer.title";
        public const string Body = "tutorial.offer.body";
        public const string Play = "tutorial.offer.play";
        public const string Skip = "tutorial.offer.skip";

        public TutorialOffer() : this(new TutorialRun())
        {
        }

        public TutorialOffer(TutorialRun run)
        {
            Run = run;
        }

        public TutorialRun Run { get; }
        public bool Asking { get; private set; }
        public bool Offered { get; private set; }

        public event Action Changed;

        public void Follow(AppFlow flow)
        {
            flow.Changed += OnFlowChanged;
            Read(flow.State);
        }

        public bool Read(AppState state)
        {
            if (state != AppState.Lobby) return Withdraw();
            if (Offered || !TutorialProgressStore.ShouldOffer) return false;
            Offered = true;
            Asking = true;
            Changed?.Invoke();
            return true;
        }

        public bool Accept()
        {
            if (!Asking) return false;
            Asking = false;
            Run.Start();
            Changed?.Invoke();
            return true;
        }

        public bool Decline()
        {
            if (!Asking) return false;
            Asking = false;
            Run.Skip();
            TutorialProgressStore.Keep(Run);
            Changed?.Invoke();
            return true;
        }

        bool Withdraw()
        {
            if (!Asking) return false;
            Asking = false;
            Changed?.Invoke();
            return true;
        }

        void OnFlowChanged(AppState state) => Read(state);
    }
}
