import argparse
import math
import os
import re

import numpy as np
from PIL import Image, ImageDraw
from scipy import ndimage, optimize

SLUG = "hummingbird_clip"
CANONICAL = "20261116-01.png"
BLINK_EDIT = "20261116-02.png"
CROP = (63, -186, 1184, 935)
SUPER = 768
CANVAS = 192
FRAMES = 16
STATIC_FRAME = 9
CIRCLE_FEATHER = 2.25
HEAD = (385, 380, 320, 262)
HEAD_FALLOFF = 0.3
NECK_Y = 630
NECK_BAND = 34
SHOULDER_Y = 730
HEAD_RIDE = 0.4
BREATH = 7.36
BREATH_WIDTH = 0.012
TILT = 1.4
NOD = 1.84
EYES = [(341, 450, 32, 18), (463, 432, 33, 18)]
EYE_FEATHER = 4.6
BLINK = {4: 1.0, 5: 1.0}
PARTS = [
    {"name": "earring_left", "polygon": [(276, 544), (284, 544), (288, 556), (292, 570), (292, 592), (268, 592), (268, 570), (272, 556)], "pivot": (280, 544), "length": 46, "swing": 6.0, "lag": 0.4, "grow": 1, "behind": "fill"},
    {"name": "earring_right", "polygon": [(533, 519), (541, 519), (546, 530), (549, 545), (549, 567), (525, 567), (525, 545), (528, 530)], "pivot": (537, 519), "length": 46, "swing": 6.0, "lag": 0.8, "grow": 1, "behind": "fill"},
]
GLINTS = []
GLINT_PEAK = 0.55
GLINT_FLOOR = 0.3
RIPPLES = [
]
GLINT_COLOUR = (255, 247, 226)
REGISTER_SCALE = 4
FRAME_NAME = re.compile(r"^\d{4}\.png$")


def load_rgba(path):
    return np.asarray(Image.open(path).convert("RGBA")).astype(np.float64) / 255


def premultiply(rgba):
    return np.dstack([rgba[..., :3] * rgba[..., 3:], rgba[..., 3]])


def smoothstep(edge0, edge1, value):
    x = np.clip((value - edge0) / (edge1 - edge0), 0, 1)
    return x * x * (3 - 2 * x)


def grid(size):
    ys, xs = np.mgrid[0:size, 0:size].astype(np.float64)
    return xs + 0.5, ys + 0.5


def ellipse_weight(xs, ys, ellipse, falloff):
    cx, cy, rx, ry = ellipse
    radius = np.hypot((xs - cx) / rx, (ys - cy) / ry)
    return 1 - smoothstep(1, 1 + falloff, radius)


def affine_matrix(params, centre):
    tx, ty, scale, angle = params
    cos, sin = math.cos(angle), math.sin(angle)
    matrix = scale * np.array([[cos, -sin], [sin, cos]])
    offset = np.array(centre) - matrix @ np.array(centre) + np.array([ty, tx])
    return matrix, offset


def warp_affine(image, params, centre, order):
    matrix, offset = affine_matrix(params, centre)
    channels = [ndimage.affine_transform(image[..., c], matrix, offset, order=order, mode="nearest") for c in range(image.shape[2])]
    return np.dstack(channels)


def raw_eye_mask(shape):
    side = (CROP[2] - CROP[0]) / SUPER
    ys, xs = np.mgrid[0:shape[0], 0:shape[1]].astype(np.float64) + 0.5
    mask = np.zeros(shape, dtype=bool)
    for x, y, rx, ry in EYES:
        cx, cy = CROP[0] + x * side, CROP[1] + y * side
        mask |= np.hypot((xs - cx) / (rx * side * 1.6), (ys - cy) / (ry * side * 2.2)) < 1
    return mask


def register(canonical, edit):
    small_canonical = canonical[::REGISTER_SCALE, ::REGISTER_SCALE].mean(axis=2)
    small_edit = edit[::REGISTER_SCALE, ::REGISTER_SCALE].mean(axis=2)
    side = (CROP[2] - CROP[0])
    region = np.zeros(canonical.shape[:2], dtype=bool)
    region[max(CROP[1], 0):CROP[1] + side, max(CROP[0], 0):CROP[0] + side] = True
    region &= ~raw_eye_mask(canonical.shape[:2])
    weight = region[::REGISTER_SCALE, ::REGISTER_SCALE]
    centre = (small_canonical.shape[0] / 2, small_canonical.shape[1] / 2)

    def cost(params):
        moved = warp_affine(small_edit[..., None], (params[0], params[1], 1 + params[2], params[3]), centre, 1)[..., 0]
        return float(np.mean((moved - small_canonical)[weight] ** 2))

    start = np.zeros(4)
    result = optimize.minimize(cost, start, method="Powell", options={"xtol": 1e-4, "ftol": 1e-7, "maxiter": 4000})
    tx, ty, scale, angle = result.x
    full = (tx * REGISTER_SCALE, ty * REGISTER_SCALE, 1 + scale, angle)
    full_centre = (canonical.shape[0] / 2, canonical.shape[1] / 2)
    return warp_affine(edit, full, full_centre, 3), full, cost(start), result.fun


def to_portrait(raw):
    image = Image.fromarray(np.clip(raw * 255, 0, 255).round().astype(np.uint8), mode="RGB")
    return np.asarray(image.crop(CROP).resize((SUPER, SUPER), Image.LANCZOS)).astype(np.float64) / 255


def eye_mask(xs, ys, scale):
    mask = np.zeros(xs.shape)
    for x, y, rx, ry in EYES:
        inside = np.hypot((xs - x) / (rx * scale), (ys - y) / (ry * scale)) < 1
        mask = np.maximum(mask, inside.astype(np.float64))
    return mask


def closed_portrait(portrait, closed_rgb, xs, ys):
    core = ndimage.gaussian_filter(eye_mask(xs, ys, 1.0), EYE_FEATHER)
    ring = (eye_mask(xs, ys, 1.6) > 0) & (eye_mask(xs, ys, 1.15) == 0)
    offset = (portrait[..., :3][ring] - closed_rgb[ring]).mean(axis=0)
    matched = np.clip(closed_rgb + offset, 0, 1)
    rgb = portrait[..., :3] * (1 - core[..., None]) + matched * core[..., None]
    return np.dstack([rgb, portrait[..., 3]]), core, offset


def polygon_mask(points, feather, grow):
    canvas = Image.new("L", (SUPER, SUPER), 0)
    ImageDraw.Draw(canvas).polygon(points, fill=255)
    mask = np.asarray(canvas) > 127
    if grow > 0:
        mask = ndimage.binary_dilation(mask, iterations=grow)
    mask = mask.astype(np.float64)
    return np.clip(ndimage.gaussian_filter(mask, feather), 0, 1) if feather > 0 else mask


def fill_holes(premultiplied, hole):
    known = (~hole).astype(np.float64)
    estimate = np.zeros_like(premultiplied)
    for sigma in (48, 24, 12, 6, 3):
        weight = ndimage.gaussian_filter(known, sigma)
        blurred = np.dstack([ndimage.gaussian_filter(premultiplied[..., c] * known, sigma) for c in range(4)])
        reliable = weight > 0.05
        estimate = np.where(reliable[..., None], blurred / np.maximum(weight, 1e-6)[..., None], estimate)
    cover = estimate[..., 3]
    solid = smoothstep(0.35, 0.65, cover)
    estimate = np.dstack([estimate[..., :3] / np.maximum(cover, 1e-6)[..., None] * solid[..., None], solid])
    return np.where(hole[..., None], estimate, premultiplied)


def split_parts(open_premultiplied, closed_premultiplied):
    layers = []
    hole = np.zeros((SUPER, SUPER), dtype=bool)
    keep = np.ones((SUPER, SUPER))
    for part in PARTS:
        mask = polygon_mask(part["polygon"], 1.5, part.get("grow", 0))
        layers.append((open_premultiplied * mask[..., None], part))
        if part["behind"] == "clear":
            keep *= 1 - mask
        else:
            hole |= ndimage.binary_dilation(mask > 0.02, iterations=3) & (open_premultiplied[..., 3] > 0.02)
    bases = [fill_holes(base, hole) * keep[..., None] for base in (open_premultiplied, closed_premultiplied)]
    return layers, bases[0], bases[1], hole | (keep < 0.98)


def phase(frame):
    return 2 * math.pi * (frame - (STATIC_FRAME - 1)) / FRAMES


def breath(frame):
    return (1 - math.cos(phase(frame))) / 2


def sample(premultiplied, source_x, source_y):
    coordinates = [source_y - 0.5, source_x - 0.5]
    return np.dstack([ndimage.map_coordinates(premultiplied[..., c], coordinates, order=1, mode="constant", cval=0.0) for c in range(4)])


def swayed(layer, part, frame, xs, ys):
    px, py = part["pivot"]
    t = phase(frame)
    lag = part.get("lag", 0.0)
    angle = math.radians(part["swing"]) * (math.sin(lag) - math.sin(t + lag) + part.get("flutter", 0.0) * math.sin(2 * t))
    shift_x, shift_y = displacement(frame, np.array([px]), np.array([py]))
    dx, dy = xs - px - shift_x[0], ys - py - shift_y[0]
    bend = angle * smoothstep(0, part["length"], np.hypot(dx, dy))
    cos, sin = np.cos(-bend), np.sin(-bend)
    fold = 1 - part.get("flap", 0.0) * (1 - math.cos(2 * t)) / 2
    return sample(layer, px + (cos * dx - sin * dy) / fold, py + sin * dx + cos * dy)


def glint(frame, xs, ys):
    layer = np.zeros((SUPER, SUPER, 4))
    for cx, cy, rx, ry, start, length in GLINTS:
        step = (frame - start) % FRAMES
        if step >= length:
            continue
        progress = (step + 0.5) / length
        strength = math.sin(math.pi * progress)
        gem = 1 - smoothstep(0.75, 1.0, np.hypot((xs - cx) / rx, (ys - cy) / ry))
        along = (xs - cx) / rx + (ys - cy) / ry
        centre = -1.6 + 3.2 * progress
        streak = np.exp(-((along - centre) / 0.28) ** 2)
        spot = np.exp(-(((xs - cx + 0.25 * rx) / (0.22 * rx)) ** 2 + ((ys - cy + 0.25 * ry) / (0.22 * ry)) ** 2))
        alpha = np.clip(GLINT_PEAK * strength * gem * np.maximum(streak, spot * strength), 0, 1)
        colour = np.array(GLINT_COLOUR) / 255
        layer = np.dstack([colour * alpha[..., None] + layer[..., :3] * (1 - alpha[..., None]), alpha + layer[..., 3] * (1 - alpha)])
    return layer


def over(top, bottom):
    return top + bottom * (1 - top[..., 3:])


def displacement(frame, xs, ys):
    t = phase(frame)
    lift = breath(frame)
    head = ellipse_weight(xs, ys, HEAD, HEAD_FALLOFF) * (1 - smoothstep(NECK_Y - NECK_BAND, NECK_Y + NECK_BAND, ys))
    profile = np.where(ys < NECK_Y, HEAD_RIDE, HEAD_RIDE + (1 - HEAD_RIDE) * smoothstep(NECK_Y, SHOULDER_Y, ys))
    profile = profile * (1 - smoothstep(SHOULDER_Y, SUPER, ys))
    breath_x = BREATH_WIDTH * lift * (xs - HEAD[0]) * profile
    breath_y = -BREATH * lift * profile
    angle = math.radians(TILT) * math.sin(t)
    cos, sin = math.cos(angle), math.sin(angle)
    px, py = HEAD[0], NECK_Y
    nod = NOD * math.sin(2 * t)
    head_x = cos * (xs - px) - sin * (ys - py) + px - xs
    head_y = sin * (xs - px) + cos * (ys - py) + py - ys + nod
    return breath_x + head * head_x, breath_y + head * head_y


def ripple_fields(ys):
    return [polygon_mask(ripple["polygon"], ripple["feather"], 0) * smoothstep(ripple["top"], ripple["top"] + ripple["ramp"], ys) for ripple in RIPPLES]


def ripple_flow(frame, ys, fields):
    t = phase(frame)
    flow = np.zeros_like(ys)
    for ripple, field in zip(RIPPLES, fields):
        wave = 2 * math.pi * ys / ripple["wavelength"]
        flow = flow + ripple["amplitude"] * field * (np.sin(wave - ripple["cycles"] * t) - np.sin(wave))
    return flow


def warp(premultiplied, frame, xs, ys, flow):
    source_x, source_y = xs - flow, ys
    for _ in range(3):
        dx, dy = displacement(frame, source_x, source_y)
        source_x, source_y = xs - flow - dx, ys - dy
    return sample(premultiplied, source_x, source_y)


def circle(xs, ys):
    radius = np.hypot(xs - SUPER / 2, ys - SUPER / 2)
    return np.clip((SUPER / 2 - radius) / CIRCLE_FEATHER + 0.5, 0, 1)


def downsample(premultiplied):
    factor = SUPER // CANVAS
    return premultiplied.reshape(CANVAS, factor, CANVAS, factor, 4).mean(axis=(1, 3))


def to_image(premultiplied):
    alpha = premultiplied[..., 3]
    rgb = np.where(alpha[..., None] > 0, premultiplied[..., :3] / np.maximum(alpha, 1e-9)[..., None], 0)
    rgba = np.dstack([np.clip(rgb, 0, 1), alpha]) * 255
    return Image.fromarray(rgba.round().astype(np.uint8), mode="RGBA")


def render(frame, bases, layers, xs, ys, ring, fields, shine):
    weight = BLINK.get(frame, 0.0)
    flow = ripple_flow(frame, ys, fields)
    image = warp(bases[0] + weight * (bases[1] - bases[0]), frame, xs, ys, flow)
    if GLINTS:
        image = over(warp(glint(frame, xs, ys) * shine[..., None], frame, xs, ys, flow), image)
    for layer, part in layers:
        image = over(swayed(layer, part, frame, xs, ys), image)
    return to_image(downsample(image * ring[..., None]))


def save_layer(premultiplied, path):
    to_image(premultiplied).save(path)


def clear_frames(folder):
    os.makedirs(folder, exist_ok=True)
    for name in os.listdir(folder):
        if FRAME_NAME.match(name):
            os.remove(os.path.join(folder, name))


def main():
    parser = argparse.ArgumentParser(prog=f"{SLUG}_loop.py")
    parser.add_argument("concept")
    parser.add_argument("work")
    parser.add_argument("export")
    arguments = parser.parse_args()
    canonical = np.asarray(Image.open(os.path.join(arguments.concept, CANONICAL)).convert("RGB")).astype(np.float64) / 255
    edit = np.asarray(Image.open(os.path.join(arguments.concept, BLINK_EDIT)).convert("RGB").resize(canonical.shape[1::-1], Image.LANCZOS)).astype(np.float64) / 255
    aligned, params, before, after = register(canonical, edit)
    portrait = load_rgba(os.path.join(arguments.work, f"{SLUG}_portrait.png"))
    xs, ys = grid(SUPER)
    closed, eyes, offset = closed_portrait(portrait, to_portrait(aligned), xs, ys)
    open_premultiplied, closed_premultiplied = premultiply(portrait), premultiply(closed)
    layers, base_open, base_closed, hole = split_parts(open_premultiplied, closed_premultiplied)
    head = ellipse_weight(xs, ys, HEAD, HEAD_FALLOFF) * (1 - smoothstep(NECK_Y - NECK_BAND, NECK_Y + NECK_BAND, ys))
    save_layer(open_premultiplied * head[..., None], os.path.join(arguments.work, f"{SLUG}_head.png"))
    save_layer(open_premultiplied * (1 - head[..., None]), os.path.join(arguments.work, f"{SLUG}_torso.png"))
    save_layer(open_premultiplied * eyes[..., None], os.path.join(arguments.work, f"{SLUG}_eyes_open.png"))
    save_layer(closed_premultiplied * eyes[..., None], os.path.join(arguments.work, f"{SLUG}_eyes_closed.png"))
    save_layer(base_open, os.path.join(arguments.work, f"{SLUG}_base.png"))
    for layer, part in layers:
        save_layer(layer, os.path.join(arguments.work, f"{SLUG}_{part['name']}.png"))
    ring = circle(xs, ys)
    fields = ripple_fields(ys)
    shine = smoothstep(GLINT_FLOOR, GLINT_FLOOR + 0.2, portrait[..., :3].mean(axis=2))
    anim = os.path.join(arguments.export, "anim")
    clear_frames(anim)
    for index in range(1, FRAMES + 1):
        frame = render(index - 1, (base_open, base_closed), layers, xs, ys, ring, fields, shine)
        frame.save(os.path.join(anim, f"{index:04d}.png"))
        if index == STATIC_FRAME:
            frame.save(os.path.join(arguments.export, "static.png"))
    shift = ", ".join(f"{value:.4f}" for value in params)
    print(f"{SLUG}_loop: blink edit registered (tx, ty, scale, angle) = ({shift}), residual {before:.5f} -> {after:.5f}, eye colour offset {np.round(offset * 255, 1).tolist()}, parts {len(layers)}, ripples {len(RIPPLES)}, filled {int(hole.sum())} px")
    print(f"{SLUG}_loop: wrote {FRAMES} frames and the static (frame {STATIC_FRAME}) at {CANVAS}x{CANVAS}")


main()
