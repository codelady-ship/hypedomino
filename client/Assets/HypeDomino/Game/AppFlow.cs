using System;
using Hypedomino.V1;

namespace HypeDomino.Game
{
    public sealed class AppFlow
    {
        bool matchFinished;
        bool inRoom;

        public AppState State { get; private set; } = AppState.Boot;

        public event Action<AppState> Changed;

        public void SignInNeeded()
        {
            if (State == AppState.UpdateRequired) return;
            Enter(AppState.Login);
        }

        public void Hello(ServerHello hello)
        {
            switch (hello.Status)
            {
                case HelloStatus.Accepted:
                    matchFinished = false;
                    inRoom = false;
                    Enter(hello.InMatch ? AppState.Match : AppState.Lobby);
                    return;
                case HelloStatus.UpdateRequired:
                    Enter(AppState.UpdateRequired);
                    return;
                case HelloStatus.Maintenance:
                    Enter(AppState.Maintenance);
                    return;
            }
        }

        public void View(MatchState state)
        {
            var finished = state == MatchState.Finished;
            if (State == AppState.Match)
            {
                matchFinished = finished;
                return;
            }
            if ((State != AppState.Lobby && State != AppState.Room) || finished) return;
            matchFinished = false;
            Enter(AppState.Match);
        }

        public void Room(LobbyState state)
        {
            inRoom = !string.IsNullOrEmpty(state.Code);
            switch (State)
            {
                case AppState.Match when inRoom && state.Phase != LobbyPhase.Match:
                    matchFinished = false;
                    Enter(AppState.Room);
                    return;
                case AppState.Lobby when inRoom:
                    Enter(AppState.Room);
                    return;
                case AppState.Room when !inRoom:
                    Enter(AppState.Lobby);
                    return;
            }
        }

        public bool LeaveMatch()
        {
            if (State != AppState.Match || !matchFinished) return false;
            Enter(inRoom ? AppState.Room : AppState.Lobby);
            return true;
        }

        void Enter(AppState next)
        {
            if (next == State) return;
            State = next;
            Changed?.Invoke(next);
        }
    }
}
