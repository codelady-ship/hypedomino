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


SLUG = "honey_comb"
DRIPS = [((443, 304), "hold"), ((544, 276), "hold"), ((621, 362), "hold"), ((703, 276), "hold"), ((800, 304), "hold"), ((212, 855), "creep"), ((1034, 855), "swell"), ((306, 996), "swell"), ((938, 996), "creep"), ((498, 1120), "creep"), ((750, 1120), "swell"), ((623, 1187), "hold")]
DRIP_STRETCH = 14.0
DRIP_HANG = 70.0
DRIP_HALF_WIDTH = 30.0
DROP_RADIUS = 10.0
DROP_FALL = 75.0
DROP_LIFE = 5
DROP_BODY = np.array([0.98, 0.62, 0.1])
DROP_SHINE = np.array([1.0, 0.93, 0.7])


def stretch(frame, kind):
    if kind == "creep":
        return ((frame - STATIC_FRAME + 1) % FRAMES) / FRAMES
    return math.sin(math.pi * (frame - STATIC_FRAME + 1) / FRAMES) ** 2


def release(kind):
    return {"creep": STATIC_FRAME - 1, "swell": 0}.get(kind)


def drip_pull(frame, drips):
    rows, columns = grid()
    dy = np.zeros((WORK, WORK))
    for (x, y), kind in drips:
        amount = DRIP_STRETCH * stretch(frame, kind)
        if amount == 0:
            continue
        across = 1 - smoothstep(DRIP_HALF_WIDTH * 0.6, DRIP_HALF_WIDTH, np.abs(columns - x))
        down = smoothstep(y - DRIP_HANG, y, rows) * (rows < y + DRIP_STRETCH * 2)
        dy += amount * across * down
    return dy, np.zeros((WORK, WORK))


def paint_drop(pixels, x, y, radius, strength):
    area = window(x, y, radius * 2.5)
    if area is None:
        return
    region, xs, ys = area
    taper = radius * np.clip(1 + ys / (2 * radius), 0, 1) ** 1.2
    shape = np.where(ys < 0, np.where(ys < -2 * radius, radius, np.abs(xs) - taper), np.hypot(xs, ys) - radius)
    shine = edge(np.hypot(xs + radius * 0.35, ys + radius * 0.1) - radius * 0.28)
    colour = DROP_BODY * (1 - shine[..., None]) + DROP_SHINE * shine[..., None]
    paint(pixels, region, colour, edge(shape) * strength)


def load_parts(work, rng):
    master = Image.open(os.path.join(work, f"{SLUG}_master.png")).convert("RGBA")
    box = box_of(master)
    return {"ring": framed(master, box), "drips": [(to_work(tip, box), kind) for tip, kind in DRIPS]}


def compose(frame, parts, ambient):
    pixels = warped(parts["ring"], *drip_pull(frame, parts["drips"]))
    if not ambient:
        return pixels
    for (x, y), kind in parts["drips"]:
        start = release(kind)
        if start is None:
            continue
        age = life(frame, start, DROP_LIFE)
        if age is None:
            continue
        paint_drop(pixels, x, y - DROP_RADIUS + DROP_FALL * age * age, DROP_RADIUS * (1 - 0.3 * age), 1 - age * age)
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
