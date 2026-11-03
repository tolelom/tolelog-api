package upload

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/google/uuid"
)

// publicURLPrefix 는 라우터가 업로드 디렉터리를 내보내는 경로다.
const publicURLPrefix = "/uploads/images/"

func PublicURL(filename string) string {
	return publicURLPrefix + filename
}

const MaxFileSize = 5 * 1024 * 1024

var AllowedMIMETypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// DetectMIME 은 클라이언트가 보낸 Content-Type 대신 파일 앞 512바이트로 형식을 판단한다.
func DetectMIME(file multipart.File) (string, error) {
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("파일 읽기 실패: %w", err)
	}

	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return "", fmt.Errorf("파일 탐색 실패: %w", err)
		}
	}

	return http.DetectContentType(buf[:n]), nil
}

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

	// 정적 서빙은 확장자로 Content-Type 을 정하므로, 확장자는 클라이언트 파일명이 아니라 감지한 형식을 따른다.
	return fmt.Sprintf("%s%s", uuid.New().String(), ext), nil
}
