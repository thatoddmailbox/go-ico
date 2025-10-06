package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/thatoddmailbox/go-ico"
)

var (
	outputDir = flag.String("o", ".", "Output directory for extracted images")
	bestOnly  = flag.Bool("best", false, "Extract only the best (highest resolution) image")
	sizeSpec  = flag.String("size", "", "Extract image closest to specified size (e.g., '32x32')")
	listOnly  = flag.Bool("list", false, "List available images without extracting")
	prefix    = flag.String("prefix", "", "Prefix for output filenames")
	verbose   = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <ico-file> [ico-file...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Extract images from ICO files and save them as PNG files.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s favicon.ico                    # Extract all images\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -best favicon.ico              # Extract only the best image\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -size=32x32 favicon.ico        # Extract image closest to 32x32\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -list favicon.ico              # List available images\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -o=icons -prefix=app_ *.ico    # Extract to icons/ with prefix\n", os.Args[0])
	}

	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Process each ICO file
	for _, icoPath := range flag.Args() {
		if err := processICOFile(icoPath); err != nil {
			log.Printf("Error processing %s: %v", icoPath, err)
		}
	}
}

func processICOFile(icoPath string) error {
	if *verbose {
		fmt.Printf("Processing: %s\n", icoPath)
	}

	file, err := os.Open(icoPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// If we only need to list, use DecodeConfig for efficiency
	if *listOnly {
		return listImages(file, icoPath)
	}

	// Decode the full ICO file
	icoFile, err := ico.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode ICO: %w", err)
	}

	if *verbose {
		fmt.Printf("  Found %d images\n", len(icoFile.Images))
	}

	baseFilename := strings.TrimSuffix(filepath.Base(icoPath), filepath.Ext(icoPath))

	if *bestOnly {
		return extractBestImage(icoFile, baseFilename)
	}

	if *sizeSpec != "" {
		return extractImageBySize(icoFile, baseFilename, *sizeSpec)
	}

	// Extract all images
	return extractAllImages(icoFile, baseFilename)
}

func listImages(file *os.File, icoPath string) error {
	config, err := ico.DecodeConfig(file)
	if err != nil {
		return err
	}

	fmt.Printf("%s:\n", icoPath)
	fmt.Printf("  Images: %d\n", config.Count)
	fmt.Printf("  Largest: %dx%d\n", config.Width, config.Height)

	// For detailed listing, we need to decode the full file
	file.Seek(0, 0)
	icoFile, err := ico.Decode(file)
	if err != nil {
		return err
	}

	for i, entry := range icoFile.Entries {
		fmt.Printf("  Image %d: %dx%d, %d bpp, %d bytes\n",
			i+1, entry.GetWidth(), entry.GetHeight(), entry.BitsPerPixel, entry.Size)
	}
	fmt.Println()

	return nil
}

func extractBestImage(icoFile *ico.ICO, baseFilename string) error {
	img := icoFile.GetBestImage()
	if img == nil {
		return fmt.Errorf("no images found")
	}

	bounds := img.Bounds()
	filename := fmt.Sprintf("%s%s_best_%dx%d.png", *prefix, baseFilename, bounds.Dx(), bounds.Dy())
	outputPath := filepath.Join(*outputDir, filename)

	if err := savePNG(img, outputPath); err != nil {
		return err
	}

	fmt.Printf("Extracted best image: %s (%dx%d)\n", outputPath, bounds.Dx(), bounds.Dy())
	return nil
}

func extractImageBySize(icoFile *ico.ICO, baseFilename, sizeSpec string) error {
	parts := strings.Split(sizeSpec, "x")
	if len(parts) != 2 {
		return fmt.Errorf("invalid size specification: %s (use format like '32x32')", sizeSpec)
	}

	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid width: %s", parts[0])
	}

	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid height: %s", parts[1])
	}

	img := icoFile.GetImageBySize(width, height)
	if img == nil {
		return fmt.Errorf("no images found")
	}

	bounds := img.Bounds()
	filename := fmt.Sprintf("%s%s_%dx%d.png", *prefix, baseFilename, bounds.Dx(), bounds.Dy())
	outputPath := filepath.Join(*outputDir, filename)

	if err := savePNG(img, outputPath); err != nil {
		return err
	}

	fmt.Printf("Extracted image closest to %dx%d: %s (actual: %dx%d)\n",
		width, height, outputPath, bounds.Dx(), bounds.Dy())
	return nil
}

func extractAllImages(icoFile *ico.ICO, baseFilename string) error {
	if len(icoFile.Images) == 0 {
		return fmt.Errorf("no images found")
	}

	for i, img := range icoFile.Images {
		bounds := img.Bounds()
		entry := icoFile.Entries[i]

		filename := fmt.Sprintf("%s%s_%d_%dx%d_%dbpp.png",
			*prefix, baseFilename, i+1, bounds.Dx(), bounds.Dy(), entry.BitsPerPixel)
		outputPath := filepath.Join(*outputDir, filename)

		if err := savePNG(img, outputPath); err != nil {
			log.Printf("Failed to save image %d: %v", i+1, err)
			continue
		}

		fmt.Printf("Extracted image %d: %s (%dx%d, %d bpp)\n",
			i+1, outputPath, bounds.Dx(), bounds.Dy(), entry.BitsPerPixel)
	}

	return nil
}

func savePNG(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	return nil
}
