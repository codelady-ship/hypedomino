import argparse
import os

import numpy as np
from PIL import Image
from scipy import ndimage

BORDER_STRIP = 6
BACKGROUND_TOLERANCE = 14
EDGE_SIGMA = 1.0
MASK_OPENING = 3
MASK_EROSION = 2
HEAD_SHARE = 0.35
CROP_SHARE = 0.935
HEADROOM = 0.17
CANVAS = 512
CIRCLE_FEATHER = 1.5
POCKET_TOLERANCE = 10
POCKET_MIN_PIXELS = 20
POCKET_REACH = 80


def border_colour(pixels):
    strip = np.concatenate([
        pixels[:BORDER_STRIP].reshape(-1, 3),
        pixels[:, :BORDER_STRIP].reshape(-1, 3),
        pixels[:, -BORDER_STRIP:].reshape(-1, 3),
    ])
    return np.median(strip, axis=0)


def background_mask(pixels, colour):
    near = np.linalg.norm(pixels - colour, axis=2) < BACKGROUND_TOLERANCE
    labels, _ = ndimage.label(near)
    touching = np.unique(np.concatenate([labels[0], labels[:, 0], labels[:, -1]]))
    background = np.isin(labels, touching[touching > 0])
    return ndimage.binary_opening(background, iterations=MASK_OPENING)


def enclosed_pockets(pixels, colour, background):
    close = (np.linalg.norm(pixels - colour, axis=2) < POCKET_TOLERANCE) & ~background
    cut_off = ndimage.binary_fill_holes(~background) & background
    candidates = close | cut_off
    labels, count = ndimage.label(candidates)
    if count == 0:
        return candidates
    sizes = ndimage.sum(candidates, labels, range(1, count + 1))
    large = np.isin(labels, np.where(sizes >= POCKET_MIN_PIXELS)[0] + 1)
    return large & (ndimage.distance_transform_edt(~background) <= POCKET_REACH)


def subject_alpha(background, pockets):
    subject = ndimage.binary_fill_holes(~background) & ~pockets
    labels, count = ndimage.label(subject)
    if count > 1:
        sizes = ndimage.sum(subject, labels, range(1, count + 1))
        subject = labels == (np.argmax(sizes) + 1)
    subject = ndimage.binary_erosion(subject, iterations=MASK_EROSION)
    return np.clip(ndimage.gaussian_filter(subject.astype(np.float64), EDGE_SIGMA), 0, 1)


def remove_fringe(pixels, alpha, colour):
    weight = np.maximum(alpha, 1e-3)[..., None]
    clean = (pixels - (1 - alpha[..., None]) * colour) / weight
    partial = (alpha > 0) & (alpha < 1)
    return np.where(partial[..., None], np.clip(clean, 0, 255), pixels)


def crop_box(alpha):
    rows = np.where(alpha.max(axis=1) > 0.5)[0]
    top, bottom = rows[0], rows[-1]
    head = alpha[top:top + int((bottom - top) * HEAD_SHARE)] > 0.5
    columns = np.where(head.any(axis=0))[0]
    centre_x = (columns[0] + columns[-1]) / 2
    side = CROP_SHARE * min(alpha.shape)
    left = int(round(np.clip(centre_x - side / 2, 0, alpha.shape[1] - side)))
    upper = int(round(min(top - HEADROOM * side, alpha.shape[0] - side)))
    return left, upper, left + int(side), upper + int(side)


def circle_alpha(size):
    ys, xs = np.mgrid[0:size, 0:size] + 0.5
    radius = np.hypot(xs - size / 2, ys - size / 2)
    return np.clip((size / 2 - radius) / CIRCLE_FEATHER + 0.5, 0, 1)


def save_rgba(pixels, alpha, path):
    rgba = np.dstack([pixels, alpha * 255]).round().astype(np.uint8)
    Image.fromarray(rgba, mode="RGBA").save(path)


def main():
    parser = argparse.ArgumentParser(prog="avatar_cleanup.py")
    parser.add_argument("slug")
    parser.add_argument("concept")
    parser.add_argument("work")
    parser.add_argument("export")
    arguments = parser.parse_args()
    concept = Image.open(arguments.concept).convert("RGB")
    pixels = np.asarray(concept).astype(np.float64)
    colour = border_colour(pixels)
    background = background_mask(pixels, colour)
    pockets = enclosed_pockets(pixels, colour, background)
    alpha = subject_alpha(background, pockets)
    clean = remove_fringe(pixels, alpha, colour)
    save_rgba(clean, alpha, os.path.join(arguments.work, f"{arguments.slug}_master.png"))
    box = crop_box(alpha)
    master = Image.fromarray(np.dstack([clean, alpha * 255]).round().astype(np.uint8), mode="RGBA")
    portrait = np.asarray(master.crop(box).resize((CANVAS, CANVAS), Image.LANCZOS)).astype(np.float64)
    save_rgba(portrait[..., :3], portrait[..., 3] / 255 * circle_alpha(CANVAS), os.path.join(arguments.export, "static.png"))
    print(f"{arguments.slug}: background {colour.round(1).tolist()}, pockets {int(pockets.sum())} px, crop {box}, canvas {CANVAS}")


main()
