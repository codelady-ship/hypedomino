import argparse
import os

import numpy as np
from PIL import Image

CANVAS = (64, 256)
SUPERSAMPLE = 4
BANDS = ((0.40, 0.11, 1.0), (0.72, 0.035, 0.55))
TAPER = 0.18


def profile(positions, centre, width, strength):
    return strength * np.exp(-0.5 * ((positions - centre) / width) ** 2)


def taper(positions):
    edge = np.minimum(positions, 1 - positions) / TAPER
    ramp = np.clip(edge, 0, 1)
    return ramp * ramp * (3 - 2 * ramp)


def streak():
    width, height = CANVAS[0] * SUPERSAMPLE, CANVAS[1] * SUPERSAMPLE
    across = (np.arange(width) + 0.5) / width
    along = (np.arange(height) + 0.5) / height
    band = np.clip(sum(profile(across, *spec) for spec in BANDS), 0, 1)
    alpha = taper(along)[:, None] * band[None, :]
    small = np.asarray(Image.fromarray((alpha * 255).astype(np.float32), mode="F").resize(CANVAS, Image.BOX))
    white = np.full(small.shape, 255.0)
    return Image.fromarray(np.dstack([white, white, white, small]).round().clip(0, 255).astype(np.uint8), mode="RGBA")


def main():
    parser = argparse.ArgumentParser(prog="sheen.py")
    parser.add_argument("export")
    arguments = parser.parse_args()
    os.makedirs(arguments.export, exist_ok=True)
    result = streak()
    result.save(os.path.join(arguments.export, "static.png"), optimize=True)
    print(f"sheen: {result.size[0]}x{result.size[1]}, bands {BANDS}, taper {TAPER}")


main()
