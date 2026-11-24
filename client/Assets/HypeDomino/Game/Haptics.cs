using System;
using System.Runtime.InteropServices;
using HypeDomino.Game.Match;
using UnityEngine;
using UnityEngine.Android;

namespace HypeDomino.Game
{
    public static class Haptics
    {
        const string EnabledKey = "haptics.enabled";
        const int AndroidVirtualKey = 1;
        const int AndroidLongPress = 0;
        const int AndroidConfirm = 16;
        const int FirstSdkWithConfirm = 30;

        static HapticsPreference preference;
        static AndroidView androidView;

        public static HapticsPreference Preference => preference ??= Load();

        public static void Play(HapticCue cue)
        {
            if (cue == HapticCue.None || !Preference.Enabled || Application.isEditor) return;
            if (Application.platform == RuntimePlatform.Android) PlayAndroid(cue);
            if (Application.platform == RuntimePlatform.IPhonePlayer) PlayIos(cue);
        }

        static HapticsPreference Load()
        {
            var loaded = new HapticsPreference(PlayerPrefs.GetInt(EnabledKey, 1) == 1);
            loaded.Changed += Save;
            return loaded;
        }

        static void Save()
        {
            PlayerPrefs.SetInt(EnabledKey, preference.Enabled ? 1 : 0);
            PlayerPrefs.Save();
        }

        static void PlayAndroid(HapticCue cue)
        {
            androidView ??= new AndroidView();
            androidView.Perform(cue switch
            {
                HapticCue.TilePlace => AndroidVirtualKey,
                HapticCue.Score => androidView.Sdk >= FirstSdkWithConfirm ? AndroidConfirm : AndroidVirtualKey,
                _ => AndroidLongPress,
            });
        }

#if UNITY_IOS && !UNITY_EDITOR
        [DllImport("__Internal")]
        static extern void HypeDominoPlayHaptic(int cue);

        static void PlayIos(HapticCue cue) => HypeDominoPlayHaptic((int)cue);
#else
        static void PlayIos(HapticCue cue)
        {
        }
#endif

        sealed class AndroidView
        {
            readonly IntPtr view;
            readonly IntPtr performHapticFeedback;
            readonly jvalue[] constant = new jvalue[1];

            public readonly int Sdk;

            public AndroidView()
            {
                using var version = new AndroidJavaClass("android.os.Build$VERSION");
                Sdk = version.GetStatic<int>("SDK_INT");
                using var window = AndroidApplication.currentActivity.Call<AndroidJavaObject>("getWindow");
                using var decorView = window.Call<AndroidJavaObject>("getDecorView");
                view = AndroidJNI.NewGlobalRef(decorView.GetRawObject());
                var viewClass = AndroidJNI.GetObjectClass(view);
                performHapticFeedback = AndroidJNI.GetMethodID(viewClass, "performHapticFeedback", "(I)Z");
                AndroidJNI.DeleteLocalRef(viewClass);
            }

            public void Perform(int feedbackConstant)
            {
                constant[0].i = feedbackConstant;
                AndroidJNI.CallBooleanMethod(view, performHapticFeedback, constant);
            }
        }
    }
}
