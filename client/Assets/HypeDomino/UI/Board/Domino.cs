namespace HypeDomino.UI.Board
{
    public readonly struct Domino
    {
        public Domino(byte first, byte second)
        {
            First = first;
            Second = second;
        }

        public byte First { get; }

        public byte Second { get; }

        public bool IsDouble => First == Second;
    }
}
