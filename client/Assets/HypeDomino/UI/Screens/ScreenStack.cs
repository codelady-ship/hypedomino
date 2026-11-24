using System.Collections.Generic;
using UnityEngine;

namespace HypeDomino.UI.Screens
{
    public sealed class ScreenStack : MonoBehaviour
    {
        const int SortingStep = 10;
        const float MillisecondsPerSecond = 1000f;

        readonly List<ScreenView> stack = new(8);
        readonly List<ScreenView> moving = new(8);

        public bool ReduceMotion { get; set; }

        public int Count => stack.Count;

        public ScreenView Top => stack.Count == 0 ? null : stack[stack.Count - 1];

        public void Push(ScreenView screen)
        {
            if (stack.Contains(screen)) return;
            Top?.TakeInput(false);
            stack.Add(screen);
            screen.Show(stack.Count * SortingStep);
            screen.TakeInput(true);
            screen.Enter();
            Track(screen);
        }

        public bool Pop()
        {
            if (stack.Count <= 1) return false;
            var leaving = Top;
            stack.RemoveAt(stack.Count - 1);
            leaving.TakeInput(false);
            leaving.Exit();
            Track(leaving);
            RevealTop();
            return true;
        }

        public void Advance(float deltaMs)
        {
            for (var at = moving.Count - 1; at >= 0; at--)
            {
                var screen = moving[at];
                if (!screen.Advance(deltaMs, ReduceMotion)) continue;
                moving.RemoveAt(at);
                Settle(screen);
            }
        }

        void Update()
        {
            if (Input.GetKeyDown(KeyCode.Escape)) Pop();
            Advance(Time.unscaledDeltaTime * MillisecondsPerSecond);
        }

        void Track(ScreenView screen)
        {
            if (!moving.Contains(screen)) moving.Add(screen);
        }

        void Settle(ScreenView screen)
        {
            if (!stack.Contains(screen)) screen.Hide();
            HideCovered();
        }

        void HideCovered()
        {
            var cover = stack.Count - 1;
            while (cover >= 0 && (stack[cover].IsModal || moving.Contains(stack[cover]))) cover--;
            for (var below = 0; below < cover; below++)
                if (!moving.Contains(stack[below])) stack[below].Hide();
        }

        void RevealTop()
        {
            for (var at = stack.Count - 1; at >= 0; at--)
            {
                stack[at].Show((at + 1) * SortingStep);
                if (!stack[at].IsModal) break;
            }
            Top.TakeInput(true);
        }
    }
}
