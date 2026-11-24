import argparse
import json
import os

import numpy as np
from PIL import Image, ImageDraw
from scipy import ndimage

REFERENCE = (1920, 1080)
KNOT = 4
SUPERSAMPLE = 2
DRAW_SCALE = 4
RIM_WIDTH = 40
RIM_RADIUS = 28
INLAY_INSET = 54
INLAY_WIDTH = 4
INLAY_CORNER = 11
VIGNETTE = 0.3
JITTER = 0.035
ABRASH = 0.03
KNOT_EDGE = 0.9
MARGIN_TOLERANCE = 30
MARGIN_SHARE = 0.98
STRETCH_LIMIT = 1.1
EDGE_FADE = 16
EDGE_LIGHT = 120
EDGE_STEPS = range(150, 486, 6)
EDGE_SHORTLIST = 40
CORNER_MARGIN = 30
SQUARE_REACH = 520


def token(tokens, path):
    value = tokens["color"]
    for key in path.split("."):
        value = value[key]
    return np.array([int(value[i:i + 2], 16) for i in (1, 3, 5)], dtype=np.float64)


def knot_grid():
    columns, rows = REFERENCE[0] // KNOT, REFERENCE[1] // KNOT
    v, u = np.mgrid[0:rows, 0:columns]
    return u.astype(np.float64), v.astype(np.float64), columns, rows


def flower4(dx, dy, size):
    centre = np.abs(dx) + np.abs(dy) <= size * 0.34
    petals = np.zeros_like(centre)
    for ox, oy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
        petals |= np.abs(dx - ox * size * 0.62) + np.abs(dy - oy * size * 0.62) <= size * 0.3
    return {"petals": petals & ~centre, "centre": centre}


def flower8(dx, dy, size):
    centre = np.hypot(dx, dy) <= size * 0.3
    petals = np.zeros_like(centre)
    for step in range(8):
        angle = step * np.pi / 4
        ox, oy = np.cos(angle) * size * 0.66, np.sin(angle) * size * 0.66
        petals |= np.hypot(dx - ox, dy - oy) <= size * 0.27
    return {"petals": petals & ~centre, "centre": centre}


def diamond(dx, dy, size):
    reach = np.abs(dx) + np.abs(dy)
    return {"outline": (reach <= size) & (reach > size - 2), "core": reach <= size * 0.42}


def rosette(dx, dy, size):
    reach = np.maximum(np.maximum(np.abs(dx), np.abs(dy)), (np.abs(dx) + np.abs(dy)) / 1.35)
    return {"ring": (reach <= size) & (reach > size - 1.6), "core": reach <= size * 0.38}


def leaf(dx, dy, size):
    body = (np.abs(dx) / size) ** 2 + (np.abs(dy) / (size * 0.42)) ** 1.3 <= 1
    vein = body & (np.abs(dy) < 0.6)
    return {"body": body & ~vein, "vein": vein}


def chevron(dx, dy, size):
    line = np.abs(np.mod(dx + np.abs(dy) * 1.2, size) - size / 2) < size * 0.16
    return {"line": line}


MOTIFS = {"flower4": flower4, "flower8": flower8, "diamond": diamond, "rosette": rosette, "leaf": leaf, "chevron": chevron}


def paint(knots, palette, mask, name):
    knots[mask] = palette[name]


def paint_motif(knots, palette, spec, dx, dy, where):
    for role, mask in MOTIFS[spec["kind"]](dx, dy, spec["size"]).items():
        paint(knots, palette, where & mask, spec[role])


def medallion_reach(kind, x, y, axes):
    ax, ay = np.abs(x) / axes[0], np.abs(y) / axes[1]
    if kind == "octagon":
        return np.maximum(np.maximum(ax, ay), (ax + ay) / 1.3)
    return ax + ay


def field(u, v, columns, rows, knots, palette, spec):
    cu, cv = u - columns / 2 + 0.5, v - rows / 2 + 0.5
    lattice = spec["lattice"]
    period = lattice["period"]
    lu = np.mod(cu, period) - period / 2
    lv = np.mod(cv, period) - period / 2
    paint(knots, palette, np.abs(np.abs(lu) + np.abs(lv) - period / 2) < lattice["width"], lattice["colour"])
    everywhere = np.ones_like(lu, dtype=bool)
    if "motif" in spec:
        paint_motif(knots, palette, spec["motif"], lu, lv, everywhere)
    if "node" in spec:
        for su, sv in ((period / 2, 0), (0, period / 2)):
            nu = np.mod(cu + su, period) - period / 2
            nv = np.mod(cv + sv, period) - period / 2
            paint_motif(knots, palette, spec["node"], nu, nv, everywhere)
    medallion = spec.get("medallion")
    if medallion:
        axes = medallion["axes"]
        reach = medallion_reach(medallion["kind"], cu, cv, axes)
        inside = reach < 1
        paint(knots, palette, inside, medallion["fill"])
        paint(knots, palette, np.abs(reach - 1) * axes[1] < 1, medallion["outline"])
        paint(knots, palette, np.abs(reach - 1 - medallion["gap"] / axes[1]) * axes[1] < 0.6, medallion["second"])


def spandrels(u, v, columns, rows, knots, palette, spec, inner):
    for corner_u, corner_v in ((inner, inner), (columns - 1 - inner, inner), (inner, rows - 1 - inner), (columns - 1 - inner, rows - 1 - inner)):
        reach = np.abs(u - corner_u) / spec["axes"][0] + np.abs(v - corner_v) / spec["axes"][1]
        paint(knots, palette, reach < 1, spec["fill"])
        paint(knots, palette, np.abs(reach - 1) * spec["axes"][1] < 0.8, spec["outline"])
        if "motif" in spec:
            paint_motif(knots, palette, spec["motif"], (u - corner_u) * np.sign(columns / 2 - corner_u) - spec["axes"][0] * 0.32, (v - corner_v) * np.sign(rows / 2 - corner_v) - spec["axes"][1] * 0.32, reach < 1)


def border_coordinates(u, v, columns, rows, middle, period):
    distances = np.stack([v, rows - 1 - v, u, columns - 1 - u])
    nearest = distances.argmin(axis=0)
    across = distances.min(axis=0)
    run = np.where(nearest < 2, columns - 1 - 2 * middle, rows - 1 - 2 * middle)
    position = np.where(nearest < 2, u, v) - middle
    count = np.maximum(2, np.round(run / period / 2) * 2)
    step = run / count
    index = np.round(position / step)
    return across, position - index * step, index


def border(u, v, columns, rows, knots, palette, spec):
    bands = spec["bands"]
    main = bands[spec["main"]]
    middle = (main[1] + main[2] - 1) / 2
    across, dx, index = border_coordinates(u, v, columns, rows, middle, spec["period"])
    for name, start, end in bands:
        paint(knots, palette, (across >= start) & (across < end), name)
    dy = across - middle
    inside = (across >= main[1]) & (across < main[2])
    motifs = spec["motifs"]
    for number, motif in enumerate(motifs):
        paint_motif(knots, palette, motif, dx, dy, inside & (np.mod(index, len(motifs)) == number))
    for guard in spec["guards"]:
        band = bands[guard["band"]]
        centre = (band[1] + band[2] - 1) / 2
        within = (across >= band[1]) & (across < band[2])
        if guard["kind"] == "dots":
            mark = np.abs(np.mod(dx, guard["step"]) - guard["step"] / 2) + np.abs(across - centre) <= 1
        else:
            mark = np.abs(np.mod(dx + np.abs(across - centre) * 2, guard["step"]) - guard["step"] / 2) < 0.8
        paint(knots, palette, within & mark, guard["colour"])


def wool(knots, rows, rng):
    jitter = 1 + rng.uniform(-JITTER, JITTER, knots.shape[:2])
    abrash = 1 + ABRASH * np.sin(np.arange(rows) / rows * np.pi * 5 + rng.uniform(0, np.pi))
    shaded = knots * (jitter * abrash[:, None])[..., None]
    pixels = np.repeat(np.repeat(shaded, KNOT, axis=0), KNOT, axis=1)
    edge = np.ones(KNOT)
    edge[0] = edge[-1] = KNOT_EDGE
    pixels *= np.outer(np.tile(edge, rows), np.tile(edge, knots.shape[1]))[..., None]
    pixels = ndimage.gaussian_filter(pixels, (0.6, 0.6, 0))
    fibre = 1 + rng.normal(0, 0.02, pixels.shape[:2])
    return pixels * fibre[..., None]


def lamp(size):
    ys, xs = np.mgrid[0:size[1], 0:size[0]] + 0.5
    dx = (xs - size[0] / 2) / (size[0] / 2)
    dy = (ys - size[1] * 0.46) / (size[1] / 2)
    falloff = np.clip(np.hypot(dx * 0.8, dy) / np.hypot(0.8, 1.08), 0, 1)
    return 1 - VIGNETTE * falloff ** 1.6


def carpet(spec, seed):
    rng = np.random.default_rng(seed)
    palette = {name: np.array(value, dtype=np.float64) for name, value in spec["palette"].items()}
    u, v, columns, rows = knot_grid()
    knots = np.empty((rows, columns, 3))
    knots[:] = palette[spec["field"]["base"]]
    field(u, v, columns, rows, knots, palette, spec["field"])
    if "spandrels" in spec:
        inner = max(end for _, _, end in spec["border"]["bands"])
        spandrels(u, v, columns, rows, knots, palette, spec["spandrels"], inner)
    border(u, v, columns, rows, knots, palette, spec["border"])
    return wool(knots, rows, rng) * lamp(REFERENCE)[..., None]


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
    return int(columns[0]), int(rows[0]), int(columns[-1] + 1), int(rows[-1] + 1)


def fit_to_canvas(image, size):
    width, height = image.size
    target = size[0] / size[1]
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
    return image.resize(size, Image.LANCZOS, box=box), stretch, tuple(round(value, 1) for value in box)


def gold_threads(pixels, spec, size):
    rgb = pixels / 255
    value = rgb.max(axis=2)
    low = rgb.min(axis=2)
    saturation = np.where(value > 0, (value - low) / np.maximum(value, 1e-6), 0)
    red, green, blue = rgb[..., 0], rgb[..., 1], rgb[..., 2]
    hue = np.where(value == red, (green - blue) / np.maximum(value - low, 1e-6) / 6, np.where(value == green, (2 + (blue - red) / np.maximum(value - low, 1e-6)) / 6, (4 + (red - green) / np.maximum(value - low, 1e-6)) / 6))
    hue = np.mod(hue, 1)
    mask = (hue >= spec["hue"][0]) & (hue <= spec["hue"][1]) & (saturation >= spec["saturation"]) & (value >= spec["value"])
    return np.asarray(Image.fromarray((mask * 255).astype(np.uint8)).resize(size, Image.BOX)).astype(np.float64) / 255


def tone(pixels, spec):
    if not spec:
        return pixels
    height, width = pixels.shape[:2]
    ys, xs = np.mgrid[0:height, 0:width] + 0.5
    dx = (xs - width / 2) / (width * spec["radius"][0])
    dy = (ys - height / 2) / (height * spec["radius"][1])
    weight = (np.clip(1 - np.hypot(dx, dy), 0, spec["edge"]) / spec["edge"])[..., None]
    base = ndimage.gaussian_filter(pixels, (spec["sigma"], spec["sigma"], 0))
    calmed = base + (pixels - base) * (1 - spec["soften"] * weight)
    return calmed * (1 - (1 - spec["gain"]) * weight)


def concept_back(item, spec):
    concept = Image.open(os.path.join(item, spec["concept"])).convert("RGB")
    trim = margin_trim(np.asarray(concept).astype(np.float64))
    back, stretch, box = fit_to_canvas(concept.crop(trim), REFERENCE)
    pixels = tone(np.asarray(back).astype(np.float64), spec.get("tone"))
    return pixels, f"trim {trim}, stretch {stretch:.3f}, crop {box}"


def to_image(pixels):
    opaque = np.full(pixels.shape[:2], 255.0)
    return Image.fromarray(np.dstack([pixels, opaque]).round().clip(0, 255).astype(np.uint8), mode="RGBA")


def resample(image, size):
    if image.size == size:
        return image
    return image.resize(size, Image.BOX)


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


def downsample(colours, alpha, size):
    premultiplied = np.dstack([colours * alpha[..., None], alpha * 255]).astype(np.float32)
    small = np.stack([np.asarray(Image.fromarray(premultiplied[..., channel], mode="F").resize(size, Image.BOX)) for channel in range(4)], axis=-1)
    opacity = np.maximum(small[..., 3:], 1e-6) / 255
    straight = np.where(small[..., 3:] > 0, small[..., :3] / opacity, 0)
    return Image.fromarray(np.dstack([straight, small[..., 3:]]).round().clip(0, 255).astype(np.uint8), mode="RGBA")


def big_canvas(size):
    scale = SUPERSAMPLE * size[1] / REFERENCE[1]
    return (size[0] * SUPERSAMPLE, size[1] * SUPERSAMPLE), scale


def rim_layer(tokens, size):
    big, scale = big_canvas(size)
    rim = ~rounded_rectangle(big, RIM_WIDTH * scale, RIM_RADIUS * scale)
    light = light_from_above(rim, RIM_WIDTH * scale)
    stops = {-1: token(tokens, "wood.lo"), 0: token(tokens, "wood.default"), 1: token(tokens, "wood.hi")}
    return downsample(shade(light, stops), rim.astype(np.float64), size)


def inlay_mask(size, inset, width, corner):
    line = rounded_rectangle(size, inset, 0.5) & ~rounded_rectangle(size, inset + width, 0.5)
    ys, xs = np.mgrid[0:size[1], 0:size[0]] + 0.5
    centre = inset + width / 2
    for corner_x in (centre, size[0] - centre):
        for corner_y in (centre, size[1] - centre):
            line |= np.abs(xs - corner_x) + np.abs(ys - corner_y) <= corner
    return line


def gold_stops(tokens):
    return {-1: token(tokens, "gold.lo"), -0.4: token(tokens, "gold.mid"), 0.2: token(tokens, "gold.default"), 1: token(tokens, "gold.hi")}


def inlay_layer(tokens, size):
    big, scale = big_canvas(size)
    line = inlay_mask(big, INLAY_INSET * scale, INLAY_WIDTH * scale, INLAY_CORNER * scale)
    light = light_from_above(line, INLAY_WIDTH * scale)
    return downsample(shade(light, gold_stops(tokens)), line.astype(np.float64), size)


def motes(spec, size, frame, frames, rng_seed):
    rng = np.random.default_rng(rng_seed)
    unit = size[1] / REFERENCE[1]
    count = spec["count"]
    margin = (RIM_WIDTH + 6) * unit
    xs = rng.uniform(margin, size[0] - margin, count)
    ys = rng.uniform(margin, size[1] - margin, count)
    radius = rng.uniform(*spec["radius"], count) * unit
    orbit = rng.uniform(*spec["orbit"], (count, 2)) * unit
    turns = rng.choice([-1, 1], count)
    phase = rng.uniform(0, 1, count)
    twinkle = rng.uniform(0, 1, count)
    strength = rng.uniform(*spec["alpha"], count)
    t = frame / frames
    angle = 2 * np.pi * (turns * t + phase)
    px = xs + orbit[:, 0] * np.cos(angle)
    py = ys + orbit[:, 1] * np.sin(angle)
    light = lamp(size)
    alpha = np.zeros((size[1], size[0]))
    for index in range(count):
        reach = int(np.ceil(radius[index] * 3)) + 1
        cx, cy = px[index], py[index]
        left, top = max(int(cx) - reach, 0), max(int(cy) - reach, 0)
        right, bottom = min(int(cx) + reach + 1, size[0]), min(int(cy) + reach + 1, size[1])
        gy, gx = np.mgrid[top:bottom, left:right] + 0.5
        dot = np.exp(-((gx - cx) ** 2 + (gy - cy) ** 2) / (2 * (radius[index] * 0.45) ** 2))
        level = strength[index] * (0.55 + 0.45 * np.cos(2 * np.pi * (t + twinkle[index]))) * light[int(min(cy, size[1] - 1)), int(min(cx, size[0] - 1))]
        alpha[top:bottom, left:right] = 1 - (1 - alpha[top:bottom, left:right]) * (1 - dot * level)
    colour = np.empty((size[1], size[0], 3))
    colour[:] = spec["colour"]
    return Image.fromarray(np.dstack([colour, alpha * 255]).round().clip(0, 255).astype(np.uint8), mode="RGBA")


def shimmer(spec, size, frame, frames, threads, seed, tokens):
    rng = np.random.default_rng(seed)
    t = frame / frames
    height, width = threads.shape
    ys, xs = np.mgrid[0:height, 0:width] + 0.5
    angle = np.arctan2(ys - height / 2, np.abs(xs - width / 2) * height / width)
    wave = np.clip(np.cos(spec["bands"] * angle - 2 * np.pi * t), 0, 1) ** spec["sharpness"]
    cell = max(1, int(round(KNOT * height / REFERENCE[1] * spec["cell"])))
    phases = rng.uniform(0, 1, (height // cell + 1, width // cell + 1))
    phase = np.repeat(np.repeat(phases, cell, axis=0), cell, axis=1)[:height, :width]
    twinkle = (0.5 + 0.5 * np.cos(2 * np.pi * (t + phase))) ** 4
    level = threads * np.clip(spec["wave"] * wave + spec["twinkle"] * twinkle, 0, 1)
    keep = rounded_rectangle((width, height), (RIM_WIDTH + 2) * height / REFERENCE[1], RIM_RADIUS * height / REFERENCE[1])
    level *= keep * lamp((width, height))
    colour = np.empty((height, width, 3))
    colour[:] = token(tokens, "gold.hi")
    return Image.fromarray(np.dstack([colour, level * spec["alpha"] * 255]).round().clip(0, 255).astype(np.uint8), mode="RGBA")


def anchors(places, inset):
    width, height = REFERENCE
    across, down = inset
    corners = [((across, down), 45), ((width - across, down), 135), ((width - across, height - down), 225), ((across, height - down), 315)]
    ends = [((across, height / 2), 0), ((width - across, height / 2), 180)]
    return corners + (ends if places == "ends" else [])


def strand_points(origin, direction, across, length, bend, steps=12):
    ux, uy = np.cos(direction), np.sin(direction)
    nx, ny = -uy, ux
    points = []
    for step in range(steps + 1):
        s = step / steps
        offset = across * (0.35 + 0.65 * s) + bend * s * s
        points.append((origin[0] + ux * length * s + nx * offset, origin[1] + uy * length * s + ny * offset))
    return points


def graded_line(draw, points, colour, width):
    last = len(points) - 1
    for step in range(last):
        factor = 1.06 - 0.32 * step / last
        shade = tuple(int(min(255, channel * factor)) for channel in colour)
        draw.line(points[step:step + 2], fill=shade, width=width)
        reach = width / 2
        draw.ellipse([points[step + 1][0] - reach, points[step + 1][1] - reach, points[step + 1][0] + reach, points[step + 1][1] + reach], fill=shade)


def silk_colours(spec):
    return [tuple(int(value) for value in colour) for colour in spec["silk"]]


def tassels(spec, size, frame, frames, seed, tokens):
    rng = np.random.default_rng(seed)
    t = frame / frames
    scale = DRAW_SCALE * size[1] / REFERENCE[1]
    big = (size[0] * DRAW_SCALE, size[1] * DRAW_SCALE)
    canvas = Image.new("RGBA", big, (0, 0, 0, 0))
    draw = ImageDraw.Draw(canvas)
    silk = silk_colours(spec)
    cap = tuple(int(value) for value in token(tokens, "gold.default"))
    cap_dark = tuple(int(value) for value in token(tokens, "gold.mid"))
    cap_light = tuple(int(value) for value in token(tokens, "gold.hi"))
    places = anchors(spec["places"], spec["inset"])
    phases = rng.uniform(0, 1, len(places))
    for (anchor, degrees), phase in zip(places, phases):
        origin = (anchor[0] * scale, anchor[1] * scale)
        direction = np.radians(degrees + spec["sway"] * np.sin(2 * np.pi * (t + phase)))
        bend = spec["bend"] * scale * np.sin(2 * np.pi * (t + phase) - 0.9)
        length = spec["length"] * scale
        lead = spec["cap"] * scale
        start = (origin[0] + np.cos(direction) * lead, origin[1] + np.sin(direction) * lead)
        strands = spec["strands"]
        for index in range(strands):
            across = (index / (strands - 1) - 0.5) * spec["spread"] * scale
            points = strand_points(start, direction, across, length, bend)
            graded_line(draw, points, silk[index % len(silk)], max(1, int(round(spec["thread"] * scale))))
        band = (origin[0] + np.cos(direction) * lead * 1.5, origin[1] + np.sin(direction) * lead * 1.5)
        draw.ellipse([band[0] - lead * 0.9, band[1] - lead * 0.9, band[0] + lead * 0.9, band[1] + lead * 0.9], fill=cap_dark)
        draw.ellipse([origin[0] - lead, origin[1] - lead, origin[0] + lead, origin[1] + lead], fill=cap)
        draw.ellipse([origin[0] - lead * 0.55, origin[1] - lead * 0.75, origin[0] + lead * 0.25, origin[1] - lead * 0.05], fill=cap_light)
    return shaded_layer(canvas, size)


def fringe(spec, size, frame, frames, tokens):
    t = frame / frames
    scale = DRAW_SCALE * size[1] / REFERENCE[1]
    big = (size[0] * DRAW_SCALE, size[1] * DRAW_SCALE)
    canvas = Image.new("RGBA", big, (0, 0, 0, 0))
    draw = ImageDraw.Draw(canvas)
    silk = silk_colours(spec)
    width, height = REFERENCE
    bead = tuple(int(value) for value in token(tokens, "gold.default"))
    positions = np.arange(spec["top"], height - spec["top"] + 0.01, spec["step"])
    for side, direction, sign in ((spec["origin"], 0.0, 1), (width - spec["origin"], np.pi, -1)):
        for index, y in enumerate(positions):
            swing = spec["swing"] * np.sin(2 * np.pi * (t - sign * y / spec["wavelength"]))
            points = strand_points((side * scale, y * scale), direction, 0, spec["length"] * scale, sign * swing * scale)
            graded_line(draw, points, silk[index % len(silk)], max(1, int(round(spec["thread"] * scale))))
        reach = spec["bead"] * scale
        for y in positions[spec["group"] // 2::spec["group"]]:
            cx = (side + sign * spec["bead"]) * scale
            draw.ellipse([cx - reach, y * scale - reach, cx + reach, y * scale + reach], fill=bead)
    return shaded_layer(canvas, size)


def shaded_layer(canvas, size):
    pixels = np.asarray(canvas).astype(np.float64)
    alpha = pixels[..., 3] / 255
    small = downsample(pixels[..., :3], alpha, size)
    shaded = np.asarray(small).astype(np.float64)
    shaded[..., :3] *= lamp(size)[..., None]
    return Image.fromarray(shaded.round().clip(0, 255).astype(np.uint8), mode="RGBA")


def motion_part(spec, size, frame, frames, threads, seed, tokens):
    kind = spec["kind"]
    if kind == "motes":
        return motes(spec, size, frame, frames, seed)
    if kind == "shimmer":
        return shimmer(spec, size, frame, frames, threads, seed, tokens)
    if kind == "tassels":
        return tassels(spec, size, frame, frames, seed, tokens)
    return fringe(spec, size, frame, frames, tokens)


def motion_frame(motion, size, frame, threads, seed, tokens):
    layer = Image.new("RGBA", size, (0, 0, 0, 0))
    for part in motion["parts"]:
        layer = Image.alpha_composite(layer, motion_part(part, size, frame, motion["frames"], threads, seed, tokens))
    return layer


def save(image, path):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    image.save(path, optimize=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("config")
    parser.add_argument("tokens")
    parser.add_argument("item")
    parser.add_argument("out")
    arguments = parser.parse_args()
    with open(arguments.config, encoding="utf-8") as source:
        config = json.load(source)
    with open(arguments.tokens, encoding="utf-8") as source:
        tokens = json.load(source)
    slug = config["slug"]
    size = tuple(config["canvas"])
    seed = config["seed"]
    work = os.path.join(arguments.out, "work")
    export = os.path.join(arguments.out, "export")
    if "carpet" in config:
        pixels, report = carpet(config["carpet"], seed), f"knot {KNOT} px"
    else:
        pixels, report = concept_back(arguments.item, config["back"])
    depth = config["edges"]["depth"]
    pixels, step, start, score = periodic_edges(pixels, depth)
    report += f", edges depth {depth} step {step} start {start} score {score:.4f}"
    back = resample(to_image(pixels), size)
    middle = inlay_layer(tokens, size)
    front = rim_layer(tokens, size)
    for name, layer in (("back", back), ("middle", middle), ("front", front)):
        save(layer, os.path.join(work, f"{slug}_{name}.png"))
    flattened = Image.alpha_composite(Image.alpha_composite(back, middle), front)
    write_slice(export, depth, step, size)
    motion = config.get("motion")
    if not motion:
        save(flattened, os.path.join(export, "static.png"))
        print(f"{slug}: {report}, canvas {size[0]}x{size[1]}, seed {seed}, static")
        return
    threads = gold_threads(pixels, config["back"]["threads"], size) if "threads" in config["back"] else None
    frames = motion["frames"]
    save(flattened, os.path.join(export, "layer", "base", "0001.png"))
    for frame in range(frames):
        layer = motion_frame(motion, size, frame, threads, seed, tokens)
        save(layer, os.path.join(export, "layer", "motion", f"{frame + 1:04d}.png"))
        if frame + 1 == motion["key"]:
            save(layer, os.path.join(work, f"{slug}_motion_key.png"))
            save(Image.alpha_composite(flattened, layer), os.path.join(export, "static.png"))
    print(f"{slug}: {report}, canvas {size[0]}x{size[1]}, seed {seed}, {'+'.join(part['kind'] for part in motion['parts'])} {frames} frames, key frame {motion['key']}")


main()
