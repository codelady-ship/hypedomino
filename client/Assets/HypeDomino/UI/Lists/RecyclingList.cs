using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Lists
{
    [RequireComponent(typeof(ScrollRect))]
    public sealed class RecyclingList : MonoBehaviour
    {
        const int Unbound = -1;

        ScrollRect scroll;
        RectTransform content;
        RecyclingLayout layout;
        RectTransform[] rows;
        int[] boundIndex;
        IRowBinder binder;
        int itemCount;

        public int RowCount => rows?.Length ?? 0;

        public void Build(RectTransform rowTemplate, float rowHeight, float gap)
        {
            scroll = GetComponent<ScrollRect>();
            content = scroll.content;
            EnsureOwnCanvas(content.gameObject);
            PinToTop(content);
            layout = new RecyclingLayout(rowHeight, gap, scroll.viewport.rect.height);
            rows = new RectTransform[layout.RowCount];
            boundIndex = new int[layout.RowCount];
            for (var slot = 0; slot < rows.Length; slot++)
            {
                rows[slot] = Instantiate(rowTemplate, content);
                PinToTop(rows[slot]);
                rows[slot].sizeDelta = new Vector2(0f, rowHeight);
                rows[slot].gameObject.SetActive(false);
                boundIndex[slot] = Unbound;
            }
            scroll.onValueChanged.AddListener(OnScrolled);
        }

        public void Show(int count, IRowBinder rowBinder)
        {
            binder = rowBinder;
            itemCount = count;
            content.sizeDelta = new Vector2(content.sizeDelta.x, layout.ContentHeight(count));
            for (var slot = 0; slot < boundIndex.Length; slot++) boundIndex[slot] = Unbound;
            Place();
        }

        public void Refresh(int index)
        {
            var slot = index % rows.Length;
            if (boundIndex[slot] == index) binder.Bind(rows[slot], index);
        }

        void OnDestroy()
        {
            if (scroll != null) scroll.onValueChanged.RemoveListener(OnScrolled);
        }

        void OnScrolled(Vector2 normalized) => Place();

        void Place()
        {
            var first = layout.FirstRow(content.anchoredPosition.y, itemCount);
            for (var offset = 0; offset < rows.Length; offset++)
            {
                var index = first + offset;
                var slot = index % rows.Length;
                var row = rows[slot];
                if (index >= itemCount)
                {
                    if (row.gameObject.activeSelf) row.gameObject.SetActive(false);
                    boundIndex[slot] = Unbound;
                    continue;
                }
                if (boundIndex[slot] != index)
                {
                    row.anchoredPosition = new Vector2(0f, -layout.RowTop(index));
                    binder.Bind(row, index);
                    boundIndex[slot] = index;
                }
                if (!row.gameObject.activeSelf) row.gameObject.SetActive(true);
            }
        }

        static void EnsureOwnCanvas(GameObject target)
        {
            if (!target.TryGetComponent<Canvas>(out _)) target.AddComponent<Canvas>();
            if (!target.TryGetComponent<GraphicRaycaster>(out _)) target.AddComponent<GraphicRaycaster>();
        }

        static void PinToTop(RectTransform rect)
        {
            rect.anchorMin = new Vector2(0f, 1f);
            rect.anchorMax = new Vector2(1f, 1f);
            rect.pivot = new Vector2(0.5f, 1f);
        }
    }
}
