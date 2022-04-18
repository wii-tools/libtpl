package main

import (
	"bytes"
	"encoding/binary"
	"image"
)

const TPLMagic uint32 = 0x0020AF30

type FileHeader struct {
	Magic         uint32
	NumOfImages   uint32
	ImageTableOff uint32
}

type ImageHeader struct {
	Height     uint16
	Width      uint16
	Format     uint32
	DataOffset uint32
	WrapS      uint32
	WrapT      uint32
	MinFilter  uint32
	MagFilter  uint32
	LODBias    float32
	EdgeLOD    uint8
	MidLOD     uint8
	MaxLOD     uint8
	Unpacked   uint8
}

type TPL struct {
	Header     FileHeader
	ImageOff   uint32
	PaletteOff uint32
	Image      ImageHeader
}

// TextureFormat is a format that an image can be converted into
type TextureFormat uint32

const (
	I4 TextureFormat = iota
	I8
	IA4
	IA8
	RGB565
	RGB5A3
	RGBA8
	CI4 = 8
	CI8 = 9
	CI14X2
	CMP = 14
)

// makeTPLHeader makes the TPL header.
func makeTPLHeader(raw []byte, format TextureFormat, width, height int) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	tpl := TPL{
		Header: FileHeader{
			Magic:         TPLMagic,
			NumOfImages:   1,
			ImageTableOff: 0x0C,
		},
		ImageOff:   20,
		PaletteOff: 0,
		Image: ImageHeader{
			Height:     uint16(height),
			Width:      uint16(width),
			Format:     uint32(format),
			DataOffset: 64,
			WrapS:      0,
			WrapT:      0,
			MinFilter:  1,
			MagFilter:  1,
			LODBias:    0,
			EdgeLOD:    0,
			MidLOD:     0,
			MaxLOD:     0,
			Unpacked:   0,
		},
	}

	err := binary.Write(buf, binary.BigEndian, tpl)
	if err != nil {
		return nil, err
	}

	buf.Write(raw)

	return buf.Bytes(), nil
}

// ToI4 converts an image.Image to I4 TPL format
func ToI4(img image.Image) ([]byte, error) {
	raw := imageToRGBA(img)

	width := img.Bounds().Max.X
	height := img.Bounds().Max.Y
	inp := 0
	output := make([]byte, addPadding(width, 8)*addPadding(height, 8)/2)

	for y1 := 0; y1 < height; y1 += 8 {
		for x1 := 0; x1 < width; x1 += 8 {
			for y := y1; y < y1+8; y++ {
				for x := x1; x < x1+8; x += 2 {
					var newPixel byte

					if x >= width || y >= height {
						newPixel = 0
					} else {
						rgba := raw[x+(y*width)]

						r := (rgba >> 0) & 0xff
						g := (rgba >> 8) & 0xff
						b := (rgba >> 16) & 0xff

						i1 := ((r + g + b) / 3) & 0xff

						if (x + (y * width) + 1) >= len(raw) {
							rgba = 0
						} else {
							rgba = raw[x+(y*width)+1]
						}

						r = (rgba >> 0) & 0xff
						g = (rgba >> 8) & 0xff
						b = (rgba >> 16) & 0xff

						i2 := ((r + g + b) / 3) & 0xff

						newPixel = (byte)((((i1 * 15) / 255) << 4) | (((i2 * 15) / 255) & 0xf))
					}

					output[inp] = newPixel
					inp++
				}
			}
		}
	}

	return makeTPLHeader(output, I4, width, height)
}

// ToIA4 converts an image.Image to IA4 TPL format
func ToIA4(img image.Image) ([]byte, error) {
	raw := imageToRGBA(img)

	width := img.Bounds().Max.X
	height := img.Bounds().Max.Y
	inp := 0
	output := make([]byte, addPadding(width, 8)*addPadding(height, 4))

	for y1 := 0; y1 < height; y1 += 4 {
		for x1 := 0; x1 < width; x1 += 8 {
			for y := y1; y < y1+4; y++ {
				for x := x1; x < x1+8; x++ {
					var newPixel byte

					if y >= height || x >= width {
						newPixel = 0
					} else {
						rgba := raw[x+(y*width)]

						r := (rgba >> 0) & 0xff
						g := (rgba >> 8) & 0xff
						b := (rgba >> 16) & 0xff

						i := ((r + g + b) / 3) & 0xff
						a := (rgba >> 24) & 0xff

						newPixel = byte((((i * 15) / 255) & 0xf) | (((a * 15) / 255) << 4))
					}

					output[inp] = newPixel
					inp++
				}
			}
		}
	}

	return makeTPLHeader(output, IA4, width, height)
}

func ToRGB5A3(img image.Image) ([]byte, error) {
	raw := imageToRGBA(img)

	width := img.Bounds().Max.X
	height := img.Bounds().Max.Y
	z := -1
	output := make([]byte, addPadding(width, 4)*addPadding(height, 4)*2)

	for y1 := 0; y1 < height; y1 += 4 {
		for x1 := 0; x1 < width; x1 += 4 {
			for y := y1; y < y1+4; y++ {
				for x := x1; x < x1+4; x++ {
					var newPixel int

					if y >= height || x >= width {
						newPixel = 0
					} else {
						rgba := raw[x+(y*width)]
						newPixel = 0

						r := (rgba >> 16) & 0xff
						g := (rgba >> 8) & 0xff
						b := (rgba >> 0) & 0xff
						a := (rgba >> 24) & 0xff

						if a <= 0xda {
							newPixel &= ^(1 << 15)

							r = ((r * 15) / 255) & 0xf
							g = ((g * 15) / 255) & 0xf
							b = ((b * 15) / 255) & 0xf
							a = ((a * 7) / 255) & 0x7

							newPixel |= int((a << 12) | (r << 8) | (g << 4) | b)
						} else {

							newPixel |= 1 << 15

							r = ((r * 31) / 255) & 0x1f
							g = ((g * 31) / 255) & 0x1f
							b = ((b * 31) / 255) & 0x1f

							newPixel |= int((r << 10) | (g << 5) | b)
						}

						z++
						output[z] = byte(newPixel >> 8)
						z++
						output[z] = byte(newPixel & 0xff)
					}
				}
			}
		}
	}

	return makeTPLHeader(output, RGB5A3, width, height)
}

// ToRGB565 converts an image.Image to RGB565 TPL format
func ToRGB565(img image.Image) ([]byte, error) {
	raw := imageToRGBA(img)

	width := img.Bounds().Max.X
	height := img.Bounds().Max.Y
	z := -1
	output := make([]byte, addPadding(width, 4)*addPadding(height, 4)*2)

	for y1 := 0; y1 < height; y1 += 4 {
		for x1 := 0; x1 < width; x1 += 4 {
			for y := y1; y < y1+4; y++ {
				for x := x1; x < x1+4; x++ {
					var newPixel uint16

					if y >= height || x >= width {
						newPixel = 0
					} else {
						rgb := raw[x+y*width]

						b := (rgb >> 16) & 0xff
						g := (rgb >> 8) & 0xff
						r := (rgb >> 0) & 0xff

						newPixel = uint16(((r >> 3) << 0) | ((g >> 2) << 5) | ((b >> 3) << 11))
					}

					z += 1
					output[z] = byte(newPixel >> 8)
					z += 1
					output[z] = byte(newPixel & 0xff)
				}
			}
		}
	}

	return makeTPLHeader(output, RGB565, width, height)
}

// addPadding calculates the amount of padding the width or height needs.
func addPadding(value, padding int) int {
	if value%padding != 0 {
		value = value + (padding - (value % padding))
	}

	return value
}

// imageToRGBA converts an image.Image value to an RGBA bitmap.
func imageToRGBA(img image.Image) []uint32 {
	size := img.Bounds()
	raw := make([]uint32, (size.Max.X-size.Min.X)*(size.Max.Y-size.Min.Y))
	idx := 0

	for y := size.Min.Y; y < size.Max.Y; y++ {
		for x := size.Min.X; x < size.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			raw[idx] = ((a & 0xff) << 24) | ((r & 0xff) << 16) | ((g & 0xff) << 8) | ((b & 0xff) << 0)
			idx += 1
		}
	}

	return raw
}
