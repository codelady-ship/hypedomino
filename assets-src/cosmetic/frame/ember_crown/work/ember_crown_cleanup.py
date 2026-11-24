import argparse
import os

import numpy as np
from PIL import Image
from scipy import ndimage

BORDER_STRIP = 6
BACKGROUND_TOLERANCE = 30
POCKET_TOLERANCE = 12
POCKET_MIN_PIXELS = 20
SPECK_MIN_PIXELS = 200
MASK_EROSION = 1
EDGE_SIGMA = 0.8
MARGIN = 0.02
CANVAS = 512


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


def pockets(distance):
    close = distance < POCKET_TOLERANCE
    labels, count = ndimage.label(close)
    if count == 0:
        return close
    sizes = ndimage.sum(close, labels, range(1, count + 1))
    return np.isin(labels, np.where(sizes >= POCKET_MIN_PIXELS)[0] + 1)


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
    background = edge_connected(near) | opening | pockets(distance)
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
    parser.add_argument("export")
    arguments = parser.parse_args()
    pixels = np.asarray(Image.open(arguments.concept).convert("RGB")).astype(np.float64)
    colour = border_colour(pixels)
    alpha, opening = frame_alpha(pixels, colour)
    master = rgba(remove_fringe(pixels, alpha, colour), alpha)
    master.save(os.path.join(arguments.work, f"{arguments.slug}_master.png"))
    box = square_box(alpha)
    master.crop(box).resize((CANVAS, CANVAS), Image.LANCZOS).save(os.path.join(arguments.export, "static.png"))
    opening_radius = np.sqrt(opening.sum() / np.pi)
    outer_radius = (box[2] - box[0]) / 2
    print(f"{arguments.slug}: background {colour.round(1).tolist()}, box {box}, opening radius {opening_radius / outer_radius:.3f} of half canvas, canvas {CANVAS}")


main()
