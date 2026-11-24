import argparse
import json
import os

import numpy as np
from PIL import Image
from scipy import ndimage

CANVAS = (1920, 1080)
SUPERSAMPLE = 2
MARGIN_TOLERANCE = 30
MARGIN_SHARE = 0.98
STRETCH_LIMIT = 1.1
RIM_WIDTH = 40
RIM_RADIUS = 28
INLAY_INSET = 54
INLAY_WIDTH = 4
INLAY_CORNER = 11
REFERENCE = CANVAS
EDGE_FADE = 16
EDGE_LIGHT = 120
EDGE_STEPS = range(150, 486, 6)
EDGE_SHORTLIST = 40
CORNER_MARGIN = 30
SQUARE_REACH = 520
WOOD = {"hi": "#5A2414", "default": "#3A150B", "lo": "#240C06"}
GOLD = {"hi": "#FFF4C2", "default": "#E6C068", "mid": "#C99A3C", "lo": "#7A5418"}
CALM_RADIUS = (0.44, 0.40)
CALM_EDGE = 0.25
CALM_SOFTEN = 0.3
CALM_GAIN = 0.75
CALM_SIGMA = 12
WIDEN_REACH = (0.15, 0.45)
WIDEN_SPREAD = 0.2


def rgb(hex_colour):
    return np.array([int(hex_colour[i:i + 2], 16) for i in (1, 3, 5)], dtype=np.float64)


def edge_light(band):
    low = ndimage.gaussian_filter1d(band, EDGE_LIGHT, axis=1, mode="reflect") + 1
    return band / low, (low + low[:, ::-1]) / 2


def seam_cost(detail, step, start, reach):
    return np.abs(detail[:, start + step - reach:start + step + reach] - detail[:, start - reach:start + reach]).mean()


def mirror_cost(detail, start, reach=24):
    return np.abs(detail[:, start - reach:start][:, ::-1] - detail[:, start:start + reach]).mean()


def repeat_band(detail, step, start):
    width = detail.shape[1]
    end = width - start
    xs = np.arange(width)
    inside = (xs >= start) & (xs < end)
    result = detail[:, np.where(inside, start + (xs - start) % step, xs)]
    offsets = np.arange(-EDGE_FADE, EDGE_FADE)
    weight = ((offsets + EDGE_FADE + 0.5) / (2 * EDGE_FADE)).reshape((1, -1) + (1,) * (detail.ndim - 2))
    carried = detail[:, start + step + offsets]
    for seam in [*range(start + step, end, step), end]:
        arriving = detail[:, (start if seam < end else end) + offsets]
        result[:, seam + offsets] = carried * (1 - weight) + arriving * weight
    return result


def best_edges(details, depth, width):
    candidates = []
    for step in EDGE_STEPS:
        for count in range(2, width // step + 1):
            start, odd = divmod(width - count * step, 2)
            if odd or start < depth + CORNER_MARGIN or start > SQUARE_REACH - step // 4:
                continue
            cost = sum(seam_cost(detail, step, start, 12) + seam_cost(detail, step, start, 48) / 2 + mirror_cost(detail, start) / 2 for detail in details)
            candidates.append((cost, step, start))
    scored = []
    for cost, step, start in sorted(candidates)[:EDGE_SHORTLIST]:
        drift = sum(np.abs(repeat_band(detail, step, start) - detail)[:, start:width - start].mean() for detail in details)
        scored.append((cost + drift / 2, step, start))
    return min(scored)


def periodic_edges(pixels, depth):
    height, width = pixels.shape[:2]
    bands = (slice(0, depth), slice(height - depth, height))
    parts = [edge_light(pixels[rows]) for rows in bands]
    score, step, start = best_edges([detail.mean(axis=2) for detail, _ in parts], depth, width)
    inward = np.clip((depth - np.arange(depth) - 0.5) / EDGE_FADE, 0, 1)
    result = pixels.copy()
    for rows, (detail, light), weight in zip(bands, parts, (inward, inward[::-1])):
        rebuilt = repeat_band(detail, step, start) * light
        result[rows] = rebuilt * weight[:, None, None] + pixels[rows] * (1 - weight[:, None, None])
    return result, step, start, score


def write_slice(export, depth, step, size):
    across, down = size[0] / REFERENCE[0], size[1] / REFERENCE[1]
    values = {"depth": depth * down, "step": step * across}
    if any(value != int(value) for value in values.values()):
        raise ValueError(f"slice {values} is not whole pixels on the {size[0]}x{size[1]} canvas")
    os.makedirs(export, exist_ok=True)
    with open(os.path.join(export, "slice.json"), "w", encoding="utf-8", newline="\n") as target:
        json.dump({key: int(value) for key, value in values.items()}, target)
        target.write("\n")


def margin_trim(pixels):
    ring = np.concatenate([pixels[:2].reshape(-1, 3), pixels[-2:].reshape(-1, 3), pixels[:, :2].reshape(-1, 3), pixels[:, -2:].reshape(-1, 3)])
    near = np.linalg.norm(pixels - np.median(ring, axis=0), axis=2) < MARGIN_TOLERANCE
    rows = np.where(near.mean(axis=1) < MARGIN_SHARE)[0]
    columns = np.where(near.mean(axis=0) < MARGIN_SHARE)[0]
    return columns[0], rows[0], columns[-1] + 1, rows[-1] + 1


def fit_to_canvas(image):
    width, height = image.size
    target = CANVAS[0] / CANVAS[1]
    stretch = min(max(target / (width / height), 1 / STRETCH_LIMIT), STRETCH_LIMIT)
    stretched_width = width * stretch
    if stretched_width / height > target:
        crop_width = height * target / stretch
        left = (width - crop_width) / 2
        box = (left, 0, left + crop_width, height)
    else:
        crop_height = stretched_width / target
        upper = (height - crop_height) / 2
        box = (0, upper, width, upper + crop_height)
    return image.resize(CANVAS, Image.LANCZOS, box=box), stretch, tuple(round(value, 1) for value in box)


def strip_cost(grey, width, step, reach):
    difference = np.abs(grey[:, :-step] - grey[:, step:]).mean(axis=0)
    running = np.concatenate([[0], np.cumsum(difference)])
    seams = np.arange(reach[0], reach[1])
    left = running[seams] - running[seams - step]
    right = running[width - seams + step] - running[width - seams]
    return (left + right) / step, seams


def widen_field(image, depth):
    pixels = np.asarray(image)
    height, width = pixels.shape[:2]
    extra = round(height * CANVAS[0] / CANVAS[1]) - width
    if extra <= 0:
        return image, 0, 0
    band = int(np.ceil(depth * height / CANVAS[1]))
    grey = pixels[band:height - band].astype(np.float64).mean(axis=2)
    reach = (int(width * WIDEN_REACH[0]), int(width * WIDEN_REACH[1]))
    best = None
    for step in range(int(extra * (0.5 - WIDEN_SPREAD)), int(extra * (0.5 + WIDEN_SPREAD)) + 1):
        costs, seams = strip_cost(grey, width, step, (max(reach[0], 2 * step), reach[1]))
        index = int(np.argmin(costs))
        if best is None or costs[index] < best[0]:
            best = (costs[index], step, int(seams[index]))
    _, step, seam = best
    field = np.concatenate([pixels[:, :seam], pixels[:, seam - step:width - seam + step], pixels[:, width - seam:]], axis=1)
    middle = width // 2
    edges = np.concatenate([pixels[:, :middle + step], pixels[:, middle - step:]], axis=1)
    rows = np.zeros(height, dtype=bool)
    rows[:band] = rows[height - band:] = True
    return Image.fromarray(np.where(rows[:, None, None], edges, field)), step, seam


def calm_centre(image):
    pixels = np.asarray(image).astype(np.float64)
    height, width = pixels.shape[:2]
    ys, xs = np.mgrid[0:height, 0:width] + 0.5
    dx = (xs - width / 2) / (width * CALM_RADIUS[0])
    dy = (ys - height / 2) / (height * CALM_RADIUS[1])
    weight = (np.clip(1 - np.hypot(dx, dy), 0, CALM_EDGE) / CALM_EDGE)[..., None]
    base = ndimage.gaussian_filter(pixels, (CALM_SIGMA, CALM_SIGMA, 0))
    calmed = base + (pixels - base) * (1 - CALM_SOFTEN * weight)
    toned = calmed * (1 - (1 - CALM_GAIN) * weight)
    return Image.fromarray(toned.round().clip(0, 255).astype(np.uint8), mode="RGB")


def rounded_rectangle(size, inset, radius):
    width, height = size
    ys, xs = np.mgrid[0:height, 0:width] + 0.5
    inner_x = np.clip(xs, inset + radius, width - inset - radius)
    inner_y = np.clip(ys, inset + radius, height - inset - radius)
    return np.hypot(xs - inner_x, ys - inner_y) <= radius


def light_from_above(mask, width):
    depth = np.minimum(ndimage.distance_transform_edt(mask), width / 2) / (width / 2)
    height = np.sin(depth * np.pi / 2)
    slope = np.gradient(height, axis=0)
    return np.clip(slope * width / 2, -1, 1)


def shade(light, stops):
    keys = sorted(stops)
    positions = np.interp(light, keys, range(len(keys)))
    lower = np.floor(positions).astype(int).clip(0, len(keys) - 2)
    weight = (positions - lower)[..., None]
    colours = np.array([stops[key] for key in keys])
    return colours[lower] * (1 - weight) + colours[lower + 1] * weight


def front_layer(size):
    scale = SUPERSAMPLE
    big = (size[0] * scale, size[1] * scale)
    opening = rounded_rectangle(big, RIM_WIDTH * scale, RIM_RADIUS * scale)
    rim = ~opening
    light = light_from_above(rim, RIM_WIDTH * scale)
    colour = shade(light, {-1: rgb(WOOD["lo"]), 0: rgb(WOOD["default"]), 1: rgb(WOOD["hi"])})
    return downsample(colour, rim, size)


def inlay_mask(size, inset, width, corner):
    outer = rounded_rectangle(size, inset, 0.5)
    inner = rounded_rectangle(size, inset + width, 0.5)
    line = outer & ~inner
    ys, xs = np.mgrid[0:size[1], 0:size[0]] + 0.5
    centre = inset + width / 2
    for corner_x in (centre, size[0] - centre):
        for corner_y in (centre, size[1] - centre):
            line |= np.abs(xs - corner_x) + np.abs(ys - corner_y) <= corner
    return line


def middle_layer(size):
    scale = SUPERSAMPLE
    big = (size[0] * scale, size[1] * scale)
    line = inlay_mask(big, INLAY_INSET * scale, INLAY_WIDTH * scale, INLAY_CORNER * scale)
    light = light_from_above(line, INLAY_WIDTH * scale)
    colour = shade(light, {-1: rgb(GOLD["lo"]), -0.4: rgb(GOLD["mid"]), 0.2: rgb(GOLD["default"]), 1: rgb(GOLD["hi"])})
    return downsample(colour, line, size)


def downsample(colour, mask, size):
    alpha = mask.astype(np.float64) * 255
    premultiplied = np.dstack([colour * mask[..., None], alpha]).astype(np.float32)
    small = np.stack([np.asarray(Image.fromarray(premultiplied[..., channel], mode="F").resize(size, Image.BOX)) for channel in range(4)], axis=-1)
    opacity = np.maximum(small[..., 3:], 1e-6) / 255
    straight = np.where(small[..., 3:] > 0, small[..., :3] / opacity, 0)
    return Image.fromarray(np.dstack([straight, small[..., 3:]]).round().clip(0, 255).astype(np.uint8), mode="RGBA")


def main():
    parser = argparse.ArgumentParser(prog="table_cleanup.py")
    parser.add_argument("slug")
    parser.add_argument("depth", type=int)
    parser.add_argument("back")
    parser.add_argument("work")
    parser.add_argument("export")
    arguments = parser.parse_args()
    concept = Image.open(arguments.back).convert("RGB")
    trim = margin_trim(np.asarray(concept).astype(np.float64))
    widened, strip, seam = widen_field(concept.crop(trim), arguments.depth)
    back, stretch, box = fit_to_canvas(widened)
    back = calm_centre(back)
    pixels, step, start, score = periodic_edges(np.asarray(back).astype(np.float64), arguments.depth)
    back = Image.fromarray(pixels.round().clip(0, 255).astype(np.uint8), mode="RGB").convert("RGBA")
    middle = middle_layer(CANVAS)
    front = front_layer(CANVAS)
    for name, layer in (("back", back), ("middle", middle), ("front", front)):
        layer.save(os.path.join(arguments.work, f"{arguments.slug}_{name}.png"))
    flattened = Image.alpha_composite(Image.alpha_composite(back, middle), front)
    flattened.save(os.path.join(arguments.export, "static.png"))
    write_slice(arguments.export, arguments.depth, step, CANVAS)
    print(f"{arguments.slug}: trim {tuple(int(value) for value in trim)}, widen strip {strip} seam {seam}, stretch {stretch:.3f}, crop {box}, edges depth {arguments.depth} step {step} start {start} score {score:.4f}, canvas {CANVAS[0]}x{CANVAS[1]}")


main()
