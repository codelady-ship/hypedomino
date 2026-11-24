import argparse
import os

import numpy as np
from PIL import Image
from scipy import ndimage

GEARS = [(630, 160, 128, 156), (311, 284, 78, 99), (940, 282, 78, 99), (196, 427, 45, 61), (1054, 428, 45, 61), (228, 875, 70, 96), (1021, 877, 70, 96)]
DIFFERENCE = 0.12
GROWTH = 2
GOLD_MARGIN = 0.04
EDGE_SIGMA = 0.8


def load(path):
    return np.asarray(Image.open(path).convert("RGBA")).astype(np.float64) / 255


def premultiplied(image):
    return np.dstack([image[..., :3] * image[..., 3:], image[..., 3]])


def tooth_mask(master, plate):
    rows, columns = np.mgrid[0:master.shape[0], 0:master.shape[1]].astype(np.float64)
    annulus = np.zeros(master.shape[:2], dtype=bool)
    for centre_x, centre_y, root, tip in GEARS:
        distance = np.hypot(columns - centre_x, rows - centre_y)
        annulus |= (distance >= root) & (distance <= tip)
    changed = np.abs(premultiplied(master) - premultiplied(plate)).max(axis=2) > DIFFERENCE
    golden = master[..., 0] > master[..., 1] + GOLD_MARGIN
    teeth = ndimage.binary_opening(annulus & changed & golden & (master[..., 3] > 0.5), iterations=1)
    teeth = ndimage.binary_dilation(teeth, iterations=GROWTH) & annulus
    return np.clip(ndimage.gaussian_filter(teeth.astype(np.float64), EDGE_SIGMA), 0, 1) * annulus


def straight(pixels):
    alpha = pixels[..., 3:]
    colour = np.where(alpha > 0, pixels[..., :3] / np.maximum(alpha, 1e-6), 0)
    return Image.fromarray((np.dstack([colour, alpha]).clip(0, 1) * 255).round().astype(np.uint8))


def main():
    parser = argparse.ArgumentParser(prog="brass_gears_parts.py")
    parser.add_argument("work")
    arguments = parser.parse_args()
    master = load(os.path.join(arguments.work, "brass_gears_master.png"))
    plate = load(os.path.join(arguments.work, "brass_gears_plate_master.png"))
    weight = tooth_mask(master, plate)[..., None]
    teeth = premultiplied(master) * weight
    under = premultiplied(master) * (1 - weight) + premultiplied(plate) * weight
    straight(teeth).save(os.path.join(arguments.work, "brass_gears_teeth.png"))
    straight(under).save(os.path.join(arguments.work, "brass_gears_under.png"))
    print(f"brass_gears_parts: {len(GEARS)} gear rims, tooth layer covers {int((weight[..., 0] > 0.5).sum())} px")


main()
