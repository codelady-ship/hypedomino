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


def life(frame, start, length):
    age = ((frame - start) % FRAMES) / length
    return age if age < 1 else None


SLUG = "opal_wings"
FLAP = 0.42
FLAP_SHADE = 0.22
FLAP_BEATS_TOP = 1
FLAP_BEATS_SIDE = 2
WING_MIN_PIXELS = 400
WING_PADDING = 8
WING_JOIN = 24
DUST = 12
DUST_RISE = (40.0, 90.0)
DUST_SIZE = (9.0, 14.0)
DUST_LIFE = (5, 9)
DUST_COLOUR = np.array([1.0, 0.95, 1.0])


def butterflies(wings):
    solid = ndimage.binary_dilation(wings[..., 3] > 0.3, iterations=WING_JOIN)
    labels, count = ndimage.label(solid)
    sizes = ndimage.sum(solid, labels, range(1, count + 1))
    found = []
    for (rows, columns), size in zip(ndimage.find_objects(labels), sizes):
        if size < WING_MIN_PIXELS:
            continue
        found.append({
            "rows": slice(max(rows.start + WING_JOIN - WING_PADDING, 0), min(rows.stop - WING_JOIN + WING_PADDING, WORK)),
            "columns": slice(max(columns.start + WING_JOIN - WING_PADDING, 0), min(columns.stop - WING_JOIN + WING_PADDING, WORK)),
            "axis": (columns.start + columns.stop) / 2,
            "top": rows.start + WING_JOIN < WORK * 0.2,
        })
    return found


def flap(frame, beats):
    return FLAP * math.sin(beats * math.pi * (frame - STATIC_FRAME + 1) / FRAMES) ** 2


def flapped(wings, found, frame):
    result = np.zeros_like(wings)
    for butterfly in found:
        closing = flap(frame, FLAP_BEATS_TOP if butterfly["top"] else FLAP_BEATS_SIDE)
        width = 1 - closing
        rows, columns = butterfly["rows"], butterfly["columns"]
        ys, xs = np.mgrid[rows, columns].astype(np.float64)
        source_x = butterfly["axis"] + (xs + 0.5 - butterfly["axis"]) / width - 0.5
        local = np.dstack([ndimage.map_coordinates(wings[..., channel], [ys, source_x], order=1, mode="constant") for channel in range(4)])
        local[..., :3] *= 1 - FLAP_SHADE * closing / FLAP
        result[rows, columns] = local + result[rows, columns] * (1 - local[..., 3:4])
    return result


def dust_tracks(rng, found):
    tracks = []
    for index in range(DUST):
        butterfly = found[index % len(found)]
        rows, columns = butterfly["rows"], butterfly["columns"]
        tracks.append({
            "x": rng.uniform(columns.start, columns.stop),
            "y": rng.uniform(rows.start, (rows.start + rows.stop) / 2),
            "rise": rng.uniform(*DUST_RISE),
            "size": rng.uniform(*DUST_SIZE),
            "length": int(rng.integers(*DUST_LIFE)),
            "start": int(rng.integers(0, FRAMES)),
        })
    return tracks


def load_parts(work, rng):
    master = Image.open(os.path.join(work, f"{SLUG}_master.png")).convert("RGBA")
    box = box_of(master)
    under = framed(Image.open(os.path.join(work, f"{SLUG}_under.png")), box)
    wings = framed(Image.open(os.path.join(work, f"{SLUG}_butterflies.png")), box)
    found = butterflies(wings)
    return {"under": under, "wings": wings, "found": found, "dust": dust_tracks(rng, found)}


def compose(frame, parts, ambient):
    pixels = over(parts["under"], flapped(parts["wings"], parts["found"], frame))
    if not ambient:
        return pixels
    for track in parts["dust"]:
        age = life(frame, track["start"], track["length"])
        if age is None:
            continue
        paint_glint(pixels, track["x"], track["y"] - track["rise"] * age, track["size"] * math.sin(math.pi * age), DUST_COLOUR, math.sin(math.pi * age))
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
