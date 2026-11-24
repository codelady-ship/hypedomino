using HypeDomino.UI.Audio;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Settings
{
    public sealed class VolumeSliderRow : MonoBehaviour
    {
        [SerializeField] Slider slider;
        [SerializeField] AudioGroup group;

        AudioVolumes volumes;

        public void Configure(Slider levelSlider, AudioGroup audioGroup)
        {
            Unbind();
            slider = levelSlider;
            group = audioGroup;
        }

        public void Bind(AudioVolumes audioVolumes)
        {
            Unbind();
            volumes = audioVolumes;
            slider.minValue = 0f;
            slider.maxValue = AudioVolumes.Full;
            slider.SetValueWithoutNotify(volumes.Level(group));
            slider.onValueChanged.AddListener(OnSlid);
        }

        void OnSlid(float level) => volumes.Set(group, level);

        void Unbind()
        {
            if (volumes == null) return;
            slider.onValueChanged.RemoveListener(OnSlid);
            volumes = null;
        }

        void OnDestroy() => Unbind();
    }
}
