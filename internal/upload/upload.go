package upload

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
)

// MaxFileSize is the maximum allowed file size (5MB).
const MaxFileSize = 5 * 1024 * 1024

// AllowedMIMETypes maps MIME types to their canonical file extensions.
var AllowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// DetectMIME reads the first 512 bytes of the file to detect its actual MIME type.
// This is more reliable than trusting the client-provided Content-Type header.
func DetectMIME(file multipart.File) (string, error) {
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("파일 읽기 실패: %w", err)
	}

	// Seek back to beginning so the file can be saved later
	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return "", fmt.Errorf("파일 탐색 실패: %w", err)
		}
	}

	return http.DetectContentType(buf[:n]), nil
}

// ValidateAndGenerateFilename validates the file size, detects the real MIME type,
// and returns a UUID-based filename with the correct extension.
func ValidateAndGenerateFilename(header *multipart.FileHeader, file multipart.File) (string, error) {
	if header.Size > MaxFileSize {
		return "", fmt.Errorf("파일 크기는 5MB 이하여야 합니다")
	}

	detectedMIME, err := DetectMIME(file)
	if err != nil {
		return "", err
	}

	ext, ok := AllowedMIMETypes[detectedMIME]
	if !ok {
		return "", fmt.Errorf("허용되는 파일 형식: jpeg, png, gif, webp")
	}

	// Prefer original extension if it matches an allowed type
	origExt := filepath.Ext(header.Filename)
	for _, allowedExt := range AllowedMIMETypes {
		if origExt == allowedExt {
			ext = origExt
			break
		}
	}

	return fmt.Sprintf("%s%s", uuid.New().String(), ext), nil
}
