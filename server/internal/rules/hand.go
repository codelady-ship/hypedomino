package rules

type Hand []Tile

func (h Hand) Contains(tile Tile) bool {
	return h.indexOf(tile) >= 0
}

func (h Hand) Doubles() int {
	count := 0
	for _, tile := range h {
		if tile.IsDouble() {
			count++
		}
	}
	return count
}

func (h Hand) Pips() int {
	total := 0
	for _, tile := range h {
		total += tile.Pips()
	}
	return total
}

func (h *Hand) Remove(tile Tile) bool {
	index := h.indexOf(tile)
	if index < 0 {
		return false
	}
	last := len(*h) - 1
	(*h)[index] = (*h)[last]
	*h = (*h)[:last]
	return true
}

func (h *Hand) Add(tile Tile) {
	*h = append(*h, tile)
}

func (h Hand) indexOf(tile Tile) int {
	for index, held := range h {
		if held == tile {
			return index
		}
	}
	return -1
}

type Boneyard []Tile

func (b *Boneyard) Draw() (Tile, bool) {
	if len(*b) == 0 {
		return Tile{}, false
	}
	tile := (*b)[0]
	*b = (*b)[1:]
	return tile, true
}
