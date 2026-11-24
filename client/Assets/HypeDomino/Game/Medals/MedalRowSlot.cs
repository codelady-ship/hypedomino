using UnityEngine;

namespace HypeDomino.Game.Medals
{
    public sealed class MedalRowSlot : MonoBehaviour
    {
        [SerializeField] MedalCellSlot[] cells;

        public int Columns => cells.Length;

        public void Configure(MedalCellSlot[] rowCells) => cells = rowCells;

        public MedalCellSlot CellAt(int column) => cells[column];
    }
}
