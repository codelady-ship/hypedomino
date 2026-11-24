import argparse
import os

import numpy as np
from PIL import Image
from scipy import ndimage

DIFFERENCE = 0.12
BODY_DIFFERENCE = 0.2
OPENING = 4
WING_SHARE = 0.2
BODY_CLOSING = 12
CLOSING = 6
GROWTH = 3
EDGE_SIGMA = 0.8
SPECK_MIN_PIXELS = 2000
PADDING = 14


def load(path):
    return np.asarray(Image.open(path).convert("RGBA")).astype(np.float64) / 255


def premultiplied(image):
    return np.dstack([image[..., :3] * image[..., 3:], image[..., 3]])


def butterfly_boxes(master, plate):
    changed = np.abs(premultiplied(master) - premultiplied(plate)).max(axis=2) > DIFFERENCE
    changed = ndimage.binary_fill_holes(ndimage.binary_closing(ndimage.binary_opening(changed, iterations=2), iterations=CLOSING))
    labels, count = ndimage.label(changed)
    sizes = ndimage.sum(changed, labels, range(1, count + 1))
    boxes = [found for found, size in zip(ndimage.find_objects(labels), sizes) if size >= SPECK_MIN_PIXELS]
    return [(max(rows.start - PADDING, 0), rows.stop + PADDING, max(columns.start - PADDING, 0), columns.stop + PADDING) for rows, columns in boxes]


def butterfly_mask(master, plate, boxes):
    changed = np.abs(premultiplied(master) - premultiplied(plate)).max(axis=2) > BODY_DIFFERENCE
    wings = np.zeros(changed.shape, dtype=bool)
    for top, bottom, left, right in boxes:
        local = ndimage.binary_opening(changed[top:bottom, left:right], iterations=OPENING)
        labels, count = ndimage.label(local)
        if count == 0:
            continue
        sizes = ndimage.sum(local, labels, range(1, count + 1))
        parts = np.isin(labels, np.where(sizes >= WING_SHARE * sizes.max())[0] + 1)
        body = ndimage.binary_fill_holes(ndimage.binary_closing(parts, iterations=BODY_CLOSING))
        wings[top:bottom, left:right] |= ndimage.binary_dilation(body, iterations=OPENING + GROWTH)
    wings &= master[..., 3] > 0.02
    return np.clip(ndimage.gaussian_filter(wings.astype(np.float64), EDGE_SIGMA), 0, 1)


def straight(pixels):
    alpha = pixels[..., 3:]
    colour = np.where(alpha > 0, pixels[..., :3] / np.maximum(alpha, 1e-6), 0)
    return Image.fromarray((np.dstack([colour, alpha]).clip(0, 1) * 255).round().astype(np.uint8))


def main():
    parser = argparse.ArgumentParser(prog="opal_wings_parts.py")
    parser.add_argument("work")
    arguments = parser.parse_args()
    master = load(os.path.join(arguments.work, "opal_wings_master.png"))
    plate = load(os.path.join(arguments.work, "opal_wings_plate_master.png"))
    boxes = butterfly_boxes(master, plate)
    weight = butterfly_mask(master, plate, boxes)[..., None]
    straight(premultiplied(master) * weight).save(os.path.join(arguments.work, "opal_wings_butterflies.png"))
    straight(premultiplied(master) * (1 - weight) + premultiplied(plate) * weight).save(os.path.join(arguments.work, "opal_wings_under.png"))
    print(f"opal_wings_parts: {len(boxes)} butterflies, boxes {boxes}")


main()
