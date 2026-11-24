import argparse
import os

import numpy as np
from PIL import Image
from scipy import ndimage

BORDER_STRIP = 6
BACKGROUND_TOLERANCE = 30
POCKET_TOLERANCE = 6
POCKET_FLATNESS = 1.2
POCKET_REGION_TOLERANCE = 14
POCKET_GROWTH = 2
GROWTH_TOLERANCE = 40
GROWTH_SATURATION = 12
GROWTH_SPREAD = 4.0
FLATNESS_WINDOW = 5
POCKET_MIN_PIXELS = 20
POCKET_SATURATION = 4
SPECK_MIN_PIXELS = 200
MASK_EROSION = 1
EDGE_SIGMA = 0.8
MARGIN = 0.02


def border_colour(pixels):
    strip = np.concatenate([
        pixels[:BORDER_STRIP].reshape(-1, 3),
        pixels[-BORDER_STRIP:].reshape(-1, 3),
        pixels[:, :BORDER_STRIP].reshape(-1, 3),
        pixels[:, -BORDER_STRIP:].reshape(-1, 3),
    ])
    return np.median(strip, axis=0)


def edge_connected(mask):
    labels, _ = ndimage.label(mask)
    touching = np.unique(np.concatenate([labels[0], labels[-1], labels[:, 0], labels[:, -1]]))
    return np.isin(labels, touching[touching > 0])


def centre_opening(near):
    labels, _ = ndimage.label(near)
    centre = labels[near.shape[0] // 2, near.shape[1] // 2]
    return labels == centre if centre else np.zeros_like(near)


def local_spread(pixels):
    luminance = pixels.mean(axis=2)
    mean = ndimage.uniform_filter(luminance, FLATNESS_WINDOW)
    square = ndimage.uniform_filter(luminance * luminance, FLATNESS_WINDOW)
    return np.sqrt(np.maximum(square - mean * mean, 0))


def pockets(distance, spread, saturation):
    neutral = saturation < POCKET_SATURATION
    strict = (distance < POCKET_TOLERANCE) & (spread < POCKET_FLATNESS) & neutral
    loose = (distance < POCKET_REGION_TOLERANCE) & neutral
    labels, count = ndimage.label(loose)
    if count == 0:
        return strict
    indices = range(1, count + 1)
    sizes = ndimage.sum(loose, labels, indices)
    flatness = ndimage.median(spread, labels, indices)
    flat = np.isin(labels, np.where((sizes >= POCKET_MIN_PIXELS) & (flatness < POCKET_FLATNESS))[0] + 1)
    flat = ndimage.binary_dilation(flat, iterations=POCKET_GROWTH) & (distance < BACKGROUND_TOLERANCE)
    labels, count = ndimage.label(strict)
    if count == 0:
        return flat
    sizes = ndimage.sum(strict, labels, range(1, count + 1))
    return flat | np.isin(labels, np.where(sizes >= POCKET_MIN_PIXELS)[0] + 1)


def without_specks(subject):
    labels, count = ndimage.label(subject)
    if count == 0:
        return subject
    sizes = ndimage.sum(subject, labels, range(1, count + 1))
    return np.isin(labels, np.where(sizes >= SPECK_MIN_PIXELS)[0] + 1)


def frame_alpha(pixels, colour):
    distance = np.linalg.norm(pixels - colour, axis=2)
    near = distance < BACKGROUND_TOLERANCE
    opening = centre_opening(near)
    spread = local_spread(pixels)
    saturation = pixels.max(axis=2) - pixels.min(axis=2)
    seeds = pockets(distance, spread, saturation)
    smooth_grey = (distance < GROWTH_TOLERANCE) & (saturation < GROWTH_SATURATION) & (spread < GROWTH_SPREAD)
    background = edge_connected(near) | opening | ndimage.binary_propagation(seeds, mask=smooth_grey | seeds)
    subject = without_specks(~background)
    subject = ndimage.binary_erosion(subject, iterations=MASK_EROSION)
    return np.clip(ndimage.gaussian_filter(subject.astype(np.float64), EDGE_SIGMA), 0, 1), opening


def remove_fringe(pixels, alpha, colour):
    weight = np.maximum(alpha, 1e-3)[..., None]
    clean = (pixels - (1 - alpha[..., None]) * colour) / weight
    partial = (alpha > 0) & (alpha < 1)
    return np.where(partial[..., None], np.clip(clean, 0, 255), pixels)


def square_box(alpha):
    rows = np.where(alpha.max(axis=1) > 0.5)[0]
    columns = np.where(alpha.max(axis=0) > 0.5)[0]
    centre_x = (columns[0] + columns[-1]) / 2
    centre_y = (rows[0] + rows[-1]) / 2
    side = max(columns[-1] - columns[0], rows[-1] - rows[0]) * (1 + 2 * MARGIN)
    return tuple(int(round(value)) for value in (centre_x - side / 2, centre_y - side / 2, centre_x + side / 2, centre_y + side / 2))


def rgba(pixels, alpha):
    return Image.fromarray(np.dstack([pixels, alpha * 255]).round().clip(0, 255).astype(np.uint8), mode="RGBA")


def main():
    parser = argparse.ArgumentParser(prog="frame_cleanup.py")
    parser.add_argument("slug")
    parser.add_argument("concept")
    parser.add_argument("work")
    arguments = parser.parse_args()
    pixels = np.asarray(Image.open(arguments.concept).convert("RGB")).astype(np.float64)
    colour = border_colour(pixels)
    alpha, opening = frame_alpha(pixels, colour)
    rgba(remove_fringe(pixels, alpha, colour), alpha).save(os.path.join(arguments.work, f"{arguments.slug}_master.png"))
    box = square_box(alpha)
    opening_radius = np.sqrt(opening.sum() / np.pi)
    outer_radius = (box[2] - box[0]) / 2
    print(f"{arguments.slug}: background {colour.round(1).tolist()}, box {box}, opening radius {opening_radius / outer_radius:.3f} of half box")


main()
