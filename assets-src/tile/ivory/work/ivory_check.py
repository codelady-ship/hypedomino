import argparse
import json
import os

import numpy as np
from PIL import Image
from scipy import ndimage

SIDE_INSET = 0.12
END_INSET = 0.08
DIVIDER_BAND = 0.1
GAP = 2
PIP_DIAMETER = 0.176
PIP_GAP = 0.094
PIP_LEVEL = 30


def faces():
    return [(low, high) for low in range(7) for high in range(low, 7)]


def premultiplied(image):
    pixels = np.asarray(image.convert("RGBA")).astype(np.float64) / 255
    return np.dstack([pixels[..., :3] * pixels[..., 3:], pixels[..., 3:]])


def straight(pixels):
    alpha = pixels[..., 3:]
    colour = np.where(alpha > 0, pixels[..., :3] / np.maximum(alpha, 1e-6), 0)
    return Image.fromarray((np.dstack([colour, alpha]) * 255).round().clip(0, 255).astype(np.uint8), mode="RGBA")


def box(image, size):
    pixels = premultiplied(image).astype(np.float32)
    small = np.stack([np.asarray(Image.fromarray(pixels[..., channel], mode="F").resize(size, Image.BOX)) for channel in range(4)], axis=-1)
    return straight(small.astype(np.float64))


def bilinear(image, size):
    pixels = premultiplied(image)
    height, width = pixels.shape[:2]
    ys = (np.arange(size[1]) + 0.5) * height / size[1] - 0.5
    xs = (np.arange(size[0]) + 0.5) * width / size[0] - 0.5
    grid = np.meshgrid(ys, xs, indexing="ij")
    sampled = np.stack([ndimage.map_coordinates(pixels[..., channel], grid, order=1, mode="nearest") for channel in range(4)], axis=-1)
    return straight(sampled)


def luminance(image, backdrop):
    canvas = Image.new("RGBA", image.size, backdrop)
    canvas.alpha_composite(image)
    return np.asarray(canvas.convert("L")).astype(np.float64)


def halves(grey, width):
    side = round(width * SIDE_INSET)
    end = round(width * END_INSET)
    band = round(width * DIVIDER_BAND)
    columns = slice(side, width - side)
    return grey[end:width - band, columns], grey[width + band:2 * width - end, columns]


def count_pips(grey, width, threshold):
    counts = []
    for half in halves(grey, width):
        labels, found = ndimage.label(half < threshold)
        sizes = ndimage.sum(np.ones_like(half), labels, range(1, found + 1))
        counts.append(int(np.sum(sizes >= 2)))
    return counts


def measure(export, width):
    ivory = []
    pip = []
    failures = []
    for sampler in (box, bilinear):
        for low, high in faces():
            small = sampler(Image.open(os.path.join(export, f"{low}_{high}.png")), (width, 2 * width))
            grey = luminance(small, (255, 255, 255, 255))
            top, bottom = halves(grey, width)
            body = np.concatenate([top.ravel(), bottom.ravel()])
            face_level = np.median(body)
            ivory.append(face_level)
            if low or high:
                pip.append(body.min())
            threshold = (face_level + PIP_LEVEL) / 2
            counts = count_pips(grey, width, threshold)
            if counts != [low, high]:
                failures.append(f"{sampler.__name__} {low}_{high}: found {counts}")
    return float(np.median(ivory)), float(np.median(pip)), failures


def sheet(export, width, table, shadow, scale, tokens):
    columns, rows = 7, 4
    step = (width + GAP * 4, 2 * width + GAP * 4)
    size = (columns * step[0] + GAP * 8, rows * step[1] + GAP * 8)
    backdrop = Image.open(table).convert("RGBA")
    backdrop = box(backdrop, (round(backdrop.width * scale / backdrop.height), scale))
    left = (backdrop.width - size[0]) // 2
    top = (backdrop.height - size[1]) // 2
    canvas = backdrop.crop((left, top, left + size[0], top + size[1]))
    shade = Image.open(shadow).convert("RGBA")
    shade_size = (round(shade.width * width / 128), round(shade.height * width / 128))
    shade = box(shade, shade_size)
    shade_pixels = np.asarray(shade).astype(np.float64)
    tint = tokens["color"]["shadow"]
    shade_pixels[..., :3] = [int(tint[i:i + 2], 16) for i in (1, 3, 5)]
    shade_pixels[..., 3] *= tokens["elevation"]["e1"]["alpha"]
    shade = Image.fromarray(shade_pixels.round().astype(np.uint8), mode="RGBA")
    offset = round(tokens["elevation"]["e1"]["offsetY"] * scale / 1080)
    for index, (low, high) in enumerate(faces()):
        x = GAP * 6 + (index % columns) * step[0]
        y = GAP * 6 + (index // columns) * step[1]
        canvas.alpha_composite(shade, (x - (shade.width - width) // 2, y - (shade.height - 2 * width) // 2 + offset))
        canvas.alpha_composite(box(Image.open(os.path.join(export, f"{low}_{high}.png")), (width, 2 * width)), (x, y))
    return canvas.convert("RGB")


def main():
    parser = argparse.ArgumentParser(prog="ivory_check.py")
    parser.add_argument("faces")
    parser.add_argument("table")
    parser.add_argument("shadow")
    parser.add_argument("tokens")
    parser.add_argument("out")
    parser.add_argument("--width", type=int, nargs="+", required=True)
    parser.add_argument("--screen", type=int, default=828)
    arguments = parser.parse_args()
    os.makedirs(arguments.out, exist_ok=True)
    with open(arguments.tokens, encoding="utf-8") as source:
        tokens = json.load(source)
    for width in arguments.width:
        ivory, pip, failures = measure(arguments.faces, width)
        image = sheet(arguments.faces, width, arguments.table, arguments.shadow, arguments.screen, tokens)
        image.save(os.path.join(arguments.out, f"check_{width}.png"))
        image.resize((image.width * 4, image.height * 4), Image.NEAREST).save(os.path.join(arguments.out, f"check_{width}_x4.png"))
        print(f"{width}x{2 * width} px: pip diameter {PIP_DIAMETER * width:.1f} px, gap between neighbouring pips {PIP_GAP * width:.1f} px, ivory luminance {ivory:.0f}, darkest pip {pip:.0f}, faces with a wrong pip count {len(failures)} of 56")
        for failure in failures:
            print(f"  {failure}")


main()
