using UnityEngine;

namespace HypeDomino.UI.Lists
{
    public interface IRowBinder
    {
        void Bind(RectTransform row, int index);
    }
}
