using System;
using HypeDomino.UI.Motion;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Cosmetics
{
    [RequireComponent(typeof(AnimatedImage))]
    public sealed class CosmeticImage : MonoBehaviour
    {
        AnimatedImage player;
        CosmeticStage stage;
        CosmeticArt art;
        Sprite still;
        LoopArt loop;
        bool holdsStill;
        bool holdsLoop;
        int showTicket;
        int loopTicket;

        public event Action Rested;

        public bool IsPlaying => Player.IsPlaying;

        public Sprite Shown => Player.Shown;

        AnimatedImage Player => player != null ? player : player = GetComponent<AnimatedImage>();

        public void Show(CosmeticStage onStage, CosmeticArt item, Sprite builtIn)
        {
            if (ReferenceEquals(stage, onStage) && holdsStill && art.IsSameAs(item))
            {
                Present();
                return;
            }
            ReleaseHeld();
            stage = onStage;
            art = item;
            still = null;
            showTicket++;
            Rest(builtIn);
            if (string.IsNullOrEmpty(item.StaticAddress)) return;
            var ticket = showTicket;
            holdsStill = true;
            stage.Stills.Acquire(item.StaticAddress, sprite => OnStill(ticket, sprite));
        }

        public void Refresh()
        {
            if (stage != null) Present();
        }

        void OnEnable() => Refresh();

        void OnDestroy() => ReleaseHeld();

        void OnStill(int ticket, Sprite sprite)
        {
            if (ticket != showTicket || sprite == null) return;
            still = sprite;
            Rest(sprite);
            Present();
        }

        void OnLoop(int ticket, LoopArt loaded)
        {
            if (ticket != loopTicket || loaded == null) return;
            loop = loaded;
            Present();
        }

        void Present()
        {
            if (still == null) return;
            if (!art.IsAnimated || !stage.AllowsAnimation)
            {
                DropLoop();
                Rest(still);
                return;
            }
            if (loop != null)
            {
                Player.Show(loop, stage.Level, stage.ReduceMotion, stage.Budget, Time.unscaledTime);
                return;
            }
            if (!holdsLoop && stage.Budget.HasRoom) AcquireLoop();
        }

        void AcquireLoop()
        {
            var ticket = ++loopTicket;
            holdsLoop = true;
            stage.Loops.Acquire(art.AnimationAddress, loaded => OnLoop(ticket, loaded));
        }

        void DropLoop()
        {
            loopTicket++;
            loop = null;
            if (!holdsLoop) return;
            holdsLoop = false;
            stage.Loops.Release(art.AnimationAddress);
        }

        void ReleaseHeld()
        {
            if (stage == null) return;
            DropLoop();
            if (!holdsStill) return;
            holdsStill = false;
            stage.Stills.Release(art.StaticAddress);
        }

        void Rest(Sprite sprite)
        {
            Player.ShowStill(sprite);
            GetComponent<Image>().enabled = sprite != null;
            Rested?.Invoke();
        }
    }
}
