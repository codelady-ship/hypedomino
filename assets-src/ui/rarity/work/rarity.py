import json
import math
import sys
from pathlib import Path

import numpy as np
from PIL import Image

SIZE = 192
SUPER = 4
HALF_DIAGONAL = 92.0
PIP_RADIUS = 0.14
PIP_STEP = 0.4
FACETS = {(-1, -1): 1.0, (1, -1): 0.9, (-1, 1): 0.84, (1, 1): 0.74}
TABLE = 0.55
TABLE_LIGHT = 0.97
DOMINO_HALF = {
    1: [(0, 0)],
    2: [(-1, -1), (1, 1)],
    3: [(-1, -1), (0, 0), (1, 1)],
    4: [(-1, -1), (1, -1), (-1, 1), (1, 1)],
    5: [(-1, -1), (1, -1), (0, 0), (-1, 1), (1, 1)],
    6: [(-1, -1), (-1, 0), (-1, 1), (1, -1), (1, 0), (1, 1)],
}


def grid():
    offsets = (np.arange(SUPER) + 0.5) / SUPER
    axis = (np.arange(SIZE)[:, None] + offsets[None, :]).reshape(-1) - SIZE / 2
    return np.meshgrid(axis, axis)


def image(luminance, alpha):
    rgba = np.stack([luminance * alpha] * 3 + [alpha], axis=-1)
    small = rgba.reshape(SIZE, SUPER, SIZE, SUPER, 4).mean(axis=(1, 3))
    out_alpha = small[..., 3]
    safe = np.where(out_alpha > 0, out_alpha, 1)
    grey = np.where(out_alpha > 0, small[..., 0] / safe, 1.0)
    data = np.stack([grey, grey, grey, out_alpha], axis=-1)
    return Image.fromarray(np.round(np.clip(data, 0, 1) * 255).astype(np.uint8), "RGBA")


def diamond():
    x, y = grid()
    inside = (np.abs(x) + np.abs(y) <= HALF_DIAGONAL).astype(np.float64)
    side_x = np.where(x < 0, -1, 1)
    side_y = np.where(y < 0, -1, 1)
    luminance = np.zeros_like(x)
    for (sx, sy), light in FACETS.items():
        luminance = np.where((side_x == sx) & (side_y == sy), light, luminance)
    luminance = np.where(np.abs(x) + np.abs(y) <= HALF_DIAGONAL * TABLE, TABLE_LIGHT, luminance)
    return image(luminance, inside)


def pips(count):
    x, y = grid()
    turn = math.radians(45)
    step = PIP_STEP * HALF_DIAGONAL
    alpha = np.zeros_like(x)
    for column, row in DOMINO_HALF[count]:
        px = (column * math.cos(turn) - row * math.sin(turn)) * step
        py = (column * math.sin(turn) + row * math.cos(turn)) * step
        alpha = np.maximum(alpha, (np.hypot(x - px, y - py) <= PIP_RADIUS * HALF_DIAGONAL).astype(np.float64))
    return image(np.ones_like(x), alpha)


def main(tokens_path, out_dir):
    tokens = json.loads(Path(tokens_path).read_text(encoding="utf-8"))
    counts = sorted({tier["pips"] for tier in tokens["rarity"]["tiers"].values()} | {6})
    out = Path(out_dir)
    out.mkdir(parents=True, exist_ok=True)
    diamond().save(out / "diamond.png", optimize=False)
    for count in counts:
        pips(count).save(out / f"pips_{count}.png", optimize=False)
    print(f"diamond and pip overlays {counts} at {SIZE} x {SIZE}")


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
