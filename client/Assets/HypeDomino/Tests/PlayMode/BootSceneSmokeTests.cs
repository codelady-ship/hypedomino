using System.Collections;
using HypeDomino.Core.Diagnostics;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.SceneManagement;
using UnityEngine.TestTools;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class BootSceneSmokeTests
    {
        const string BootScene = "Boot";
        const int FramesToRun = 60;

        [UnityTearDown]
        public IEnumerator UnloadBoot()
        {
            var boot = SceneManager.GetSceneByName(BootScene);
            if (boot.isLoaded) yield return SceneManager.UnloadSceneAsync(boot);
        }

        [UnityTest]
        public IEnumerator TheFirstBuildSceneLoadsAndRunsWithoutErrors()
        {
            Assert.AreEqual(BootScene, System.IO.Path.GetFileNameWithoutExtension(SceneUtility.GetScenePathByBuildIndex(0)));

            yield return SceneManager.LoadSceneAsync(BootScene, LoadSceneMode.Additive);
            for (var frame = 0; frame < FramesToRun; frame++) yield return null;

            var boot = SceneManager.GetSceneByName(BootScene);
            Assert.IsTrue(boot.isLoaded);
            Camera camera = null;
            FrameTimingSampler sampler = null;
            foreach (var root in boot.GetRootGameObjects())
            {
                if (camera == null) camera = root.GetComponentInChildren<Camera>();
                if (sampler == null) sampler = root.GetComponentInChildren<FrameTimingSampler>();
            }
            Assert.IsNotNull(camera, "Boot has no camera");
            Assert.IsTrue(camera.isActiveAndEnabled);
            Assert.IsNotNull(sampler, "Boot has no FrameTimingSampler");
            Assert.IsTrue(sampler.isActiveAndEnabled);
        }
    }
}
