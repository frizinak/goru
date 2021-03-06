package image

import (
	"image"
	"image/color"

	"github.com/frizinak/goru/data"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

func face(b []byte, size float64) (font.Face, error) {
	col, err := opentype.ParseCollection(b)
	if err != nil {
		return nil, err
	}
	fnt, err := col.Font(0)
	if err != nil {
		return nil, err
	}

	cursiveface, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})

	return cursiveface, err
}

func Image(height int, normal, cursive string, margins bool, fg, bg color.NRGBA) (*image.NRGBA, error) {
	startX := height / 8
	stopX := height / 8
	startY := height / 8
	stopY := height / 8
	padding := height / 8
	if normal == "" || cursive == "" {
		startY /= 2
		stopY /= 2
		padding = 0
	}

	if !margins {
		startY, stopY = 0, 0
	}

	rest := height - startY - padding - stopY
	if rest < 0 {
		rest = 0
	}
	fsize := float64(rest) * 0.4
	if normal == "" || cursive == "" {
		fsize = float64(rest) * 0.75
	}
	img := image.NewNRGBA(image.Rect(0, 0, 0, 0))

	cursiveSize := fsize
	printSize := fsize
	fcursive, err := face(data.FontLobster, cursiveSize)
	if err != nil {
		return nil, err
	}

	fprint, err := face(data.FontOpenSans, printSize)
	if err != nil {
		return nil, err
	}

	fontLSrc := image.NewUniform(fg)
	do := func(startX1, startX2 int) (int, int) {
		dwr := font.Drawer{
			Dst:  img,
			Src:  fontLSrc,
			Face: fprint,
		}

		if normal != "" {
			y := startY + int(printSize)
			if cursive == "" {
				y = height/2 + int(printSize/3)
			}
			dwr.Dot = fixed.P(startX1, y)
			dwr.DrawString(normal)
		}
		width1 := int(dwr.Dot.X>>6) - startX

		if cursive != "" {
			dwr.Face = fcursive
			y := startY + padding + int(cursiveSize) + int(printSize)
			if normal == "" {
				y = height/2 + int(cursiveSize/3)
			}
			dwr.Dot = fixed.P(startX2, y)
			dwr.DrawString(cursive)
		}
		width2 := int(dwr.Dot.X>>6) - startX
		return width1, width2
	}

	w1, w2 := do(startX, startX)
	w := w1 + startX + stopX
	startX1, startX2 := startX, startX+(w1-w2)/2
	if w2 > w {
		w = w2 + startX + stopX
		startX1, startX2 = startX+(w2-w1)/2, startX
	}

	img = image.NewNRGBA(image.Rect(0, 0, w, height))

	if bg.A != 0 {
		for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
			for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
				o := img.PixOffset(x, y)
				img.Pix[o+0] = bg.R
				img.Pix[o+1] = bg.G
				img.Pix[o+2] = bg.B
				img.Pix[o+3] = bg.A
			}
		}
	}
	do(startX1, startX2)

	return img, nil
}
