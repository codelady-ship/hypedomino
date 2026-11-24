using System;
using HypeDomino.Content;
using HypeDomino.Core.Diagnostics;
using HypeDomino.Game.FriendLobby;
using HypeDomino.Game.Inventory;
using HypeDomino.Game.Lobby;
using HypeDomino.Game.Login;
using HypeDomino.Game.Medals;
using HypeDomino.Game.Missions;
using HypeDomino.Game.Pass;
using HypeDomino.Game.Privacy;
using HypeDomino.Game.Profile;
using HypeDomino.Game.Shop;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.Net.Integrity;
using Hypedomino.V1;
using UnityEngine;
using Object = UnityEngine.Object;
using Random = System.Random;

namespace HypeDomino.Game
{
    public static class NetBoot
    {
        public static AuthSession Auth { get; private set; }
        public static ApiClient Api { get; private set; }
        public static IntegrityProofs Integrity { get; private set; }
        public static Connection Connection { get; private set; }
        public static MatchQueue Queue { get; private set; }
        public static MaintenanceScreen Maintenance { get; private set; }
        public static LoginScreen Login { get; private set; }
        public static LobbyCommands Lobby { get; private set; }
        public static FriendRoom Room { get; private set; }
        public static ShopLoader Shop { get; private set; }
        public static ShopCheckout Checkout { get; private set; }
        public static PassLoader Pass { get; private set; }
        public static PassLevelBuyer PassLevels { get; private set; }
        public static MissionLoader Missions { get; private set; }
        public static InventoryLoader Inventory { get; private set; }
        public static MedalLoader Medals { get; private set; }
        public static ProfileLoader Profile { get; private set; }
        public static LanguageSync Language { get; private set; }
        public static LobbyScreen LobbyScreen { get; private set; }
        public static LobbyHeaderLoader LobbyHeaderLoader { get; private set; }

        static readonly Action<LobbyHeaderLoadOutcome> headerLoaded = HeaderLoaded;

        static NetStart start;
        static SessionStartProof startProof;
        static Uri socket;
        static NetStartPhases phases;

        [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.AfterSceneLoad)]
        static void Start()
        {
            var endpoint = ServerEndpoint.Load(Application.isEditor);
            if (endpoint == null)
            {
                Debug.LogWarning("Net no server endpoint");
                return;
            }
            socket = endpoint.SocketUri;
            phases = new NetStartPhases(StartupPhases.App);
            var http = new UnityHttpTransport(endpoint.serverUrl, endpoint.httpTimeoutSeconds);
            var identity = new NoIdentityProvider();
            Auth = new AuthSession(http, identity, new MemorySecureStore());
            Api = new ApiClient(http, Auth);
            AnalyticsBoot.Start(Api);
            Integrity = new IntegrityProofs(Api, new NoDeviceAttestor());
            startProof = new SessionStartProof(Integrity.Request, StartConnection);
            Connection = new Connection(OpenSocket, NextHello, () => Time.realtimeSinceStartupAsDouble, new Random());
            var session = Connection.Session;
            session.HelloReceived += startProof.Read;
            DeviceRiskProbe.Collect(session.DeviceRiskSignals);
            session.CollectHookSignals = DeviceRiskProbe.CollectHooks;
            AppBoot.Listen(session, AppBoot.Flow, ContentBoot.Catalog, LanguageBoot.Strings, LanguageBoot.Bundles, ContentBoot.CosmeticsUpdater, Haptics.Play, AudioBoot.PlayMatch);
            MatchBoot.Start(AppBoot.Flow, Connection);
            Queue = new MatchQueue();
            Queue.Follow(session, AppBoot.Flow);
            Lobby = new LobbyCommands();
            Room = new FriendRoom();
            session.LobbyStateReceived += state => Room.Read(state, Auth.AccountId);
            FriendLobbyBoot.Start(session, Room, Lobby, Api);
            BuildLoaders(Api);
            Checkout = new ShopCheckout(new ShopBuyer(Api), new ShopPurchase(Api, new NoStoreBilling(), Integrity.Request, AnalyticsBoot.Queue));
            Language = new LanguageSync(Api, LanguageBoot.Choice);
            session.HelloReceived += SyncLanguage;
            PrivacyBoot.Start(Api, Auth, session);
            BuildLobby(Api, Integrity, Queue, AppBoot.Flow);
            start = new NetStart(Auth, AppBoot.Flow, Dial, outcome => Debug.LogWarning($"Net resume outcome={outcome}"));
            Login = new LoginScreen(Auth, new AccountDeletion(http, identity), SignInOptions.Current, Dial);
            session.ErrorReceived += OnError;
            Maintenance = new MaintenanceScreen(Dial, () => Time.realtimeSinceStartupAsDouble);
            session.HelloReceived += Maintenance.Read;
            var driver = new GameObject(nameof(NetBoot)) { hideFlags = HideFlags.HideInHierarchy };
            Object.DontDestroyOnLoad(driver);
            driver.AddComponent<Driver>();
            Debug.Log($"Net server={endpoint.serverUrl} socket={socket}");
            StartupPhases.App.Mark(nameof(NetBoot));
            AnalyticsBoot.Follow(Auth, Queue, session);
            start.Begin();
        }

        public static ClientHello Hello(string accessToken, RuntimePlatform platform, ulong clientTimeUnixMs) => new()
        {
            AppVersion = Application.version,
            Platform = PlatformOf(platform),
            OsVersion = SystemInfo.operatingSystem,
            DeviceModel = SystemInfo.deviceModel,
            SessionToken = accessToken ?? string.Empty,
            ClientTimeUnixMs = clientTimeUnixMs,
        };

        public static Platform PlatformOf(RuntimePlatform platform) => platform switch
        {
            RuntimePlatform.Android => Platform.Android,
            RuntimePlatform.IPhonePlayer => Platform.Ios,
            _ => Platform.Unspecified,
        };

        static ClientHello NextHello()
        {
            var hello = Hello(Auth.AccessToken, Application.platform, (ulong)DateTimeOffset.UtcNow.ToUnixTimeMilliseconds());
            startProof.Attach(hello);
            return hello;
        }

        static void BuildLoaders(ApiClient api)
        {
            Shop = new ShopLoader(api);
            Pass = new PassLoader(api);
            PassLevels = new PassLevelBuyer(api);
            Missions = new MissionLoader(api);
            Inventory = new InventoryLoader(api);
            Medals = new MedalLoader(api);
            Profile = new ProfileLoader(api);
        }

        static void SyncLanguage(ServerHello hello)
        {
            if (hello.Status == HelloStatus.Accepted) Language.SignedIn(Auth.AccountId);
        }

        public static LobbyScreen NewLobbyScreen(MatchQueue queue, IntegrityProofs integrity, LobbyHeader header) => new(header, new LobbyPlay(queue, integrity.Request), queue);

        public static bool ReloadHeader(AppState state, LobbyHeaderLoader loader, Action<LobbyHeaderLoadOutcome> loaded) => state == AppState.Lobby && loader.Load(loaded);

        static void BuildLobby(ApiClient api, IntegrityProofs integrity, MatchQueue queue, AppFlow flow)
        {
            var header = new LobbyHeader();
            LobbyHeaderLoader = new LobbyHeaderLoader(api, header);
            LobbyScreen = NewLobbyScreen(queue, integrity, header);
            flow.Changed += OnFlowChanged;
        }

        static void OnFlowChanged(AppState state) => ReloadHeader(state, LobbyHeaderLoader, headerLoaded);

        static void HeaderLoaded(LobbyHeaderLoadOutcome outcome)
        {
            if (outcome != LobbyHeaderLoadOutcome.Loaded) Debug.LogWarning($"Lobby header outcome={outcome}");
        }

        static void Dial()
        {
            phases.Resumed();
            if (!Connection.Rejoin()) startProof.Start();
        }

        static void StartConnection()
        {
            phases.Proved();
            Connection.Start(socket);
        }

        static ISocketTransport OpenSocket() => phases.Dialing(new WebSocketTransport());

        static void OnError(Error error)
        {
            if (error.Code == ErrorCode.Unauthenticated) start.Unauthenticated();
            else startProof.Read(error);
        }

        sealed class Driver : MonoBehaviour
        {
            void Update()
            {
                phases.Tick();
                Connection.Tick();
                Maintenance.Tick();
            }
        }
    }
}
