import argparse
import json
import os

import numpy as np
from PIL import Image
from scipy import ndimage

BORDER_STRIP = 6
BACKGROUND_TOLERANCE = 30
MASK_OPENING = 2
MASK_EROSION = 1
EDGE_SIGMA = 1.0
POCKET_TOLERANCE = 10
POCKET_MIN_PIXELS = 20
PART_MIN_PIXELS = 400
MASTER_PAD = 4
CANVAS = 256
CONTENT = 240
PIP_DARK = 70
PIP_REACH = 1.5
FILL_RING = 2
FILL_ITERATIONS = 800
PIP_LAYOUTS = {
    0: [],
    1: [(0, 0)],
    2: [(-1, -1), (1, 1)],
    3: [(-1, -1), (0, 0), (1, 1)],
    4: [(-1, -1), (1, -1), (-1, 1), (1, 1)],
    5: [(-1, -1), (1, -1), (0, 0), (-1, 1), (1, 1)],
    6: [(-1, -1), (-1, 0), (-1, 1), (1, -1), (1, 0), (1, 1)],
}


def border_colour(pixels):
    strip = np.concatenate([
        pixels[:BORDER_STRIP].reshape(-1, 3),
        pixels[-BORDER_STRIP:].reshape(-1, 3),
        pixels[:, :BORDER_STRIP].reshape(-1, 3),
        pixels[:, -BORDER_STRIP:].reshape(-1, 3),
    ])
    return np.median(strip, axis=0)


def background_mask(pixels, colour):
    near = np.linalg.norm(pixels - colour, axis=2) < BACKGROUND_TOLERANCE
    labels, _ = ndimage.label(near)
    touching = np.unique(np.concatenate([labels[0], labels[-1], labels[:, 0], labels[:, -1]]))
    background = np.isin(labels, touching[touching > 0])
    return ndimage.binary_opening(background, iterations=MASK_OPENING)


def enclosed_pockets(pixels, colour, background, minimum):
    close = (np.linalg.norm(pixels - colour, axis=2) < POCKET_TOLERANCE) & ~background
    cut_off = ndimage.binary_fill_holes(~background) & background
    candidates = close | cut_off
    labels, count = ndimage.label(candidates)
    if count == 0:
        return candidates
    sizes = ndimage.sum(candidates, labels, range(1, count + 1))
    return np.isin(labels, np.where(sizes >= minimum)[0] + 1)


def cell_subject(subject, rows, columns, cell):
    labels, count = ndimage.label(subject)
    sizes = ndimage.sum(subject, labels, range(1, count + 1))
    centres = ndimage.center_of_mass(subject, labels, range(1, count + 1))
    height, width = subject.shape
    keep = []
    for index, (size, (y, x)) in enumerate(zip(sizes, centres), start=1):
        row = min(int(y / height * rows), rows - 1)
        column = min(int(x / width * columns), columns - 1)
        if size >= PART_MIN_PIXELS and row * columns + column == cell - 1:
            keep.append(index)
    return np.isin(labels, keep)


def subject_alpha(subject):
    eroded = ndimage.binary_erosion(subject, iterations=MASK_EROSION)
    return np.clip(ndimage.gaussian_filter(eroded.astype(np.float64), EDGE_SIGMA), 0, 1)


def remove_fringe(pixels, alpha, colour):
    weight = np.maximum(alpha, 1e-3)[..., None]
    clean = (pixels - (1 - alpha[..., None]) * colour) / weight
    partial = (alpha > 0) & (alpha < 1)
    return np.where(partial[..., None], np.clip(clean, 0, 255), pixels)


def bounding_box(alpha):
    rows = np.where(alpha.max(axis=1) > 0)[0]
    columns = np.where(alpha.max(axis=0) > 0)[0]
    top = max(rows[0] - MASTER_PAD, 0)
    left = max(columns[0] - MASTER_PAD, 0)
    bottom = min(rows[-1] + MASTER_PAD + 1, alpha.shape[0])
    right = min(columns[-1] + MASTER_PAD + 1, alpha.shape[1])
    return left, top, right, bottom


def to_image(pixels, alpha):
    return Image.fromarray(np.dstack([pixels, alpha * 255]).round().astype(np.uint8), mode="RGBA")


def fit_canvas(master):
    content = master.getbbox()
    trimmed = master.crop(content)
    scale = CONTENT / max(trimmed.size)
    size = (max(1, round(trimmed.width * scale)), max(1, round(trimmed.height * scale)))
    resized = trimmed.convert("RGBa").resize(size, Image.LANCZOS).convert("RGBA")
    canvas = Image.new("RGBA", (CANVAS, CANVAS), (0, 0, 0, 0))
    canvas.paste(resized, ((CANVAS - size[0]) // 2, (CANVAS - size[1]) // 2))
    return canvas


def tile_axes(angle):
    turn = np.radians(angle)
    return np.array([np.cos(turn), np.sin(turn)]), np.array([-np.sin(turn), np.cos(turn)])


def painted_pips(pixels, spec):
    dark = pixels.mean(axis=2) < PIP_DARK
    labels, count = ndimage.label(dark)
    sizes = ndimage.sum(dark, labels, range(1, count + 1))
    centres = ndimage.center_of_mass(dark, labels, range(1, count + 1))
    found = []
    for tile in spec["tiles"]:
        across, down = tile_axes(tile["angle"])
        for size, (y, x) in zip(sizes, centres):
            offset = np.array([x, y]) - tile["centre"]
            inside = abs(offset @ across) <= spec["step"][0] * PIP_REACH and abs(offset @ down) <= spec["step"][1] * PIP_REACH
            if size >= spec["pipMinPixels"] and inside:
                found.append((x, y))
    return found


def discs(shape, centres, radius):
    ys, xs = np.mgrid[0:shape[0], 0:shape[1]].astype(np.float64)
    mask = np.zeros(shape, dtype=bool)
    for x, y in centres:
        mask |= np.hypot(xs - x, ys - y) <= radius
    return mask


def harmonic_fill(pixels, mask):
    ring = ndimage.binary_dilation(mask, iterations=FILL_RING) & ~mask
    filled = pixels.copy()
    filled[mask] = pixels[ring].mean(axis=0)
    for _ in range(FILL_ITERATIONS):
        neighbours = (np.roll(filled, 1, 0) + np.roll(filled, -1, 0) + np.roll(filled, 1, 1) + np.roll(filled, -1, 1)) / 4
        filled[mask] = neighbours[mask]
    return filled


def pip_targets(spec):
    targets = []
    for tile in spec["tiles"]:
        across, down = tile_axes(tile["angle"])
        for column, row in PIP_LAYOUTS[tile["value"]]:
            targets.append(np.array(tile["centre"]) + column * spec["step"][0] * across + row * spec["step"][1] * down)
    return targets


def repaint_pips(master, spec):
    rgba = np.asarray(master).astype(np.float64)
    pixels = rgba[..., :3]
    radius = spec["pipRadius"]
    found = painted_pips(pixels, spec)
    filled = harmonic_fill(pixels, discs(pixels.shape[:2], found, radius))
    source = spec["stampPip"]
    window = int(np.ceil(radius)) + 2
    left, top = round(source[0]), round(source[1])
    stamp = (pixels - filled)[top - window:top + window + 1, left - window:left + window + 1]
    stamp = stamp * discs(stamp.shape[:2], [(window + source[0] - left, window + source[1] - top)], radius)[..., None]
    painted = filled.copy()
    targets = pip_targets(spec)
    for target in targets:
        x, y = round(target[0]), round(target[1])
        shift = (target[1] - y - (source[1] - top), target[0] - x - (source[0] - left), 0)
        painted[y - window:y + window + 1, x - window:x + window + 1] += ndimage.shift(stamp, shift, order=1, mode="constant")
    repainted = np.dstack([np.clip(painted, 0, 255), rgba[..., 3]]).round().astype(np.uint8)
    return Image.fromarray(repainted, mode="RGBA"), len(found), len(targets)


def main():
    parser = argparse.ArgumentParser(prog="medal_cleanup.py")
    parser.add_argument("slug")
    parser.add_argument("concept")
    parser.add_argument("rows", type=int)
    parser.add_argument("columns", type=int)
    parser.add_argument("cell", type=int)
    parser.add_argument("work")
    parser.add_argument("export")
    parser.add_argument("--pocket-min", type=int, default=POCKET_MIN_PIXELS)
    parser.add_argument("--pips")
    arguments = parser.parse_args()
    pixels = np.asarray(Image.open(arguments.concept).convert("RGB")).astype(np.float64)
    colour = border_colour(pixels)
    background = background_mask(pixels, colour)
    pockets = enclosed_pockets(pixels, colour, background, arguments.pocket_min)
    subject = ndimage.binary_fill_holes(~background) & ~pockets
    medal = cell_subject(subject, arguments.rows, arguments.columns, arguments.cell)
    alpha = subject_alpha(medal)
    clean = remove_fringe(pixels, alpha, colour)
    box = bounding_box(alpha)
    master = to_image(clean, alpha).crop(box)
    repaint = ""
    if arguments.pips:
        with open(arguments.pips, encoding="utf-8") as source:
            spec = json.load(source)
        master, erased, painted = repaint_pips(master, spec)
        values = "-".join(str(tile["value"]) for tile in spec["tiles"])
        repaint = f", pips {erased} erased, {painted} painted, values {values}"
    master.save(os.path.join(arguments.work, f"{arguments.slug}_master.png"))
    fit_canvas(master).save(os.path.join(arguments.export, "static.png"))
    print(f"{arguments.slug}: background {colour.round(1).tolist()}, pockets {int(pockets.sum())} px of at least {arguments.pocket_min}, cell {arguments.cell} of {arguments.rows}x{arguments.columns}, box {tuple(int(v) for v in box)}, canvas {CANVAS}{repaint}")


main()
