using System;
using HypeDomino.UI.Tokens;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Board
{
    public sealed class BoardTilesView : MonoBehaviour
    {
        [SerializeField] TileFaceSet faceSet;
        [SerializeField] Image[] tiles = Array.Empty<Image>();
        [SerializeField] Sprite shadowSprite;
        [SerializeField] Image[] shadows = Array.Empty<Image>();

        public int Count { get; private set; }

        public int Capacity => tiles.Length;

        public void Configure(TileFaceSet faces, Image[] tileImages, Sprite shadow, Image[] shadowImages)
        {
            faceSet = faces;
            tiles = tileImages ?? Array.Empty<Image>();
            shadowSprite = shadow;
            shadows = shadowImages ?? Array.Empty<Image>();
            if (shadows.Length != tiles.Length) throw new ArgumentException($"{shadows.Length} shadows for {tiles.Length} tiles", nameof(shadowImages));
            Color restingShadow = TokenColor.Shadow;
            restingShadow.a = TokenElevation.E1Alpha;
            for (var i = 0; i < tiles.Length; i++)
            {
                tiles[i].raycastTarget = false;
                shadows[i].raycastTarget = false;
                shadows[i].sprite = shadowSprite;
                shadows[i].color = restingShadow;
            }
        }

        public void Show(ChainLayout layout)
        {
            Count = Mathf.Min(layout.Count, tiles.Length);
            for (var i = 0; i < tiles.Length; i++)
            {
                var shown = i < Count;
                tiles[i].enabled = shown;
                shadows[i].enabled = shown;
                if (shown) Place(i, layout[i], layout.OnBoard(layout[i].Center), layout.Scale);
            }
        }

        void Place(int slot, PlacedDomino tile, Vector2 position, float shortSide)
        {
            var face = faceSet.Face(new Domino(tile.Near, tile.Far));
            var rotation = Quaternion.Euler(0f, 0f, TileFaces.Angle(tile));
            var image = tiles[slot];
            image.sprite = face;
            var rect = image.rectTransform;
            rect.anchoredPosition = position;
            rect.sizeDelta = new Vector2(shortSide, shortSide * 2f);
            rect.localRotation = rotation;
            var shadow = shadows[slot].rectTransform;
            shadow.anchoredPosition = new Vector2(position.x, position.y - TokenElevation.E1OffsetY);
            shadow.sizeDelta = shadowSprite.rect.size * (shortSide / face.rect.width);
            shadow.localRotation = rotation;
        }
    }
}
