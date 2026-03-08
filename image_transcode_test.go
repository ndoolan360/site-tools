package sitetools

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/gif"
	"strings"
	"testing"

	genavif "github.com/gen2brain/avif"
	genwebp "github.com/gen2brain/webp"
)

// ---------- Test Assets ----------

//go:embed test_assets/images/rgb.png
var pngRGB []byte

//go:embed test_assets/images/rgba.png
var pngRGBA []byte

//go:embed test_assets/images/rgb.jpg
var jpgRGB []byte

//go:embed test_assets/images/rgb.gif
var gifRGB []byte

//go:embed test_assets/images/rgba.gif
var gifRGBA []byte

//go:embed test_assets/images/rgb_frames.gif
var gifAnimRGB []byte

//go:embed test_assets/images/rgba_frames.gif
var gifAnimRGBA []byte

//go:embed test_assets/images/rgb.webp
var webpRGB []byte

//go:embed test_assets/images/rgba.webp
var webpRGBA []byte

//go:embed test_assets/images/rgb_frames.webp
var webpAnimRGB []byte

//go:embed test_assets/images/rgba_frames.webp
var webpAnimRGBA []byte

//go:embed test_assets/images/rgb.avif
var avifRGB []byte

//go:embed test_assets/images/rgba.avif
var avifRGBA []byte

//go:embed test_assets/images/rgb_frames.avif
var avifAnimRGB []byte

//go:embed test_assets/images/rgba_frames.avif
var avifAnimRGBA []byte

// ---------- Test Asset Registry ----------

type testImage struct {
	name     string
	data     []byte
	ext      string
	format   string
	alpha    bool
	animated bool
}

var allTestImages = []testImage{
	// --- PNG ---
	{"pngRGB", pngRGB, ".png", ImageFormatPNG, false, false},
	{"pngRGBA", pngRGBA, ".png", ImageFormatPNG, true, false},

	// --- JPEG ---
	{"jpgRGB", jpgRGB, ".jpg", ImageFormatJPEG, false, false},

	// --- GIF ---
	{"gifRGB", gifRGB, ".gif", ImageFormatGIF, false, false},
	{"gifRGBA", gifRGBA, ".gif", ImageFormatGIF, true, false},
	{"gifAnimRGB", gifAnimRGB, ".gif", ImageFormatGIF, false, true},
	{"gifAnimRGBA", gifAnimRGBA, ".gif", ImageFormatGIF, true, true},

	// --- WEBP ---
	{"webpRGB", webpRGB, ".webp", ImageFormatWEBP, false, false},
	{"webpRGBA", webpRGBA, ".webp", ImageFormatWEBP, true, false},
	{"webpAnimRGB", webpAnimRGB, ".webp", ImageFormatWEBP, false, true},
	{"webpAnimRGBA", webpAnimRGBA, ".webp", ImageFormatWEBP, true, true},

	// --- AVIF ---
	{"avifRGB", avifRGB, ".avif", ImageFormatAVIF, false, false},
	{"avifRGBA", avifRGBA, ".avif", ImageFormatAVIF, true, false},
	{"avifAnimRGB", avifAnimRGB, ".avif", ImageFormatAVIF, false, true},
	{"avifAnimRGBA", avifAnimRGBA, ".avif", ImageFormatAVIF, true, true},
}

func staticImages() []testImage {
	var out []testImage
	for _, img := range allTestImages {
		if !img.animated {
			out = append(out, img)
		}
	}
	return out
}

func animatedImages() []testImage {
	var out []testImage
	for _, img := range allTestImages {
		if img.animated {
			out = append(out, img)
		}
	}
	return out
}

// ---------- Helpers ----------

func decodeAny(t *testing.T, data []byte) []image.Image {
	t.Helper()

	if isWebP(data) {
		w, err := genwebp.DecodeAll(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("failed to decode WebP: %v", err)
		}
		return w.Image
	}

	if isGIF(data) {
		g, err := gif.DecodeAll(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("failed to decode GIF: %v", err)
		}
		frames := make([]image.Image, len(g.Image))
		for i, f := range g.Image {
			frames[i] = f
		}
		return frames
	}

	if isAVIF(data) {
		a, err := genavif.DecodeAll(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("failed to decode AVIF: %v", err)
		}
		return a.Image
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to decode image: %v", err)
	}
	return []image.Image{img}
}

func assertPathExtension(t *testing.T, path string, wantExt string) {
	t.Helper()
	if !strings.HasSuffix(path, wantExt) {
		t.Errorf("expected path to end with %q, got %q", wantExt, path)
	}
}

func assertDimensions(t *testing.T, img image.Image, source image.Image) {
	t.Helper()
	got := img.Bounds().Size()
	want := source.Bounds().Size()
	if got != want {
		t.Errorf("dimensions mismatch: got %v, want %v", got, want)
	}
}

func assertDecodableAny(t *testing.T, data []byte, animated bool) image.Image {
	t.Helper()

	imageList := decodeAny(t, data)
	if len(imageList) == 0 {
		t.Fatal("expected at least one decoded image, got zero")
	}

	if animated && len(imageList) < 2 {
		t.Fatalf("expected animated image with multiple frames, got %d", len(imageList))
	}

	return imageList[0] // Return first frame for dimension checks, etc.
}

func isGIF(data []byte) bool {
	return len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a")
}

func isWebP(data []byte) bool {
	return len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
}

func isAVIF(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	brand := string(data[8:12])
	return string(data[4:8]) == "ftyp" && (brand == "avif" || brand == "avis")
}

// ---------- Static → All Formats ----------

func TestTranscode_StaticToAllFormats(t *testing.T) {
	targets := []struct {
		format string
		ext    string
	}{
		{ImageFormatPNG, ".png"},
		{ImageFormatJPEG, ".jpg"},
		{ImageFormatGIF, ".gif"},
		{ImageFormatWEBP, ".webp"},
		{ImageFormatAVIF, ".avif"},
	}

	for _, src := range staticImages() {
		for _, tgt := range targets {
			name := fmt.Sprintf("%s_to_%s", src.name, tgt.format)
			t.Run(name, func(t *testing.T) {
				asset := &Asset{
					Data: append([]byte(nil), src.data...),
					Path: "img/test" + src.ext,
				}

				err := (ImageTranscoder{ToFormat: tgt.format, Quality: 100}).Transform(asset)
				if err != nil {
					t.Fatalf("Transform() error: %v", err)
				}

				assertPathExtension(t, asset.Path, tgt.ext)

				srcImg := decodeAny(t, src.data)[0]
				outImg := assertDecodableAny(t, asset.Data, src.animated)
				assertDimensions(t, outImg, srcImg)
			})
		}
	}
}

// ---------- Animated → Animated Formats ----------

func TestTranscode_AnimatedToAnimatedFormats(t *testing.T) {
	targets := []struct {
		format string
		ext    string
	}{
		{ImageFormatGIF, ".gif"},
		{ImageFormatWEBP, ".webp"},
	}

	for _, src := range animatedImages() {
		for _, tgt := range targets {
			name := fmt.Sprintf("%s_to_%s", src.name, tgt.format)
			t.Run(name, func(t *testing.T) {
				asset := &Asset{
					Data: append([]byte(nil), src.data...),
					Path: "img/test" + src.ext,
				}

				err := (ImageTranscoder{
					ToFormat:         tgt.format,
					Quality:          100,
					PreserveAnimated: true,
				}).Transform(asset)
				if err != nil {
					t.Fatalf("Transform() error: %v", err)
				}

				assertPathExtension(t, asset.Path, tgt.ext)
				// Output should be decodable at minimum
				assertDecodableAny(t, asset.Data, src.animated)
			})
		}
	}
}

// ---------- Animated → Static (first frame) ----------

func TestTranscode_AnimatedToStatic_FirstFrame(t *testing.T) {
	targets := []struct {
		format string
		ext    string
	}{
		{ImageFormatPNG, ".png"},
		{ImageFormatJPEG, ".jpg"},
	}

	for _, src := range animatedImages() {
		for _, tgt := range targets {
			name := fmt.Sprintf("%s_to_%s", src.name, tgt.format)
			t.Run(name, func(t *testing.T) {
				asset := &Asset{
					Data: append([]byte(nil), src.data...),
					Path: "img/test" + src.ext,
				}

				err := (ImageTranscoder{
					ToFormat:         tgt.format,
					Quality:          100,
					PreserveAnimated: false,
				}).Transform(asset)
				if err != nil {
					t.Fatalf("Transform() error: %v", err)
				}

				assertPathExtension(t, asset.Path, tgt.ext)
				assertDecodableAny(t, asset.Data, false) // Output should be static
			})
		}
	}
}

// ---------- Animated → Static (PreserveAnimated = error) ----------

func TestTranscode_AnimatedToStatic_PreserveError(t *testing.T) {
	targets := []string{ImageFormatPNG, ImageFormatJPEG, ImageFormatAVIF}

	for _, src := range animatedImages() {
		for _, tgt := range targets {
			name := fmt.Sprintf("%s_to_%s_preserved", src.name, tgt)
			t.Run(name, func(t *testing.T) {
				asset := &Asset{
					Data: append([]byte(nil), src.data...),
					Path: "img/test" + src.ext,
				}

				err := (ImageTranscoder{
					ToFormat:         tgt,
					PreserveAnimated: true,
				}).Transform(asset)
				if err == nil {
					t.Fatal("expected error when transcoding animated image to non-animated format with PreserveAnimated=true")
				}
				if !strings.Contains(err.Error(), "cannot transcode animated image") {
					t.Fatalf("unexpected error message: %v", err)
				}
			})
		}
	}
}

// ---------- Identity Transcode ----------

var oneWayAnimatedFormats = map[string]bool{
	ImageFormatAVIF: true, // AVIF doesn't support encoding animations yet
}

func TestTranscode_IdentityRoundTrip(t *testing.T) {
	for _, src := range allTestImages {
		if oneWayAnimatedFormats[src.format] && src.animated {
			continue
		}

		name := fmt.Sprintf("%s_to_same", src.name)
		t.Run(name, func(t *testing.T) {
			asset := &Asset{
				Data: append([]byte(nil), src.data...),
				Path: "img/test" + src.ext,
			}

			err := (ImageTranscoder{
				ToFormat:         src.format,
				Quality:          100,
				PreserveAnimated: true,
			}).Transform(asset)
			if err != nil {
				t.Fatalf("Transform() error: %v", err)
			}

			assertPathExtension(t, asset.Path, src.ext)
			img := assertDecodableAny(t, asset.Data, src.animated)
			size := img.Bounds().Size()
			srcSize := decodeAny(t, src.data)[0].Bounds().Size()
			if size != srcSize {
				t.Errorf("size mismatch: got %v, want %v", size, srcSize)
			}
		})
	}
}

// ---------- Edge Cases ----------

func TestTranscode_EmptyToFormat(t *testing.T) {
	asset := &Asset{Data: pngRGB, Path: "test.png"}
	err := (ImageTranscoder{}).Transform(asset)
	if err == nil {
		t.Fatal("expected error for empty ToFormat")
	}
}

func TestTranscode_InvalidImageData(t *testing.T) {
	asset := &Asset{Data: []byte("not an image"), Path: "test.png"}
	err := (ImageTranscoder{ToFormat: ImageFormatPNG}).Transform(asset)
	if err == nil {
		t.Fatal("expected error for invalid image data")
	}
}

func TestTranscode_UnsupportedFormat(t *testing.T) {
	asset := &Asset{Data: pngRGB, Path: "test.png"}
	err := (ImageTranscoder{ToFormat: "image/tiff"}).Transform(asset)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestTranscode_QualityClamping(t *testing.T) {
	tests := []struct {
		name    string
		quality int
	}{
		{"negative", -10},
		{"zero", 0},
		{"max", 100},
		{"over_max", 200},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			asset := &Asset{
				Data: append([]byte(nil), jpgRGB...),
				Path: "test.jpg",
			}
			err := (ImageTranscoder{ToFormat: ImageFormatJPEG, Quality: tc.quality}).Transform(asset)
			if err != nil {
				t.Fatalf("Transform() error: %v", err)
			}
			assertDecodableAny(t, asset.Data, false)
		})
	}
}
