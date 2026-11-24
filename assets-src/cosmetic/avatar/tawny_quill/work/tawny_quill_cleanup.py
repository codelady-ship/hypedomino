import argparse
import os

import numpy as np
from PIL import Image
from scipy import ndimage

BORDER_STRIP = 6
BACKGROUND_TOLERANCE = 30
EDGE_SIGMA = 1.0
MASK_OPENING = 3
MASK_EROSION = 2
HEAD_SHARE = 0.35
CROP_SHARE = 0.935
HEADROOM = 0.17
PORTRAIT = 768
POCKET_TOLERANCE = 24
POCKET_MIN_PIXELS = 20
POCKET_REACH = 140
POCKET_FLATNESS = 12.0
KEY_BAND = 6
KEY_LOW = 14
KEY_HIGH = 60
INTERIOR_SIGMA = 6
SPAN_MIN = 40
POCKET_KEEP = [(710, 793), (761, 699)]


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
    flatness = ndimage.median(np.linalg.norm(pixels - colour, axis=2), labels, range(1, count + 1))
    keep = np.where((sizes >= POCKET_MIN_PIXELS) & ((np.asarray(flatness) < POCKET_FLATNESS)))[0] + 1
    keep = np.setdiff1d(keep, [labels[y, x] for x, y in POCKET_KEEP])
    return np.isin(labels, keep) & (ndimage.distance_transform_edt(~background) <= POCKET_REACH) | cut_off


def subject_alpha(background, pockets):
    subject = ndimage.binary_fill_holes(~background) & ~pockets
    labels, count = ndimage.label(subject)
    if count > 1:
        sizes = ndimage.sum(subject, labels, range(1, count + 1))
        subject = labels == (np.argmax(sizes) + 1)
    subject = ndimage.binary_erosion(subject, iterations=MASK_EROSION)
    return np.clip(ndimage.gaussian_filter(subject.astype(np.float64), EDGE_SIGMA), 0, 1)


def nearby_colour(pixels, alpha, outside):
    interior = (ndimage.distance_transform_edt(~outside) > KEY_BAND) & (alpha > 0.99)
    weight = ndimage.gaussian_filter(interior.astype(np.float64), INTERIOR_SIGMA)
    nearby = np.dstack([ndimage.gaussian_filter(pixels[..., c] * interior, INTERIOR_SIGMA) for c in range(3)]) / np.maximum(weight, 1e-6)[..., None]
    return nearby, weight


def keyed_edge(pixels, colour, alpha, outside, nearby, weight):
    depth = ndimage.distance_transform_edt(~outside)
    band = depth <= KEY_BAND
    span = nearby - colour
    reach = np.sum(span * span, axis=2)
    projected = np.clip(np.sum((pixels - colour) * span, axis=2) / np.maximum(reach, 1e-6), 0, 1)
    key = np.clip((np.linalg.norm(pixels - colour, axis=2) - KEY_LOW) / (KEY_HIGH - KEY_LOW), 0, 1)
    trusted = (weight > 0.05) & (reach >= SPAN_MIN ** 2)
    estimate = np.where(trusted, projected, key)
    return np.where(band, np.minimum(alpha, estimate), alpha)


def remove_fringe(pixels, alpha, colour, nearby, weight):
    share = np.maximum(alpha, 1e-3)[..., None]
    clean = np.clip((pixels - (1 - alpha[..., None]) * colour) / share, 0, 255)
    settled = np.where((weight > 0.05)[..., None], share * clean + (1 - share) * nearby, clean)
    partial = (alpha > 0) & (alpha < 1)
    return np.where(partial[..., None], settled, pixels)


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


def save_rgba(pixels, alpha, path):
    rgba = np.dstack([pixels, alpha * 255]).round().astype(np.uint8)
    Image.fromarray(rgba, mode="RGBA").save(path)


def main():
    parser = argparse.ArgumentParser(prog="avatar_cleanup.py")
    parser.add_argument("slug")
    parser.add_argument("concept")
    parser.add_argument("work")
    arguments = parser.parse_args()
    concept = Image.open(arguments.concept).convert("RGB")
    pixels = np.asarray(concept).astype(np.float64)
    colour = border_colour(pixels)
    background = background_mask(pixels, colour)
    pockets = enclosed_pockets(pixels, colour, background)
    rough = subject_alpha(background, pockets)
    nearby, weight = nearby_colour(pixels, rough, background | pockets)
    alpha = keyed_edge(pixels, colour, rough, background | pockets, nearby, weight)
    clean = remove_fringe(pixels, alpha, colour, nearby, weight)
    save_rgba(clean, alpha, os.path.join(arguments.work, f"{arguments.slug}_master.png"))
    box = crop_box(alpha)
    master = Image.fromarray(np.dstack([clean, alpha * 255]).round().astype(np.uint8), mode="RGBA")
    master.crop(box).resize((PORTRAIT, PORTRAIT), Image.LANCZOS).save(os.path.join(arguments.work, f"{arguments.slug}_portrait.png"))
    print(f"{arguments.slug}: background {colour.round(1).tolist()}, pockets {int(pockets.sum())} px, crop {box}, portrait {PORTRAIT}")


main()
