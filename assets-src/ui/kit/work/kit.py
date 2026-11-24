import json
import sys
from pathlib import Path

import numpy as np
from PIL import Image

SCALE = 2
SUPER = 4
CENTER = 8


def token(tokens, path):
    value = tokens
    for part in path.split("."):
        value = value[part]
    return value


def rgb(hex_value):
    return np.array([int(hex_value[index:index + 2], 16) for index in (1, 3, 5)], dtype=np.float64) / 255


def mix(a, b, t):
    return a * (1 - t) + b * t


def sample_grid(width, height):
    offsets = (np.arange(SUPER) + 0.5) / SUPER
    xs = (np.arange(width)[:, None] + offsets[None, :]).reshape(-1)
    ys = (np.arange(height)[:, None] + offsets[None, :]).reshape(-1)
    return np.meshgrid(xs, ys)


def downsample(values, width, height):
    return values.reshape(height, SUPER, width, SUPER, -1).mean(axis=(1, 3))


def rounded_box(x, y, left, top, right, bottom, radius):
    cx, cy = (left + right) / 2, (top + bottom) / 2
    hx, hy = (right - left) / 2 - radius, (bottom - top) / 2 - radius
    qx, qy = np.abs(x - cx) - hx, np.abs(y - cy) - hy
    outside = np.hypot(np.maximum(qx, 0), np.maximum(qy, 0))
    return outside + np.minimum(np.maximum(qx, qy), 0) - radius


def vertical(stops, t):
    positions = [position for position, _ in stops]
    colours = np.stack([colour for _, colour in stops])
    return np.stack([np.interp(t, positions, colours[:, channel]) for channel in range(3)], axis=-1)


def layer_over(base_rgb, base_alpha, rgb_values, alpha):
    out_alpha = alpha + base_alpha * (1 - alpha)
    safe = np.where(out_alpha > 0, out_alpha, 1)
    out_rgb = (rgb_values * alpha[..., None] + base_rgb * (base_alpha * (1 - alpha))[..., None]) / safe[..., None]
    return out_rgb, out_alpha


def finish(rgb_values, alpha, width, height):
    packed = np.concatenate([rgb_values * alpha[..., None], alpha[..., None]], axis=-1)
    small = downsample(packed, width, height)
    out_alpha = small[..., 3]
    safe = np.where(out_alpha > 0, out_alpha, 1)
    out_rgb = np.where(out_alpha[..., None] > 0, small[..., :3] / safe[..., None], 1.0)
    data = np.concatenate([out_rgb, out_alpha[..., None]], axis=-1)
    return Image.fromarray(np.round(np.clip(data, 0, 1) * 255).astype(np.uint8), "RGBA")


def lit_box(width, height, radius, body, rim, rim_width, inlay=None):
    x, y = sample_grid(width, height)
    t = y / height
    distance = rounded_box(x, y, 0, 0, width, height, radius)
    shape = (distance <= 0).astype(np.float64)
    colour = vertical(body, t)
    rim_mask = ((distance > -rim_width) & (distance <= 0)).astype(np.float64)
    colour, _ = layer_over(colour, shape, vertical(rim, t), rim_mask)
    if inlay is not None:
        inset, line, inlay_colour = inlay
        inner = rounded_box(x, y, inset, inset, width - inset, height - inset, max(radius - inset, 1))
        line_mask = (np.abs(inner + line / 2) <= line / 2).astype(np.float64)
        colour, _ = layer_over(colour, shape, np.broadcast_to(inlay_colour, colour.shape), line_mask)
    return finish(colour, shape, width, height)


def soft_box(width, height, margin, radius, softness):
    x, y = sample_grid(width, height)
    distance = rounded_box(x, y, margin, margin, width - margin, height - margin, radius)
    edge = np.clip(0.5 - distance / softness, 0, 1)
    alpha = edge * edge * (3 - 2 * edge)
    return finish(np.ones(alpha.shape + (3,)), alpha, width, height)


def white_box(width, height, radius):
    x, y = sample_grid(width, height)
    alpha = (rounded_box(x, y, 0, 0, width, height, radius) <= 0).astype(np.float64)
    return finish(np.ones(alpha.shape + (3,)), alpha, width, height)


def divider(colours):
    width, height = 128, 8
    x, y = sample_grid(width, height)
    u = x / width
    colour = vertical([(0.0, colours[0]), (0.5, colours[1]), (1.0, colours[0])], u)
    fade = np.clip(np.minimum(u, 1 - u) / 0.3, 0, 1)
    fade = fade * fade * (3 - 2 * fade)
    core = np.clip(1 - np.abs(y - height / 2) / 2.5, 0, 1)
    return finish(colour, fade * np.clip(core * 1.6, 0, 1), width, height)


def radial_shadow(size):
    x, y = sample_grid(size, size)
    r = np.hypot(x - size / 2, y - size / 2) / (size / 2)
    alpha = np.exp(-((r / 0.55) ** 2) * 2.2) * np.clip((1 - r) / 0.15, 0, 1)
    return finish(np.ones(alpha.shape + (3,)), alpha, size, size)


def build(tokens):
    px = lambda units: units * SCALE
    c = lambda path: rgb(token(tokens, path))
    gold_hi, gold, gold_mid, gold_lo = c("color.gold.hi"), c("color.gold.default"), c("color.gold.mid"), c("color.gold.lo")
    surface, surface_raised = c("color.surface.default"), c("color.surface.raised")
    ground_raised = c("color.background.raised")
    touch = px(token(tokens, "size.touchTargetMin"))
    md, lg = px(token(tokens, "radius.md")), px(token(tokens, "radius.lg"))
    rim = [(0.0, gold_hi), (0.5, gold), (1.0, gold_lo)]
    gold_body = lambda fill: [(0.0, gold_hi), (0.45, fill), (1.0, gold_mid)]
    button_side = md + 4
    pill_height, pill_radius = touch, touch // 2
    soft = px(16)
    margin = soft // 2
    sprites = {}

    def add(name, image, border, note):
        sprites[name] = (image, border, note)

    add("button_primary", lit_box(pill_radius * 2 + CENTER * 2, pill_height, pill_radius, gold_body(c("color.role.primary.fill")), rim, 3),
        [pill_radius, pill_radius - CENTER // 2, pill_radius, pill_radius - CENTER // 2], "Primary pill, role.primary fill with the gold gradient")
    add("button_secondary", lit_box(button_side * 2 + CENTER, touch, md, gold_body(c("color.role.secondary.fill")), rim, 3),
        [button_side, touch // 2 - CENTER // 2, button_side, touch // 2 - CENTER // 2], "Secondary button, radius.md, role.secondary fill")
    grey = lambda value: np.array([value, value, value])
    add("button_tint", lit_box(button_side * 2 + CENTER, touch, md, [(0.0, grey(1.0)), (0.45, grey(0.93)), (1.0, grey(0.8))], [(0.0, grey(1.0)), (1.0, grey(0.62))], 3),
        [button_side, touch // 2 - CENTER // 2, button_side, touch // 2 - CENTER // 2], "Greyscale button, radius.md, tinted with a role fill")
    panel_side = lg + 8
    add("panel", lit_box(panel_side * 2 + CENTER, panel_side * 2 + CENTER, lg, [(0.0, mix(surface, surface_raised, 0.3)), (0.5, surface), (1.0, mix(surface, ground_raised, 0.5))], [(0.0, gold_mid), (1.0, gold_lo)], 2, (10, 2, gold_mid)),
        [panel_side] * 4, "Panel, radius.lg, surface.default lacquer with a gold inlay line")
    side = md + 8
    add("panel_raised", lit_box(side * 2 + CENTER, side * 2 + CENTER, md, [(0.0, mix(surface_raised, gold_lo, 0.18)), (1.0, surface_raised)], [(0.0, gold_lo), (1.0, gold_lo)], 2),
        [side] * 4, "Panel inside a panel, list row, input; radius.md, surface.raised")
    add("tab_on", lit_box(side * 2 + CENTER, side * 2 + CENTER, md, [(0.0, mix(surface_raised, gold_lo, 0.3)), (1.0, surface_raised)], rim, 3),
        [side] * 4, "Selected tab, radius.md, surface.raised with the gold rim")
    add("tab_off", lit_box(side * 2 + CENTER, side * 2 + CENTER, md, [(0.0, mix(surface, surface_raised, 0.2)), (1.0, surface)], [(0.0, gold_lo), (1.0, gold_lo)], 2),
        [side] * 4, "Tab at rest, radius.md, surface.default with a gold.lo rim")
    for name, radius, pill, note in (
        ("shadow_md", md, False, "radius.md owners: secondary and tinted buttons, raised panels, tabs"),
        ("shadow_lg", lg, False, "radius.lg owners: panels"),
        ("shadow_pill", pill_radius, True, "pill owners: the primary button"),
    ):
        border_x = radius + margin * 2
        width = border_x * 2 + CENTER
        height = pill_height + margin * 2 if pill else width
        border_y = (height - CENTER) // 2
        add(name, soft_box(width, height, margin, radius, soft), [border_x, border_y, border_x, border_y],
            f"White elevation shadow for {note}; tint color.shadow at the elevation alpha, rect grown by {margin // SCALE} units per side")
    backdrop_height = px(64)
    backdrop_radius = backdrop_height // 2
    add("backdrop_soft", soft_box(backdrop_radius * 2 + CENTER, backdrop_height, 12, backdrop_radius - 12, 24),
        [backdrop_radius, backdrop_radius - CENTER // 2, backdrop_radius, backdrop_radius - CENTER // 2], "White soft pill behind seat names and hints; tint color.scrim at opacity.scrim")
    add("pill_fill", white_box(backdrop_radius * 2 + CENTER, backdrop_height, backdrop_radius),
        [backdrop_radius, backdrop_radius - CENTER // 2, backdrop_radius, backdrop_radius - CENTER // 2], "White crisp pill: toggle track, progress bar, badge, dot")
    add("divider", divider([gold_lo, gold_hi]), [0, 0, 0, 0], "Gold divider line, stretched as a simple sprite")
    add("icon_shadow", radial_shadow(96), [0, 0, 0, 0], "White soft shadow behind an icon; tint color.shadow")
    return sprites


def main(tokens_path, out_dir):
    tokens = json.loads(Path(tokens_path).read_text(encoding="utf-8"))
    out = Path(out_dir)
    out.mkdir(parents=True, exist_ok=True)
    sprites = build(tokens)
    entries = []
    for name, (image, border, note) in sorted(sprites.items()):
        image.save(out / f"{name}.png", optimize=False)
        entries.append({"name": name, "width": image.size[0], "height": image.size[1], "border": border, "pixelsPerUnit": 100 * SCALE, "use": note})
        print(f"{name}: {image.size[0]}x{image.size[1]} border left, bottom, right, top {border}")
    (out / "kit.json").write_text(json.dumps({"sprites": entries}, indent=2) + "\n", encoding="utf-8", newline="\n")


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
