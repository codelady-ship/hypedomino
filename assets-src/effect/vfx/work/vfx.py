import argparse
import os

import numpy as np
from PIL import Image

SUPERSAMPLE = 4
ALPHA_FLOOR = 2.0 / 255.0
LEAF_OUTLINE = (0.92, 0.70, 0.88, 0.62, 0.95, 0.74, 0.86, 0.66, 0.90, 0.72, 0.84)
LEAF_FACETS = (1.00, 0.78, 0.92, 0.70, 0.96, 0.82, 0.74, 0.88, 0.80, 0.94, 0.76)


def smoothstep(edge0, edge1, value):
    ramp = np.clip((value - edge0) / (edge1 - edge0), 0, 1)
    return ramp * ramp * (3 - 2 * ramp)


def grid(width, height):
    xs = (np.arange(width * SUPERSAMPLE) + 0.5) / (width * SUPERSAMPLE) * 2 - 1
    ys = 1 - (np.arange(height * SUPERSAMPLE) + 0.5) / (height * SUPERSAMPLE) * 2
    return np.meshgrid(xs, ys)


def polar(width, height):
    x, y = grid(width, height)
    return x, y, np.hypot(x, y), np.arctan2(y, x)


def glow(size):
    _, _, radius, _ = polar(size, size)
    falloff = np.exp(-0.5 * (radius / 0.38) ** 2)
    edge = np.exp(-0.5 * (1 / 0.38) ** 2)
    return np.clip((falloff - edge) / (1 - edge), 0, 1) * (radius < 1)


def ring_profile(radius):
    inner = smoothstep(0.40, 0.66, radius)
    outer = (1 - smoothstep(0.66, 1.0, radius)) ** 1.6
    return inner * outer


def halo(size):
    _, _, radius, _ = polar(size, size)
    return ring_profile(radius)


def flow_halo(size):
    _, _, radius, angle = polar(size, size)
    from_top = np.angle(np.exp(1j * (angle - np.pi / 2)))
    arc_band = np.exp(-0.5 * (from_top / 0.38) ** 2)
    arc_ring = smoothstep(0.46, 0.68, radius) * (1 - smoothstep(0.70, 0.99, radius))
    base = 0.6 * ring_profile(radius)
    arc = arc_band * arc_ring
    return 1 - (1 - base) * (1 - arc)


def star(size):
    x, y, radius, _ = polar(size, size)

    def ray(along, across):
        reach = np.clip(1 - np.abs(along), 0, 1)
        width = 0.075 * reach + 0.012
        return reach ** 1.4 * np.exp(-0.5 * (across / width) ** 2)

    core = np.exp(-0.5 * (radius / 0.16) ** 2)
    return np.clip(np.maximum(ray(x, y), ray(y, x)) + 0.55 * core, 0, 1)


def spark(size):
    _, _, radius, _ = polar(size, size)
    shape = np.exp(-0.5 * (radius / 0.2) ** 2) + 0.35 * np.exp(-0.5 * (radius / 0.5) ** 2)
    return np.clip(shape, 0, 1) * (1 - smoothstep(0.85, 1.0, radius))


def dust(size):
    _, _, radius, _ = polar(size, size)
    return 0.8 * (1 - smoothstep(0.0, 1.0, radius)) ** 2


def streak(width, height):
    x, y = grid(width, height)
    along = (x + 1) / 2
    thickness = 0.1 + 0.85 * along ** 1.3
    body = along ** 1.2 * np.exp(-0.5 * (y / (0.45 * thickness)) ** 2)
    head = 1 - smoothstep(0.93, 1.0, along)
    return np.clip(body * head, 0, 1)


def ring(size):
    _, _, radius, _ = polar(size, size)
    return np.exp(-0.5 * ((radius - 0.8) / 0.06) ** 2) * (1 - smoothstep(0.96, 1.0, radius))


def rays(size):
    _, _, radius, angle = polar(size, size)
    long_ray = (0.5 + 0.5 * np.cos(8 * angle)) ** 18
    short_ray = (0.5 + 0.5 * np.cos(8 * angle + np.pi)) ** 18
    long_reach = np.clip(1 - radius, 0, 1) ** 1.5
    short_reach = np.clip(1 - radius / 0.68, 0, 1) ** 1.5
    hollow = smoothstep(0.12, 0.3, radius)
    return np.clip(long_ray * long_reach + 0.8 * short_ray * short_reach, 0, 1) * hollow


def rounded_box(x, y, half_width, half_height, corner):
    dx = np.maximum(np.abs(x) - (half_width - corner), 0)
    dy = np.maximum(np.abs(y) - (half_height - corner), 0)
    inside = (np.abs(x) <= half_width) & (np.abs(y) <= half_height)
    return inside & (np.hypot(dx, dy) <= corner)


def stamp(size):
    x, y, radius, angle = polar(size, size)
    notched = 0.94 - 0.025 * (0.5 + 0.5 * np.cos(24 * angle))
    seal = (radius <= notched) & (radius >= 0.8)
    body = rounded_box(x, y + 0.2, 0.32, 0.22, 0.06)
    shackle_y = y - 0.1
    shackle_radius = np.hypot(x, shackle_y)
    arch = (shackle_radius >= 0.13) & (shackle_radius <= 0.22) & (shackle_y >= 0)
    legs = (np.abs(x) >= 0.13) & (np.abs(x) <= 0.22) & (y <= 0.1) & (y >= 0.0)
    keyhole = (np.hypot(x, y + 0.14) <= 0.06) | ((np.abs(x) <= 0.025) & (y <= -0.14) & (y >= -0.3))
    lock = (body | arch | legs) & ~keyhole
    return ((seal | lock) & (radius < 1)).astype(float)


def paper(width, height):
    x, y = grid(width, height)
    alpha = rounded_box(x, y, 0.9, 0.92, 0.08).astype(float)
    curl = 0.8 + 0.2 * np.cos((x - 0.25) * 1.6)
    return alpha, curl


def leaf(size):
    _, _, radius, angle = polar(size, size)
    count = len(LEAF_OUTLINE)
    position = (angle + np.pi) / (2 * np.pi) * count
    sector = np.floor(position).astype(int) % count
    blend = position - np.floor(position)
    outline = np.asarray(LEAF_OUTLINE)
    edge = outline[sector] * (1 - blend) + outline[(sector + 1) % count] * blend
    alpha = (radius <= edge).astype(float)
    shade = np.asarray(LEAF_FACETS)[sector] * (0.9 + 0.1 * (1 - radius))
    return alpha, shade


def to_image(width, height, alpha, value=None):
    canvas = (width, height)

    def shrink(channel):
        source = Image.fromarray((channel * 255).astype(np.float32), mode="F")
        return np.asarray(source.resize(canvas, Image.BOX))

    small_alpha = shrink(np.clip(alpha, 0, 1))
    if value is None:
        small_value = np.full(small_alpha.shape, 255.0)
    else:
        weighted = shrink(np.clip(alpha * value, 0, 1))
        small_value = np.where(small_alpha > 0, weighted / np.maximum(small_alpha, 1e-6) * 255, 255.0)
    small_alpha = np.where(small_alpha < ALPHA_FLOOR * 255, 0, small_alpha)
    pixels = np.dstack([small_value, small_value, small_value, small_alpha]).round().clip(0, 255).astype(np.uint8)
    return Image.fromarray(pixels, mode="RGBA")


def sprites():
    paper_alpha, paper_shade = paper(24, 32)
    leaf_alpha, leaf_shade = leaf(32)
    return {
        "glow": to_image(128, 128, glow(128)),
        "halo": to_image(256, 256, halo(256)),
        "flow_halo": to_image(256, 256, flow_halo(256)),
        "star": to_image(64, 64, star(64)),
        "spark": to_image(32, 32, spark(32)),
        "dust": to_image(32, 32, dust(32)),
        "streak": to_image(128, 32, streak(128, 32)),
        "ring": to_image(128, 128, ring(128)),
        "rays": to_image(256, 256, rays(256)),
        "stamp": to_image(256, 256, stamp(256)),
        "flake_paper": to_image(24, 32, paper_alpha, paper_shade),
        "flake_leaf": to_image(32, 32, leaf_alpha, leaf_shade),
    }


def main():
    parser = argparse.ArgumentParser(prog="vfx.py")
    parser.add_argument("export")
    arguments = parser.parse_args()
    os.makedirs(arguments.export, exist_ok=True)
    for name, image in sprites().items():
        image.save(os.path.join(arguments.export, f"{name}.png"), optimize=True)
        print(f"{name}: {image.size[0]}x{image.size[1]}")


main()
