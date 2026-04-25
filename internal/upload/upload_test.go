package upload

import (
	"bytes"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"testing"
)

// mockFile implements multipart.File backed by a bytes.Reader.
type mockFile struct {
	*bytes.Reader
}

func (m *mockFile) Close() error                                         { return nil }
func (m *mockFile) ReadAt(p []byte, off int64) (int, error)             { return m.Reader.ReadAt(p, off) }

func newMockFile(data []byte) *mockFile {
	return &mockFile{bytes.NewReader(data)}
}

// jpegHeader is the magic bytes for a JPEG file.
var jpegHeader = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F', 0}

// pngHeader is the magic bytes for a PNG file.
var pngHeader = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}

func makeFileData(magic []byte, size int) []byte {
	data := make([]byte, size)
	copy(data, magic)
	return data
}

// --- DetectMIME ---

func TestDetectMIME_JPEG(t *testing.T) {
	f := newMockFile(makeFileData(jpegHeader, 512))
	mime, err := DetectMIME(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %q", mime)
	}
	// Confirm seeker rewound
	pos, _ := f.Seek(0, io.SeekCurrent)
	if pos != 0 {
		t.Errorf("expected file position 0 after DetectMIME, got %d", pos)
	}
}

func TestDetectMIME_PNG(t *testing.T) {
	f := newMockFile(makeFileData(pngHeader, 512))
	mime, err := DetectMIME(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "image/png" {
		t.Errorf("expected image/png, got %q", mime)
	}
}

func TestDetectMIME_ShortFile(t *testing.T) {
	f := newMockFile(jpegHeader)
	mime, err := DetectMIME(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime == "" {
		t.Error("expected non-empty MIME for short file")
	}
}

func TestDetectMIME_EmptyFile(t *testing.T) {
	f := newMockFile([]byte{})
	_, err := DetectMIME(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ValidateAndGenerateFilename ---

func newHeader(filename string, size int64) *multipart.FileHeader {
	return &multipart.FileHeader{
		Filename: filename,
		Size:     size,
	}
}

func TestValidateAndGenerateFilename_JPEG_Success(t *testing.T) {
	header := newHeader("photo.jpg", 1024)
	f := newMockFile(makeFileData(jpegHeader, 512))

	name, err := ValidateAndGenerateFilename(header, f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Ext(name) != ".jpg" {
		t.Errorf("expected .jpg extension, got %q", filepath.Ext(name))
	}
	// UUID part should be 36 chars + extension
	withoutExt := strings.TrimSuffix(name, filepath.Ext(name))
	if len(withoutExt) != 36 {
		t.Errorf("expected UUID length 36, got %d", len(withoutExt))
	}
}

func TestValidateAndGenerateFilename_PNG_Success(t *testing.T) {
	header := newHeader("image.png", 512)
	f := newMockFile(makeFileData(pngHeader, 512))

	name, err := ValidateAndGenerateFilename(header, f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Ext(name) != ".png" {
		t.Errorf("expected .png extension, got %q", filepath.Ext(name))
	}
}

func TestValidateAndGenerateFilename_FileTooLarge(t *testing.T) {
	header := newHeader("big.jpg", MaxFileSize+1)
	f := newMockFile(makeFileData(jpegHeader, 512))

	_, err := ValidateAndGenerateFilename(header, f)
	if err == nil {
		t.Error("expected error for oversized file")
	}
	if !strings.Contains(err.Error(), "5MB") {
		t.Errorf("expected size error message, got %q", err.Error())
	}
}

func TestValidateAndGenerateFilename_ExactMaxSize(t *testing.T) {
	header := newHeader("ok.jpg", MaxFileSize)
	f := newMockFile(makeFileData(jpegHeader, 512))

	_, err := ValidateAndGenerateFilename(header, f)
	if err != nil {
		t.Fatalf("unexpected error for file at exactly MaxFileSize: %v", err)
	}
}

func TestValidateAndGenerateFilename_DisallowedMIME(t *testing.T) {
	header := newHeader("doc.txt", 100)
	f := newMockFile([]byte("hello world plain text content"))

	_, err := ValidateAndGenerateFilename(header, f)
	if err == nil {
		t.Error("expected error for disallowed MIME type")
	}
}

func TestValidateAndGenerateFilename_OrigExtPreferred(t *testing.T) {
	// MIME detects jpeg but original extension is .jpeg — still allowed
	header := newHeader("photo.jpeg", 512)
	f := newMockFile(makeFileData(jpegHeader, 512))

	name, err := ValidateAndGenerateFilename(header, f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ext := filepath.Ext(name)
	if ext != ".jpg" && ext != ".jpeg" {
		t.Errorf("expected .jpg or .jpeg extension, got %q", ext)
	}
}

func TestValidateAndGenerateFilename_UniqueNames(t *testing.T) {
	header := newHeader("a.png", 100)

	name1, _ := ValidateAndGenerateFilename(header, newMockFile(makeFileData(pngHeader, 512)))
	name2, _ := ValidateAndGenerateFilename(header, newMockFile(makeFileData(pngHeader, 512)))

	if name1 == name2 {
		t.Error("expected unique filenames for each call")
	}
}

// --- Constants/vars ---

func TestMaxFileSizeValue(t *testing.T) {
	if MaxFileSize != 5*1024*1024 {
		t.Errorf("expected MaxFileSize 5MB, got %d", MaxFileSize)
	}
}

func TestAllowedMIMETypes_ContainsExpected(t *testing.T) {
	expected := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	for _, mime := range expected {
		if _, ok := AllowedMIMETypes[mime]; !ok {
			t.Errorf("expected %q in AllowedMIMETypes", mime)
		}
	}
}
