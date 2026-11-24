import json
import math
import re
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

import numpy as np
from PIL import Image

SIZE = 96
VIEW = 24.0
SUPER = 16
CURVE_STEPS = 24
SVG = "{http://www.w3.org/2000/svg}"
CLIP_TOLERANCE = 0.01
TRANSLATE = re.compile(r"translate\(\s*([-+.\deE]+)(?:[\s,]+([-+.\deE]+))?\s*\)")
TOKEN = re.compile(r"[MmLlHhVvCcSsQqTtAaZz]|[-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?")
ARGS = {"M": 2, "L": 2, "H": 1, "V": 1, "C": 6, "S": 4, "Q": 4, "T": 2, "A": 7, "Z": 0}


def cubic(p0, p1, p2, p3):
    t = np.linspace(0.0, 1.0, CURVE_STEPS + 1)[1:, None]
    u = 1.0 - t
    return list(map(tuple, u**3 * p0 + 3 * u * u * t * p1 + 3 * u * t * t * p2 + t**3 * p3))


def arc(start, rx, ry, angle, large, sweep, end):
    if rx == 0 or ry == 0 or start == end:
        return [end]
    phi = math.radians(angle)
    cos_phi, sin_phi = math.cos(phi), math.sin(phi)
    dx, dy = (start[0] - end[0]) / 2, (start[1] - end[1]) / 2
    x1 = cos_phi * dx + sin_phi * dy
    y1 = -sin_phi * dx + cos_phi * dy
    rx, ry = abs(rx), abs(ry)
    scale = x1 * x1 / (rx * rx) + y1 * y1 / (ry * ry)
    if scale > 1:
        rx, ry = rx * math.sqrt(scale), ry * math.sqrt(scale)
    numerator = rx * rx * ry * ry - rx * rx * y1 * y1 - ry * ry * x1 * x1
    denominator = rx * rx * y1 * y1 + ry * ry * x1 * x1
    factor = math.sqrt(max(0.0, numerator / denominator))
    if large == sweep:
        factor = -factor
    cx1, cy1 = factor * rx * y1 / ry, -factor * ry * x1 / rx
    cx = cos_phi * cx1 - sin_phi * cy1 + (start[0] + end[0]) / 2
    cy = sin_phi * cx1 + cos_phi * cy1 + (start[1] + end[1]) / 2
    theta = math.atan2((y1 - cy1) / ry, (x1 - cx1) / rx)
    delta = math.atan2((-y1 - cy1) / ry, (-x1 - cx1) / rx) - theta
    if sweep and delta < 0:
        delta += 2 * math.pi
    if not sweep and delta > 0:
        delta -= 2 * math.pi
    steps = max(4, int(abs(delta) / (2 * math.pi) * CURVE_STEPS * 4))
    points = []
    for index in range(1, steps + 1):
        a = theta + delta * index / steps
        ex, ey = rx * math.cos(a), ry * math.sin(a)
        points.append((cos_phi * ex - sin_phi * ey + cx, sin_phi * ex + cos_phi * ey + cy))
    points[-1] = end
    return points


def subpaths(d):
    tokens = TOKEN.findall(d)
    contours, contour = [], []
    current = start = (0.0, 0.0)
    last_control, last_command = None, ""
    command, index = None, 0
    while index < len(tokens):
        if tokens[index].isalpha():
            command = tokens[index]
            index += 1
            if command in "Zz":
                if contour:
                    contours.append(contour)
                contour, current, last_control, last_command = [], start, None, "Z"
                continue
        count = ARGS[command.upper()]
        if count == 0:
            raise ValueError("numbers after a close path command")
        values = [float(value) for value in tokens[index:index + count]]
        index += count
        relative = command.islower()
        upper = command.upper()
        ox, oy = current if relative else (0.0, 0.0)
        if upper == "M":
            if contour:
                contours.append(contour)
            current = start = (values[0] + ox, values[1] + oy)
            contour = [current]
            command = "l" if relative else "L"
            last_control = None
        elif upper == "L":
            current = (values[0] + ox, values[1] + oy)
            contour.append(current)
            last_control = None
        elif upper == "H":
            current = (values[0] + ox, current[1])
            contour.append(current)
            last_control = None
        elif upper == "V":
            current = (current[0], values[0] + (current[1] if relative else 0.0))
            contour.append(current)
            last_control = None
        elif upper in "CS":
            if upper == "C":
                c1 = (values[0] + ox, values[1] + oy)
                rest = values[2:]
            else:
                c1 = (2 * current[0] - last_control[0], 2 * current[1] - last_control[1]) if last_command in "CS" and last_control else current
                rest = values
            c2 = (rest[0] + ox, rest[1] + oy)
            end = (rest[2] + ox, rest[3] + oy)
            contour.extend(cubic(np.array(current), np.array(c1), np.array(c2), np.array(end)))
            current, last_control = end, c2
        elif upper in "QT":
            if upper == "Q":
                q = (values[0] + ox, values[1] + oy)
                end = (values[2] + ox, values[3] + oy)
            else:
                q = (2 * current[0] - last_control[0], 2 * current[1] - last_control[1]) if last_command in "QT" and last_control else current
                end = (values[0] + ox, values[1] + oy)
            c1 = (current[0] + 2 / 3 * (q[0] - current[0]), current[1] + 2 / 3 * (q[1] - current[1]))
            c2 = (end[0] + 2 / 3 * (q[0] - end[0]), end[1] + 2 / 3 * (q[1] - end[1]))
            contour.extend(cubic(np.array(current), np.array(c1), np.array(c2), np.array(end)))
            current, last_control = end, q
        elif upper == "A":
            end = (values[5] + ox, values[6] + oy)
            contour.extend(arc(current, values[0], values[1], values[2], int(values[3]), int(values[4]), end))
            current, last_control = end, None
        last_command = upper
    if contour:
        contours.append(contour)
    return contours


def coverage(contours, even_odd):
    grid = SIZE * SUPER
    unit = grid / VIEW
    x0s, y0s, x1s, y1s = [], [], [], []
    for contour in contours:
        points = np.array(contour + [contour[0]], dtype=np.float64) * unit
        x0s.append(points[:-1, 0])
        y0s.append(points[:-1, 1])
        x1s.append(points[1:, 0])
        y1s.append(points[1:, 1])
    x0, y0, x1, y1 = (np.concatenate(values) for values in (x0s, y0s, x1s, y1s))
    moving = y0 != y1
    x0, y0, x1, y1 = x0[moving], y0[moving], x1[moving], y1[moving]
    direction = np.where(y1 > y0, 1, -1)
    top, bottom = np.minimum(y0, y1), np.maximum(y0, y1)
    first = np.clip(np.ceil(top - 0.5), 0, grid).astype(np.int64)
    last = np.clip(np.ceil(bottom - 0.5), 0, grid).astype(np.int64)
    spans = last - first
    edge = np.repeat(np.arange(len(spans)), spans)
    row = first[edge] + np.arange(spans.sum()) - np.repeat(np.cumsum(spans) - spans, spans)
    center = row + 0.5
    x = x0[edge] + (center - y0[edge]) * (x1[edge] - x0[edge]) / (y1[edge] - y0[edge])
    column = np.clip(np.floor(x - 0.5) + 1, 0, grid).astype(np.int64)
    delta = np.zeros((grid, grid + 1), dtype=np.int32)
    np.add.at(delta, (row, column), direction[edge])
    winding = np.cumsum(delta[:, :grid], axis=1)
    inside = (winding % 2 != 0) if even_odd else (winding != 0)
    return inside.reshape(SIZE, SUPER, SIZE, SUPER).mean(axis=(1, 3))


def clip_bounds(rect):
    tx, ty = 0.0, 0.0
    transform = rect.attrib.get("transform")
    if transform:
        match = TRANSLATE.fullmatch(transform)
        if not match:
            raise ValueError(f"unsupported clip transform {transform}")
        tx, ty = float(match.group(1)), float(match.group(2) or 0)
    x = float(rect.attrib.get("x", "0")) + tx
    y = float(rect.attrib.get("y", "0")) + ty
    return x, y, x + float(rect.attrib["width"]), y + float(rect.attrib["height"])


def check_clip(root):
    for clip in root.iter(SVG + "clipPath"):
        for rect in clip:
            left, top, right, bottom = clip_bounds(rect)
            if left > CLIP_TOLERANCE or top > CLIP_TOLERANCE or right < VIEW - CLIP_TOLERANCE or bottom < VIEW - CLIP_TOLERANCE:
                raise ValueError(f"clip {rect.attrib} does not cover the view box")


def layers(element, opacity, out):
    tag = element.tag.replace(SVG, "")
    if tag in ("defs", "clipPath"):
        return
    if "transform" in element.attrib or "stroke" in element.attrib:
        raise ValueError(f"unsupported {tag} attributes {element.attrib}")
    opacity *= float(element.attrib.get("opacity", "1"))
    if tag == "path":
        if element.attrib.get("fill", "none") != "none":
            out.append((element.attrib["d"], opacity, element.attrib.get("fill-rule") == "evenodd"))
        return
    if tag not in ("svg", "g"):
        raise ValueError(f"unsupported element {tag}")
    for child in element:
        layers(child, opacity, out)


def rasterize(svg):
    root = ET.fromstring(svg)
    if root.attrib.get("viewBox") != "0 0 24 24":
        raise ValueError(f"unsupported viewBox {root.attrib.get('viewBox')}")
    check_clip(root)
    paths = []
    layers(root, 1.0, paths)
    alpha = np.zeros((SIZE, SIZE))
    for d, opacity, even_odd in paths:
        source = coverage(subpaths(d), even_odd) * opacity
        alpha = source + alpha * (1.0 - source)
    rgba = np.zeros((SIZE, SIZE, 4), dtype=np.uint8)
    rgba[..., :3] = 255
    rgba[..., 3] = np.round(np.clip(alpha, 0.0, 1.0) * 255).astype(np.uint8)
    return Image.fromarray(rgba, "RGBA")


def main(source_path, out_dir):
    sources = json.loads(Path(source_path).read_text(encoding="utf-8"))
    out = Path(out_dir)
    out.mkdir(parents=True, exist_ok=True)
    for key in sorted(sources):
        rasterize(sources[key]).save(out / f"{key}.png", optimize=False)
    print(f"{len(sources)} icons at {SIZE} x {SIZE} into {out}")


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
