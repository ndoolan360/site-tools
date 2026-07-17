#!/usr/bin/env bash
#
# Generates sample test images for the image transcoding test suite.
#
# Requirements:
#   - ImageMagick 7+  (brew install imagemagick)
#   - libwebp tools    (brew install webp)
#
# Usage:
#   cd test_assets/images && bash generate.sh
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

SIZE="4x4"
ALT_SIZE="2x2"

# ── Temporary PNGs used as intermediate sources ──────────────────────────

# RGB: solid red (no alpha channel)
magick -size "$SIZE" xc:red rgb_temp.png

# RGBA: checkerboard of red and transparent (alpha channel present)
magick -size 2x2 xc:red -size 2x2 xc:none +append \
  \( -size 2x2 xc:none -size 2x2 xc:red +append \) -append rgba_temp.png

# Second frames for animations (different colour so frames are distinct)
magick -size "$SIZE" xc:blue rgb_temp2.png
magick -size 2x2 xc:blue -size 2x2 xc:none +append \
  \( -size 2x2 xc:none -size 2x2 xc:blue +append \) -append rgba_temp2.png

# ── Static images ───────────────────────────────────────────────────────

# PNG
cp rgb_temp.png  rgb.png
cp rgba_temp.png rgba.png

# JPEG (RGB only — JPEG does not support alpha)
magick rgb_temp.png -quality 95 rgb.jpg

# GIF — static, opaque
magick rgb_temp.png rgb.gif

# GIF — static, with transparency
magick rgba_temp.png rgba.gif

# WebP — static, opaque
cwebp -lossless rgb_temp.png -o rgb.webp -quiet

# WebP — static, with alpha
cwebp -lossless rgba_temp.png -o rgba.webp -quiet

# AVIF — static
avifenc --lossless rgb_temp.png rgb.avif 2>/dev/null
avifenc --lossless rgba_temp.png rgba.avif 2>/dev/null

# ── Animated GIF ────────────────────────────────────────────────────────

# Animated GIF — RGB (2 frames, 100 ms delay)
magick -delay 10 -loop 0 rgb_temp.png rgb_temp2.png rgb_frames.gif

# Animated GIF — RGBA (2 frames, 100 ms delay)
magick -delay 10 -loop 0 rgba_temp.png rgba_temp2.png rgba_frames.gif

# ── Animated WebP ───────────────────────────────────────────────────────

cwebp -lossless rgb_temp.png  -o rgb_f1.webp  -quiet
cwebp -lossless rgb_temp2.png -o rgb_f2.webp  -quiet
cwebp -lossless rgba_temp.png  -o rgba_f1.webp -quiet
cwebp -lossless rgba_temp2.png -o rgba_f2.webp -quiet

# Animated WebP — RGB (2 frames, 100 ms delay each)
img2webp -loop 0 -d 100 rgb_f1.webp -d 100 rgb_f2.webp -o rgb_frames.webp

# Animated WebP — RGBA (2 frames, 100 ms delay each)
img2webp -loop 0 -d 100 rgba_f1.webp -d 100 rgba_f2.webp -o rgba_frames.webp

# ── Animated AVIF ───────────────────────────────────────────────────────

# avifenc accepts multiple PNGs as sequential frames with --duration

# Animated AVIF — RGB (2 frames, 100 ms delay each)
avifenc --lossless --fps 10 --duration 1 rgb_temp.png --duration 1 rgb_temp2.png \
  rgb_frames.avif 2>/dev/null

# Animated AVIF — RGBA (2 frames, 100 ms delay each)
avifenc --lossless --fps 10 --duration 1 rgba_temp.png --duration 1 rgba_temp2.png \
  rgba_frames.avif 2>/dev/null

# ── Cleanup temporaries ─────────────────────────────────────────────────

rm -f rgb_temp.png rgba_temp.png rgb_temp2.png rgba_temp2.png
rm -f rgb_f1.webp rgb_f2.webp rgba_f1.webp rgba_f2.webp

# ── Summary ──────────────────────────────────────────────────────────────

echo "Generated test images:"
ls -lh *.avif *.gif *.jpg *.png *.webp
