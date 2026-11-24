using System.Runtime.InteropServices;
using HypeDomino.Core.Diagnostics;
using HypeDomino.UI.Motion;
using UnityEngine;
using UnityEngine.Android;

namespace HypeDomino.Game
{
    public static class MotionBoot
    {
        const string ReduceKey = "motion.reduce";
        const int FollowSystem = -1;
        const string AnimatorDurationScale = "animator_duration_scale";
        const float DefaultAnimatorDurationScale = 1f;

        static bool? savedChoice;

        public static MotionPreference Motion { get; private set; }

        [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.BeforeSceneLoad)]
        static void Start()
        {
            Build();
            StartupPhases.App.Mark(nameof(MotionBoot));
        }

        static void Build()
        {
            savedChoice = LoadChoice();
            Motion = new MotionPreference(savedChoice, SystemReduce());
            Motion.Changed += OnChanged;
            Debug.Log($"Motion reduce={Motion.ReduceMotion} choice={Motion.Choice?.ToString() ?? "system"} system={Motion.SystemReduce}");
            if (Application.isEditor) return;
            var driver = new GameObject(nameof(MotionBoot)) { hideFlags = HideFlags.HideInHierarchy };
            Object.DontDestroyOnLoad(driver);
            driver.AddComponent<Driver>();
        }

        static void OnChanged()
        {
            if (Motion.Choice != savedChoice) SaveChoice();
            Debug.Log($"Motion reduce={Motion.ReduceMotion} choice={Motion.Choice?.ToString() ?? "system"} system={Motion.SystemReduce}");
        }

        static void SaveChoice()
        {
            savedChoice = Motion.Choice;
            PlayerPrefs.SetInt(ReduceKey, savedChoice.HasValue ? (savedChoice.Value ? 1 : 0) : FollowSystem);
            PlayerPrefs.Save();
        }

        static bool? LoadChoice()
        {
            var saved = PlayerPrefs.GetInt(ReduceKey, FollowSystem);
            if (saved == FollowSystem) return null;
            return saved == 1;
        }

        static bool SystemReduce()
        {
            if (Application.isEditor) return false;
            if (Application.platform == RuntimePlatform.Android) return AndroidAnimationsOff();
            if (Application.platform == RuntimePlatform.IPhonePlayer) return IosReduceMotion();
            return false;
        }

        static bool AndroidAnimationsOff()
        {
            using var global = new AndroidJavaClass("android.provider.Settings$Global");
            using var resolver = AndroidApplication.currentContext.Call<AndroidJavaObject>("getContentResolver");
            return global.CallStatic<float>("getFloat", resolver, AnimatorDurationScale, DefaultAnimatorDurationScale) == 0f;
        }

#if UNITY_IOS && !UNITY_EDITOR
        [DllImport("__Internal")]
        static extern int HypeDominoReduceMotion();

        static bool IosReduceMotion() => HypeDominoReduceMotion() != 0;
#else
        static bool IosReduceMotion() => false;
#endif

        sealed class Driver : MonoBehaviour
        {
            void OnApplicationFocus(bool hasFocus)
            {
                if (hasFocus) Motion.SetSystemReduce(SystemReduce());
            }
        }
    }
}
