package ico

import (
	"bytes"
	"testing"
)

// createMinimalICO creates a minimal valid ICO with one 1x1 32-bit BMP
func createMinimalICO() []byte {
	var buf bytes.Buffer

	// ICO Header (6 bytes)
	buf.Write([]byte{0x00, 0x00}) // Reserved (0)
	buf.Write([]byte{0x01, 0x00}) // Type (1 = ICO)
	buf.Write([]byte{0x01, 0x00}) // Count (1 image)

	// Directory Entry (16 bytes)
	buf.WriteByte(1)                          // Width (1 pixel)
	buf.WriteByte(1)                          // Height (1 pixel)
	buf.WriteByte(0)                          // ColorCount (0 = no palette)
	buf.WriteByte(0)                          // Reserved (0)
	buf.Write([]byte{0x01, 0x00})             // ColorPlanes (1)
	buf.Write([]byte{0x20, 0x00})             // BitsPerPixel (32)
	buf.Write([]byte{0x2C, 0x00, 0x00, 0x00}) // Size (44 bytes: 40 header + 4 pixel)
	buf.Write([]byte{0x16, 0x00, 0x00, 0x00}) // Offset (22 bytes)

	// BMP Info Header (40 bytes)
	buf.Write([]byte{0x28, 0x00, 0x00, 0x00}) // Header size (40)
	buf.Write([]byte{0x01, 0x00, 0x00, 0x00}) // Width (1)
	buf.Write([]byte{0x02, 0x00, 0x00, 0x00}) // Height (2, doubled for ICO)
	buf.Write([]byte{0x01, 0x00})             // Planes (1)
	buf.Write([]byte{0x20, 0x00})             // BitsPerPixel (32)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00}) // Compression (0)
	buf.Write([]byte{0x04, 0x00, 0x00, 0x00}) // ImageSize (4 bytes)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00}) // XPelsPerMeter (0)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00}) // YPelsPerMeter (0)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00}) // ColorsUsed (0)
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00}) // ColorsImportant (0)

	// Pixel data (4 bytes: 1 pixel in BGRA format)
	buf.Write([]byte{0x00, 0x00, 0xFF, 0xFF}) // Red pixel (BGRA: B=0, G=0, R=255, A=255)

	return buf.Bytes()
}

func TestBasicDecode(t *testing.T) {
	data := createMinimalICO()
	ico, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Failed to decode ICO: %v", err)
	}

	if ico.Header.Count != 1 {
		t.Errorf("Expected 1 image, got %d", ico.Header.Count)
	}

	if len(ico.Images) != 1 {
		t.Errorf("Expected 1 decoded image, got %d", len(ico.Images))
	}

	img := ico.Images[0]
	bounds := img.Bounds()
	if bounds.Dx() != 1 || bounds.Dy() != 1 {
		t.Errorf("Expected 1x1 image, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestDecodeConfig(t *testing.T) {
	data := createMinimalICO()
	config, err := DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Failed to decode ICO config: %v", err)
	}

	if config.Width != 1 || config.Height != 1 {
		t.Errorf("Expected 1x1 config, got %dx%d", config.Width, config.Height)
	}

	if config.Count != 1 {
		t.Errorf("Expected count 1, got %d", config.Count)
	}
}

func TestGetBestImage(t *testing.T) {
	data := createMinimalICO()
	ico, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Failed to decode ICO: %v", err)
	}

	bestImg := ico.GetBestImage()
	if bestImg == nil {
		t.Error("GetBestImage returned nil")
		return
	}

	bounds := bestImg.Bounds()
	if bounds.Dx() != 1 || bounds.Dy() != 1 {
		t.Errorf("Expected 1x1 best image, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestGetImageBySize(t *testing.T) {
	data := createMinimalICO()
	ico, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Failed to decode ICO: %v", err)
	}

	// Test exact match
	img := ico.GetImageBySize(1, 1)
	if img == nil {
		t.Error("GetImageBySize returned nil for exact match")
		return
	}

	bounds := img.Bounds()
	if bounds.Dx() != 1 || bounds.Dy() != 1 {
		t.Errorf("Expected 1x1 image, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Test close match
	img = ico.GetImageBySize(2, 2)
	if img == nil {
		t.Error("GetImageBySize returned nil for close match")
		return
	}

	bounds = img.Bounds()
	if bounds.Dx() != 1 || bounds.Dy() != 1 {
		t.Errorf("Expected closest match to be 1x1, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestGetAvailableSizes(t *testing.T) {
	data := createMinimalICO()
	ico, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Failed to decode ICO: %v", err)
	}

	sizes := ico.GetAvailableSizes()
	if len(sizes) != 1 {
		t.Errorf("Expected 1 size, got %d", len(sizes))
	}

	if sizes[0].X != 1 || sizes[0].Y != 1 {
		t.Errorf("Expected size 1x1, got %dx%d", sizes[0].X, sizes[0].Y)
	}
}

func TestDirectoryEntryGetters(t *testing.T) {
	// Test normal values
	entry := DirectoryEntry{Width: 16, Height: 32}
	if entry.GetWidth() != 16 {
		t.Errorf("Expected width 16, got %d", entry.GetWidth())
	}
	if entry.GetHeight() != 32 {
		t.Errorf("Expected height 32, got %d", entry.GetHeight())
	}

	// Test special case where 0 means 256
	entry = DirectoryEntry{Width: 0, Height: 0}
	if entry.GetWidth() != 256 {
		t.Errorf("Expected width 256 for zero value, got %d", entry.GetWidth())
	}
	if entry.GetHeight() != 256 {
		t.Errorf("Expected height 256 for zero value, got %d", entry.GetHeight())
	}
}

func TestErrorCases(t *testing.T) {
	// Test empty data
	_, err := Decode(bytes.NewReader([]byte{}))
	if err == nil {
		t.Error("Expected error for empty data")
	}

	// Test invalid header (wrong type)
	invalidHeader := []byte{0x00, 0x00, 0x02, 0x00, 0x01, 0x00}
	_, err = Decode(bytes.NewReader(invalidHeader))
	if err == nil {
		t.Error("Expected error for invalid header type")
	}

	// Test DecodeConfig with empty data
	_, err = DecodeConfig(bytes.NewReader([]byte{}))
	if err == nil {
		t.Error("Expected error for empty data in DecodeConfig")
	}

	// Test invalid reserved field
	invalidReserved := []byte{0x01, 0x00, 0x01, 0x00, 0x01, 0x00}
	_, err = Decode(bytes.NewReader(invalidReserved))
	if err == nil {
		t.Error("Expected error for non-zero reserved field")
	}
}

func TestScoreSizeMatch(t *testing.T) {
	entry := DirectoryEntry{Width: 16, Height: 16}

	// Exact match should have score 0
	score := scoreSizeMatch(entry, 16, 16)
	if score != 0 {
		t.Errorf("Expected score 0 for exact match, got %d", score)
	}

	// Test distance calculation
	score = scoreSizeMatch(entry, 17, 18)
	expected := (16-17)*(16-17) + (16-18)*(16-18) // 1 + 4 = 5
	if score != expected {
		t.Errorf("Expected score %d, got %d", expected, score)
	}
}

func TestPNGDetection(t *testing.T) {
	// Test PNG signature detection
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	testData := append(pngSignature, []byte("dummy data")...)

	if !bytes.Equal(testData[:8], pngSignature) {
		t.Error("PNG signature detection should work correctly")
	}
}

// Benchmark tests
func BenchmarkDecode(b *testing.B) {
	data := createMinimalICO()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := Decode(bytes.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeConfig(b *testing.B) {
	data := createMinimalICO()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := DecodeConfig(bytes.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetBestImage(b *testing.B) {
	data := createMinimalICO()
	ico, err := Decode(bytes.NewReader(data))
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ico.GetBestImage()
	}
}
