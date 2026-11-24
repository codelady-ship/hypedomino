using UnityEngine;
using UnityEngine.EventSystems;
using UnityEngine.UI;

namespace HypeDomino.UI.Audio
{
    [RequireComponent(typeof(Selectable))]
    public sealed class ClickSound : MonoBehaviour, IPointerClickHandler
    {
        Selectable selectable;

        void Awake() => selectable = GetComponent<Selectable>();

        public void OnPointerClick(PointerEventData eventData)
        {
            if (eventData.button != PointerEventData.InputButton.Left || !selectable.IsInteractable()) return;
            var player = SoundPlayer.Main;
            if (player != null) player.Play(Sound.UiClick);
        }
    }
}
