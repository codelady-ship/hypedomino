using System;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using Hypedomino.V1;
using UnityEngine;

namespace HypeDomino.Game.Match
{
    public interface IMatchViews
    {
        bool Ready { get; }
        IMatchTable Table { get; }
        ILoadingScreen LoadingScreen { get; }
        CosmeticArtCache<Sprite> Stills { get; }
        MatchEmoji Emoji { get; }
    }

    public sealed class MatchHost
    {
        readonly AppFlow flow;
        readonly Connection connection;
        readonly ActiveCosmeticManifest manifest;
        IMatchViews views;

        public MatchHost(AppFlow flow, Connection connection, ActiveCosmeticManifest manifest)
        {
            this.flow = flow ?? throw new ArgumentNullException(nameof(flow));
            this.connection = connection ?? throw new ArgumentNullException(nameof(connection));
            this.manifest = manifest ?? throw new ArgumentNullException(nameof(manifest));
            flow.Changed += Follow;
        }

        public MatchHud Hud { get; } = new();
        public MatchRoot Root { get; private set; }
        public bool Live => Root != null;

        public void Attach(IMatchViews source)
        {
            if (source == null) throw new ArgumentNullException(nameof(source));
            Stop();
            views = source;
            Follow(flow.State);
        }

        public void Detach(IMatchViews source)
        {
            if (source == null || source != views) return;
            Stop();
            views = null;
        }

        public void Tick(double nowSeconds, bool reduceMotion)
        {
            if (Root == null) return;
            Root.Tick(connection, nowSeconds, reduceMotion, views.Ready);
        }

        void Follow(AppState state)
        {
            if (state == AppState.Match) Begin();
            else Stop();
        }

        void Begin()
        {
            if (Root != null || views == null) return;
            Hud.Clear();
            Root = new MatchRoot(views.Table, Hud, views.LoadingScreen, views.Stills, manifest, views.Emoji);
            Root.Bind(connection);
            var view = connection.Session.View;
            if (view.State != MatchState.Unspecified && view.State != MatchState.Finished) Root.Show(view);
        }

        void Stop()
        {
            if (Root == null) return;
            Root.Unbind();
            Root.Loading.ReleaseArt();
            Root = null;
            Hud.Clear();
        }
    }
}
