# Usage Guide for go-ico

This guide shows you how to use the go-ico library to decode ICO files in your Go applications.

## Table of Contents

1. [Basic Usage](#basic-usage)
2. [Getting Metadata Only](#getting-metadata-only)
3. [Working with Multiple Images](#working-with-multiple-images)
4. [Finding Specific Image Sizes](#finding-specific-image-sizes)
5. [Common Use Cases](#common-use-cases)
6. [Error Handling](#error-handling)
7. [Performance Tips](#performance-tips)
8. [Complete Examples](#complete-examples)

## Basic Usage

### Decoding an ICO File

The most basic operation is to decode an ICO file and access its images:

```go
package main

import (
    "fmt"
    "os"

    "github.com/thatoddmailbox/go-ico"
)

func main() {
    // Open the ICO file
    file, err := os.Open("favicon.ico")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    // Decode the ICO file
    icoFile, err := ico.Decode(file)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Successfully decoded ICO with %d images\n", len(icoFile.Images))

    // Access the first image
    if len(icoFile.Images) > 0 {
        img := icoFile.Images[0]
        bounds := img.Bounds()
        fmt.Printf("First image: %dx%d pixels\n", bounds.Dx(), bounds.Dy())
    }
}
```

### Saving Images as PNG

```go
import (
    "image/png"
    "os"

    "github.com/thatoddmailbox/go-ico"
)

func saveICOAsPNG(icoPath, pngPath string) error {
    // Open and decode ICO
    file, err := os.Open(icoPath)
    if err != nil {
        return err
    }
    defer file.Close()

    icoFile, err := ico.Decode(file)
    if err != nil {
        return err
    }

    // Get the best image
    img := icoFile.GetBestImage()
    if img == nil {
        return fmt.Errorf("no images found in ICO")
    }

    // Save as PNG
    outFile, err := os.Create(pngPath)
    if err != nil {
        return err
    }
    defer outFile.Close()

    return png.Encode(outFile, img)
}
```

## Getting Metadata Only

When you only need information about the ICO file without decoding the actual images, use `DecodeConfig` for better performance:

```go
func analyzeICO(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    // Get metadata without decoding images
    config, err := ico.DecodeConfig(file)
    if err != nil {
        return err
    }

    fmt.Printf("File: %s\n", filename)
    fmt.Printf("  Images: %d\n", config.Count)
    fmt.Printf("  Largest: %dx%d\n", config.Width, config.Height)

    return nil
}
```

## Working with Multiple Images

ICO files often contain multiple images at different sizes. Here's how to work with all of them:

```go
func listAllImages(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    icoFile, err := ico.Decode(file)
    if err != nil {
        return err
    }

    fmt.Printf("Images in %s:\n", filename)

    // List all available sizes
    sizes := icoFile.GetAvailableSizes()
    for i, size := range sizes {
        entry := icoFile.Entries[i]
        fmt.Printf("  %d: %dx%d (%d bpp, %d bytes)\n",
            i+1, size.X, size.Y, entry.BitsPerPixel, entry.Size)
    }

    return nil
}
```

### Extracting All Images

```go
func extractAllImages(icoPath string) error {
    file, err := os.Open(icoPath)
    if err != nil {
        return err
    }
    defer file.Close()

    icoFile, err := ico.Decode(file)
    if err != nil {
        return err
    }

    // Extract each image
    for i, img := range icoFile.Images {
        bounds := img.Bounds()
        entry := icoFile.Entries[i]

        filename := fmt.Sprintf("image_%d_%dx%d_%dbpp.png",
            i+1, bounds.Dx(), bounds.Dy(), entry.BitsPerPixel)

        outFile, err := os.Create(filename)
        if err != nil {
            fmt.Printf("Failed to create %s: %v\n", filename, err)
            continue
        }

        err = png.Encode(outFile, img)
        outFile.Close()

        if err != nil {
            fmt.Printf("Failed to encode %s: %v\n", filename, err)
        } else {
            fmt.Printf("Saved: %s\n", filename)
        }
    }

    return nil
}
```

## Finding Specific Image Sizes

### Getting the Best Image

The "best" image is the one with the highest resolution:

```go
func getBestIcon(icoPath string) (image.Image, error) {
    file, err := os.Open(icoPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    icoFile, err := ico.Decode(file)
    if err != nil {
        return nil, err
    }

    return icoFile.GetBestImage(), nil
}
```

### Finding Images by Size

Use `GetImageBySize()` to find the image closest to your desired dimensions:

```go
func getIconForSize(icoPath string, width, height int) (image.Image, error) {
    file, err := os.Open(icoPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    icoFile, err := ico.Decode(file)
    if err != nil {
        return nil, err
    }

    img := icoFile.GetImageBySize(width, height)
    if img == nil {
        return nil, fmt.Errorf("no suitable image found")
    }

    return img, nil
}

// Usage examples
func main() {
    // Get image closest to 16x16 (common favicon size)
    favicon16, _ := getIconForSize("app.ico", 16, 16)

    // Get image closest to 32x32 (common icon size)
    icon32, _ := getIconForSize("app.ico", 32, 32)

    // Get image closest to 48x48 (common Windows icon)
    icon48, _ := getIconForSize("app.ico", 48, 48)
}
```

## Common Use Cases

### Web Favicon Extraction

```go
import (
    "image"
    "net/http"

    "github.com/thatoddmailbox/go-ico"
)

func extractFavicon(icoPath string) (image.Image, error) {
    file, err := os.Open(icoPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    icoFile, err := ico.Decode(file)
    if err != nil {
        return nil, err
    }

    // Try common favicon sizes in order of preference
    preferredSizes := [][]int{
        {32, 32},  // Most common
        {16, 16},  // Classic favicon
        {24, 24},  // Alternative
        {48, 48},  // Larger alternative
    }

    for _, size := range preferredSizes {
        img := icoFile.GetImageBySize(size[0], size[1])
        bounds := img.Bounds()

        // Check if we got an exact or close match
        if bounds.Dx() <= size[0]+8 && bounds.Dy() <= size[1]+8 {
            return img, nil
        }
    }

    // Fallback to best available
    return icoFile.GetBestImage(), nil
}
```

### Creating a Favicon Server

```go
func faviconHandler(w http.ResponseWriter, r *http.Request) {
    // Extract favicon from ICO file
    img, err := extractFavicon("static/favicon.ico")
    if err != nil {
        http.Error(w, "Favicon not found", http.StatusNotFound)
        return
    }

    // Serve as PNG
    w.Header().Set("Content-Type", "image/png")
    w.Header().Set("Cache-Control", "public, max-age=31536000")

    if err := png.Encode(w, img); err != nil {
        http.Error(w, "Failed to encode image", http.StatusInternalServerError)
    }
}

func main() {
    http.HandleFunc("/favicon.png", faviconHandler)
    http.ListenAndServe(":8080", nil)
}
```

### Desktop Application Icon Loading

```go
import (
    "image"

    "github.com/thatoddmailbox/go-ico"
)

type IconSet struct {
    Small  image.Image  // 16x16
    Medium image.Image  // 32x32
    Large  image.Image  // 48x48
}

func loadApplicationIcons(icoPath string) (*IconSet, error) {
    file, err := os.Open(icoPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    icoFile, err := ico.Decode(file)
    if err != nil {
        return nil, err
    }

    return &IconSet{
        Small:  icoFile.GetImageBySize(16, 16),
        Medium: icoFile.GetImageBySize(32, 32),
        Large:  icoFile.GetImageBySize(48, 48),
    }, nil
}
```

## Error Handling

### Robust ICO Processing

```go
import (
    "errors"
    "fmt"
    "os"
    "strings"
)

func processICOSafely(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        if os.IsNotExist(err) {
            return fmt.Errorf("ICO file not found: %s", filename)
        }
        return fmt.Errorf("cannot open ICO file: %w", err)
    }
    defer file.Close()

    // Try to get config first (faster, less likely to fail)
    config, err := ico.DecodeConfig(file)
    if err != nil {
        return fmt.Errorf("invalid ICO file %s: %w", filename, err)
    }

    if config.Count == 0 {
        return errors.New("ICO file contains no images")
    }

    fmt.Printf("ICO file %s: %d images, largest %dx%d\n",
        filename, config.Count, config.Width, config.Height)

    // Reset file position for full decode
    file.Seek(0, 0)

    // Decode full file
    icoFile, err := ico.Decode(file)
    if err != nil {
        // Handle specific error types
        switch {
        case strings.Contains(err.Error(), "unsupported"):
            return fmt.Errorf("unsupported ICO format in %s: %w", filename, err)
        case strings.Contains(err.Error(), "truncated"):
            return fmt.Errorf("corrupted ICO file %s: %w", filename, err)
        default:
            return fmt.Errorf("failed to decode ICO %s: %w", filename, err)
        }
    }

    // Process images...
    for i, img := range icoFile.Images {
        if img == nil {
            fmt.Printf("Warning: image %d failed to decode\n", i+1)
            continue
        }

        bounds := img.Bounds()
        fmt.Printf("  Image %d: %dx%d\n", i+1, bounds.Dx(), bounds.Dy())
    }

    return nil
}
```

### Validation Helper

```go
func validateICO(filename string) (bool, error) {
    file, err := os.Open(filename)
    if err != nil {
        return false, err
    }
    defer file.Close()

    // Quick validation using DecodeConfig
    _, err = ico.DecodeConfig(file)
    return err == nil, err
}
```

## Performance Tips

### 1. Use DecodeConfig When Possible

```go
// Fast: Only reads header and directory entries
config, err := ico.DecodeConfig(file)

// Slower: Decodes all image data
icoFile, err := ico.Decode(file)
```

### 2. Process Large Files Efficiently

```go
func processLargeICO(filename string) error {
    // First check if it's worth processing
    config, err := ico.DecodeConfig(file)
    if err != nil {
        return err
    }

    // Skip files with too many images or very large images
    if config.Count > 20 {
        return fmt.Errorf("ICO has too many images: %d", config.Count)
    }
    if config.Width > 512 || config.Height > 512 {
        return fmt.Errorf("ICO images too large: %dx%d", config.Width, config.Height)
    }

    // Now safe to decode
    file.Seek(0, 0)
    icoFile, err := ico.Decode(file)
    // ... process
}
```

### 3. Reuse Decoded Data

```go
type ICOCache struct {
    file *ico.ICO
    best image.Image
    bySize map[string]image.Image
}

func NewICOCache(filename string) (*ICOCache, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    icoFile, err := ico.Decode(file)
    if err != nil {
        return nil, err
    }

    cache := &ICOCache{
        file:   icoFile,
        best:   icoFile.GetBestImage(),
        bySize: make(map[string]image.Image),
    }

    return cache, nil
}

func (c *ICOCache) GetImageBySize(width, height int) image.Image {
    key := fmt.Sprintf("%dx%d", width, height)

    if img, exists := c.bySize[key]; exists {
        return img
    }

    img := c.file.GetImageBySize(width, height)
    c.bySize[key] = img
    return img
}
```

## Complete Examples

### Command-Line ICO Converter

```go
package main

import (
    "flag"
    "fmt"
    "image/png"
    "os"
    "path/filepath"
    "strconv"
    "strings"

    "github.com/thatoddmailbox/go-ico"
)

func main() {
    var (
        outputDir = flag.String("o", ".", "Output directory")
        size      = flag.String("size", "", "Specific size to extract (e.g., '32x32')")
        best      = flag.Bool("best", false, "Extract only the best image")
        list      = flag.Bool("list", false, "List available sizes")
    )
    flag.Parse()

    if flag.NArg() == 0 {
        fmt.Println("Usage: ico-convert [options] <ico-file> [ico-file...]")
        flag.PrintDefaults()
        return
    }

    for _, filename := range flag.Args() {
        if err := processFile(filename, *outputDir, *size, *best, *list); err != nil {
            fmt.Printf("Error processing %s: %v\n", filename, err)
        }
    }
}

func processFile(filename, outputDir, sizeSpec string, bestOnly, listOnly bool) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    if listOnly {
        return listImages(file, filename)
    }

    icoFile, err := ico.Decode(file)
    if err != nil {
        return err
    }

    baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))

    if bestOnly {
        return saveBestImage(icoFile, outputDir, baseName)
    }

    if sizeSpec != "" {
        return saveSpecificSize(icoFile, outputDir, baseName, sizeSpec)
    }

    return saveAllImages(icoFile, outputDir, baseName)
}

// ... implementation functions ...
```

### Web Service for ICO Processing

```go
package main

import (
    "encoding/json"
    "fmt"
    "image/png"
    "net/http"
    "strconv"
    "io"

    "github.com/thatoddmailbox/go-ico"
)

type ICOInfo struct {
    Count   int    `json:"count"`
    Width   int    `json:"width"`
    Height  int    `json:"height"`
    Images  []ImageInfo `json:"images"`
}

type ImageInfo struct {
    Width  int `json:"width"`
    Height int `json:"height"`
    BPP    int `json:"bpp"`
    Size   int `json:"size"`
}

func main() {
    http.HandleFunc("/analyze", analyzeHandler)
    http.HandleFunc("/extract", extractHandler)

    fmt.Println("ICO service running on :8080")
    http.ListenAndServe(":8080", nil)
}

func analyzeHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "POST required", http.StatusMethodNotAllowed)
        return
    }

    // Read uploaded file
    file, _, err := r.FormFile("ico")
    if err != nil {
        http.Error(w, "No ICO file provided", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Decode ICO
    icoFile, err := ico.Decode(file)
    if err != nil {
        http.Error(w, "Invalid ICO file", http.StatusBadRequest)
        return
    }

    // Build response
    info := ICOInfo{
        Count:  len(icoFile.Images),
        Images: make([]ImageInfo, len(icoFile.Images)),
    }

    sizes := icoFile.GetAvailableSizes()
    for i, size := range sizes {
        entry := icoFile.Entries[i]
        info.Images[i] = ImageInfo{
            Width:  size.X,
            Height: size.Y,
            BPP:    int(entry.BitsPerPixel),
            Size:   int(entry.Size),
        }

        if size.X*size.Y > info.Width*info.Height {
            info.Width = size.X
            info.Height = size.Y
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(info)
}

func extractHandler(w http.ResponseWriter, r *http.Request) {
    // ... similar implementation for extracting images ...
}
```

This usage guide covers the most common scenarios you'll encounter when working with ICO files in Go. The library is designed to be simple to use while providing the flexibility needed for various applications.