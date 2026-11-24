namespace HypeDomino.UI.Cosmetics
{
    public readonly struct TableLayer
    {
        public TableLayer(string name, string address, bool animated)
        {
            Name = name;
            Address = address;
            Animated = animated;
        }

        public string Name { get; }

        public string Address { get; }

        public bool Animated { get; }
    }

    public readonly struct TableLook
    {
        static readonly TableLayer[] NoLayers = new TableLayer[0];

        readonly TableLayer[] layers;

        public TableLook(string staticAddress, TableLayer[] backToFront)
        {
            StaticAddress = staticAddress;
            layers = backToFront;
        }

        public string StaticAddress { get; }

        public TableLayer[] Layers => layers ?? NoLayers;

        public bool IsAnimated => Layers.Length > 0;

        public int AnimatedLayerCount
        {
            get
            {
                var count = 0;
                foreach (var layer in Layers)
                    if (layer.Animated) count++;
                return count;
            }
        }

        public bool IsSameAs(TableLook other)
        {
            if (StaticAddress != other.StaticAddress || Layers.Length != other.Layers.Length) return false;
            for (var at = 0; at < Layers.Length; at++)
            {
                var mine = Layers[at];
                var theirs = other.Layers[at];
                if (mine.Name != theirs.Name || mine.Address != theirs.Address || mine.Animated != theirs.Animated) return false;
            }
            return true;
        }
    }
}
