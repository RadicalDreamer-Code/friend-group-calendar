package avatar

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

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

	out, err := NewProcessor().Process(context.Background(), in.Bytes())
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
	if _, err := NewProcessor().Process(context.Background(), []byte("<svg xmlns='http://www.w3.org/2000/svg'/>")); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

func TestDecodeCostKeepsLargeImagesWithinBudget(t *testing.T) {
	const w, h = 7745, 5164 // just under maxPixels
	for model, fits := range map[color.Model]bool{
		color.YCbCrModel:   true,  // 40 MP JPEG
		color.NRGBAModel:   true,  // 40 MP PNG
		color.NRGBA64Model: false, // 40 MP 16-bit PNG
	} {
		cost := decodeCost(image.Config{ColorModel: model, Width: w, Height: h})
		if got := cost <= MemoryBudget; got != fits {
			t.Errorf("%T: cost %d MB, fits in the budget = %v, want %v", model, cost>>20, got, fits)
		}
	}
}

func TestProcessRejectsImagesOverTheMemoryBudget(t *testing.T) {
	// 16-bit, 6000 x 6000: tiny as a PNG, about 430 MB once decoded.
	var in bytes.Buffer
	if err := png.Encode(&in, image.NewNRGBA64(image.Rect(0, 0, 6000, 6000))); err != nil {
		t.Fatal(err)
	}
	if _, err := NewProcessor().Process(context.Background(), in.Bytes()); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

func TestProcessWaitsForMemory(t *testing.T) {
	p := NewProcessor()
	if err := p.mem.Acquire(context.Background(), MemoryBudget); err != nil {
		t.Fatal(err)
	}
	var in bytes.Buffer
	if err := png.Encode(&in, image.NewRGBA(image.Rect(0, 0, 64, 64))); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := p.Process(ctx, in.Bytes()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("with the budget used up: err = %v, want DeadlineExceeded", err)
	}
	p.mem.Release(MemoryBudget)
	if _, err := p.Process(context.Background(), in.Bytes()); err != nil {
		t.Fatalf("after the budget is free: %v", err)
	}
}
