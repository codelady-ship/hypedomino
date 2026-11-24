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


def opening_of(ring):
    labels, _ = ndimage.label(ring[..., 3] < 0.5)
    inside = labels == labels[WORK // 2, WORK // 2]
    rows, columns = np.nonzero(inside)
    return columns.mean() + 0.5, rows.mean() + 0.5, math.sqrt(inside.sum() / math.pi)


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


def life(frame, start, length):
    age = ((frame - start) % FRAMES) / length
    return age if age < 1 else None


SLUG = "tide_pearl"
RIPPLE = 6.0
RIPPLE_WAVES = 6
RIPPLE_INNER_CALM = 40.0
BUBBLES = 16
BUBBLE_ANGLES = (100, 168)
BUBBLE_RADII = (10.0, 18.0)
BUBBLE_RISE = (90.0, 150.0)
BUBBLE_LIFE = (9, 14)
BUBBLE_WOBBLE = 5.0
BUBBLE_START_RADIUS = 55.0
BUBBLE_RIM = np.array([0.88, 0.98, 0.96])
BUBBLE_FILL = np.array([0.7, 0.92, 0.9])
BUBBLE_SHINE = np.array([1.0, 1.0, 1.0])


def ripple(frame, opening):
    centre_x, centre_y, radius = opening
    rows, columns = grid()
    across, down = columns - centre_x, rows - centre_y
    distance = np.hypot(across, down)
    angle = np.arctan2(across, -down)
    swing = RIPPLE * smoothstep(radius, radius + RIPPLE_INNER_CALM, distance)
    push = swing * (math.sin(phase(frame)) * np.cos(RIPPLE_WAVES * angle) + 0.5 * math.sin(2 * phase(frame)) * np.sin(RIPPLE_WAVES * angle))
    return push * down / np.maximum(distance, 1), push * across / np.maximum(distance, 1)


def bubble_tracks(rng, opening):
    centre_x, centre_y, radius = opening
    tracks = []
    for index in range(BUBBLES):
        side = 1 if index % 2 else -1
        angle = math.radians(rng.uniform(*BUBBLE_ANGLES)) * side
        start = radius + BUBBLE_START_RADIUS + rng.uniform(0, 40)
        tracks.append({
            "x": centre_x + math.sin(angle) * start,
            "y": centre_y - math.cos(angle) * start,
            "size": rng.uniform(*BUBBLE_RADII),
            "rise": rng.uniform(*BUBBLE_RISE),
            "length": int(rng.integers(*BUBBLE_LIFE)),
            "start": int(rng.integers(0, FRAMES)),
            "drift": side * rng.uniform(0, 14),
            "wobble": rng.uniform(0, 2 * math.pi),
        })
    return tracks


def paint_bubble(pixels, x, y, size, strength):
    area = window(x, y, size + 2)
    if area is None:
        return
    region, xs, ys = area
    distance = np.hypot(xs, ys)
    disc = edge(distance - size)
    rim = smoothstep(size * 0.55, size, distance) * disc
    shine = edge(np.hypot(xs + size * 0.38, ys + size * 0.38) - size * 0.24)
    alpha = np.maximum(disc * 0.24 + rim * 0.8, shine * 0.95) * strength
    weight = np.clip(rim / np.maximum(disc, 1e-6), 0, 1)[..., None]
    colour = BUBBLE_FILL * (1 - weight) + BUBBLE_RIM * weight
    colour = colour * (1 - shine[..., None]) + BUBBLE_SHINE * shine[..., None]
    paint(pixels, region, colour, alpha)


def load_parts(work, rng):
    master = Image.open(os.path.join(work, f"{SLUG}_master.png")).convert("RGBA")
    ring = framed(master, box_of(master))
    opening = opening_of(ring)
    return {"ring": ring, "opening": opening, "bubbles": bubble_tracks(rng, opening)}


def compose(frame, parts, ambient):
    pixels = warped(parts["ring"], *ripple(frame, parts["opening"]))
    if not ambient:
        return pixels
    for track in parts["bubbles"]:
        age = life(frame, track["start"], track["length"])
        if age is None:
            continue
        x = track["x"] + track["drift"] * age + BUBBLE_WOBBLE * math.sin(track["wobble"] + age * 3 * math.pi)
        y = track["y"] - track["rise"] * age
        paint_bubble(pixels, x, y, track["size"] * (0.7 + 0.3 * age), math.sin(math.pi * age))
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
