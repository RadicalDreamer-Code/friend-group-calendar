package avatar

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/gen2brain/webp"
)

func TestProcessMakesSquareWebP(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1200, 800))
	for y := range 800 {
		for x := range 1200 {
			src.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var in bytes.Buffer
	if err := png.Encode(&in, src); err != nil {
		t.Fatal(err)
	}

	out, err := Process(in.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := webp.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("output is not WebP: %v", err)
	}
	if cfg.Width != Size || cfg.Height != Size {
		t.Fatalf("output is %dx%d, want %dx%d", cfg.Width, cfg.Height, Size, Size)
	}
}

func TestProcessRejectsNonImages(t *testing.T) {
	if _, err := Process([]byte("<svg xmlns='http://www.w3.org/2000/svg'/>")); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}
