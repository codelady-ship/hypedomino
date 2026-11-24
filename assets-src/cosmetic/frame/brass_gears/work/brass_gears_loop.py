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
SEED = 20261115
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


def to_work(point, box):
    scale = WORK / (box[2] - box[0])
    return (point[0] - box[0]) * scale, (point[1] - box[1]) * scale


def framed(image, box):
    resized = image.convert("RGBA").crop(box).convert("RGBa").resize((WORK, WORK), Image.Resampling.LANCZOS)
    return np.asarray(resized).astype(np.float64) / 255


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


def pulse(frame, start, length):
    age = ((frame - start) % FRAMES) / length
    return math.sin(math.pi * age) if age < 1 else 0.0


SLUG = "brass_gears"
GEARS = [((630, 160), 156, 24, 1), ((311, 284), 99, 20, -1), ((940, 282), 99, 20, 1), ((196, 427), 61, 16, 1), ((1054, 428), 61, 16, -1), ((228, 875), 96, 19, -1), ((1021, 877), 96, 19, 1)]
GEAR_REACH = 6
RUBY_MIN_PIXELS = 150
RUBY_SPACING = 50.0
GLINT_SIZE = 26.0
GLINT_LENGTH = 5
GLINT_COLOUR = np.array([1.0, 0.94, 0.92])


def turned(layer, centre, reach, angle):
    centre_x, centre_y = centre
    top, bottom = max(int(centre_y - reach), 0), min(int(centre_y + reach) + 1, WORK)
    left, right = max(int(centre_x - reach), 0), min(int(centre_x + reach) + 1, WORK)
    ys, xs = np.mgrid[top:bottom, left:right].astype(np.float64) + 0.5
    cosine, sine = math.cos(angle), math.sin(angle)
    across, down = xs - centre_x, ys - centre_y
    source_x = centre_x + across * cosine + down * sine - 0.5
    source_y = centre_y - across * sine + down * cosine - 0.5
    inside = (np.hypot(across, down) <= reach)[..., None]
    local = np.dstack([ndimage.map_coordinates(layer[..., channel], [source_y, source_x], order=1, mode="constant") for channel in range(4)])
    return (slice(top, bottom), slice(left, right)), local * inside


def gear_teeth(teeth, gears, frame):
    result = np.zeros_like(teeth)
    progress = ((frame - STATIC_FRAME + 1) % FRAMES) / FRAMES
    for centre, reach, count, direction in gears:
        pitch = direction * 2 * math.pi / count
        region, ahead = turned(teeth, centre, reach, pitch * progress)
        _, behind = turned(teeth, centre, reach, pitch * (progress - 1))
        result[region] += ahead * (1 - progress) + behind * progress
    return result


def rubies(ring):
    alpha = np.maximum(ring[..., 3], 1e-6)
    red, green, blue = (ring[..., channel] / alpha for channel in range(3))
    stone = (red > green + 0.35) & (red > blue + 0.2) & (green < 0.25) & (ring[..., 3] > 0.9)
    return apart(blobs(ndimage.binary_opening(stone, iterations=2), RUBY_MIN_PIXELS), RUBY_SPACING)


def load_parts(work, rng):
    master = Image.open(os.path.join(work, f"{SLUG}_master.png")).convert("RGBA")
    box = box_of(master)
    scale = WORK / (box[2] - box[0])
    gears = [(to_work(centre, box), (reach + GEAR_REACH) * scale, count, direction) for centre, reach, count, direction in GEARS]
    stones = rubies(framed(master, box))
    starts = [int(value) for value in rng.permutation(FRAMES)[:len(stones)]]
    return {
        "under": framed(Image.open(os.path.join(work, f"{SLUG}_under.png")), box),
        "teeth": framed(Image.open(os.path.join(work, f"{SLUG}_teeth.png")), box),
        "gears": gears,
        "rubies": list(zip(stones, starts)),
    }


def compose(frame, parts, ambient):
    pixels = over(parts["under"], gear_teeth(parts["teeth"], parts["gears"], frame))
    if not ambient:
        return pixels
    for (x, y), start in parts["rubies"]:
        strength = pulse(frame, start, GLINT_LENGTH)
        if strength > 0:
            paint_glint(pixels, x, y, GLINT_SIZE * (0.6 + 0.4 * strength), GLINT_COLOUR, strength)
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
    print(f"{SLUG}_loop: wrote {FRAMES} frames and the static (frame {STATIC_FRAME} without ambient detail) at {CANVAS}x{CANVAS}")


main()
