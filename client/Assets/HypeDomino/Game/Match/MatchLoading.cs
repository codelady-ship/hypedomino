using System;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using Hypedomino.V1;

namespace HypeDomino.Game.Match
{
    public interface ILoadingScreen
    {
        void ShowCards(MatchSnapshot view, LoadingCards cards);
        void Close();
    }

    public sealed class MatchLoading
    {
        readonly ILoadingScreen screen;
        readonly MatchPrefetch prefetch;
        readonly ActiveCosmeticManifest manifest;
        readonly LoadingCards cards = new();
        readonly LoadingReport report = new();
        readonly Action<MatchSnapshot> onView;
        Session session;

        public MatchLoading(ILoadingScreen screen, MatchPrefetch prefetch, ActiveCosmeticManifest manifest)
        {
            this.screen = screen ?? throw new ArgumentNullException(nameof(screen));
            this.prefetch = prefetch ?? throw new ArgumentNullException(nameof(prefetch));
            this.manifest = manifest ?? throw new ArgumentNullException(nameof(manifest));
            onView = Show;
        }

        public bool Open { get; private set; }

        public void Bind(Session source)
        {
            Unbind();
            session = source ?? throw new ArgumentNullException(nameof(source));
            session.ViewReceived += onView;
        }

        public void Unbind()
        {
            if (session == null) return;
            session.ViewReceived -= onView;
            session = null;
        }

        public void Show(MatchSnapshot view)
        {
            var loading = view.State == MatchState.Loading;
            if (!loading && !Open) return;
            if (loading && !Open) Begin(view);
            cards.Read(view);
            screen.ShowCards(view, cards);
            if (loading) return;
            Open = false;
            screen.Close();
        }

        public void Tick(bool sceneReady)
        {
            if (!Open || session == null || !session.Accepted) return;
            if (report.TryReport(prefetch.Loaded, prefetch.Done && sceneReady, out var intent))
                session.Send(Envelope.PayloadOneofCase.LoadingIntent, intent);
        }

        public void ReleaseArt() => prefetch.Release();

        void Begin(MatchSnapshot view)
        {
            Open = true;
            report.Reset();
            prefetch.Start(view, manifest.Current);
        }
    }
}
