import argparse
import math
import os
import re

import numpy as np
from PIL import Image

CANVAS = (128, 256)
FRAMES = 16
STATIC_FRAME = 9
STAR_CENTRES = [(481, 720), (130, 868), (759, 868), (443, 392), (443, 543), (443, 1140), (443, 1320)]
STAR_BOX = 96
STAR_PULSES = [(0, 2), (1, 5), (2, 5), (4, 11), (5, 11), (3, 14), (6, 14)]
STAR_PULSE_HALF_WIDTH = 2.5
STAR_PULSE_SCALE = 0.18
SPARKLE_CELL = 12
SPARKLE_DIM = 0.7
SPARKLE_SEED = 20261112
FRAME_NAME = re.compile(r"^\d{4}\.png$")


def pulse(frame, centre):
    distance = abs(frame - centre)
    distance = min(distance, FRAMES - distance)
    if distance >= STAR_PULSE_HALF_WIDTH:
        return 0.0
    return 0.5 * (1 + math.cos(math.pi * distance / STAR_PULSE_HALF_WIDTH))


def sparkle_groups(shape):
    cells = np.random.default_rng(SPARKLE_SEED).integers(0, 2, size=(shape[0] // SPARKLE_CELL + 1, shape[1] // SPARKLE_CELL + 1))
    return np.kron(cells, np.ones((SPARKLE_CELL, SPARKLE_CELL)))[:shape[0], :shape[1]]


def sparkle_dim(frame, groups):
    slow = math.sin(math.pi * frame / 8) ** 2
    fast = math.sin(math.pi * frame / 4) ** 2
    return SPARKLE_DIM * np.where(groups > 0, slow, fast)


def scaled_star(stars, centre, scale):
    x, y = centre
    box = (x - STAR_BOX, y - STAR_BOX, x + STAR_BOX, y + STAR_BOX)
    patch = stars.crop(box)
    inverse = 1 / scale
    offset = STAR_BOX * (1 - inverse)
    grown = patch.transform(patch.size, Image.Transform.AFFINE, (inverse, 0, offset, 0, inverse, offset), resample=Image.Resampling.BICUBIC)
    return grown, box[:2]


def silhouette_alpha(face):
    alpha = Image.open(face).convert("RGBA").getchannel("A")
    if alpha.size != CANVAS:
        raise SystemExit(f"sema_loop: silhouette {face} is {alpha.size[0]}x{alpha.size[1]}, the canvas is {CANVAS[0]}x{CANVAS[1]}")
    return alpha


def compose(frame, master, stars, sparkles, groups, silhouette):
    pixels = np.asarray(master).astype(np.float64)
    weight = np.asarray(sparkles)[..., 3].astype(np.float64) / 255
    pixels *= (1 - weight * sparkle_dim(frame, groups))[..., None]
    image = Image.fromarray(pixels.round().astype(np.uint8), mode="RGB").convert("RGBA")
    for star, centre in STAR_PULSES:
        strength = pulse(frame, centre)
        if strength == 0:
            continue
        grown, origin = scaled_star(stars, STAR_CENTRES[star], 1 + STAR_PULSE_SCALE * strength)
        image.alpha_composite(grown, origin)
    frame_image = image.resize(CANVAS, Image.Resampling.LANCZOS)
    frame_image.putalpha(silhouette)
    return frame_image


def clear_frames(folder):
    os.makedirs(folder, exist_ok=True)
    for name in os.listdir(folder):
        if FRAME_NAME.match(name):
            os.remove(os.path.join(folder, name))


def main():
    parser = argparse.ArgumentParser(prog="sema_loop.py")
    parser.add_argument("work")
    parser.add_argument("export")
    parser.add_argument("silhouette")
    arguments = parser.parse_args()
    master = Image.open(os.path.join(arguments.work, "sema_master.png")).convert("RGB")
    stars = Image.open(os.path.join(arguments.work, "sema_stars.png")).convert("RGBA")
    sparkles = Image.open(os.path.join(arguments.work, "sema_sparkles.png")).convert("RGBA")
    groups = sparkle_groups((master.height, master.width))
    silhouette = silhouette_alpha(arguments.silhouette)
    anim = os.path.join(arguments.export, "anim")
    clear_frames(anim)
    for index in range(1, FRAMES + 1):
        frame = compose(index - 1, master, stars, sparkles, groups, silhouette)
        frame.save(os.path.join(anim, f"{index:04d}.png"))
        if index == STATIC_FRAME:
            frame.save(os.path.join(arguments.export, "static.png"))
    print(f"sema_loop: wrote {FRAMES} frames and the static (frame {STATIC_FRAME}) at {CANVAS[0]}x{CANVAS[1]}, masked by {arguments.silhouette}")


main()
