import argparse
import json
import os

import numpy as np
from PIL import Image

WIDTH = 1080
MARGIN = 24
GAP = 24
CELL_HEIGHT = 360
LEVELS = ("low", "high")
ROWS = (("tilePlace", 0), ("scoreCascade", 1), ("scoreCascade", 3), ("scoreCascade", 6), ("domino", 0), ("block", 0))
TILE_SCALE = 0.5
SEED = 176
SPRITE_SIZE = {"dust": 22, "spark": 24, "star": 44, "streak": 96, "flake_paper": 18, "flake_leaf": 22}
DIM_OPACITY = "disabled"


def hex_colour(value):
    return np.array([int(value[at:at + 2], 16) for at in (1, 3, 5)], dtype=np.float64) / 255


def token(tokens, path):
    node = tokens
    for key in path.split("."):
        node = node[key]
    return node


def load(path):
    return np.asarray(Image.open(path).convert("RGBA"), dtype=np.float64) / 255


class Canvas:
    def __init__(self, width, height, colour):
        self.rgb = np.ones((height, width, 3)) * colour

    def place(self, sprite, centre, size, tint, alpha=1.0, angle=0.0, additive=True):
        image = Image.fromarray((sprite * 255).round().astype(np.uint8), mode="RGBA")
        image = image.resize((max(1, int(round(size[0]))), max(1, int(round(size[1])))), Image.BICUBIC)
        if angle:
            image = image.rotate(angle, resample=Image.BICUBIC, expand=True)
        pixels = np.asarray(image, dtype=np.float64) / 255
        height, width = pixels.shape[:2]
        left = int(round(centre[0] - width / 2))
        top = int(round(centre[1] - height / 2))
        x0, y0 = max(left, 0), max(top, 0)
        x1, y1 = min(left + width, self.rgb.shape[1]), min(top + height, self.rgb.shape[0])
        if x0 >= x1 or y0 >= y1:
            return
        patch = pixels[y0 - top:y1 - top, x0 - left:x1 - left]
        colour = patch[..., :3] * tint
        weight = patch[..., 3:4] * alpha
        target = self.rgb[y0:y1, x0:x1]
        self.rgb[y0:y1, x0:x1] = target + colour * weight if additive else target * (1 - weight) + colour * weight

    def image(self):
        return Image.fromarray((np.clip(self.rgb, 0, 1) * 255).round().astype(np.uint8), mode="RGB")


class Board:
    def __init__(self, root):
        self.tokens = json.load(open(os.path.join(root, "docs/design-tokens.json"), encoding="utf8"))
        self.effects = {entry["name"]: entry for entry in json.load(open(os.path.join(root, "client/Assets/HypeDomino/UI/Effects/effects.json"), encoding="utf8"))["effects"]}
        sprites = os.path.join(root, "assets-src/effect/vfx/export")
        self.sprites = {name[:-4]: load(os.path.join(sprites, name)) for name in sorted(os.listdir(sprites))}
        self.sprites["tile_shadow"] = load(os.path.join(root, "assets-src/effect/tile_shadow/export/static.png"))
        faces = os.path.join(root, "assets-src/tile/ivory/export/face")
        self.faces = {name[:-4]: load(os.path.join(faces, name)) for name in ("4_6.png", "6_6.png", "1_6.png")}
        carpet = Image.open(os.path.join(root, "assets-src/cosmetic/table/crimson_carpet/export/static.png")).convert("RGB")
        cell_width = (WIDTH - 2 * MARGIN - GAP) // 2
        left = (carpet.width - cell_width) // 2
        top = (carpet.height - CELL_HEIGHT) // 2
        self.carpet = np.asarray(carpet.crop((left, top, left + cell_width, top + CELL_HEIGHT)), dtype=np.float64) / 255
        self.cell_width = cell_width

    def colour(self, path):
        if path == "team":
            path = "color.role.teamA.fill"
        return hex_colour(token(self.tokens, path))

    def chain(self, canvas, dim):
        middle_x, middle_y = self.cell_width / 2, CELL_HEIGHT / 2
        long_side, short_side = 256 * TILE_SCALE, 128 * TILE_SCALE
        tiles = (("4_6", (middle_x - long_side / 2 - short_side / 2, middle_y), 90), ("6_6", (middle_x, middle_y), 0), ("1_6", (middle_x + long_side / 2 + short_side / 2, middle_y), -90))
        shadow = self.sprites["tile_shadow"]
        elevation = self.tokens["elevation"]["e1"]
        for _, centre, angle in tiles:
            canvas.place(shadow, (centre[0], centre[1] + elevation["offsetY"]), (shadow.shape[1] * TILE_SCALE, shadow.shape[0] * TILE_SCALE), self.colour("color.shadow"), elevation["alpha"], angle, additive=False)
        for name, centre, angle in tiles:
            canvas.place(self.faces[name], centre, (short_side, long_side), np.ones(3), 1.0, angle, additive=False)
        if dim:
            canvas.rgb = canvas.rgb * (1 - dim) + self.colour("color.scrim") * dim
        return tiles[-1][1]

    def glow(self, canvas, centre, size, tint, level):
        glow = self.tokens["glow"][level]
        canvas.place(self.sprites["glow"], centre, (size * glow["scale"], size * glow["scale"]), tint, glow["alpha"])

    def scatter(self, canvas, random, name, count, tint, centre, radius, toward=None):
        size = SPRITE_SIZE[name]
        for _ in range(count):
            angle = random.uniform(0, 2 * np.pi)
            distance = radius * np.sqrt(random.uniform(0.15, 1.0))
            point = (centre[0] + distance * np.cos(angle), centre[1] + distance * np.sin(angle) * 0.75)
            scale = random.uniform(0.6, 1.1)
            if name == "streak":
                heading = np.degrees(np.arctan2(-(toward[1] - point[1]), toward[0] - point[0]))
                canvas.place(self.sprites[name], point, (size * scale, size * scale / 4), tint, 1.0, heading)
                continue
            spin = random.uniform(0, 360) if name.startswith("flake") or name == "star" else 0.0
            canvas.place(self.sprites[name], point, (size * scale, size * scale * (1.33 if name == "flake_paper" else 1.0)), tint, random.uniform(0.55, 1.0), spin, additive=not name.startswith("flake"))

    def cascade_counts(self, level, step):
        budget = self.tokens["effects"]["budgets"]["scoreCascade"][level]["particles"]
        percent = round(self.tokens["scoreCascade"]["steps"][step - 1]["particleShare"] * 100)
        total = (budget * percent + 50) // 100
        mix = list(self.effects["scoreCascade"]["stepMix"].items())
        counts = {sprite: (total * share + 50) // 100 for sprite, share in mix[:-1]}
        counts[mix[-1][0]] = total - sum(counts.values())
        return counts

    def cell(self, effect, step, level, random):
        canvas = Canvas(self.cell_width, CELL_HEIGHT, np.zeros(3))
        canvas.rgb = self.carpet.copy()
        definition = self.effects[effect]["levels"][level]
        dim = self.tokens["opacity"][DIM_OPACITY] if effect == "block" else 0.0
        last = self.chain(canvas, dim)
        middle = (self.cell_width / 2, CELL_HEIGHT / 2)
        meter = (self.cell_width - 30, 30)
        if effect == "tilePlace":
            for particle in definition["particles"]:
                if particle["sprite"] == "streak":
                    canvas.place(self.sprites["streak"], (last[0] - 120, last[1]), (150, 40), self.colour(particle["tint"]), 0.9)
                    continue
                self.scatter(canvas, random, particle["sprite"], particle["count"], self.colour(particle["tint"]), (last[0], last[1] + 50), 80)
        if effect == "scoreCascade":
            tint = self.colour("color.role.score.fill")
            popup = (last[0], last[1] - 70)
            self.glow(canvas, popup, 220, tint, self.tokens["scoreCascade"]["steps"][step - 1]["glow"])
            for sprite, count in self.cascade_counts(level, step).items():
                self.scatter(canvas, random, sprite, count, tint, popup, 110, meter)
        if effect == "domino":
            for element in definition["sprites"]:
                self.glow(canvas, last, 260, self.colour(element["tint"]), element["glow"])
            for particle in definition["particles"]:
                self.scatter(canvas, random, particle["sprite"], particle["count"], self.colour(particle["tint"]), last, 170)
        if effect == "block":
            for element in definition["sprites"]:
                size = 300 if element["sprite"] == "ring" else 220
                additive = element["sprite"] != "stamp"
                canvas.place(self.sprites[element["sprite"]], middle, (size, size), self.colour(element["tint"]), 0.9, 0.0, additive)
            for particle in definition["particles"]:
                self.scatter(canvas, random, particle["sprite"], particle["count"], self.colour(particle["tint"]), middle, 170)
        return canvas.image()

    def panel(self):
        height = 2 * MARGIN + len(ROWS) * CELL_HEIGHT + (len(ROWS) - 1) * GAP
        sheet = Image.new("RGB", (WIDTH, height), tuple(int(v) for v in (hex_colour(token(self.tokens, "color.background.base")) * 255).round()))
        random = np.random.default_rng(SEED)
        for row, (effect, step) in enumerate(ROWS):
            for column, level in enumerate(LEVELS):
                left = MARGIN + column * (self.cell_width + GAP)
                top = MARGIN + row * (CELL_HEIGHT + GAP)
                sheet.paste(self.cell(effect, step, level, random), (left, top))
        return sheet


def main():
    parser = argparse.ArgumentParser(prog="b_8.py")
    parser.add_argument("root")
    parser.add_argument("output")
    arguments = parser.parse_args()
    os.makedirs(os.path.dirname(arguments.output), exist_ok=True)
    panel = Board(arguments.root).panel()
    panel.save(arguments.output, quality=82)
    print(f"b_8: {panel.size[0]}x{panel.size[1]}")


main()
