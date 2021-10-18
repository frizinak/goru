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

func Image(height int, word string) (*image.NRGBA, error) {
	startX := 50
	stopX := 50
	startY := 50
	stopY := 50
	padding := 50
	rest := height - startY - padding - stopY
	if rest < 0 {
		rest = 0
	}
	fsize := float64((rest / 3))
	img := image.NewNRGBA(image.Rect(0, 0, 0, 0))

	cursiveSize := fsize
	printSize := fsize
	cursive, err := face(data.FontLobster, cursiveSize)
	if err != nil {
		return nil, err
	}

	print, err := face(data.FontOpenSans, printSize)
	if err != nil {
		return nil, err
	}

	fontLSrc := image.NewUniform(color.NRGBA{0, 0, 0, 255})
	do := func() int {
		dwr := font.Drawer{
			Dst:  img,
			Src:  fontLSrc,
			Face: print,
		}

		dwr.Dot = fixed.P(startX, startY+int(cursiveSize))
		dwr.DrawString(word)
		width1 := int(dwr.Dot.X>>6) - startX

		dwr.Face = cursive
		dwr.Dot = fixed.P(startX, startY+padding+int(cursiveSize)+int(printSize))
		dwr.DrawString(word)
		width2 := int(dwr.Dot.X>>6) - startX
		width := width1
		if width2 > width {
			width = width2
		}
		return width + startX + stopX
	}

	w := do()
	img = image.NewNRGBA(image.Rect(0, 0, w, height))

	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			o := img.PixOffset(x, y)
			img.Pix[o+0] = 255
			img.Pix[o+1] = 255
			img.Pix[o+2] = 255
			img.Pix[o+3] = 255
		}
	}
	do()

	return img, nil
}
