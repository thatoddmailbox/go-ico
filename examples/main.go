package main

import (
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/thatoddmailbox/go-ico"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <ico-file>")
		fmt.Println("Example: go run main.go favicon.ico")
		os.Exit(1)
	}

	icoPath := os.Args[1]

	// Example 1: Get metadata without decoding images
	fmt.Println("=== ICO Metadata ===")
	file, err := os.Open(icoPath)
	if err != nil {
		log.Fatalf("Failed to open ICO file: %v", err)
	}

	config, err := ico.DecodeConfig(file)
	if err != nil {
		log.Fatalf("Failed to decode ICO config: %v", err)
	}
	file.Close()

	fmt.Printf("Largest image size: %dx%d\n", config.Width, config.Height)
	fmt.Printf("Number of images: %d\n", config.Count)

	// Example 2: Decode full ICO file
	fmt.Println("\n=== Full ICO Decoding ===")
	file, err = os.Open(icoPath)
	if err != nil {
		log.Fatalf("Failed to open ICO file: %v", err)
	}
	defer file.Close()

	icoFile, err := ico.Decode(file)
	if err != nil {
		log.Fatalf("Failed to decode ICO: %v", err)
	}

	fmt.Printf("Successfully decoded ICO with %d images\n", len(icoFile.Images))

	// Example 3: Show all available sizes
	fmt.Println("\n=== Available Sizes ===")
	sizes := icoFile.GetAvailableSizes()
	for i, size := range sizes {
		entry := icoFile.Entries[i]
		fmt.Printf("Image %d: %dx%d, %d bits per pixel, %d bytes\n",
			i+1, size.X, size.Y, entry.BitsPerPixel, entry.Size)
	}

	// Example 4: Extract the best (highest resolution) image
	fmt.Println("\n=== Extracting Best Image ===")
	bestImg := icoFile.GetBestImage()
	if bestImg != nil {
		bounds := bestImg.Bounds()
		fmt.Printf("Best image size: %dx%d\n", bounds.Dx(), bounds.Dy())

		// Save the best image as PNG
		bestPath := "best_" + filepath.Base(icoPath) + ".png"
		outFile, err := os.Create(bestPath)
		if err != nil {
			log.Printf("Failed to create output file: %v", err)
		} else {
			defer outFile.Close()
			err = png.Encode(outFile, bestImg)
			if err != nil {
				log.Printf("Failed to encode PNG: %v", err)
			} else {
				fmt.Printf("Saved best image as: %s\n", bestPath)
			}
		}
	}

	// Example 5: Get image by specific size
	fmt.Println("\n=== Finding Specific Sizes ===")

	// Try to find a 16x16 image
	img16 := icoFile.GetImageBySize(16, 16)
	if img16 != nil {
		bounds := img16.Bounds()
		fmt.Printf("Found image closest to 16x16: %dx%d\n", bounds.Dx(), bounds.Dy())

		// Save as PNG
		img16Path := "16x16_" + filepath.Base(icoPath) + ".png"
		outFile, err := os.Create(img16Path)
		if err != nil {
			log.Printf("Failed to create 16x16 output file: %v", err)
		} else {
			defer outFile.Close()
			err = png.Encode(outFile, img16)
			if err != nil {
				log.Printf("Failed to encode 16x16 PNG: %v", err)
			} else {
				fmt.Printf("Saved 16x16 image as: %s\n", img16Path)
			}
		}
	}

	// Try to find a 32x32 image
	img32 := icoFile.GetImageBySize(32, 32)
	if img32 != nil {
		bounds := img32.Bounds()
		fmt.Printf("Found image closest to 32x32: %dx%d\n", bounds.Dx(), bounds.Dy())

		// Save as PNG
		img32Path := "32x32_" + filepath.Base(icoPath) + ".png"
		outFile, err := os.Create(img32Path)
		if err != nil {
			log.Printf("Failed to create 32x32 output file: %v", err)
		} else {
			defer outFile.Close()
			err = png.Encode(outFile, img32)
			if err != nil {
				log.Printf("Failed to encode 32x32 PNG: %v", err)
			} else {
				fmt.Printf("Saved 32x32 image as: %s\n", img32Path)
			}
		}
	}

	// Example 6: Extract all images
	fmt.Println("\n=== Extracting All Images ===")
	for i, img := range icoFile.Images {
		bounds := img.Bounds()
		entry := icoFile.Entries[i]

		filename := fmt.Sprintf("image_%d_%dx%d_%s.png",
			i+1, bounds.Dx(), bounds.Dy(), filepath.Base(icoPath))

		outFile, err := os.Create(filename)
		if err != nil {
			log.Printf("Failed to create file %s: %v", filename, err)
			continue
		}

		err = png.Encode(outFile, img)
		outFile.Close()

		if err != nil {
			log.Printf("Failed to encode %s: %v", filename, err)
		} else {
			fmt.Printf("Saved image %d (%dx%d, %d bpp) as: %s\n",
				i+1, bounds.Dx(), bounds.Dy(), entry.BitsPerPixel, filename)
		}
	}

	// Example 7: Show pixel data from a corner of the best image
	fmt.Println("\n=== Pixel Analysis ===")
	if bestImg != nil {
		bounds := bestImg.Bounds()
		fmt.Printf("Analyzing corner pixels of %dx%d image:\n", bounds.Dx(), bounds.Dy())

		// Show colors of the four corners
		corners := []struct {
			name string
			x, y int
		}{
			{"Top-left", bounds.Min.X, bounds.Min.Y},
			{"Top-right", bounds.Max.X - 1, bounds.Min.Y},
			{"Bottom-left", bounds.Min.X, bounds.Max.Y - 1},
			{"Bottom-right", bounds.Max.X - 1, bounds.Max.Y - 1},
		}

		for _, corner := range corners {
			c := bestImg.At(corner.x, corner.y)
			r, g, b, a := c.RGBA()
			// Convert from 16-bit to 8-bit
			r8, g8, b8, a8 := uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8)
			fmt.Printf("%s (%d,%d): RGBA(%d,%d,%d,%d)\n",
				corner.name, corner.x, corner.y, r8, g8, b8, a8)
		}
	}

	fmt.Println("\n=== Done ===")
	fmt.Println("All extracted images have been saved as PNG files.")
}
