using System;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Settings
{
    public sealed class SettingToggleRow : MonoBehaviour
    {
        [SerializeField] Toggle toggle;

        Action<bool> choose;

        public void Configure(Toggle onOff)
        {
            Unbind();
            toggle = onOff;
        }

        public void Bind(bool on, Action<bool> chosen)
        {
            Unbind();
            choose = chosen;
            toggle.SetIsOnWithoutNotify(on);
            toggle.onValueChanged.AddListener(OnToggled);
        }

        public void Show(bool on) => toggle.SetIsOnWithoutNotify(on);

        void OnToggled(bool on) => choose(on);

        void Unbind()
        {
            if (choose == null) return;
            toggle.onValueChanged.RemoveListener(OnToggled);
            choose = null;
        }

        void OnDestroy() => Unbind();
    }
}
