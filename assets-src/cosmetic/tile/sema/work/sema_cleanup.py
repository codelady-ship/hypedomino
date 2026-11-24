import argparse
import os

import numpy as np
from PIL import Image
from scipy import ndimage

GEM_CENTRE = (478, 878)
GEM_RADIUS = 128
EMBLEM_CENTRE = (450, 870)
HAZE_INNER = 300
HAZE_OUTER = 360
REFERENCE_BAND = (380, 520)
HAZE_SIGMA = 10
HAZE_KEEP = 0.35
METAL_GROWTH = 9
SKY_FEATHER = 2
STARS = [
    (481, 720, 36, 48),
    (130, 868, 60, 70),
    (759, 868, 60, 70),
    (443, 392, 68, 68),
    (443, 543, 46, 66),
    (443, 1140, 68, 62),
    (443, 1320, 50, 72),
]
STAR_SHAPE_POWER = 0.75
STAR_FEATHER = 2
SPARKLE_SIGMA = 2
SPARKLE_THRESHOLD = 0.04


def hsv_planes(image):
    hsv = np.asarray(image.convert("HSV")).astype(np.float64)
    return hsv[..., 0] * 360 / 255, hsv[..., 1] / 255, hsv[..., 2] / 255


def gold_mask(image):
    hue, saturation, value = hsv_planes(image)
    gold = (hue > 15) & (hue < 60) & (saturation > 0.35) & (value > 0.35)
    highlight = (saturation < 0.35) & (value > 0.7)
    return ndimage.maximum_filter(gold | highlight, size=METAL_GROWTH)


def blur(plane, sigma):
    return ndimage.gaussian_filter(plane, sigma)


def distance_from(shape, centre):
    ys, xs = np.mgrid[0:shape[0], 0:shape[1]]
    return np.hypot(xs - centre[0], ys - centre[1])


def sky_mask(image, gold):
    return ~gold & (distance_from(gold.shape, GEM_CENTRE) > GEM_RADIUS)


def remove_haze(pixels, sky):
    luminance = pixels @ np.array([0.2126, 0.7152, 0.0722])
    weight = sky.astype(np.float64)
    low = blur(luminance * weight, HAZE_SIGMA) / np.maximum(blur(weight, HAZE_SIGMA), 1e-6)
    radius = distance_from(sky.shape, EMBLEM_CENTRE)
    band = sky & (radius > REFERENCE_BAND[0]) & (radius < REFERENCE_BAND[1])
    reference = np.median(low[band])
    excess = np.maximum(low - reference, 0)
    target = low - excess * (1 - HAZE_KEEP)
    zone = np.clip((HAZE_OUTER - radius) / (HAZE_OUTER - HAZE_INNER), 0, 1)
    soft = np.clip(blur(weight, SKY_FEATHER), 0, 1) * weight
    factor = 1 + (target / np.maximum(low, 1e-6) - 1) * zone * soft
    return np.clip(pixels * factor[..., None], 0, 255), reference


def star_alpha(shape):
    ys, xs = np.mgrid[0:shape[0], 0:shape[1]]
    alpha = np.zeros(shape)
    for x, y, half_width, half_height in STARS:
        reach = (np.abs(xs - x) / half_width) ** STAR_SHAPE_POWER + (np.abs(ys - y) / half_height) ** STAR_SHAPE_POWER
        alpha = np.maximum(alpha, reach <= 1)
    return np.clip(blur(alpha, STAR_FEATHER), 0, 1)


def sparkle_alpha(pixels, sky):
    luminance = pixels @ np.array([0.2126, 0.7152, 0.0722]) / 255
    detail = luminance - blur(luminance, SPARKLE_SIGMA)
    return np.clip((detail - SPARKLE_THRESHOLD) / SPARKLE_THRESHOLD, 0, 1) * sky


def save_layer(pixels, alpha, path):
    rgba = np.dstack([pixels, alpha * 255]).round().astype(np.uint8)
    Image.fromarray(rgba, mode="RGBA").save(path)


def main():
    parser = argparse.ArgumentParser(prog="sema_cleanup.py")
    parser.add_argument("concept")
    parser.add_argument("work")
    arguments = parser.parse_args()
    concept = Image.open(arguments.concept).convert("RGB")
    gold = gold_mask(concept)
    sky = sky_mask(concept, gold)
    pixels, reference = remove_haze(np.asarray(concept).astype(np.float64), sky)
    master = pixels.round().astype(np.uint8)
    Image.fromarray(master, mode="RGB").save(os.path.join(arguments.work, "sema_master.png"))
    save_layer(master, star_alpha(sky.shape), os.path.join(arguments.work, "sema_stars.png"))
    save_layer(master, sparkle_alpha(pixels, sky), os.path.join(arguments.work, "sema_sparkles.png"))
    print(f"sema_cleanup: sky reference luminance {reference:.2f}")


main()
