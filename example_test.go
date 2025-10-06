package ico_test

import (
	"bytes"
	"fmt"
	"image/color"
	"log"

	"github.com/thatoddmailbox/go-ico"
)

// ExampleDecode demonstrates how to decode an ICO file and access its images.
func ExampleDecode() {
	// Create a simple ICO with one 2x2 image for demonstration
	icoData := createSampleICO()

	// Decode the ICO file
	icoFile, err := ico.Decode(bytes.NewReader(icoData))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Number of images: %d\n", len(icoFile.Images))
	fmt.Printf("First image size: %dx%d\n",
		icoFile.Entries[0].GetWidth(),
		icoFile.Entries[0].GetHeight())
	fmt.Printf("Bits per pixel: %d\n", icoFile.Entries[0].BitsPerPixel)

	// Output:
	// Number of images: 1
	// First image size: 2x2
	// Bits per pixel: 32
}

// ExampleDecodeConfig demonstrates how to quickly get metadata about an ICO file.
func ExampleDecodeConfig() {
	// Create a sample ICO for demonstration
	icoData := createSampleICO()

	// Get just the metadata without decoding images
	config, err := ico.DecodeConfig(bytes.NewReader(icoData))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Largest image: %dx%d\n", config.Width, config.Height)
	fmt.Printf("Total images: %d\n", config.Count)

	// Output:
	// Largest image: 2x2
	// Total images: 1
}

// ExampleICO_GetBestImage shows how to get the highest resolution image.
func ExampleICO_GetBestImage() {
	// Create a sample ICO
	icoData := createSampleICO()

	icoFile, err := ico.Decode(bytes.NewReader(icoData))
	if err != nil {
		log.Fatal(err)
	}

	// Get the best (highest resolution) image
	bestImg := icoFile.GetBestImage()
	bounds := bestImg.Bounds()
	fmt.Printf("Best image dimensions: %dx%d\n", bounds.Dx(), bounds.Dy())

	// Output:
	// Best image dimensions: 2x2
}

// ExampleICO_GetImageBySize shows how to find an image closest to a desired size.
func ExampleICO_GetImageBySize() {
	// Create a sample ICO
	icoData := createSampleICO()

	icoFile, err := ico.Decode(bytes.NewReader(icoData))
	if err != nil {
		log.Fatal(err)
	}

	// Try to find an image close to 16x16
	img := icoFile.GetImageBySize(16, 16)
	bounds := img.Bounds()
	fmt.Printf("Requested 16x16, got: %dx%d\n", bounds.Dx(), bounds.Dy())

	// Output:
	// Requested 16x16, got: 2x2
}

// ExampleICO_GetAvailableSizes shows how to list all available image sizes.
func ExampleICO_GetAvailableSizes() {
	// Create a sample ICO
	icoData := createSampleICO()

	icoFile, err := ico.Decode(bytes.NewReader(icoData))
	if err != nil {
		log.Fatal(err)
	}

	// Get all available sizes
	sizes := icoFile.GetAvailableSizes()
	for i, size := range sizes {
		fmt.Printf("Image %d: %dx%d\n", i+1, size.X, size.Y)
	}

	// Output:
	// Image 1: 2x2
}

// createSampleICO creates a minimal ICO file for testing purposes.
// This creates a 2x2 image with a simple pattern.
func createSampleICO() []byte {
	var buf bytes.Buffer

	// ICO Header (6 bytes)
	buf.Write([]byte{0x00, 0x00}) // Reserved (0)
	buf.Write([]byte{0x01, 0x00}) // Type (1 = ICO)
	buf.Write([]byte{0x01, 0x00}) // Count (1 image)

	// Directory Entry (16 bytes)
	buf.WriteByte(2)                          // Width (2 pixels)
	buf.WriteByte(2)                          // Height (2 pixels)
	buf.WriteByte(0)                          // ColorCount (0 = no palette)
	buf.WriteByte(0)                          // Reserved (0)
	buf.Write([]byte{0x01, 0x00})             // ColorPlanes (1)
	buf.Write([]byte{0x20, 0x00})             // BitsPerPixel (32)
	buf.Write([]byte{0x38, 0x00, 0x00, 0x00}) // Size (56 bytes: 40 header + 16 pixel)
	buf.Write([]byte{0x16, 0x00, 0x00, 0x00}) // Offset (22 bytes)

	// BMP Info Header (40 bytes)
	buf.Write([]byte{0x28, 0x00, 0x00, 0x00}) // Header size (40)
	buf.Write([]byte{0x02, 0x00, 0x00, 0x00}) // Width (2)
	buf.Write([]byte{0x04, 0x00, 0x00, 0x00}) // Height (4, doubled for ICO)
	buf.Write([]byte{0x01, 0x00})             // Planes (1)
	buf.Write([]byte{0x20, 0x00})             // BitsPerPixel (32)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00}) // Compression (0)
	buf.Write([]byte{0x10, 0x00, 0x00, 0x00}) // ImageSize (16 bytes)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00}) // XPelsPerMeter (0)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00}) // YPelsPerMeter (0)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00}) // ColorsUsed (0)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00}) // ColorsImportant (0)

	// Pixel data (16 bytes: 2x2 pixels, 4 bytes each, stored bottom-to-top)
	// Bottom row: red, green
	buf.Write([]byte{0x00, 0x00, 0xFF, 0xFF}) // Red pixel (BGRA format)
	buf.Write([]byte{0x00, 0xFF, 0x00, 0xFF}) // Green pixel
	// Top row: blue, white
	buf.Write([]byte{0xFF, 0x00, 0x00, 0xFF}) // Blue pixel
	buf.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF}) // White pixel

	return buf.Bytes()
}

// Example demonstrates a complete real-world scenario.
func Example() {
	// Simulate reading an ICO file (in real usage, you'd use os.Open)
	icoData := createSampleICO()

	// First, check what's in the file without decoding all images
	config, err := ico.DecodeConfig(bytes.NewReader(icoData))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ICO file contains %d images\n", config.Count)
	fmt.Printf("Largest image is %dx%d\n", config.Width, config.Height)

	// Now decode the full file
	icoFile, err := ico.Decode(bytes.NewReader(icoData))
	if err != nil {
		log.Fatal(err)
	}

	// Get the best image for display
	bestImg := icoFile.GetBestImage()
	if bestImg != nil {
		bounds := bestImg.Bounds()
		fmt.Printf("Using %dx%d image for display\n", bounds.Dx(), bounds.Dy())

		// Check a pixel color (top-left corner)
		c := bestImg.At(0, 0)
		r, g, b, a := c.RGBA()
		// Convert from 16-bit to 8-bit for display
		fmt.Printf("Top-left pixel: RGBA(%d,%d,%d,%d)\n",
			uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8))
	}

	// Try to get a favicon-sized image (16x16 or close)
	faviconImg := icoFile.GetImageBySize(16, 16)
	faviconBounds := faviconImg.Bounds()
	fmt.Printf("Favicon image: %dx%d\n", faviconBounds.Dx(), faviconBounds.Dy())

	// Output:
	// ICO file contains 1 images
	// Largest image is 2x2
	// Using 2x2 image for display
	// Top-left pixel: RGBA(0,0,255,255)
	// Favicon image: 2x2
}

// Example_pixelAccess shows how to access individual pixels from decoded images.
func Example_pixelAccess() {
	// Create and decode a sample ICO
	icoData := createSampleICO()
	icoFile, err := ico.Decode(bytes.NewReader(icoData))
	if err != nil {
		log.Fatal(err)
	}

	img := icoFile.Images[0]
	bounds := img.Bounds()

	// Access each pixel and show its color
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()

			// Convert to standard color.RGBA for easier reading
			rgba := color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			}

			var colorName string
			switch rgba {
			case color.RGBA{255, 0, 0, 255}:
				colorName = "red"
			case color.RGBA{0, 255, 0, 255}:
				colorName = "green"
			case color.RGBA{0, 0, 255, 255}:
				colorName = "blue"
			case color.RGBA{255, 255, 255, 255}:
				colorName = "white"
			default:
				colorName = "unknown"
			}

			fmt.Printf("Pixel at (%d,%d): %s\n", x, y, colorName)
		}
	}

	// Output:
	// Pixel at (0,0): blue
	// Pixel at (1,0): white
	// Pixel at (0,1): red
	// Pixel at (1,1): green
}
