import argparse
import os

import numpy as np
from PIL import Image

CANVAS = (128, 256)


def downsample(path):
    source = np.asarray(Image.open(path).convert("RGBA")).astype(np.float64) / 255.0
    alpha = source[..., 3:]
    premultiplied = np.dstack([source[..., :3] * alpha, alpha]).astype(np.float32)
    small = np.stack([np.asarray(Image.fromarray(premultiplied[..., channel], mode="F").resize(CANVAS, Image.BOX)) for channel in range(4)], axis=-1)
    opacity = small[..., 3:]
    straight = np.where(opacity > 0, small[..., :3] / np.maximum(opacity, 1e-6), 0)
    pixels = np.dstack([straight, opacity]) * 255
    return Image.fromarray(pixels.round().clip(0, 255).astype(np.uint8), mode="RGBA")


def main():
    parser = argparse.ArgumentParser(prog="ivory_export.py")
    parser.add_argument("renders")
    parser.add_argument("export")
    arguments = parser.parse_args()
    target = os.path.join(arguments.export, "face")
    os.makedirs(target, exist_ok=True)
    count = 0
    for low in range(7):
        for high in range(low, 7):
            name = f"{low}_{high}.png"
            downsample(os.path.join(arguments.renders, name)).save(os.path.join(target, name), optimize=True)
            count += 1
    print(f"exported {count} faces at {CANVAS[0]}x{CANVAS[1]} into {target}")


main()
