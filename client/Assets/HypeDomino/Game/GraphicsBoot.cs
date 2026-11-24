using System;
using System.Collections.Generic;
using HypeDomino.Core.Diagnostics;
using HypeDomino.UI.Graphics;
using UnityEngine;
using UnityEngine.Rendering;
using UnityEngine.Rendering.Universal;

namespace HypeDomino.Game
{
    public static class GraphicsBoot
    {
        const string ChoiceKey = "graphics.choice";
        const string ProbeDropsKey = "graphics.probeDrops";
        const string ProbeSettledKey = "graphics.probeSettled";
        const string ProbeVersionKey = "graphics.probeVersion";
        const string FrameRateKey = "graphics.frameRate";
        const string BatterySaverKey = "graphics.batterySaver";
        const int AutoChoice = -1;

        static GraphicsLevel? savedChoice;
        static int savedProbeDrops;
        static int? savedFrameRate;
        static bool savedBatterySaver;

        public static GraphicsQuality Quality { get; private set; }

        public static FrameRatePolicy FrameRate { get; private set; }

        [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.BeforeSceneLoad)]
        static void Start()
        {
            var refreshRatesHz = RefreshRatesHz();
            var device = new DeviceFacts(SystemInfo.systemMemorySize, SystemInfo.graphicsDeviceName, Highest(refreshRatesHz));
            var sameVersion = PlayerPrefs.GetString(ProbeVersionKey) == Application.version;
            Quality = new GraphicsQuality(
                DeviceClass.TierOf(device),
                LoadChoice(),
                sameVersion ? PlayerPrefs.GetInt(ProbeDropsKey, 0) : 0,
                sameVersion && PlayerPrefs.GetInt(ProbeSettledKey, 0) == 1);
            savedChoice = Quality.Choice;
            savedProbeDrops = Quality.ProbeDrops;
            Quality.Changed += OnChanged;
            savedFrameRate = LoadFrameRate();
            savedBatterySaver = PlayerPrefs.GetInt(BatterySaverKey, 0) == 1;
            FrameRate = new FrameRatePolicy(Quality, refreshRatesHz, savedFrameRate, savedBatterySaver);
            FrameRate.Changed += OnFrameRateChanged;
            ThermalWatch.StartWatching();
            PowerSaveWatch.StartWatching();
            var renderScale = ApplyRenderScale();
            Application.targetFrameRate = FrameRate.Target;
            Debug.Log($"Graphics tier={Quality.Tier} level={Quality.Level} effects={Quality.Effects} renderScale={renderScale:F2} probeDrops={Quality.ProbeDrops} " +
                      $"targetFps={FrameRate.Target} highestFps={FrameRate.Highest} memoryMb={device.MemoryMb} gpu=\"{device.GpuName}\" " +
                      $"refreshHz={string.Join("/", refreshRatesHz)} shortSide={NativeShortSide()}");
            StartupPhases.App.Mark(nameof(GraphicsBoot));
        }

        public static void SettleProbe(double workTimeP95Ms)
        {
            var gateMs = Quality.WorkTimeGateMs;
            var result = Quality.SettleProbe(workTimeP95Ms);
            SaveState();
            Debug.Log($"GraphicsProbe workP95={workTimeP95Ms:F2}ms gate={gateMs:F2}ms result={result} level={Quality.Level} probeDrops={Quality.ProbeDrops}");
        }

        static void OnChanged()
        {
            if (Quality.Choice != savedChoice || Quality.ProbeDrops != savedProbeDrops) SaveState();
            var renderScale = ApplyRenderScale();
            Debug.Log($"Graphics level={Quality.Level} effects={Quality.Effects} renderScale={renderScale:F2} heat={Quality.Heat} probeDrops={Quality.ProbeDrops}");
        }

        static void SaveState()
        {
            savedChoice = Quality.Choice;
            savedProbeDrops = Quality.ProbeDrops;
            PlayerPrefs.SetInt(ChoiceKey, Quality.Choice.HasValue ? (int)Quality.Choice.Value : AutoChoice);
            PlayerPrefs.SetInt(ProbeDropsKey, Quality.ProbeDrops);
            PlayerPrefs.SetInt(ProbeSettledKey, Quality.ProbeSettled ? 1 : 0);
            PlayerPrefs.SetString(ProbeVersionKey, Application.version);
            PlayerPrefs.Save();
        }

        static void OnFrameRateChanged()
        {
            if (FrameRate.Choice != savedFrameRate || FrameRate.BatterySaver != savedBatterySaver) SaveFrameRate();
            if (Application.targetFrameRate == FrameRate.Target) return;
            Application.targetFrameRate = FrameRate.Target;
            Debug.Log($"Graphics targetFps={FrameRate.Target} inMatch={FrameRate.InMatch} batterySaver={FrameRate.BatterySaver} systemSaver={FrameRate.SystemSaver} level={Quality.Level}");
        }

        static void SaveFrameRate()
        {
            savedFrameRate = FrameRate.Choice;
            savedBatterySaver = FrameRate.BatterySaver;
            PlayerPrefs.SetInt(FrameRateKey, FrameRate.Choice ?? AutoChoice);
            PlayerPrefs.SetInt(BatterySaverKey, FrameRate.BatterySaver ? 1 : 0);
            PlayerPrefs.Save();
        }

        static int? LoadFrameRate()
        {
            var saved = PlayerPrefs.GetInt(FrameRateKey, AutoChoice);
            return saved > 0 ? saved : null;
        }

        static GraphicsLevel? LoadChoice()
        {
            var saved = PlayerPrefs.GetInt(ChoiceKey, AutoChoice);
            if (saved < (int)GraphicsLevel.Low || saved > (int)GraphicsLevel.Ultra) return null;
            return (GraphicsLevel)saved;
        }

        static float ApplyRenderScale()
        {
            var renderScale = Quality.RenderScale(NativeShortSide());
            if (Application.isEditor) return renderScale;
            if (GraphicsSettings.currentRenderPipeline is UniversalRenderPipelineAsset pipeline) pipeline.renderScale = renderScale;
            return renderScale;
        }

        static int NativeShortSide() => Math.Min(Display.main.systemWidth, Display.main.systemHeight);

        static int[] RefreshRatesHz()
        {
            var rates = new List<int>();
            AddRate(rates, Screen.currentResolution);
            foreach (var resolution in Screen.resolutions) AddRate(rates, resolution);
            rates.Sort();
            return rates.ToArray();
        }

        static void AddRate(List<int> rates, Resolution resolution)
        {
            var rate = (int)Math.Round(resolution.refreshRateRatio.value);
            if (rate > 0 && !rates.Contains(rate)) rates.Add(rate);
        }

        static int Highest(int[] rates) => rates.Length == 0 ? 0 : rates[rates.Length - 1];
    }
}
