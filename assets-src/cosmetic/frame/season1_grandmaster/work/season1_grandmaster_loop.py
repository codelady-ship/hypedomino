import argparse
import math
import os
import re

import numpy as np
from PIL import Image
from scipy import ndimage

WORK = 1024
CANVAS = 256
FRAMES = 16
STATIC_FRAME = 9
MARGIN = 0.02
SEED = 20261116
FRAME_NAME = re.compile(r"^\d{4}\.png$")


def square_box(alpha):
    rows = np.where(alpha.max(axis=1) > 0.5)[0]
    columns = np.where(alpha.max(axis=0) > 0.5)[0]
    centre_x = (columns[0] + columns[-1]) / 2
    centre_y = (rows[0] + rows[-1]) / 2
    side = max(columns[-1] - columns[0], rows[-1] - rows[0]) * (1 + 2 * MARGIN)
    return tuple(int(round(value)) for value in (centre_x - side / 2, centre_y - side / 2, centre_x + side / 2, centre_y + side / 2))


def box_of(image):
    return square_box(np.asarray(image)[..., 3] / 255)


def framed(image, box):
    resized = image.convert("RGBA").crop(box).convert("RGBa").resize((WORK, WORK), Image.Resampling.LANCZOS)
    return np.asarray(resized).astype(np.float64) / 255


def grid():
    rows, columns = np.mgrid[0:WORK, 0:WORK].astype(np.float64)
    return rows + 0.5, columns + 0.5


def warped(layer, dy, dx):
    rows, columns = np.mgrid[0:WORK, 0:WORK].astype(np.float64)
    coordinates = [rows - dy, columns - dx]
    return np.dstack([ndimage.map_coordinates(layer[..., channel], coordinates, order=1, mode="constant") for channel in range(4)])


def smoothstep(edge0, edge1, value):
    x = np.clip((value - edge0) / (edge1 - edge0), 0, 1)
    return x * x * (3 - 2 * x)


def phase(frame, offset=0.0):
    return 2 * math.pi * (frame / FRAMES + offset)


def over(base, top):
    return base * (1 - top[..., 3:4]) + top


def window(x, y, reach):
    left = max(int(math.floor(x - reach)), 0)
    top = max(int(math.floor(y - reach)), 0)
    right = min(int(math.ceil(x + reach)), WORK)
    bottom = min(int(math.ceil(y + reach)), WORK)
    if right <= left or bottom <= top:
        return None
    ys, xs = np.mgrid[top:bottom, left:right].astype(np.float64) + 0.5
    return (slice(top, bottom), slice(left, right)), xs - x, ys - y


def paint(pixels, region, colour, alpha):
    if alpha.max() <= 0:
        return
    layer = np.dstack([colour * alpha[..., None], alpha])
    pixels[region] = pixels[region] * (1 - alpha[..., None]) + layer


def edge(signed_distance):
    return np.clip(0.5 - signed_distance, 0, 1)


def paint_glint(pixels, x, y, size, colour, strength):
    area = window(x, y, size + 2)
    if area is None or size < 1:
        return
    region, xs, ys = area
    star = np.sqrt(np.abs(xs) / size) + np.sqrt(np.abs(ys) / size)
    paint(pixels, region, colour, edge((star - 1) * size * 0.25) * strength)


def pulse(frame, start, length):
    age = ((frame - start) % FRAMES) / length
    return math.sin(math.pi * age) if age < 1 else 0.0


def blobs(mask, minimum):
    labels, count = ndimage.label(mask)
    if count == 0:
        return []
    indices = range(1, count + 1)
    sizes = ndimage.sum(mask, labels, indices)
    centres = ndimage.center_of_mass(mask, labels, indices)
    return [(column + 0.5, row + 0.5) for (row, column), size in zip(centres, sizes) if size >= minimum]


def apart(points, spacing):
    kept = []
    for point in points:
        if all(math.hypot(point[0] - other[0], point[1] - other[1]) >= spacing for other in kept):
            kept.append(point)
    return kept


def opening(ring):
    labels, _ = ndimage.label(ring[..., 3] < 0.5)
    inside = labels == labels[WORK // 2, WORK // 2]
    rows, columns = np.nonzero(inside)
    return (columns.mean() + 0.5, rows.mean() + 0.5), math.sqrt(inside.sum() / math.pi)


def straight_colour(ring):
    alpha = np.maximum(ring[..., 3:4], 1e-6)
    return ring[..., :3] / alpha


SLUG = "season1_grandmaster"
ORBIT_GEMS = 12
GEM_MIN_PIXELS = 400
GEM_PAD = 6
GEM_GROWTH = 3
ORBIT_DIRECTION = 1


def orbit_parts(ring):
    labels, count = ndimage.label(ring[..., 3] > 0.05)
    sizes = ndimage.sum(np.ones(labels.shape), labels, range(1, count + 1))
    band = int(np.argmax(sizes)) + 1
    gems = [index for index, size in enumerate(sizes, start=1) if index != band and size >= GEM_MIN_PIXELS]
    centres = ndimage.center_of_mass(np.ones(labels.shape), labels, gems)
    return labels, gems, [(column + 0.5, row + 0.5) for row, column in centres]


def gem_sprite(ring, labels, index):
    mask = ndimage.binary_dilation(labels == index, iterations=GEM_GROWTH)
    rows, columns = np.nonzero(mask)
    top, bottom = rows.min() - GEM_PAD, rows.max() + GEM_PAD + 1
    left, right = columns.min() - GEM_PAD, columns.max() + GEM_PAD + 1
    sprite = ring[top:bottom, left:right] * mask[top:bottom, left:right, None]
    rows_mass, columns_mass = ndimage.center_of_mass(sprite[..., 3])
    return sprite, (columns_mass + 0.5, rows_mass + 0.5)


def place(pixels, sprite, anchor, x, y):
    left = x - anchor[0]
    top = y - anchor[1]
    column, row = int(math.floor(left)), int(math.floor(top))
    padded = np.pad(sprite, ((1, 1), (1, 1), (0, 0)))
    shifted = ndimage.shift(padded, (top - row, left - column, 0), order=1, mode="constant")
    height, width = shifted.shape[:2]
    row, column = row - 1, column - 1
    target = (slice(max(row, 0), min(row + height, WORK)), slice(max(column, 0), min(column + width, WORK)))
    source = shifted[target[0].start - row:target[0].stop - row, target[1].start - column:target[1].stop - column]
    pixels[target] = over(pixels[target], source)


def load_parts(work, rng):
    master = Image.open(os.path.join(work, f"{SLUG}_master.png")).convert("RGBA")
    ring = framed(master, box_of(master))
    centre, _ = opening(ring)
    labels, gems, centres = orbit_parts(ring)
    top = gems[int(np.argmin([y for _, y in centres]))]
    sprite, anchor = gem_sprite(ring, labels, top)
    removed = ndimage.binary_dilation(np.isin(labels, gems), iterations=GEM_GROWTH)
    radius = float(np.mean([math.hypot(x - centre[0], y - centre[1]) for x, y in centres]))
    return {"band": ring * ~removed[..., None], "sprite": sprite, "anchor": anchor, "centre": centre, "radius": radius, "found": len(gems)}


def describe(parts):
    return f"{parts['found']} orbit gems found, {ORBIT_GEMS} placed on radius {parts['radius']:.1f} work px"


def compose(frame, parts, ambient):
    pixels = parts["band"].copy()
    progress = ((frame - STATIC_FRAME + 1) % FRAMES) / FRAMES
    for gem in range(ORBIT_GEMS):
        angle = 2 * math.pi * (gem + ORBIT_DIRECTION * progress) / ORBIT_GEMS
        x = parts["centre"][0] + parts["radius"] * math.sin(angle)
        y = parts["centre"][1] - parts["radius"] * math.cos(angle)
        place(pixels, parts["sprite"], parts["anchor"], x, y)
    return pixels


def to_image(pixels):
    data = (pixels.clip(0, 1) * 255).round().astype(np.uint8)
    return Image.frombytes("RGBa", (WORK, WORK), data.tobytes()).resize((CANVAS, CANVAS), Image.Resampling.LANCZOS).convert("RGBA")


def clear_frames(folder):
    os.makedirs(folder, exist_ok=True)
    for name in os.listdir(folder):
        if FRAME_NAME.match(name):
            os.remove(os.path.join(folder, name))


def main():
    parser = argparse.ArgumentParser(prog=f"{SLUG}_loop.py")
    parser.add_argument("work")
    parser.add_argument("export")
    arguments = parser.parse_args()
    parts = load_parts(arguments.work, np.random.default_rng(SEED))
    anim = os.path.join(arguments.export, "anim")
    clear_frames(anim)
    for index in range(1, FRAMES + 1):
        to_image(compose(index - 1, parts, True)).save(os.path.join(anim, f"{index:04d}.png"))
    to_image(compose(STATIC_FRAME - 1, parts, False)).save(os.path.join(arguments.export, "static.png"))
    print(f"{SLUG}_loop: {describe(parts)}; wrote {FRAMES} frames and the static (frame {STATIC_FRAME} without ambient detail) at {CANVAS}x{CANVAS}")


main()
