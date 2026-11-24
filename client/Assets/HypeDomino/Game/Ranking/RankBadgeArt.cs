using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Game.Ranking
{
    public static class RankBadgeArt
    {
        public static void Show(Image image, Sprite[] tierSprites, RankBadge badge)
        {
            var tier = (int)badge.Tier;
            var sprite = badge.Placed && tierSprites != null && tier < tierSprites.Length ? tierSprites[tier] : null;
            image.sprite = sprite;
            image.enabled = sprite != null;
        }
    }
}
