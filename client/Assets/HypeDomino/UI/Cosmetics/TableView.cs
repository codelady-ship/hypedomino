using HypeDomino.UI.Motion;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Cosmetics
{
    public sealed class TableView : MonoBehaviour
    {
        [SerializeField] Image flat;
        [SerializeField] AnimatedImage[] layers;
        [SerializeField] Sprite builtIn;

        CosmeticStage stage;
        TableLook look;
        Sprite still;
        LoopArt[] loaded;
        int loadedCount;
        bool holdsStill;
        bool holdsLayers;
        bool layerFailed;
        bool acquiring;
        int showTicket;
        int layerTicket;

        public bool IsComposed { get; private set; }

        public Sprite Flat => flat.enabled ? flat.sprite : null;

        public int Capacity => layers.Length;

        public AnimatedImage Layer(int index) => layers[index];

        public void Configure(Image flatImage, AnimatedImage[] layerImages, Sprite builtInTable)
        {
            flat = flatImage;
            layers = layerImages;
            builtIn = builtInTable;
            HideLayers();
        }

        public void Show(CosmeticStage onStage, TableLook table)
        {
            if (ReferenceEquals(stage, onStage) && holdsStill && look.IsSameAs(table))
            {
                Present();
                return;
            }
            ReleaseHeld();
            stage = onStage;
            look = table;
            still = null;
            layerFailed = false;
            showTicket++;
            ShowFlat(builtIn);
            if (string.IsNullOrEmpty(table.StaticAddress)) return;
            var ticket = showTicket;
            holdsStill = true;
            stage.Stills.Acquire(table.StaticAddress, sprite => OnStill(ticket, sprite));
        }

        public void RefreshEffects()
        {
            if (stage != null) Present();
        }

        void OnDestroy() => ReleaseHeld();

        void OnStill(int ticket, Sprite sprite)
        {
            if (ticket != showTicket || sprite == null) return;
            still = sprite;
            if (!IsComposed) ShowFlat(sprite);
            Present();
        }

        void Present()
        {
            if (still == null) return;
            if (!CanCompose())
            {
                DropLayers();
                ShowFlat(still);
                return;
            }
            if (!holdsLayers)
            {
                if (stage.Budget.HasRoomFor(look.AnimatedLayerCount)) AcquireLayers();
                return;
            }
            if (loadedCount == look.Layers.Length) Compose();
        }

        bool CanCompose() =>
            look.IsAnimated && !layerFailed && stage.AllowsAnimation && look.Layers.Length <= layers.Length;

        void AcquireLayers()
        {
            if (loaded == null || loaded.Length < layers.Length) loaded = new LoopArt[layers.Length];
            var ticket = ++layerTicket;
            holdsLayers = true;
            acquiring = true;
            var table = look.Layers;
            for (var at = 0; at < table.Length; at++)
            {
                var index = at;
                stage.Loops.Acquire(table[at].Address, art => OnLayer(ticket, index, art));
            }
            acquiring = false;
            if (layerFailed || loadedCount == table.Length) Present();
        }

        void OnLayer(int ticket, int index, LoopArt art)
        {
            if (ticket != layerTicket) return;
            if (art == null) layerFailed = true;
            else
            {
                loaded[index] = art;
                loadedCount++;
            }
            if (acquiring) return;
            if (layerFailed || loadedCount == look.Layers.Length) Present();
        }

        void Compose()
        {
            HideLayers();
            if (!stage.Budget.HasRoomFor(look.AnimatedLayerCount))
            {
                DropLayers();
                ShowFlat(still);
                return;
            }
            var now = Time.unscaledTime;
            for (var at = 0; at < look.Layers.Length; at++)
            {
                layers[at].GetComponent<Image>().enabled = true;
                layers[at].Show(loaded[at], stage.Level, stage.ReduceMotion, stage.Budget, now);
            }
            flat.enabled = false;
            IsComposed = true;
        }

        void HideLayers()
        {
            IsComposed = false;
            if (layers == null) return;
            foreach (var layer in layers)
            {
                layer.ShowStill(null);
                layer.GetComponent<Image>().enabled = false;
            }
        }

        void DropLayers()
        {
            layerTicket++;
            HideLayers();
            if (!holdsLayers) return;
            holdsLayers = false;
            for (var at = 0; at < look.Layers.Length; at++)
            {
                stage.Loops.Release(look.Layers[at].Address);
                loaded[at] = null;
            }
            loadedCount = 0;
        }

        void ReleaseHeld()
        {
            if (stage == null) return;
            DropLayers();
            if (!holdsStill) return;
            holdsStill = false;
            stage.Stills.Release(look.StaticAddress);
        }

        void ShowFlat(Sprite sprite)
        {
            flat.sprite = sprite;
            flat.enabled = sprite != null;
        }
    }
}
