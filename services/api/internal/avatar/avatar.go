// Package avatar turns an uploaded photo into the square WebP the app shows.
package avatar

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif" // register decoders for image.DecodeConfig
	_ "image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"
)

const (
	// Size is the edge length of the stored square avatar.
	Size = 512
	// MaxUploadBytes bounds the upload before it is decoded.
	MaxUploadBytes = 10 << 20
	// maxPixels bounds the decoded image (about 160 MB as RGBA) so a small
	// file with huge dimensions cannot exhaust memory.
	maxPixels   = 40_000_000
	webpQuality = 82
)

var ErrUnsupported = errors.New("unsupported image: upload a JPEG, PNG, WebP or GIF")
var ErrTooLarge = errors.New("image dimensions are too large")

// Process center-crops the image to a square, applies its EXIF orientation,
// resizes it to Size and encodes it as WebP. Re-encoding also strips all
// metadata, such as the GPS position phones write into photos.
func Process(data []byte) ([]byte, error) {
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
