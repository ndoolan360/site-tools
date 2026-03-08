package sitetools

import (
	"bytes"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"path"
	"strings"

	"github.com/HugoSmits86/nativewebp"
	genavif "github.com/gen2brain/avif"
	genwebp "github.com/gen2brain/webp"
)

type ImageTranscoder struct {
	// ToFormat is the desired output format mimetype (e.g. "image/jpeg", "image/png", "image/webp").
	ToFormat string
	// Quality is the desired output quality [1-100] for lossy formats like JPEG and WebP.
	// For WebP, quality of 100 produces lossless output via nativewebp.
	// For values below 100, lossy encoding is used via gen2brain/webp.
	// Ignored for lossless-only formats like PNG.
	Quality int
	// PreserveAnimated controls behavior when transcoding animated input to a non-animated output:
	// - true: return an error
	// - false: use first frame
	PreserveAnimated bool
}

func (it ImageTranscoder) Transform(asset *Asset) error {
	if it.ToFormat == "" {
		return fmt.Errorf("ToFormat is required")
	}

	inFormat, err := identifyFormat(asset.Data)
	if err != nil {
		return fmt.Errorf("failed to identify input image format: %w", err)
	}

	reader := bytes.NewReader(asset.Data)
	frames, err := inFormat.decode(reader)
	if err != nil {
		return fmt.Errorf("failed to decode input image: %w", err)
	}

	if err := frames.validate(); err != nil {
		return fmt.Errorf("failed to validate decoded frames: %w", err)
	}

	outFormat, err := it.newFormat()
	if err != nil {
		return err
	}

	if frames.isAnimated() && !outFormat.supportsAnimation() {
		if it.PreserveAnimated {
			return fmt.Errorf("cannot transcode animated image to non-animated format: %s", it.ToFormat)
		}
		frames = frames.firstFrame()
	}

	var data bytes.Buffer
	if err := outFormat.encode(&data, frames); err != nil {
		return fmt.Errorf("failed to encode image to %s: %w", it.ToFormat, err)
	}

	pathExt := path.Ext(asset.Path)
	pathWithoutExt := strings.TrimSuffix(asset.Path, pathExt)

	asset.Data = data.Bytes()
	asset.Path = pathWithoutExt + outFormat.extension()

	return nil
}

// ---------- Frame Representation ----------

// disposal represents GIF-style per-frame disposal methods
const (
	__                 byte = iota
	disposalNone            // keep frame
	disposalBackground      // clear to background
	disposalPrevious        // restore to previous
)

type imageFrames struct {
	// For static images, this contains exactly one frame.
	frames []image.Image
	// Per-frame delay in milliseconds. Length matches frames.
	delay []int
	// Animation loop count. 0 means infinite.
	loopCount int
	// Per-frame GIF-style disposal values. Length matches frames.
	disposal []byte
}

func (f imageFrames) isAnimated() bool {
	return len(f.frames) > 1
}

func (f imageFrames) firstFrame() imageFrames {
	return imageFrames{
		frames:   f.frames[:1],
		delay:    f.delay[:1],
		disposal: f.disposal[:1],
	}
}

func (f imageFrames) validate() error {
	if len(f.frames) == 0 {
		return fmt.Errorf("no frames decoded")
	}
	if len(f.delay) != len(f.frames) {
		return fmt.Errorf("frame metadata mismatch: %d frames, %d delays", len(f.frames), len(f.delay))
	}
	if len(f.disposal) != len(f.frames) {
		return fmt.Errorf("frame metadata mismatch: %d frames, %d disposal entries", len(f.frames), len(f.disposal))
	}
	return nil
}

// ---------- Decoder ----------

type imageDecoder interface {
	identify(data []byte) bool
	decode(r *bytes.Reader) (imageFrames, error)
}

func identifyFormat(data []byte) (imageDecoder, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	candidates := []imageDecoder{
		pngFormat{},
		jpegFormat{},
		gifFormat{},
		webpFormat{},
		avifFormat{},
	}

	for _, f := range candidates {
		if f.identify(data) {
			return f, nil
		}
	}

	return nil, fmt.Errorf("unknown image format")
}

func standardDecoder(reader *bytes.Reader) (imageFrames, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return imageFrames{}, err
	}

	return imageFrames{
		frames:    []image.Image{img},
		delay:     []int{0},
		loopCount: 0,
		disposal:  []byte{disposalNone},
	}, nil
}

// ---------- Encoder ----------

type imageEncoder interface {
	encode(w *bytes.Buffer, frames imageFrames) error
	extension() string
	supportsAnimation() bool
}

func (it ImageTranscoder) newFormat() (imageEncoder, error) {
	quality := it.Quality
	if quality < 0 {
		quality = 0
	} else if quality > 100 {
		quality = 100
	}

	switch it.ToFormat {
	case ImageFormatPNG:
		return pngFormat{}, nil
	case ImageFormatJPEG:
		return jpegFormat{quality}, nil
	case ImageFormatGIF:
		return gifFormat{}, nil
	case ImageFormatWEBP:
		return webpFormat{quality}, nil
	case ImageFormatAVIF:
		return avifFormat{quality}, nil
	default:
		return nil, fmt.Errorf("unsupported 'ToFormat': %s", it.ToFormat)
	}
}

// ---------- Format Implementations ----------

const (
	ImageFormatJPEG = "image/jpeg"
	ImageFormatPNG  = "image/png"
	ImageFormatGIF  = "image/gif"
	ImageFormatWEBP = "image/webp"
	ImageFormatAVIF = "image/avif"
)

// ---------- PNG ----------

type pngFormat struct{}

func (pngFormat) identify(data []byte) bool {
	return len(data) >= 8 &&
		data[0] == 0x89 &&
		string(data[1:4]) == "PNG" &&
		data[4] == 0x0D &&
		data[5] == 0x0A &&
		data[6] == 0x1A &&
		data[7] == 0x0A
}

func (pngFormat) decode(r *bytes.Reader) (imageFrames, error) {
	return standardDecoder(r)
}

func (pngFormat) encode(w *bytes.Buffer, frames imageFrames) error {
	return png.Encode(w, frames.frames[0])
}

func (pngFormat) extension() string {
	return ".png"
}

func (pngFormat) supportsAnimation() bool {
	return false
}

// ---------- JPEG ----------

type jpegFormat struct {
	quality int
}

func (jpegFormat) identify(data []byte) bool {
	return len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8
}

func (jpegFormat) decode(r *bytes.Reader) (imageFrames, error) {
	return standardDecoder(r)
}

func (e jpegFormat) encode(w *bytes.Buffer, frames imageFrames) error {
	return jpeg.Encode(w, frames.frames[0], &jpeg.Options{Quality: e.quality})
}

func (jpegFormat) extension() string {
	return ".jpg"
}

func (jpegFormat) supportsAnimation() bool {
	return false
}

// ---------- GIF ----------

type gifFormat struct{}

func (gifFormat) identify(data []byte) bool {
	if len(data) < 6 {
		return false
	}
	h := string(data[:6])
	return h == "GIF87a" || h == "GIF89a"
}

func (gifFormat) decode(r *bytes.Reader) (imageFrames, error) {
	g, err := gif.DecodeAll(r)
	if err != nil {
		return imageFrames{}, err
	}

	frames := make([]image.Image, len(g.Image))
	delay := make([]int, len(g.Image))
	for i := range g.Image {
		frames[i] = g.Image[i]
		// GIF delay is in centiseconds. Convert to milliseconds.
		delay[i] = g.Delay[i] * 10
	}

	return imageFrames{
		frames:    frames,
		delay:     delay,
		loopCount: g.LoopCount,
		disposal:  g.Disposal,
	}, nil
}

func (gifFormat) encode(w *bytes.Buffer, frames imageFrames) error {
	palettedFrames := make([]*image.Paletted, len(frames.frames))
	for i, frame := range frames.frames {
		palettedFrames[i] = toPaletted(frame)
	}

	delayCS := make([]int, len(frames.delay))
	for i := range frames.delay {
		// milliseconds to centiseconds for GIF
		delayCS[i] = frames.delay[i] / 10
	}

	g := gif.GIF{
		Image:     palettedFrames,
		Delay:     delayCS,
		LoopCount: frames.loopCount,
		Disposal:  frames.disposal,
	}

	return gif.EncodeAll(w, &g)
}

func (gifFormat) extension() string {
	return ".gif"
}

func (gifFormat) supportsAnimation() bool {
	return true
}

func toPaletted(img image.Image) *image.Paletted {
	if p, ok := img.(*image.Paletted); ok {
		return p
	}
	bounds := img.Bounds()
	p := image.NewPaletted(bounds, palette.WebSafe)
	draw.FloydSteinberg.Draw(p, bounds, img, image.Point{})
	return p
}

// ---------- WEBP ----------

type webpFormat struct {
	// quality controls encoding strategy:
	// - 0 or 100: lossless via nativewebp (default behavior)
	// - 1-99: lossy via gen2brain/webp
	quality int
}

func (webpFormat) identify(data []byte) bool {
	// RIFF....WEBP
	return len(data) >= 12 &&
		string(data[:4]) == "RIFF" &&
		string(data[8:12]) == "WEBP"
}

func (webpFormat) decode(r *bytes.Reader) (imageFrames, error) {
	w, err := genwebp.DecodeAll(r)
	if err != nil {
		return imageFrames{}, err
	}

	disposal := make([]byte, len(w.Image))
	for i := range disposal {
		// gen2brain/webp does not expose disposal info
		disposal[i] = gif.DisposalNone
	}

	return imageFrames{
		frames:    w.Image,
		delay:     w.Delay, // already in milliseconds
		loopCount: 0,       // gen2brain/webp DecodeAll does not expose loop count
		disposal:  disposal,
	}, nil
}

func (e webpFormat) encode(w *bytes.Buffer, frames imageFrames) error {
	if frames.isAnimated() {
		durations := make([]uint, len(frames.frames))
		disposals := make([]uint, len(frames.frames))

		for i := range frames.frames {
			durations[i] = uint(frames.delay[i])
			disposals[i] = gifDisposalToWebP(frames.disposal[i])
		}

		ani := nativewebp.Animation{
			Images:    frames.frames,
			Durations: durations,
			LoopCount: uint16(frames.loopCount),
			Disposals: disposals,
		}

		return nativewebp.EncodeAll(w, &ani, &nativewebp.Options{UseExtendedFormat: false})
	}

	if e.quality > 0 && e.quality < 100 {
		// lossy single-frame via gen2brain/webp
		return genwebp.Encode(w, frames.frames[0], genwebp.Options{Lossless: false, Quality: e.quality})
	}

	// lossless single-frame via nativewebp
	return nativewebp.Encode(w, frames.frames[0], nil)
}

func (webpFormat) extension() string {
	return ".webp"
}

func (webpFormat) supportsAnimation() bool {
	return true
}

func gifDisposalToWebP(d byte) uint {
	switch d {
	case gif.DisposalBackground:
		return 1 // clear to background
	case gif.DisposalNone, gif.DisposalPrevious:
		fallthrough
	default:
		return 0 // keep
	}
}

// ---------- AVIF ----------

type avifFormat struct {
	// quality controls encoding quality [0-100]. 0 uses the library default (60).
	quality int
}

func (avifFormat) identify(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	boxType := string(data[4:8])
	if boxType != "ftyp" {
		return false
	}
	majorBrand := string(data[8:12])
	return majorBrand == "avif" || majorBrand == "avis"
}

func (avifFormat) decode(r *bytes.Reader) (imageFrames, error) {
	a, err := genavif.DecodeAll(r)
	if err != nil {
		return imageFrames{}, err
	}

	delay := make([]int, len(a.Image))
	disposal := make([]byte, len(a.Image))
	for i := range a.Image {
		// gen2brain/avif delays are in seconds (float64). Convert to milliseconds.
		delay[i] = int(a.Delay[i] * 1000)
		// gen2brain/avif does not expose disposal info
		disposal[i] = disposalNone
	}

	return imageFrames{
		frames:    a.Image,
		delay:     delay,
		loopCount: 0, // gen2brain/avif DecodeAll does not expose loop count
		disposal:  disposal,
	}, nil
}

func (e avifFormat) encode(w *bytes.Buffer, frames imageFrames) error {
	quality := e.quality
	if quality <= 0 {
		quality = genavif.DefaultQuality
	}

	return genavif.Encode(w, frames.frames[0], genavif.Options{
		Quality: quality,
		Speed:   genavif.DefaultSpeed,
	})
}

func (avifFormat) extension() string {
	return ".avif"
}

// supportsAnimation returns false because animated AVIF encoding
// is not supported by any available Go library.
func (avifFormat) supportsAnimation() bool {
	return false
}
