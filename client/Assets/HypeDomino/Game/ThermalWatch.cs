using HypeDomino.UI.Graphics;
using UnityEngine;
using UnityEngine.Android;
using UnityEngine.Scripting;

namespace HypeDomino.Game
{
    public sealed class ThermalWatch : MonoBehaviour
    {
        const int FirstSdkWithThermalStatus = 29;
        const int AndroidModerate = 2;
        const int AndroidSevere = 3;

        sealed class ThermalListener : AndroidJavaProxy
        {
            volatile int status;

            public ThermalListener(int initialStatus) : base("android.os.PowerManager$OnThermalStatusChangedListener") => status = initialStatus;

            public int Status => status;

            [Preserve]
            public void onThermalStatusChanged(int changedStatus) => status = changedStatus;
        }

        ThermalListener listener;
        AndroidJavaObject powerManager;

        public static void StartWatching()
        {
            if (Application.platform != RuntimePlatform.Android || AndroidSdk() < FirstSdkWithThermalStatus) return;
            var watch = new GameObject(nameof(ThermalWatch)).AddComponent<ThermalWatch>();
            DontDestroyOnLoad(watch.gameObject);
        }

        void Awake()
        {
            powerManager = AndroidApplication.currentContext.Call<AndroidJavaObject>("getSystemService", "power");
            listener = new ThermalListener(powerManager.Call<int>("getCurrentThermalStatus"));
            powerManager.Call("addThermalStatusListener", listener);
        }

        void OnDestroy()
        {
            powerManager?.Call("removeThermalStatusListener", listener);
            powerManager?.Dispose();
        }

        void Update()
        {
            GraphicsBoot.Quality?.UpdateHeat(StatusOf(listener.Status), Time.realtimeSinceStartup);
        }

        static ThermalStatus StatusOf(int androidStatus)
        {
            if (androidStatus >= AndroidSevere) return ThermalStatus.Severe;
            return androidStatus == AndroidModerate ? ThermalStatus.Moderate : ThermalStatus.Normal;
        }

        static int AndroidSdk()
        {
            using var version = new AndroidJavaClass("android.os.Build$VERSION");
            return version.GetStatic<int>("SDK_INT");
        }
    }
}
