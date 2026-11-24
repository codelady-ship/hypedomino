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


def paint(pixels, region, colour, alpha):
    if alpha.max() <= 0:
        return
    layer = np.dstack([colour * alpha[..., None], alpha])
    pixels[region] = pixels[region] * (1 - alpha[..., None]) + layer


def edge(signed_distance):
    return np.clip(0.5 - signed_distance, 0, 1)


SLUG = "storm_halo"
CLOUD_LIFT = 9.0
CLOUD_TOP_EDGE = 330.0
CLOUD_BOTTOM_EDGE = 880.0
CLOUD_BOTTOM_SHARE = 0.6
BOLT_ANGLES = [36, 324, 58, 302, 122, 238, 144, 216]
BOLT_FLASH_FRAMES = [0, 8, 5, 13, 10, 2, 15, 7]
BOLT_FADE = [1.0, 0.45]
BOLT_INNER = 418.0
BOLT_OUTER = 585.0
BOLT_SEGMENTS = 6
BOLT_JITTER = 18.0
BOLT_CORE = 4.0
BOLT_RIM = 8.0
BOLT_CORE_COLOUR = np.array([0.97, 0.95, 1.0])
BOLT_RIM_COLOUR = np.array([0.66, 0.52, 0.9])


def segment_distance(xs, ys, start, end):
    direction = np.subtract(end, start)
    length = max(float(np.dot(direction, direction)), 1e-9)
    along = np.clip(((xs - start[0]) * direction[0] + (ys - start[1]) * direction[1]) / length, 0, 1)
    return np.hypot(xs - start[0] - along * direction[0], ys - start[1] - along * direction[1]), along


def stroke(points, widths, xs, ys):
    distance = np.full(xs.shape, np.inf)
    width = np.zeros(xs.shape)
    for index in range(len(points) - 1):
        current, along = segment_distance(xs, ys, points[index], points[index + 1])
        local = widths[index] + (widths[index + 1] - widths[index]) * along
        closer = current - local < distance - width
        distance = np.where(closer, current, distance)
        width = np.where(closer, local, width)
    return distance, width


def bolt_path(angle, rng):
    radians = math.radians(angle)
    direction = np.array([math.sin(radians), -math.cos(radians)])
    normal = np.array([-direction[1], direction[0]])
    points = []
    for step in range(BOLT_SEGMENTS + 1):
        radius = BOLT_INNER + (BOLT_OUTER - BOLT_INNER) * step / BOLT_SEGMENTS
        offset = 0.0 if step == 0 else rng.uniform(-BOLT_JITTER, BOLT_JITTER)
        points.append(WORK / 2 + direction * radius + normal * offset)
    side = 1 if rng.uniform() < 0.5 else -1
    fork_at = points[BOLT_SEGMENTS // 2]
    fork = [fork_at, fork_at + direction * 30 + normal * 24 * side, fork_at + direction * 52 + normal * 30 * side]
    return points, fork


def paint_bolt(pixels, path, strength):
    points, fork = path
    every = np.array(points + fork)
    low = np.clip(np.floor(every.min(axis=0) - BOLT_RIM * 2).astype(int), 0, WORK)
    high = np.clip(np.ceil(every.max(axis=0) + BOLT_RIM * 2).astype(int), 0, WORK)
    ys, xs = np.mgrid[low[1]:high[1], low[0]:high[0]].astype(np.float64) + 0.5
    main_distance, main_width = stroke(points, np.linspace(1.0, 0.35, len(points)), xs, ys)
    fork_distance, fork_width = stroke(fork, np.array([0.7, 0.5, 0.25]), xs, ys)
    use_fork = fork_distance - fork_width < main_distance - main_width
    distance = np.where(use_fork, fork_distance, main_distance)
    width = np.where(use_fork, fork_width, main_width)
    core = edge(distance - BOLT_CORE * width)
    rim = edge(distance - BOLT_RIM * width)
    colour = BOLT_RIM_COLOUR * (1 - core[..., None]) + BOLT_CORE_COLOUR * core[..., None]
    paint(pixels, (slice(low[1], high[1]), slice(low[0], high[0])), colour, rim * strength)


def bolt_strength(frame, flash):
    since = (frame - flash) % FRAMES
    return BOLT_FADE[since] if since < len(BOLT_FADE) else 0.0


def cloud_lift(frame):
    rows, _ = grid()
    weight = (1 - smoothstep(0, CLOUD_TOP_EDGE, rows)) - smoothstep(CLOUD_BOTTOM_EDGE, WORK, rows) * CLOUD_BOTTOM_SHARE
    return -CLOUD_LIFT * math.sin(phase(frame)) * weight, np.zeros((WORK, WORK))


def load_parts(work, rng):
    master = Image.open(os.path.join(work, f"{SLUG}_master.png")).convert("RGBA")
    return {"ring": framed(master, box_of(master)), "bolts": [bolt_path(angle, rng) for angle in BOLT_ANGLES]}


def compose(frame, parts, ambient):
    pixels = warped(parts["ring"], *cloud_lift(frame))
    if not ambient:
        return pixels
    for path, flash in zip(parts["bolts"], BOLT_FLASH_FRAMES):
        strength = bolt_strength(frame, flash)
        if strength > 0:
            paint_bolt(pixels, path, strength)
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
