using System.Collections.Generic;
using HypeDomino.UI.Lists;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class RecyclingListTests
    {
        const float RowHeight = 100f;
        const float Gap = 20f;
        const float ViewportHeight = 1000f;
        const int Items = 1000;

        sealed class CountingBinder : IRowBinder
        {
            public readonly Dictionary<RectTransform, int> Shown = new();
            public int Binds;

            public void Bind(RectTransform row, int index)
            {
                Binds++;
                Shown[row] = index;
            }
        }

        GameObject root;
        ScrollRect scroll;
        RecyclingList list;
        CountingBinder binder;

        [SetUp]
        public void BuildList()
        {
            root = new GameObject("Canvas", typeof(Canvas));
            var holder = new GameObject("List", typeof(RectTransform), typeof(ScrollRect));
            holder.transform.SetParent(root.transform, false);
            ((RectTransform)holder.transform).sizeDelta = new Vector2(600f, ViewportHeight);
            var viewport = Stretched("Viewport", holder.transform);
            viewport.gameObject.AddComponent<RectMask2D>();
            var content = Stretched("Content", viewport);
            scroll = holder.GetComponent<ScrollRect>();
            scroll.viewport = viewport;
            scroll.content = content;
            scroll.horizontal = false;
            var template = new GameObject("Row", typeof(RectTransform)).GetComponent<RectTransform>();
            template.SetParent(root.transform, false);
            list = holder.AddComponent<RecyclingList>();
            list.Build(template, RowHeight, Gap);
            binder = new CountingBinder();
            list.Show(Items, binder);
        }

        [TearDown]
        public void DestroyList() => Object.Destroy(root);

        static RectTransform Stretched(string name, Transform parent)
        {
            var rect = new GameObject(name, typeof(RectTransform)).GetComponent<RectTransform>();
            rect.SetParent(parent, false);
            rect.anchorMin = Vector2.zero;
            rect.anchorMax = Vector2.one;
            rect.sizeDelta = Vector2.zero;
            return rect;
        }

        void ScrollTo(float offset)
        {
            scroll.content.anchoredPosition = new Vector2(0f, offset);
            scroll.onValueChanged.Invoke(Vector2.zero);
        }

        int ActiveRows()
        {
            var active = 0;
            foreach (Transform row in scroll.content) if (row.gameObject.activeSelf) active++;
            return active;
        }

        [Test]
        public void AThousandItemsUseOnlyTheRowsThatCoverTheViewport()
        {
            Assert.AreEqual(12, list.RowCount);
            Assert.AreEqual(12, scroll.content.childCount);
            Assert.AreEqual(12, ActiveRows());
            Assert.AreEqual(Items * (RowHeight + Gap) - Gap, scroll.content.sizeDelta.y);
        }

        [Test]
        public void TheScrollingContentIsItsOwnCanvasWithoutLayout()
        {
            Assert.IsNotNull(scroll.content.GetComponent<Canvas>());
            Assert.IsNotNull(scroll.content.GetComponent<GraphicRaycaster>());
            Assert.IsNull(scroll.content.GetComponent<LayoutGroup>());
            Assert.IsNull(scroll.content.GetComponent<ContentSizeFitter>());
        }

        [Test]
        public void ScrollingOneRowRebindsOneRowAndCreatesNone()
        {
            ScrollTo(240f);
            var before = binder.Binds;
            ScrollTo(360f);
            Assert.AreEqual(before + 1, binder.Binds);
            Assert.AreEqual(12, scroll.content.childCount);
        }

        [Test]
        public void EachShownRowSitsWhereItsItemBelongs()
        {
            ScrollTo(4900f);
            foreach (Transform child in scroll.content)
            {
                var row = (RectTransform)child;
                if (!row.gameObject.activeSelf) continue;
                Assert.AreEqual(-binder.Shown[row] * (RowHeight + Gap), row.anchoredPosition.y, 0.001f);
            }
        }

        [Test]
        public void RowsPastTheLastItemAreHidden()
        {
            list.Show(3, binder);
            Assert.AreEqual(3, ActiveRows());
        }

        [Test]
        public void RefreshRebindsOnlyAVisibleItem()
        {
            var before = binder.Binds;
            list.Refresh(2);
            list.Refresh(500);
            Assert.AreEqual(before + 1, binder.Binds);
        }

        [Test]
        public void ScrollingAllocatesNothing()
        {
            ScrollTo(120f);
            ScrollTo(0f);
            Assert.That(() =>
            {
                for (var offset = 0f; offset < 60_000f; offset += 37f) ScrollTo(offset);
            }, Allocates.Nothing);
        }
    }
}
