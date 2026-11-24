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


SLUG = "season1_diamond"
CRYSTAL_BRIGHTNESS = 0.62
CRYSTAL_SATURATION = 0.16
CRYSTAL_REACH = 1.12
CRYSTAL_MIN_PIXELS = 300
TIP_SPACING = 60.0
TIP_INSET = 14.0
TIP_GLINT_SIZE = 50.0
TIP_GLINT_LENGTH = 5
CHASE_SIZE = 24.0
CHASE_SPARKS = 2
GLINT_COLOUR = np.array([1.0, 0.97, 0.93])


def crystal_tips(ring, centre, radius):
    colour = straight_colour(ring)
    rows, columns = grid()
    distance = np.hypot(columns - centre[0], rows - centre[1])
    crystal = (colour.mean(axis=2) > CRYSTAL_BRIGHTNESS) & (colour.max(axis=2) - colour.min(axis=2) < CRYSTAL_SATURATION)
    crystal &= (ring[..., 3] > 0.9) & (distance > radius * CRYSTAL_REACH) & (rows < centre[1])
    labels, count = ndimage.label(ndimage.binary_opening(crystal, iterations=2))
    tips = []
    for index in range(1, count + 1):
        ys, xs = np.nonzero(labels == index)
        if len(ys) < CRYSTAL_MIN_PIXELS:
            continue
        top = ys.min()
        tip = np.array([xs[ys == top].mean() + 0.5, top + 0.5])
        middle = np.array([xs.mean() + 0.5, ys.mean() + 0.5])
        toward = (middle - tip) / max(np.linalg.norm(middle - tip), 1e-6)
        tips.append(tuple(tip + toward * TIP_INSET))
    return sorted(apart(sorted(tips, key=lambda point: point[1]), TIP_SPACING))


def band_radius(ring, centre, radius):
    rows, columns = grid()
    lower = (ring[..., 3] > 0.9) & (rows > centre[1] + radius * 0.5)
    return float(np.median(np.hypot(columns - centre[0], rows - centre[1])[lower]))


def load_parts(work, rng):
    master = Image.open(os.path.join(work, f"{SLUG}_master.png")).convert("RGBA")
    ring = framed(master, box_of(master))
    centre, radius = opening(ring)
    tips = crystal_tips(ring, centre, radius)
    starts = [int(value) for value in rng.permutation(FRAMES)[:len(tips)]]
    return {"ring": ring, "centre": centre, "band": band_radius(ring, centre, radius), "tips": list(zip(tips, starts))}


def describe(parts):
    return f"{len(parts['tips'])} crystal tips, band radius {parts['band']:.1f} work px"


def compose(frame, parts, ambient):
    pixels = parts["ring"].copy()
    if not ambient:
        return pixels
    for (x, y), start in parts["tips"]:
        strength = pulse(frame, start, TIP_GLINT_LENGTH)
        if strength > 0:
            paint_glint(pixels, x, y, TIP_GLINT_SIZE * (0.6 + 0.4 * strength), GLINT_COLOUR, strength)
    for spark in range(CHASE_SPARKS):
        angle = phase(frame) / CHASE_SPARKS + 2 * math.pi * spark / CHASE_SPARKS
        x = parts["centre"][0] + parts["band"] * math.sin(angle)
        y = parts["centre"][1] - parts["band"] * math.cos(angle)
        paint_glint(pixels, x, y, CHASE_SIZE, GLINT_COLOUR, 1.0)
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
