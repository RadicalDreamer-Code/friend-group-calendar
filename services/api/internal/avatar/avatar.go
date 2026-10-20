// Package avatar turns an uploaded photo into the square WebP the app shows.
package avatar

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif" // register decoders for image.DecodeConfig
	_ "image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"
	"golang.org/x/sync/semaphore"
)

const (
	// Size is the edge length of the stored square avatar.
	Size = 512
	// MaxUploadBytes bounds the upload before it is decoded.
	MaxUploadBytes = 10 << 20
	// MemoryBudget bounds the estimated memory of all decodes in flight, and
	// so of any single one. A file of well under 1 MB can decode to hundreds
	// of MB, so the upload size alone does not bound it.
	MemoryBudget = 320 << 20
	// maxPixels keeps the cost arithmetic far from overflowing.
	maxPixels   = 40_000_000
	webpQuality = 82
)

var ErrUnsupported = errors.New("unsupported image: upload a JPEG, PNG, WebP or GIF")
var ErrTooLarge = errors.New("image dimensions are too large")

// Processor converts uploads while keeping the memory of the decodes in
// flight within MemoryBudget.
type Processor struct {
	mem *semaphore.Weighted
}

func NewProcessor() *Processor {
	return &Processor{mem: semaphore.NewWeighted(MemoryBudget)}
}

// Process center-crops the image to a square, applies its EXIF orientation,
// resizes it to Size and encodes it as WebP. Re-encoding also strips all
// metadata, such as the GPS position phones write into photos. It waits
// until the decode fits in the memory budget.
func (p *Processor) Process(ctx context.Context, data []byte) ([]byte, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrUnsupported
	}
	switch format {
	case "jpeg", "png", "webp", "gif":
	default:
		return nil, ErrUnsupported
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, ErrUnsupported
	}
	if cfg.Width*cfg.Height > maxPixels {
		return nil, ErrTooLarge
	}
	cost := decodeCost(cfg)
	if cost > MemoryBudget {
		return nil, ErrTooLarge
	}
	if err := p.mem.Acquire(ctx, cost); err != nil {
		return nil, err
	}
	defer p.mem.Release(cost)

	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, ErrUnsupported
	}
	square := imaging.Fill(img, Size, Size, imaging.Center, imaging.Lanczos)

	var out bytes.Buffer
	if err := webp.Encode(&out, square, webp.Options{Quality: webpQuality}); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}
	return out.Bytes(), nil
}

// decodeCost estimates the memory of decoding and converting an image: the
// decoded pixels plus an 8-bit RGBA copy for rotating or converting it.
// Measured peaks for 40 MP images stay below it (JPEG 188 MB, PNG 281 MB,
// 16-bit PNG 426 MB, against estimates of 280, 320 and 480 MB).
func decodeCost(cfg image.Config) int64 {
	var bytesPerPixel int64
	switch cfg.ColorModel {
	case color.GrayModel:
		bytesPerPixel = 1
	case color.Gray16Model:
		bytesPerPixel = 2
	case color.YCbCrModel: // JPEG; 4:4:4 sampling is the worst case
		bytesPerPixel = 3
	case color.RGBAModel, color.NRGBAModel, color.CMYKModel:
		bytesPerPixel = 4
	default:
		if _, ok := cfg.ColorModel.(color.Palette); ok {
			bytesPerPixel = 1
		} else {
			bytesPerPixel = 8 // 16-bit RGBA, or anything unknown
		}
	}
	return int64(cfg.Width) * int64(cfg.Height) * (bytesPerPixel + 4)
}
