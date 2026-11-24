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


SLUG = "season1_challenger"
RUSTLE = 6.0
RUSTLE_WAVES = 3.0
LEAF_INNER = 1.04
LEAF_OUTER = 1.2
MOTES = 12
MOTE_RADIUS = 1.22
MOTE_RISE = 150.0
MOTE_DRIFT = 18.0
MOTE_SIZE = 22.0
MOTE_COLOUR = np.array([1.0, 0.95, 0.82])
CREST_BRIGHTNESS = 0.7
CREST_SATURATION = 0.12
CREST_MIN_PIXELS = 400
CREST_GLINT_SIZE = 56.0
CREST_GLINT_START = 2
CREST_GLINT_LENGTH = 6
CREST_GLINT_COLOUR = np.array([1.0, 0.98, 0.95])


def rustle(frame, centre, radius):
    rows, columns = grid()
    across, down = columns - centre[0], rows - centre[1]
    distance = np.maximum(np.hypot(across, down), 1e-6)
    from_bottom = np.arctan2(np.abs(across), down)
    weight = smoothstep(radius * LEAF_INNER, radius * LEAF_OUTER, distance)
    push = RUSTLE * np.sin(phase(frame) - RUSTLE_WAVES * from_bottom) * weight
    return push * down / distance, push * across / distance


def crest(ring, centre, radius):
    colour = straight_colour(ring)
    rows, _ = grid()
    stone = (colour.mean(axis=2) > CREST_BRIGHTNESS) & (colour.max(axis=2) - colour.min(axis=2) < CREST_SATURATION)
    stone &= (ring[..., 3] > 0.9) & (rows < centre[1] - radius * 0.85)
    found = blobs(ndimage.binary_opening(stone, iterations=2), CREST_MIN_PIXELS)
    return min(found, key=lambda point: point[1]) if found else (centre[0], centre[1] - radius)


def motes(rng, centre, radius):
    result = []
    for index in range(MOTES):
        side = 1 if index % 2 == 0 else -1
        from_bottom = rng.uniform(0.35, 2.6)
        x = centre[0] + side * radius * MOTE_RADIUS * math.sin(from_bottom)
        y = centre[1] + radius * MOTE_RADIUS * math.cos(from_bottom)
        result.append((x, y, side * rng.uniform(0.3, 1.0), index * FRAMES // MOTES))
    return result


def paint_motes(pixels, frame, parts):
    for x, y, drift, start in parts["motes"]:
        age = ((frame - start) % FRAMES) / FRAMES
        strength = math.sin(math.pi * age) ** 2
        if strength > 0:
            paint_glint(pixels, x + drift * MOTE_DRIFT * age, y - MOTE_RISE * age, MOTE_SIZE * (0.7 + 0.3 * strength), MOTE_COLOUR, strength)


def load_parts(work, rng):
    master = Image.open(os.path.join(work, f"{SLUG}_master.png")).convert("RGBA")
    ring = framed(master, box_of(master))
    centre, radius = opening(ring)
    return {"ring": ring, "centre": centre, "radius": radius, "crest": crest(ring, centre, radius), "motes": motes(rng, centre, radius)}


def describe(parts):
    return f"opening radius {parts['radius']:.1f} work px, crest stone at ({parts['crest'][0]:.0f}, {parts['crest'][1]:.0f})"


def compose(frame, parts, ambient):
    pixels = warped(parts["ring"], *rustle(frame, parts["centre"], parts["radius"]))
    if not ambient:
        return pixels
    paint_motes(pixels, frame, parts)
    strength = pulse(frame, CREST_GLINT_START, CREST_GLINT_LENGTH)
    if strength > 0:
        paint_glint(pixels, *parts["crest"], CREST_GLINT_SIZE * (0.6 + 0.4 * strength), CREST_GLINT_COLOUR, strength)
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
