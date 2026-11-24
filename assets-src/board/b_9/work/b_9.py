import argparse
import json
import os

import numpy as np
from PIL import Image
from scipy import ndimage

WIDTH = 1080
MARGIN = 24
GAP = 24
CANVAS_SCALE = 2
PIVOT_ARM = 24
PIVOT_STROKE = 2
SCREEN_SHORT_SIDE = 828
UNITS_SHORT_SIDE = 1080
TILE_WIDTH = 32
ICON_UNITS = 36
ICON = "settings"
KEY_FRAME = "0009"
SIZE_CELL = (112, 96)
ENLARGE = 4
GROUND_CENTRE = (0.41, 0.25)


def hex_colour(value):
    return tuple(int(value[at:at + 2], 16) for at in (1, 3, 5))


def token(tokens, path):
    node = tokens
    for key in path.split("."):
        node = node[key]
    return node


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


def draw_cross(pixels, half, arm, colour):
    middle_y, middle_x = pixels.shape[0] // 2, pixels.shape[1] // 2
    pixels[middle_y - half:middle_y + half, middle_x - arm:middle_x + arm] = colour
    pixels[middle_y - arm:middle_y + arm, middle_x - half:middle_x + half] = colour


def mark_pivot(image, colour, outline):
    pixels = np.asarray(image).copy()
    half = PIVOT_STROKE // 2
    draw_cross(pixels, half + PIVOT_STROKE, PIVOT_ARM + PIVOT_STROKE, outline + (255,))
    draw_cross(pixels, half, PIVOT_ARM, colour + (255,))
    return Image.fromarray(pixels, mode="RGBA")


def quadrants(static, key_frame):
    width, height = static.size
    split = key_frame.copy()
    split.paste(static.crop((0, 0, width // 2, height // 2)), (0, 0))
    split.paste(static.crop((width // 2, height // 2, width, height)), (width // 2, height // 2))
    return split


def gradient_tint(icon, top, bottom):
    pixels = np.asarray(icon).astype(np.float64)
    ramp = np.linspace(0, 1, icon.height)[:, None, None]
    tint = np.array(top) * (1 - ramp) + np.array(bottom) * ramp
    pixels[..., :3] = pixels[..., :3] / 255 * tint
    return Image.fromarray(pixels.round().clip(0, 255).astype(np.uint8), mode="RGBA")


class Board:
    def __init__(self, root):
        self.root = root
        self.tokens = json.load(open(os.path.join(root, "docs/design-tokens.json"), encoding="utf8"))
        skin = os.path.join(root, "assets-src/cosmetic/tile/sema/export")
        self.static = Image.open(os.path.join(skin, "static.png")).convert("RGBA")
        self.key_frame = Image.open(os.path.join(skin, "anim", f"{KEY_FRAME}.png")).convert("RGBA")

    def colour(self, path):
        return hex_colour(token(self.tokens, path))

    def canvas_cell(self, art):
        size = (art.width * CANVAS_SCALE, art.height * CANVAS_SCALE)
        cell = Image.new("RGBA", size, self.colour("color.surface.default") + (255,))
        cell.alpha_composite(art.resize(size, Image.NEAREST))
        return mark_pivot(cell, self.colour("color.text.primary"), self.colour("color.shadow")).convert("RGB")

    def ground(self):
        carpet = Image.open(os.path.join(self.root, "assets-src/cosmetic/table/crimson_carpet/export/static.png")).convert("RGBA")
        carpet = box(carpet, (round(carpet.width * SCREEN_SHORT_SIDE / carpet.height), SCREEN_SHORT_SIDE))
        left = round(carpet.width * GROUND_CENTRE[0]) - SIZE_CELL[0] // 2
        top = round(carpet.height * GROUND_CENTRE[1]) - SIZE_CELL[1] // 2
        return carpet.crop((left, top, left + SIZE_CELL[0], top + SIZE_CELL[1]))

    def shadow(self):
        shade = Image.open(os.path.join(self.root, "assets-src/effect/tile_shadow/export/static.png")).convert("RGBA")
        shade = bilinear(shade, (round(shade.width * TILE_WIDTH / self.static.width), round(shade.height * TILE_WIDTH / self.static.width)))
        pixels = np.asarray(shade).astype(np.float64)
        pixels[..., :3] = self.colour("color.shadow")
        pixels[..., 3] *= self.tokens["elevation"]["e1"]["alpha"]
        return Image.fromarray(pixels.round().astype(np.uint8), mode="RGBA")

    def size_cell(self):
        cell = self.ground()
        tile = bilinear(self.static, (TILE_WIDTH, 2 * TILE_WIDTH))
        shade = self.shadow()
        offset = round(self.tokens["elevation"]["e1"]["offsetY"] * SCREEN_SHORT_SIDE / UNITS_SHORT_SIDE)
        tile_x = (SIZE_CELL[0] // 2 - tile.width) // 2 + 4
        tile_y = (SIZE_CELL[1] - tile.height) // 2
        cell.alpha_composite(shade, (tile_x - (shade.width - tile.width) // 2, tile_y - (shade.height - tile.height) // 2 + offset))
        cell.alpha_composite(tile, (tile_x, tile_y))
        icon_side = round(ICON_UNITS * SCREEN_SHORT_SIDE / UNITS_SHORT_SIDE)
        icon = bilinear(Image.open(os.path.join(self.root, "assets-src/ui/icons/export", f"{ICON}.png")), (icon_side, icon_side))
        icon = gradient_tint(icon, self.colour("color.gold.hi"), self.colour("color.gold.lo"))
        cell.alpha_composite(icon, (SIZE_CELL[0] // 2 + (SIZE_CELL[0] // 2 - icon_side) // 2 - 4, (SIZE_CELL[1] - icon_side) // 2))
        return cell.convert("RGB")

    def panel(self):
        top_row = [self.canvas_cell(art) for art in (self.static, self.key_frame, quadrants(self.static, self.key_frame))]
        small = self.size_cell()
        large = small.resize((small.width * ENLARGE, small.height * ENLARGE), Image.NEAREST)
        height = 2 * MARGIN + top_row[0].height + GAP + large.height
        sheet = Image.new("RGB", (WIDTH, height), self.colour("color.background.base"))
        row_width = sum(cell.width for cell in top_row) + GAP * (len(top_row) - 1)
        left = (WIDTH - row_width) // 2
        for cell in top_row:
            sheet.paste(cell, (left, MARGIN))
            left += cell.width + GAP
        bottom = MARGIN + top_row[0].height + GAP
        left = (WIDTH - small.width - GAP - large.width) // 2
        sheet.paste(small, (left, bottom + (large.height - small.height) // 2))
        sheet.paste(large, (left + small.width + GAP, bottom))
        return sheet


def main():
    parser = argparse.ArgumentParser(prog="b_9.py")
    parser.add_argument("root")
    parser.add_argument("output")
    arguments = parser.parse_args()
    os.makedirs(os.path.dirname(arguments.output), exist_ok=True)
    panel = Board(arguments.root).panel()
    panel.save(arguments.output, optimize=False)
    print(f"b_9: {panel.size[0]}x{panel.size[1]}")


main()
