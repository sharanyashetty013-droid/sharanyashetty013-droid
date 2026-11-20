"""Recolour the git-koi GIF into the "sakura" look and put it in moving water.

Usage: python scripts/recolor_koi.py koi/koi.gif

git-koi draws the koi on a transparent background in one of its six bodies.
This rotates every colour's hue toward pink, boosts its saturation, and draws
an animated pixel-art pond under the fish: rippling light patterns and soft
wave lines that loop seamlessly with the GIF. The GIF is overwritten in place.
Everything git-koi decides (size, patterns, swim path) is kept as drawn.
"""

import colorsys
import math
import sys

import numpy as np
from PIL import Image

# Fish colour
HUE_SHIFT = 0.90  # fraction of the colour wheel to rotate by
SATURATION = 1.3  # multiplier, capped at full saturation

# Water colours, darkest to brightest
WATER = np.array(
    [
        (0x2A, 0x1A, 0x2E),  # deep water
        (0x34, 0x20, 0x3A),  # soft ripple
        (0x42, 0x29, 0x4A),  # light ripple
        (0x5A, 0x38, 0x62),  # bright caustic
    ],
    dtype=np.uint8,
)
PIXEL = 1  # water pixel size, in the koi's native pixels

# Wide pond. git-koi's released Action draws a 256x256 pond (exported at 2x);
# the fish's square is centred in this wider pond so it spans the README.
POND_WIDTH, POND_HEIGHT = 1024, 256
FISH_SCALE = 2  # git-koi's export scale, undone so pond and fish share pixels


def recolor(rgb):
    r, g, b = (c / 255 for c in rgb)
    h, l, s = colorsys.rgb_to_hls(r, g, b)
    r, g, b = colorsys.hls_to_rgb((h + HUE_SHIFT) % 1, l, min(1.0, s * SATURATION))
    return round(r * 255), round(g * 255), round(b * 255)


def water(width, height, frame, frames):
    """One frame of pixel-art water. Whole-number time frequencies make it loop."""
    w, h = width // PIXEL, height // PIXEL
    y, x = np.mgrid[0:h, 0:w].astype(np.float32)
    t = 2 * math.pi * frame / frames

    # Caustics: two interfering, gently warped wave fields.
    a = np.sin(x * 0.11 + np.sin(y * 0.07 + t) * 2.0 + t)
    b = np.sin(y * 0.13 + np.sin(x * 0.05 - t) * 2.0 - 2 * t)
    c = np.sin((x + y) * 0.06 + t)
    field = (a + b + c) / 3  # roughly -1 .. 1

    level = np.zeros((h, w), dtype=np.uint8)
    level[field > 0.25] = 1
    level[field > 0.50] = 2
    level[field > 0.72] = 3

    # Thin drifting wave lines across the surface.
    lines = np.sin(y * 0.45 + np.sin(x * 0.08 + t) * 1.5 - t) > 0.97
    level[lines & (level < 2)] = 2

    img = WATER[level]
    return img.repeat(PIXEL, axis=0).repeat(PIXEL, axis=1)[:height, :width]


def main(path):
    src = Image.open(path)
    frames = getattr(src, "n_frames", 1)
    cache = {}
    out_frames, durations = [], []

    for i in range(frames):
        src.seek(i)
        durations.append(src.info.get("duration", 100))
        # git-koi exports at 2x; drop back to its native pixels.
        rgba = np.array(src.convert("RGBA"))[::FISH_SCALE, ::FISH_SCALE]
        fh, fw = rgba.shape[:2]

        out = water(POND_WIDTH, POND_HEIGHT, i, frames)

        # The fish's square sits in the middle of the wide pond.
        top, left = (POND_HEIGHT - fh) // 2, (POND_WIDTH - fw) // 2
        region = out[top : top + fh, left : left + fw]

        fish = rgba[..., 3] > 0
        colours, inverse = np.unique(rgba[fish][:, :3], axis=0, return_inverse=True)
        mapped = np.array(
            [cache.setdefault(tuple(c), recolor(c)) for c in colours.tolist()],
            dtype=np.uint8,
        )
        region[fish] = mapped[inverse.reshape(-1)]
        out_frames.append(Image.fromarray(out, "RGB"))

    out_frames[0].save(
        path,
        save_all=True,
        append_images=out_frames[1:],
        duration=durations,
        loop=src.info.get("loop", 0),
        disposal=1,
        optimize=False,
    )


if __name__ == "__main__":
    main(sys.argv[1])
