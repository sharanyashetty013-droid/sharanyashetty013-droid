"""Recolour the git-koi GIF into the "neon violet" look and give it a dark pond.

Usage: python scripts/recolor_koi.py koi/koi.gif

git-koi draws the koi on a transparent background in one of its six bodies.
This rotates every colour's hue, boosts its saturation, and puts the fish on a
deep purple pond, then overwrites the GIF in place.
"""

import colorsys
import sys

import numpy as np
from PIL import Image

HUE_SHIFT = 0.78  # fraction of the colour wheel to rotate by
SATURATION = 1.6  # multiplier, capped at full saturation
POND = (0x12, 0x0E, 0x26)  # background colour, #120E26


def recolor(rgb):
    r, g, b = (c / 255 for c in rgb)
    h, l, s = colorsys.rgb_to_hls(r, g, b)
    r, g, b = colorsys.hls_to_rgb((h + HUE_SHIFT) % 1, l, min(1.0, s * SATURATION))
    return round(r * 255), round(g * 255), round(b * 255)


def main(path):
    src = Image.open(path)
    cache = {}
    frames, durations = [], []

    for i in range(getattr(src, "n_frames", 1)):
        src.seek(i)
        durations.append(src.info.get("duration", 100))
        rgba = np.array(src.convert("RGBA"))

        out = np.empty(rgba.shape[:2] + (3,), dtype=np.uint8)
        out[:] = POND

        opaque = rgba[..., 3] > 0
        colours, inverse = np.unique(rgba[opaque][:, :3], axis=0, return_inverse=True)
        mapped = np.array(
            [cache.setdefault(tuple(c), recolor(c)) for c in colours.tolist()],
            dtype=np.uint8,
        )
        out[opaque] = mapped[inverse.reshape(-1)]
        frames.append(Image.fromarray(out, "RGB"))

    frames[0].save(
        path,
        save_all=True,
        append_images=frames[1:],
        duration=durations,
        loop=src.info.get("loop", 0),
        disposal=1,
        optimize=False,
    )


if __name__ == "__main__":
    main(sys.argv[1])
