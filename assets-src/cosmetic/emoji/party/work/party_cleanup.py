import argparse
import os

import numpy as np
from PIL import Image
from scipy import ndimage

SLUG = "party"
SHEET = "20261115-02.png"
CELL = (1, 0)
WRITE_STATIC = False
GRID = 3
WORK_SIDE = 480
REDUCTION = 4
MEDALLION_WIDTH = 368
MARGIN = 8
PAD = 400
BORDER_STRIP = 6
BACKGROUND_TOLERANCE = 30
MASK_OPENING = 2
MIN_PART_PIXELS = 60
KEEP_REACH = 4
EDGE_SIGMA = 1.0
MASK_EROSION = 1
POCKET_TOLERANCE = 12
POCKET_MIN_PIXELS = 40
POCKET_FLATNESS = 6.0
KEY_BAND = 5
KEY_LOW = 14
KEY_HIGH = 60
INTERIOR_SIGMA = 5
SPAN_MIN = 40
BAND_SHARE = 0.1
LOWER_SHARE = 0.5


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


def enclosed_pockets(pixels, colour, background):
    distance = np.linalg.norm(pixels - colour, axis=2)
    candidates = (distance < POCKET_TOLERANCE) & ~background
    labels, count = ndimage.label(candidates)
    if count == 0:
        return candidates
    sizes = ndimage.sum(candidates, labels, range(1, count + 1))
    spread = ndimage.standard_deviation(distance, labels, range(1, count + 1))
    keep = np.where((sizes >= POCKET_MIN_PIXELS) & (np.asarray(spread) < POCKET_FLATNESS))[0] + 1
    return np.isin(labels, keep)


def cell_subject(background, pockets):
    subject = ndimage.binary_fill_holes(~background) & ~pockets
    labels, count = ndimage.label(subject)
    centres = ndimage.center_of_mass(subject, labels, range(1, count + 1))
    sizes = ndimage.sum(subject, labels, range(1, count + 1))
    height, width = subject.shape
    row, column = CELL
    chosen = [
        index + 1 for index, ((y, x), size) in enumerate(zip(centres, sizes))
        if size >= MIN_PART_PIXELS and int(y * GRID // height) == row and int(x * GRID // width) == column
    ]
    largest = max(chosen, key=lambda label: sizes[label - 1])
    return np.isin(labels, chosen), labels == largest


def medallion_geometry(medallion):
    rows = np.where(medallion.any(axis=1))[0]
    spans = np.array([np.ptp(np.where(medallion[row])[0]) + 1 for row in rows])
    lower = rows >= rows[-1] - LOWER_SHARE * spans.max()
    band = rows[lower & (spans >= (1 - BAND_SHARE / 4) * spans[lower].max())]
    centre_y = (band[0] + band[-1]) / 2
    columns = np.where(medallion[band].any(axis=0))[0]
    return (columns[0] + columns[-1]) / 2, centre_y, columns[-1] - columns[0] + 1


def soft_alpha(subject):
    eroded = ndimage.binary_erosion(subject, iterations=MASK_EROSION)
    return np.clip(ndimage.gaussian_filter(eroded.astype(np.float64), EDGE_SIGMA), 0, 1)


def nearby_colour(pixels, alpha, outside):
    interior = (ndimage.distance_transform_edt(~outside) > KEY_BAND) & (alpha > 0.99)
    weight = ndimage.gaussian_filter(interior.astype(np.float64), INTERIOR_SIGMA)
    nearby = np.dstack([ndimage.gaussian_filter(pixels[..., c] * interior, INTERIOR_SIGMA) for c in range(3)]) / np.maximum(weight, 1e-6)[..., None]
    return nearby, weight


def keyed_edge(pixels, colour, alpha, outside, nearby, weight):
    band = ndimage.distance_transform_edt(~outside) <= KEY_BAND
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


def placement(subject, centre_x, centre_y, width):
    scale = MEDALLION_WIDTH / width
    rows = np.where(subject.any(axis=1))[0]
    columns = np.where(subject.any(axis=0))[0]
    top, bottom, left, right = rows[0], rows[-1] + 1, columns[0], columns[-1] + 1
    usable = WORK_SIDE - 2 * MARGIN
    scale = min(scale, usable / (bottom - top), usable / (right - left))
    half = WORK_SIDE / 2 / scale
    inner = MARGIN / scale
    x0 = np.clip(centre_x - half, right + inner - 2 * half, left - inner)
    y0 = np.clip(centre_y - half, bottom + inner - 2 * half, top - inner)
    return x0, y0, scale


def resample(premultiplied, x0, y0, scale):
    side = WORK_SIDE / scale
    box = (x0 + PAD, y0 + PAD, x0 + PAD + side, y0 + PAD + side)
    channels = []
    for channel in range(4):
        padded = np.pad(premultiplied[..., channel], PAD).astype(np.float32)
        image = Image.fromarray(padded)
        channels.append(np.asarray(image.resize((WORK_SIDE, WORK_SIDE), Image.Resampling.LANCZOS, box=box)))
    stacked = np.dstack(channels).astype(np.float64)
    stacked[..., 3] = np.clip(stacked[..., 3], 0, 1)
    stacked[..., :3] = np.clip(stacked[..., :3], 0, stacked[..., 3:4] * 255)
    return stacked


def unpremultiplied(premultiplied):
    alpha = premultiplied[..., 3]
    rgb = np.where(alpha[..., None] > 0, premultiplied[..., :3] / np.maximum(alpha, 1e-6)[..., None], 0)
    return Image.fromarray(np.dstack([np.clip(rgb, 0, 255), alpha * 255]).round().astype(np.uint8))


def reduced(premultiplied):
    side = WORK_SIDE // REDUCTION
    return premultiplied.reshape(side, REDUCTION, side, REDUCTION, 4).mean(axis=(1, 3))


def main():
    parser = argparse.ArgumentParser(prog=f"{SLUG}_cleanup.py")
    parser.add_argument("item")
    arguments = parser.parse_args()
    work = os.path.join(arguments.item, "work")
    pixels = np.asarray(Image.open(os.path.join(arguments.item, "concept", SHEET)).convert("RGB")).astype(np.float64)
    colour = border_colour(pixels)
    background = background_mask(pixels, colour)
    pockets = enclosed_pockets(pixels, colour, background)
    subject, medallion = cell_subject(background, pockets)
    outside = ~subject
    rough = soft_alpha(subject)
    nearby, weight = nearby_colour(pixels, rough, outside)
    alpha = keyed_edge(pixels, colour, rough, outside, nearby, weight)
    alpha = np.where(ndimage.binary_dilation(subject, iterations=KEEP_REACH), alpha, 0)
    clean = remove_fringe(pixels, alpha, colour, nearby, weight)
    centre_x, centre_y, width = medallion_geometry(medallion)
    x0, y0, scale = placement(subject, centre_x, centre_y, width)
    premultiplied = resample(np.dstack([clean * alpha[..., None], alpha]), x0, y0, scale)
    unpremultiplied(premultiplied).save(os.path.join(work, f"{SLUG}_master.png"))
    if WRITE_STATIC:
        unpremultiplied(reduced(premultiplied)).save(os.path.join(arguments.item, "export", "static.png"))
    print(f"{SLUG}_cleanup: sheet {SHEET} cell {CELL}, background {colour.round(1).tolist()}, pockets {int(pockets.sum())} px, medallion width {width} px at ({centre_x:.1f}, {centre_y:.1f}), scale {scale:.4f}, master {WORK_SIDE}, static {'written' if WRITE_STATIC else 'from the loop'}")


main()
