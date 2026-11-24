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


def rotated(xs, ys, angle):
    cosine, sine = math.cos(angle), math.sin(angle)
    return xs * cosine + ys * sine, -xs * sine + ys * cosine


def edge(signed_distance):
    return np.clip(0.5 - signed_distance, 0, 1)


def life(frame, start, length):
    age = ((frame - start) % FRAMES) / length
    return age if age < 1 else None


SLUG = "plum_bough"
TASSELS = [((358, 1084), 1), ((892, 1084), -1)]
TASSEL_HALF_WIDTH = 46.0
TASSEL_LENGTH = 156.0
TASSEL_SWING = math.radians(5.0)
PETAL_ANGLES = (62, 140)
PETAL_RADII = (70.0, 120.0)
PETALS = 22
PETAL_FALL = (140.0, 220.0)
PETAL_LIFE = (10, 15)
PETAL_SWAY = 16.0
PETAL_SIZE = (14.0, 20.0)
PETAL_COLOURS = [np.array([0.88, 0.45, 0.6]), np.array([0.98, 0.93, 0.86])]
PETAL_BASE_SHADE = 0.72


def tassel_swing(frame, box):
    rows, columns = grid()
    dy = np.zeros((WORK, WORK))
    dx = np.zeros((WORK, WORK))
    scale = WORK / (box[2] - box[0])
    for pivot, side in TASSELS:
        pivot_x, pivot_y = to_work(pivot, box)
        angle = TASSEL_SWING * side * math.sin(phase(frame))
        below = rows - pivot_y
        across = columns - pivot_x
        weight = smoothstep(0, 30 * scale, below) * (1 - smoothstep(TASSEL_HALF_WIDTH * scale, (TASSEL_HALF_WIDTH + 20) * scale, np.abs(across))) * (below < (TASSEL_LENGTH + 30) * scale)
        dx += weight * (across * (math.cos(angle) - 1) - below * math.sin(angle))
        dy += weight * (across * math.sin(angle) + below * (math.cos(angle) - 1))
    return dy, dx


def petal_tracks(rng, box, opening):
    scale = WORK / (box[2] - box[0])
    centre_x, centre_y, radius = opening
    tracks = []
    for index in range(PETALS):
        side = 1 if index % 2 else -1
        angle = math.radians(rng.uniform(*PETAL_ANGLES)) * side
        reach = radius + rng.uniform(*PETAL_RADII)
        tracks.append({
            "x": centre_x + math.sin(angle) * reach,
            "y": centre_y - math.cos(angle) * reach,
            "fall": rng.uniform(*PETAL_FALL) * scale,
            "length": int(rng.integers(*PETAL_LIFE)),
            "start": int(rng.integers(0, FRAMES)),
            "size": rng.uniform(*PETAL_SIZE),
            "spin": rng.uniform(-1.5, 1.5) * math.pi,
            "turn": rng.uniform(0, 2 * math.pi),
            "outward": side * rng.uniform(4, 22),
            "colour": PETAL_COLOURS[int(rng.integers(0, len(PETAL_COLOURS)))],
        })
    return tracks


def paint_petal(pixels, x, y, size, angle, flip, colour, strength):
    area = window(x, y, size * 1.6 + 2)
    if area is None:
        return
    region, xs, ys = area
    along, across = rotated(xs, ys, angle)
    width = size * (0.55 + 0.25 * flip)
    shape = np.sqrt((along / size) ** 2 + (across / width) ** 2) - 1
    notch = np.hypot(along - size * 1.05, across) - size * 0.28
    distance = np.maximum(shape * min(size, width), -notch)
    alpha = edge(distance) * strength
    shade = PETAL_BASE_SHADE + (1 - PETAL_BASE_SHADE) * np.clip((along / size + 1) / 2, 0, 1)
    paint(pixels, region, colour * shade[..., None], alpha)


def load_parts(work, rng):
    master = Image.open(os.path.join(work, f"{SLUG}_master.png")).convert("RGBA")
    box = box_of(master)
    ring = framed(master, box)
    return {"ring": ring, "box": box, "petals": petal_tracks(rng, box, opening_of(ring))}


def compose(frame, parts, ambient):
    pixels = warped(parts["ring"], *tassel_swing(frame, parts["box"]))
    if not ambient:
        return pixels
    for track in parts["petals"]:
        age = life(frame, track["start"], track["length"])
        if age is None:
            continue
        x = track["x"] + track["outward"] * age + PETAL_SWAY * math.sin(track["turn"] + age * 2 * math.pi)
        y = track["y"] + track["fall"] * age
        flip = math.cos(track["turn"] + age * 3 * math.pi)
        paint_petal(pixels, x, y, track["size"], track["turn"] + track["spin"] * age, abs(flip), track["colour"], math.sin(math.pi * age) ** 0.5)
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
