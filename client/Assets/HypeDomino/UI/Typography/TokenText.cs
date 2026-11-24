using HypeDomino.UI.Tokens;
using TMPro;
using UnityEngine;

namespace HypeDomino.UI.Typography
{
    [RequireComponent(typeof(TMP_Text))]
    public sealed class TokenText : MonoBehaviour
    {
        [SerializeField] TokenTextStyle style;

        public TokenTextStyle Style => style;

        void Awake() => TypeSet.Loaded.Apply(GetComponent<TMP_Text>(), style);
    }
}
