using System.Runtime.InteropServices;
using UnityEngine;
using UnityEngine.Android;

namespace HypeDomino.Game
{
    public sealed class PowerSaveWatch : MonoBehaviour
    {
        AndroidJavaObject powerManager;

        public static bool Watches(RuntimePlatform platform) =>
            platform == RuntimePlatform.Android || platform == RuntimePlatform.IPhonePlayer;

        public static void StartWatching()
        {
            if (!Watches(Application.platform)) return;
            var watch = new GameObject(nameof(PowerSaveWatch)).AddComponent<PowerSaveWatch>();
            DontDestroyOnLoad(watch.gameObject);
        }

        void Awake()
        {
            if (Application.platform == RuntimePlatform.Android)
                powerManager = AndroidApplication.currentContext.Call<AndroidJavaObject>("getSystemService", "power");
            ReadSystemSaver();
        }

        void OnApplicationFocus(bool hasFocus)
        {
            if (hasFocus) ReadSystemSaver();
        }

        void OnDestroy() => powerManager?.Dispose();

        void ReadSystemSaver() => GraphicsBoot.FrameRate?.SetSystemSaver(SystemSaverOn());

        bool SystemSaverOn() => powerManager != null ? powerManager.Call<bool>("isPowerSaveMode") : IosLowPowerMode();

#if UNITY_IOS && !UNITY_EDITOR
        [DllImport("__Internal")]
        static extern int HypeDominoLowPowerMode();

        static bool IosLowPowerMode() => HypeDominoLowPowerMode() != 0;
#else
        static bool IosLowPowerMode() => false;
#endif
    }
}
