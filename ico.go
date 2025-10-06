// Package ico provides functionality to decode ICO (Icon) files.
// ICO files can contain multiple images at different sizes and can store
// images in either BMP or PNG format.
package ico

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
)

// Header represents the ICO file header
type Header struct {
	Reserved uint16 // Always 0
	Type     uint16 // 1 for ICO, 2 for CUR
	Count    uint16 // Number of images
}

// DirectoryEntry represents an entry in the ICO directory
type DirectoryEntry struct {
	Width        uint8  // Width in pixels (0 means 256)
	Height       uint8  // Height in pixels (0 means 256)
	ColorCount   uint8  // Number of colors in palette (0 means no palette)
	Reserved     uint8  // Always 0
	ColorPlanes  uint16 // Color planes (should be 0 or 1)
	BitsPerPixel uint16 // Bits per pixel
	Size         uint32 // Size of image data in bytes
	Offset       uint32 // Offset to image data from beginning of file
}

// ICO represents a decoded ICO file
type ICO struct {
	Header  Header
	Entries []DirectoryEntry
	Images  []image.Image
}

// GetWidth returns the actual width, handling the special case where 0 means 256
func (e DirectoryEntry) GetWidth() int {
	if e.Width == 0 {
		return 256
	}
	return int(e.Width)
}

// GetHeight returns the actual height, handling the special case where 0 means 256
func (e DirectoryEntry) GetHeight() int {
	if e.Height == 0 {
		return 256
	}
	return int(e.Height)
}

// Decode decodes an ICO file from the given reader
func Decode(r io.Reader) (*ICO, error) {
	// Read all data into memory for easier parsing
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read ICO data: %w", err)
	}

	if len(data) < 6 {
		return nil, fmt.Errorf("ICO file too short: need at least 6 bytes for header")
	}

	// Parse header
	header := Header{}
	buf := bytes.NewReader(data)
	if err := binary.Read(buf, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("failed to read ICO header: %w", err)
	}

	if header.Reserved != 0 {
		return nil, fmt.Errorf("invalid ICO file: reserved field must be 0")
	}

	if header.Type != 1 {
		return nil, fmt.Errorf("unsupported file type: %d (only ICO type 1 is supported)", header.Type)
	}

	if header.Count == 0 {
		return nil, fmt.Errorf("ICO file contains no images")
	}

	// Parse directory entries
	entries := make([]DirectoryEntry, header.Count)
	for i := 0; i < int(header.Count); i++ {
		if err := binary.Read(buf, binary.LittleEndian, &entries[i]); err != nil {
			return nil, fmt.Errorf("failed to read directory entry %d: %w", i, err)
		}
	}

	// Decode images
	images := make([]image.Image, header.Count)
	for i, entry := range entries {
		if entry.Offset >= uint32(len(data)) {
			return nil, fmt.Errorf("invalid offset for image %d: %d", i, entry.Offset)
		}

		if entry.Offset+entry.Size > uint32(len(data)) {
			return nil, fmt.Errorf("image %d extends beyond file boundary", i)
		}

		imageData := data[entry.Offset : entry.Offset+entry.Size]
		img, err := decodeImage(imageData, entry)
		if err != nil {
			return nil, fmt.Errorf("failed to decode image %d: %w", i, err)
		}
		images[i] = img
	}

	return &ICO{
		Header:  header,
		Entries: entries,
		Images:  images,
	}, nil
}

// decodeImage decodes a single image from the ICO file
func decodeImage(data []byte, entry DirectoryEntry) (image.Image, error) {
	// Check if it's a PNG (starts with PNG signature)
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return png.Decode(bytes.NewReader(data))
	}

	// Otherwise, assume it's a BMP without file header
	return decodeBMP(data, entry)
}

// decodeBMP decodes a BMP image data (without the file header)
func decodeBMP(data []byte, entry DirectoryEntry) (image.Image, error) {
	if len(data) < 40 {
		return nil, fmt.Errorf("BMP data too short: need at least 40 bytes for header")
	}

	// Read BMP info header
	buf := bytes.NewReader(data)

	var headerSize uint32
	if err := binary.Read(buf, binary.LittleEndian, &headerSize); err != nil {
		return nil, fmt.Errorf("failed to read BMP header size: %w", err)
	}

	var width, height int32
	if err := binary.Read(buf, binary.LittleEndian, &width); err != nil {
		return nil, fmt.Errorf("failed to read BMP width: %w", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &height); err != nil {
		return nil, fmt.Errorf("failed to read BMP height: %w", err)
	}

	// Height in BMP for ICO is the combined height of XOR and AND masks
	// So actual image height is height/2
	height = height / 2

	var planes uint16
	if err := binary.Read(buf, binary.LittleEndian, &planes); err != nil {
		return nil, fmt.Errorf("failed to read BMP planes: %w", err)
	}

	var bitsPerPixel uint16
	if err := binary.Read(buf, binary.LittleEndian, &bitsPerPixel); err != nil {
		return nil, fmt.Errorf("failed to read BMP bits per pixel: %w", err)
	}

	// Skip the rest of the header
	buf.Seek(int64(headerSize), io.SeekStart)

	switch bitsPerPixel {
	case 32:
		return decodeBMP32(data[headerSize:], int(width), int(height))
	case 24:
		return decodeBMP24(data[headerSize:], int(width), int(height))
	case 8:
		return decodeBMP8(data, int(width), int(height), int(headerSize))
	case 4:
		return decodeBMP4(data, int(width), int(height), int(headerSize))
	case 1:
		return decodeBMP1(data, int(width), int(height), int(headerSize))
	default:
		return nil, fmt.Errorf("unsupported BMP bit depth: %d", bitsPerPixel)
	}
}

// decodeBMP32 decodes 32-bit BMP data
func decodeBMP32(data []byte, width, height int) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// XOR mask (color data)
	xorRowSize := width * 4
	xorRowPadding := (4 - (xorRowSize % 4)) % 4
	xorTotalRowSize := xorRowSize + xorRowPadding

	// Read XOR mask first
	for y := 0; y < height; y++ {
		// BMP rows are stored bottom-to-top
		srcY := height - 1 - y
		rowOffset := srcY * xorTotalRowSize

		if rowOffset+xorRowSize > len(data) {
			return nil, fmt.Errorf("BMP data truncated at row %d", y)
		}

		for x := 0; x < width; x++ {
			pixelOffset := rowOffset + x*4
			if pixelOffset+3 >= len(data) {
				return nil, fmt.Errorf("BMP data truncated at pixel (%d,%d)", x, y)
			}

			// BMP uses BGRA format
			b := data[pixelOffset]
			g := data[pixelOffset+1]
			r := data[pixelOffset+2]
			a := data[pixelOffset+3]

			img.Set(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}

	// AND mask (transparency mask) - 1 bit per pixel
	andMaskOffset := height * xorTotalRowSize
	andRowSize := (width + 7) / 8 // 8 pixels per byte
	andRowPadding := (4 - (andRowSize % 4)) % 4
	andTotalRowSize := andRowSize + andRowPadding

	// Apply AND mask if there's enough data
	if andMaskOffset+height*andTotalRowSize <= len(data) {
		for y := 0; y < height; y++ {
			// AND mask rows are also stored bottom-to-top
			srcY := height - 1 - y
			rowOffset := andMaskOffset + srcY*andTotalRowSize

			for x := 0; x < width; x++ {
				byteOffset := rowOffset + x/8
				bitIndex := 7 - (x % 8)

				if byteOffset < len(data) {
					maskByte := data[byteOffset]
					isTransparent := (maskByte >> bitIndex) & 1

					if isTransparent == 1 {
						// AND mask bit is 1, so pixel should be fully transparent
						currentColor := img.RGBAAt(x, y)
						img.Set(x, y, color.NRGBA{R: currentColor.R, G: currentColor.G, B: currentColor.B, A: 0})
					}
				}
			}
		}
	}

	return img, nil
}

// decodeBMP24 decodes 24-bit BMP data
func decodeBMP24(data []byte, width, height int) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// XOR mask (color data)
	xorRowSize := width * 3
	xorRowPadding := (4 - (xorRowSize % 4)) % 4
	xorTotalRowSize := xorRowSize + xorRowPadding

	// Read XOR mask first
	for y := 0; y < height; y++ {
		srcY := height - 1 - y
		rowOffset := srcY * xorTotalRowSize

		if rowOffset+xorRowSize > len(data) {
			return nil, fmt.Errorf("BMP data truncated at row %d", y)
		}

		for x := 0; x < width; x++ {
			pixelOffset := rowOffset + x*3
			if pixelOffset+2 >= len(data) {
				return nil, fmt.Errorf("BMP data truncated at pixel (%d,%d)", x, y)
			}

			b := data[pixelOffset]
			g := data[pixelOffset+1]
			r := data[pixelOffset+2]

			img.Set(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}

	// AND mask (transparency mask) - 1 bit per pixel
	andMaskOffset := height * xorTotalRowSize
	andRowSize := (width + 7) / 8 // 8 pixels per byte
	andRowPadding := (4 - (andRowSize % 4)) % 4
	andTotalRowSize := andRowSize + andRowPadding

	// Apply AND mask if there's enough data
	if andMaskOffset+height*andTotalRowSize <= len(data) {
		for y := 0; y < height; y++ {
			// AND mask rows are also stored bottom-to-top
			srcY := height - 1 - y
			rowOffset := andMaskOffset + srcY*andTotalRowSize

			for x := 0; x < width; x++ {
				byteOffset := rowOffset + x/8
				bitIndex := 7 - (x % 8)

				if byteOffset < len(data) {
					maskByte := data[byteOffset]
					isTransparent := (maskByte >> bitIndex) & 1

					if isTransparent == 1 {
						// AND mask bit is 1, so pixel should be fully transparent
						currentColor := img.RGBAAt(x, y)
						img.Set(x, y, color.NRGBA{R: currentColor.R, G: currentColor.G, B: currentColor.B, A: 0})
					}
				}
			}
		}
	}

	return img, nil
}

// decodeBMP8 decodes 8-bit BMP data with palette
func decodeBMP8(data []byte, width, height int, headerSize int) (image.Image, error) {
	// Read palette (256 colors * 4 bytes each = 1024 bytes)
	paletteOffset := headerSize
	if paletteOffset+1024 > len(data) {
		return nil, fmt.Errorf("BMP palette data truncated")
	}

	palette := make([]color.NRGBA, 256)
	for i := 0; i < 256; i++ {
		offset := paletteOffset + i*4
		b := data[offset]
		g := data[offset+1]
		r := data[offset+2]
		// Skip reserved byte at offset+3
		palette[i] = color.NRGBA{R: r, G: g, B: b, A: 255}
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	pixelDataOffset := paletteOffset + 1024
	rowSize := width
	rowPadding := (4 - (rowSize % 4)) % 4
	totalRowSize := rowSize + rowPadding

	// Read XOR mask (color data)
	for y := 0; y < height; y++ {
		srcY := height - 1 - y
		rowOffset := pixelDataOffset + srcY*totalRowSize

		for x := 0; x < width; x++ {
			if rowOffset+x >= len(data) {
				return nil, fmt.Errorf("BMP data truncated at pixel (%d,%d)", x, y)
			}

			paletteIndex := data[rowOffset+x]
			img.Set(x, y, palette[paletteIndex])
		}
	}

	// AND mask (transparency mask) - 1 bit per pixel
	andMaskOffset := pixelDataOffset + height*totalRowSize
	andRowSize := (width + 7) / 8 // 8 pixels per byte
	andRowPadding := (4 - (andRowSize % 4)) % 4
	andTotalRowSize := andRowSize + andRowPadding

	// Apply AND mask if there's enough data
	if andMaskOffset+height*andTotalRowSize <= len(data) {
		for y := 0; y < height; y++ {
			// AND mask rows are also stored bottom-to-top
			srcY := height - 1 - y
			rowOffset := andMaskOffset + srcY*andTotalRowSize

			for x := 0; x < width; x++ {
				byteOffset := rowOffset + x/8
				bitIndex := 7 - (x % 8)

				if byteOffset < len(data) {
					maskByte := data[byteOffset]
					isTransparent := (maskByte >> bitIndex) & 1

					if isTransparent == 1 {
						// AND mask bit is 1, so pixel should be fully transparent
						currentColor := img.RGBAAt(x, y)
						img.Set(x, y, color.NRGBA{R: currentColor.R, G: currentColor.G, B: currentColor.B, A: 0})
					}
				}
			}
		}
	}

	return img, nil
}

// decodeBMP4 decodes 4-bit BMP data with palette
func decodeBMP4(data []byte, width, height int, headerSize int) (image.Image, error) {
	// Read palette (16 colors * 4 bytes each = 64 bytes)
	paletteOffset := headerSize
	if paletteOffset+64 > len(data) {
		return nil, fmt.Errorf("BMP palette data truncated")
	}

	palette := make([]color.NRGBA, 16)
	for i := 0; i < 16; i++ {
		offset := paletteOffset + i*4
		b := data[offset]
		g := data[offset+1]
		r := data[offset+2]
		palette[i] = color.NRGBA{R: r, G: g, B: b, A: 255}
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	pixelDataOffset := paletteOffset + 64
	rowSize := (width + 1) / 2 // 2 pixels per byte
	rowPadding := (4 - (rowSize % 4)) % 4
	totalRowSize := rowSize + rowPadding

	for y := 0; y < height; y++ {
		srcY := height - 1 - y
		rowOffset := pixelDataOffset + srcY*totalRowSize

		for x := 0; x < width; x += 2 {
			byteOffset := rowOffset + x/2
			if byteOffset >= len(data) {
				return nil, fmt.Errorf("BMP data truncated at pixel (%d,%d)", x, y)
			}

			pixelByte := data[byteOffset]

			// First pixel (high nibble)
			paletteIndex1 := (pixelByte >> 4) & 0x0F
			img.Set(x, y, palette[paletteIndex1])

			// Second pixel (low nibble), if it exists
			if x+1 < width {
				paletteIndex2 := pixelByte & 0x0F
				img.Set(x+1, y, palette[paletteIndex2])
			}
		}
	}

	// AND mask (transparency mask) - 1 bit per pixel
	andMaskOffset := pixelDataOffset + height*totalRowSize
	andRowSize := (width + 7) / 8 // 8 pixels per byte
	andRowPadding := (4 - (andRowSize % 4)) % 4
	andTotalRowSize := andRowSize + andRowPadding

	// Apply AND mask if there's enough data
	if andMaskOffset+height*andTotalRowSize <= len(data) {
		for y := 0; y < height; y++ {
			// AND mask rows are also stored bottom-to-top
			srcY := height - 1 - y
			rowOffset := andMaskOffset + srcY*andTotalRowSize

			for x := 0; x < width; x++ {
				byteOffset := rowOffset + x/8
				bitIndex := 7 - (x % 8)

				if byteOffset < len(data) {
					maskByte := data[byteOffset]
					isTransparent := (maskByte >> bitIndex) & 1

					if isTransparent == 1 {
						// AND mask bit is 1, so pixel should be fully transparent
						currentColor := img.RGBAAt(x, y)
						img.Set(x, y, color.NRGBA{R: currentColor.R, G: currentColor.G, B: currentColor.B, A: 0})
					}
				}
			}
		}
	}

	return img, nil
}

// decodeBMP1 decodes 1-bit BMP data with palette
func decodeBMP1(data []byte, width, height int, headerSize int) (image.Image, error) {
	// Read palette (2 colors * 4 bytes each = 8 bytes)
	paletteOffset := headerSize
	if paletteOffset+8 > len(data) {
		return nil, fmt.Errorf("BMP palette data truncated")
	}

	palette := make([]color.NRGBA, 2)
	for i := 0; i < 2; i++ {
		offset := paletteOffset + i*4
		b := data[offset]
		g := data[offset+1]
		r := data[offset+2]
		palette[i] = color.NRGBA{R: r, G: g, B: b, A: 255}
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	pixelDataOffset := paletteOffset + 8
	rowSize := (width + 7) / 8 // 8 pixels per byte
	rowPadding := (4 - (rowSize % 4)) % 4
	totalRowSize := rowSize + rowPadding

	for y := 0; y < height; y++ {
		srcY := height - 1 - y
		rowOffset := pixelDataOffset + srcY*totalRowSize

		for x := 0; x < width; x++ {
			byteOffset := rowOffset + x/8
			if byteOffset >= len(data) {
				return nil, fmt.Errorf("BMP data truncated at pixel (%d,%d)", x, y)
			}

			pixelByte := data[byteOffset]
			bitIndex := 7 - (x % 8)
			paletteIndex := (pixelByte >> bitIndex) & 1

			img.Set(x, y, palette[paletteIndex])
		}
	}

	// AND mask (transparency mask) - 1 bit per pixel
	andMaskOffset := pixelDataOffset + height*totalRowSize
	andRowSize := (width + 7) / 8 // 8 pixels per byte
	andRowPadding := (4 - (andRowSize % 4)) % 4
	andTotalRowSize := andRowSize + andRowPadding

	// Apply AND mask if there's enough data
	if andMaskOffset+height*andTotalRowSize <= len(data) {
		for y := 0; y < height; y++ {
			// AND mask rows are also stored bottom-to-top
			srcY := height - 1 - y
			rowOffset := andMaskOffset + srcY*andTotalRowSize

			for x := 0; x < width; x++ {
				byteOffset := rowOffset + x/8
				bitIndex := 7 - (x % 8)

				if byteOffset < len(data) {
					maskByte := data[byteOffset]
					isTransparent := (maskByte >> bitIndex) & 1

					if isTransparent == 1 {
						// AND mask bit is 1, so pixel should be fully transparent
						currentColor := img.RGBAAt(x, y)
						img.Set(x, y, color.NRGBA{R: currentColor.R, G: currentColor.G, B: currentColor.B, A: 0})
					}
				}
			}
		}
	}

	return img, nil
}

// GetBestImage returns the image with the highest resolution from the ICO file.
// If multiple images have the same resolution, it returns the first one found.
func (ico *ICO) GetBestImage() image.Image {
	if len(ico.Images) == 0 {
		return nil
	}

	bestIndex := 0
	bestSize := ico.Entries[0].GetWidth() * ico.Entries[0].GetHeight()

	for i, entry := range ico.Entries {
		size := entry.GetWidth() * entry.GetHeight()
		if size > bestSize {
			bestSize = size
			bestIndex = i
		}
	}

	return ico.Images[bestIndex]
}

// GetImageBySize returns the image that best matches the requested size.
// It finds the image with dimensions closest to the requested width and height.
func (ico *ICO) GetImageBySize(width, height int) image.Image {
	if len(ico.Images) == 0 {
		return nil
	}

	bestIndex := 0
	bestScore := scoreSizeMatch(ico.Entries[0], width, height)

	for i, entry := range ico.Entries {
		score := scoreSizeMatch(entry, width, height)
		if score < bestScore {
			bestScore = score
			bestIndex = i
		}
	}

	return ico.Images[bestIndex]
}

// GetAvailableSizes returns a slice of available image sizes in the ICO file.
// Each element contains the width and height of an available image.
func (ico *ICO) GetAvailableSizes() []image.Point {
	sizes := make([]image.Point, len(ico.Entries))
	for i, entry := range ico.Entries {
		sizes[i] = image.Point{
			X: entry.GetWidth(),
			Y: entry.GetHeight(),
		}
	}
	return sizes
}

// scoreSizeMatch calculates how well an image size matches the requested size.
// Lower scores indicate better matches.
func scoreSizeMatch(entry DirectoryEntry, targetWidth, targetHeight int) int {
	widthDiff := entry.GetWidth() - targetWidth
	heightDiff := entry.GetHeight() - targetHeight
	return widthDiff*widthDiff + heightDiff*heightDiff
}

// Config represents the metadata of an ICO file without decoding the image data.
type Config struct {
	Width  int
	Height int
	Count  int
}

// DecodeConfig decodes just the configuration (metadata) of an ICO file without
// decoding the image data. It returns the dimensions of the largest image and
// the total number of images in the file.
func DecodeConfig(r io.Reader) (Config, error) {
	// Read just enough data for header and directory entries
	headerBuf := make([]byte, 6)
	if _, err := io.ReadFull(r, headerBuf); err != nil {
		return Config{}, fmt.Errorf("failed to read ICO header: %w", err)
	}

	header := Header{}
	if err := binary.Read(bytes.NewReader(headerBuf), binary.LittleEndian, &header); err != nil {
		return Config{}, fmt.Errorf("failed to parse ICO header: %w", err)
	}

	if header.Reserved != 0 || header.Type != 1 || header.Count == 0 {
		return Config{}, fmt.Errorf("invalid ICO file")
	}

	// Read directory entries
	entryBuf := make([]byte, 16*int(header.Count))
	if _, err := io.ReadFull(r, entryBuf); err != nil {
		return Config{}, fmt.Errorf("failed to read ICO directory entries: %w", err)
	}

	// Find the largest image
	var maxWidth, maxHeight int
	buf := bytes.NewReader(entryBuf)
	for i := 0; i < int(header.Count); i++ {
		var entry DirectoryEntry
		if err := binary.Read(buf, binary.LittleEndian, &entry); err != nil {
			return Config{}, fmt.Errorf("failed to read directory entry %d: %w", i, err)
		}

		width := entry.GetWidth()
		height := entry.GetHeight()
		if width*height > maxWidth*maxHeight {
			maxWidth = width
			maxHeight = height
		}
	}

	return Config{
		Width:  maxWidth,
		Height: maxHeight,
		Count:  int(header.Count),
	}, nil
}
