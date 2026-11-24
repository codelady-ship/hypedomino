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


SLUG = "silk_ribbon"
TAIL_SWAY = 13.0
TAIL_WAVES = 0.022
TAIL_FROM = 30.0
TAIL_FULL = 180.0
BOW_LIFT = 7.0
BOW_EDGE = 280.0
STONES = [(629, 103), (173, 608), (1080, 608), (626, 990)]
GLINT_SIZE = 24.0
GLINT_LENGTH = 5
GLINT_COLOUR = np.array([1.0, 0.97, 0.9])


def ribbon_flutter(frame, opening):
    centre_x, centre_y, radius = opening
    rows, columns = grid()
    distance = np.hypot(columns - centre_x, rows - centre_y)
    outside = smoothstep(radius + TAIL_FROM, radius + TAIL_FULL, distance)
    lower = smoothstep(centre_y, centre_y + radius * 0.5, rows)
    side = np.sign(columns - centre_x)
    wave = math.sin(phase(frame)) * np.cos(TAIL_WAVES * rows) + 0.6 * math.sin(2 * phase(frame)) * np.sin(TAIL_WAVES * rows)
    dx = TAIL_SWAY * outside * lower * side * wave
    dy = -BOW_LIFT * math.sin(2 * phase(frame)) * (1 - smoothstep(0, BOW_EDGE, rows)) * smoothstep(radius * 0.15, radius * 0.6, np.abs(columns - centre_x))
    return dy, dx


def load_parts(work, rng):
    master = Image.open(os.path.join(work, f"{SLUG}_master.png")).convert("RGBA")
    box = box_of(master)
    ring = framed(master, box)
    found = [to_work(stone, box) for stone in STONES]
    starts = [int(value) for value in rng.permutation(FRAMES)[:len(found)]]
    return {"ring": ring, "opening": opening_of(ring), "stones": list(zip(found, starts))}


def compose(frame, parts, ambient):
    pixels = warped(parts["ring"], *ribbon_flutter(frame, parts["opening"]))
    if not ambient:
        return pixels
    for (x, y), start in parts["stones"]:
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
