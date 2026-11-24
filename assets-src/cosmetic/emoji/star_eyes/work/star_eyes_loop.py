import argparse
import math
import os
import re

import numpy as np
from PIL import Image, ImageDraw
from scipy import ndimage
from scipy.spatial import ConvexHull

SLUG = "star_eyes"
FRAMES = 12
STATIC_FRAME = 7
WORK_SIDE = 480
REDUCTION = 4
BODY_PIVOT = (240.0, 430.0)
FEATHER = 1.2
GROW = 2
FILL_SIGMAS = (3, 6, 12, 24, 48)
FRAME_NAME = re.compile(r"^\d{4}\.png$")
BODY = {'bounce': 8, 'beats': 2}
PARTS = [
    {'name': 'star_left', 'region': [179, 212, 52, 52], 'colour': 'gold', 'motion': 'pulse', 'amount': 0.28, 'beats': 2},
    {'name': 'star_right', 'region': [303, 212, 52, 52], 'colour': 'gold', 'motion': 'pulse', 'amount': 0.28, 'beats': 2, 'phase': 0.25},
]


def colour_test(pixels, name):
    red, green, blue = pixels[..., 0], pixels[..., 1], pixels[..., 2]
    spread = pixels.max(axis=2) - pixels.min(axis=2)
    tests = {
        "red": (red > green + 50) & (red > blue + 40),
        "crystal": (spread < 34) & (pixels.mean(axis=2) > 110),
        "gold": (red > blue + 70) & (green > blue + 35) & (red > 120),
        "dark": pixels.max(axis=2) < 110,
        "any": np.ones(red.shape, dtype=bool),
    }
    return tests[name]


def part_mask(master, part):
    centre_x, centre_y, radius_x, radius_y = part["region"]
    rows, columns = np.mgrid[0:WORK_SIDE, 0:WORK_SIDE]
    inside = ((columns - centre_x) / radius_x) ** 2 + ((rows - centre_y) / radius_y) ** 2 <= 1
    selected = inside & colour_test(master[..., :3], part.get("colour", "any")) & (master[..., 3] > 0.05)
    labels, count = ndimage.label(selected)
    if count > 1:
        sizes = ndimage.sum(selected, labels, range(1, count + 1))
        selected = np.isin(labels, np.where(sizes >= part.get("min_pixels", 30))[0] + 1)
    if part.get("hull"):
        selected = hull(selected)
    selected = ndimage.binary_fill_holes(ndimage.binary_dilation(selected, iterations=GROW)) & inside
    return np.clip(ndimage.gaussian_filter(selected.astype(np.float64), FEATHER), 0, 1) * (master[..., 3] > 0)


def hull(selected):
    rows, columns = np.nonzero(selected)
    points = np.column_stack([columns, rows])
    corners = points[ConvexHull(points).vertices]
    canvas = Image.new("L", (WORK_SIDE, WORK_SIDE), 0)
    ImageDraw.Draw(canvas).polygon([tuple(map(float, corner)) for corner in corners], fill=255)
    return np.asarray(canvas) > 0


def filled(base, hole):
    known = (hole < 0.02).astype(np.float64)
    result = base.copy()
    missing = known < 1
    for sigma in FILL_SIGMAS:
        weight = ndimage.gaussian_filter(known, sigma)
        estimate = np.dstack([ndimage.gaussian_filter(base[..., c] * known, sigma) for c in range(4)]) / np.maximum(weight, 1e-6)[..., None]
        take = missing & (weight > 0.2)
        result[take] = estimate[take]
        known = np.where(take, 1.0, known)
        missing &= ~take
    return base * (1 - hole[..., None]) + result * hole[..., None]


def matrix(pivot, offset=(0.0, 0.0), angle=0.0, scale=(1.0, 1.0)):
    radians = math.radians(angle)
    cos, sin = math.cos(radians), math.sin(radians)
    linear = np.array([[cos * scale[0], -sin * scale[1]], [sin * scale[0], cos * scale[1]]])
    result = np.eye(3)
    result[:2, :2] = linear
    result[:2, 2] = np.array(pivot) + np.array(offset) - linear @ np.array(pivot)
    return result


def body_matrix(t):
    u = 2 * math.pi * t
    beat = BODY.get("beats", 1) * u
    lift = BODY.get("bounce", 0.0) * (1 - math.cos(beat)) / 2
    stretch = BODY.get("squash", 0.0) * math.sin(beat)
    angle = BODY.get("tilt", 0.0) * math.sin(BODY.get("tilt_beats", 1) * u)
    shift = BODY.get("sway", 0.0) * math.sin(u) + BODY.get("jitter", 0.0) * math.sin(5 * u)
    return matrix(BODY_PIVOT, (shift, -lift), angle, (1 - stretch / 2, 1 + stretch))


def cycle(t, part):
    q = (t * part.get("beats", 1) + 0.25 + part.get("phase", 0.0)) % 1.0
    if q < 0.25:
        grow = q / 0.25
        return 0.0, grow, 0.6 + 0.4 * grow
    travel = ((q - 0.25) / 0.75) ** 2
    return part["amount"] * travel, 1 - travel, 1.0


def part_motion(t, part, pivot):
    u = 2 * math.pi * t
    beat = part.get("beats", 1) * u + 2 * math.pi * part.get("phase", 0.0)
    rest = beat - 2 * math.pi * part.get("phase", 0.0)
    motion, amount = part["motion"], part.get("amount", 0.0)
    if motion == "pulse":
        grow = 1 + amount * (1 - math.cos(beat)) / 2
        return matrix(pivot, scale=(grow, grow)), 1.0
    if motion == "spin":
        return matrix(pivot, angle=amount * 360 * t * part.get("beats", 1)), 1.0
    if motion == "swing":
        return matrix(pivot, angle=amount * math.sin(beat)), 1.0
    if motion == "bob":
        return matrix(pivot, offset=(0.0, -amount * math.sin(beat))), 1.0
    if motion == "tip":
        share = (1 - math.cos(rest)) / 2
        return matrix(pivot, offset=(0.0, -part.get("lift", 0.0) * share), angle=amount * share), 1.0
    if motion == "flicker":
        return matrix(pivot, scale=(1 - amount * math.sin(beat) / 3, 1 + amount * math.sin(beat))), 1.0
    if motion in ("drop", "rise"):
        travel, strength, grow = cycle(t, part)
        return matrix(pivot, offset=(0.0, travel), scale=(grow, grow)), strength
    raise ValueError(motion)


def warped(premultiplied, transform):
    inverse = np.linalg.inv(transform)
    coefficients = (inverse[0, 0], inverse[0, 1], inverse[0, 2], inverse[1, 0], inverse[1, 1], inverse[1, 2])
    channels = [
        np.asarray(Image.fromarray(premultiplied[..., c].astype(np.float32)).transform((WORK_SIDE, WORK_SIDE), Image.Transform.AFFINE, coefficients, resample=Image.Resampling.BICUBIC))
        for c in range(4)
    ]
    result = np.dstack(channels).astype(np.float64)
    result[..., 3] = np.clip(result[..., 3], 0, 1)
    result[..., :3] = np.clip(result[..., :3], 0, result[..., 3:4] * 255)
    return result


def pivot_of(part, mask):
    if "pivot" in part:
        return part["pivot"]
    y, x = ndimage.center_of_mass(mask)
    if part["motion"] in ("drop", "rise"):
        rows = np.where(mask.max(axis=1) > 0.5)[0]
        return (x, float(rows[0]))
    return (x, y)


def unpremultiplied(premultiplied):
    alpha = premultiplied[..., 3]
    rgb = np.where(alpha[..., None] > 0, premultiplied[..., :3] / np.maximum(alpha, 1e-6)[..., None], 0)
    return Image.fromarray(np.dstack([np.clip(rgb, 0, 255), alpha * 255]).round().astype(np.uint8))


def reduced(premultiplied):
    side = WORK_SIDE // REDUCTION
    return premultiplied.reshape(side, REDUCTION, side, REDUCTION, 4).mean(axis=(1, 3))


def clear_frames(folder):
    os.makedirs(folder, exist_ok=True)
    for name in os.listdir(folder):
        if FRAME_NAME.match(name):
            os.remove(os.path.join(folder, name))


def main():
    parser = argparse.ArgumentParser(prog=f"{SLUG}_loop.py")
    parser.add_argument("item")
    arguments = parser.parse_args()
    work = os.path.join(arguments.item, "work")
    export = os.path.join(arguments.item, "export")
    image = np.asarray(Image.open(os.path.join(work, f"{SLUG}_master.png")).convert("RGBA")).astype(np.float64)
    alpha = image[..., 3] / 255
    master = np.dstack([image[..., :3] * alpha[..., None], alpha])
    straight = np.dstack([image[..., :3], alpha])
    base = master.copy()
    layers = []
    for part in PARTS:
        mask = part_mask(straight, part)
        layers.append((part, master * mask[..., None], pivot_of(part, mask)))
        if part.get("hole") == "fill":
            base = filled(base, mask)
        elif part.get("hole") == "clear":
            base = base * (1 - mask[..., None])
    unpremultiplied(base).save(os.path.join(work, f"{SLUG}_base.png"))
    for part, layer, _ in layers:
        unpremultiplied(layer).save(os.path.join(work, f"{SLUG}_{part['name']}.png"))
    anim = os.path.join(export, "anim")
    clear_frames(anim)
    for index in range(1, FRAMES + 1):
        t = (index - STATIC_FRAME) / FRAMES
        body = body_matrix(t)
        frame = warped(base, body)
        for part, layer, pivot in layers:
            motion, strength = part_motion(t, part, pivot)
            moved = warped(layer, body @ motion) * strength
            frame = moved + frame * (1 - moved[..., 3:4])
        small = unpremultiplied(reduced(frame))
        small.save(os.path.join(anim, f"{index:04d}.png"))
        if index == STATIC_FRAME:
            small.save(os.path.join(export, "static.png"))
    print(f"{SLUG}_loop: parts {len(PARTS)} ({', '.join(part['name'] for part in PARTS) or 'none'}), {FRAMES} frames at {WORK_SIDE // REDUCTION}x{WORK_SIDE // REDUCTION}, static frame {STATIC_FRAME}")


main()
