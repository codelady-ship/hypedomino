import argparse
import json
import os

import numpy as np
from PIL import Image

BOARD = (1516, 521)
UNITS = 1080
MEDIAN_THRESHOLD = 7.0
BRIGHT_THRESHOLD = 4.5
CHAIN = ((0, 1), (1, 3), (3, 3), (3, 5), (5, 6), (6, 6), (6, 2), (2, 4), (4, 0))


def premultiplied_box(image, size):
    pixels = np.asarray(image.convert("RGBA")).astype(np.float32) / 255
    pixels[..., :3] *= pixels[..., 3:]
    small = np.stack([np.asarray(Image.fromarray(pixels[..., channel], mode="F").resize(size, Image.BOX)) for channel in range(4)], axis=-1)
    alpha = small[..., 3:]
    colour = np.where(alpha > 0, small[..., :3] / np.maximum(alpha, 1e-6), 0)
    return Image.fromarray((np.dstack([colour, alpha]) * 255).round().clip(0, 255).astype(np.uint8), mode="RGBA")


def relative_luminance(rgb):
    linear = rgb / 255
    linear = np.where(linear <= 0.04045, linear / 12.92, ((linear + 0.055) / 1.055) ** 2.4)
    return linear @ np.array([0.2126, 0.7152, 0.0722])


def contrast(light, dark):
    return (light + 0.05) / (dark + 0.05)


def scenes(item):
    export = os.path.join(item, "export")
    static = Image.open(os.path.join(export, "static.png")).convert("RGBA")
    yield "static", static
    layers = os.path.join(export, "layer")
    if not os.path.isdir(layers):
        return
    back = Image.open(os.path.join(layers, "base", "0001.png")).convert("RGBA")
    for name in sorted(os.listdir(os.path.join(layers, "motion"))):
        yield f"frame {name[:4]}", Image.alpha_composite(back, Image.open(os.path.join(layers, "motion", name)).convert("RGBA"))


def centre_luminance(scene):
    width, height = scene.size
    box_width, box_height = BOARD[0] * height / UNITS, BOARD[1] * height / UNITS
    left, top = round((width - box_width) / 2), round((height - box_height) / 2)
    centre = np.asarray(scene.convert("RGB"))[top:top + round(box_height), left:left + round(box_width)].astype(np.float64)
    luminance = relative_luminance(centre.reshape(-1, 3))
    return float(np.median(luminance)), float(np.percentile(luminance, 90))


def ivory_luminance(faces, width):
    levels = []
    for name in sorted(os.listdir(faces)):
        face = np.asarray(premultiplied_box(Image.open(os.path.join(faces, name)), (width, 2 * width))).astype(np.float64)
        opaque = face[..., 3] == 255
        levels.append(np.median(relative_luminance(face[..., :3][opaque])))
    return float(np.median(levels))


def tinted_shadow(shadow, width, tokens):
    shade = premultiplied_box(Image.open(shadow), (round(176 * width / 128), round(304 * width / 128)))
    pixels = np.asarray(shade).astype(np.float64)
    tint = tokens["color"]["shadow"]
    pixels[..., :3] = [int(tint[i:i + 2], 16) for i in (1, 3, 5)]
    pixels[..., 3] *= tokens["elevation"]["e1"]["alpha"]
    return Image.fromarray(pixels.round().astype(np.uint8), mode="RGBA")


def sheet(scene, faces, shadow, tokens, width, screen):
    table = premultiplied_box(scene, (round(scene.width * screen / scene.height), screen))
    shade = tinted_shadow(shadow, width, tokens)
    offset = round(tokens["elevation"]["e1"]["offsetY"] * screen / UNITS)
    pieces = []
    for low, high in CHAIN:
        face = premultiplied_box(Image.open(os.path.join(faces, f"{min(low, high)}_{max(low, high)}.png")), (width, 2 * width))
        if low == high:
            pieces.append((face, shade))
        else:
            turn = 90 if low < high else -90
            pieces.append((face.rotate(turn, expand=True), shade.rotate(turn, expand=True)))
    total = sum(face.width for face, _ in pieces) + 2 * (len(pieces) - 1)
    x = (table.width - total) // 2
    for face, shade_piece in pieces:
        y = table.height // 2 - face.height // 2
        table.alpha_composite(shade_piece, (x - (shade_piece.width - face.width) // 2, y - (shade_piece.height - face.height) // 2 + offset))
        table.alpha_composite(face, (x, y))
        x += face.width + 2
    return table.convert("RGB")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("item")
    parser.add_argument("faces")
    parser.add_argument("shadow")
    parser.add_argument("tokens")
    parser.add_argument("out")
    parser.add_argument("--width", type=int, default=32)
    parser.add_argument("--screen", type=int, default=828)
    arguments = parser.parse_args()
    with open(arguments.tokens, encoding="utf-8") as source:
        tokens = json.load(source)
    os.makedirs(arguments.out, exist_ok=True)
    ivory = ivory_luminance(arguments.faces, arguments.width)
    worst = None
    for name, scene in scenes(arguments.item):
        median, bright = centre_luminance(scene)
        result = (contrast(ivory, median), contrast(ivory, bright), name, median, bright)
        if worst is None or result[:2] < worst[:2]:
            worst = result
        if name == "static":
            image = sheet(scene, arguments.faces, arguments.shadow, tokens, arguments.width, arguments.screen)
            image.save(os.path.join(arguments.out, "check.png"))
            middle = (image.width // 2, image.height // 2)
            crop = image.crop((middle[0] - 240, middle[1] - 80, middle[0] + 240, middle[1] + 80))
            crop.resize((crop.width * 3, crop.height * 3), Image.NEAREST).save(os.path.join(arguments.out, "check_x3.png"))
    passed = worst[0] >= MEDIAN_THRESHOLD and worst[1] >= BRIGHT_THRESHOLD
    print(f"ivory face {ivory:.3f}; worst scene {worst[2]}: centre median {worst[3]:.3f}, 90th percentile {worst[4]:.3f}; contrast {worst[0]:.1f}:1 against the median (needs {MEDIAN_THRESHOLD}:1), {worst[1]:.1f}:1 against the 90th percentile (needs {BRIGHT_THRESHOLD}:1): {'pass' if passed else 'fail'}")


main()
