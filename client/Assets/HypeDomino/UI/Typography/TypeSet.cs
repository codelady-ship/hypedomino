using System;
using HypeDomino.UI.Tokens;
using TMPro;
using UnityEngine;
using UnityEngine.TextCore;

namespace HypeDomino.UI.Typography
{
    public sealed class TypeSet : ScriptableObject
    {
        public const string ResourceName = "TypeSet";
        const float LineSpacingUnitsPerEm = 100f;

        [Serializable]
        public struct Face
        {
            public TokenFontFamily family;
            public int weight;
            public TMP_FontAsset font;
        }

        [SerializeField] Face[] faces = Array.Empty<Face>();

        static TypeSet loaded;

        public static TypeSet Loaded => loaded != null ? loaded : loaded = Resources.Load<TypeSet>(ResourceName);

        public int FaceCount => faces.Length;

        public Face FaceAt(int index) => faces[index];

        public TMP_FontAsset FontOf(TokenTextStyle style)
        {
            var at = (int)style;
            return FontOf(TokenTypography.FamilyOfStyle[at], TokenTypography.WeightOfStyle[at]);
        }

        public TMP_FontAsset FontOf(TokenFontFamily family, int weight)
        {
            foreach (var face in faces)
                if (face.family == family && face.weight == weight) return face.font;
            return null;
        }

        public void Apply(TMP_Text text, TokenTextStyle style)
        {
            var at = (int)style;
            var font = FontOf(style);
            text.font = font;
            text.fontSize = TokenTypography.SizeOfStyle[at];
            text.lineSpacing = LineSpacing(font.faceInfo, TokenTypography.LineHeightOfStyle[at]);
        }

        public static float LineSpacing(FaceInfo face, float lineHeight) => (lineHeight - face.lineHeight / face.pointSize) * LineSpacingUnitsPerEm;
    }
}
