import argparse
import os

import numpy as np
from PIL import Image
from scipy import ndimage

PADDING = 24
SOFTNESS = 6.0


def shadow(face):
    alpha = np.asarray(Image.open(face).convert("RGBA"))[..., 3].astype(np.float64) / 255
    padded = np.pad(alpha, PADDING)
    soft = ndimage.gaussian_filter(padded, SOFTNESS, mode="constant")
    white = np.full(soft.shape, 255.0)
    return Image.fromarray(np.dstack([white, white, white, soft * 255]).round().clip(0, 255).astype(np.uint8), mode="RGBA")


def main():
    parser = argparse.ArgumentParser(prog="tile_shadow.py")
    parser.add_argument("face")
    parser.add_argument("export")
    arguments = parser.parse_args()
    os.makedirs(arguments.export, exist_ok=True)
    result = shadow(arguments.face)
    result.save(os.path.join(arguments.export, "static.png"), optimize=True)
    print(f"tile_shadow: {result.size[0]}x{result.size[1]}, padding {PADDING}, softness {SOFTNESS}")


main()
