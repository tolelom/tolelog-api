# Tolelog Backend

Tolelog 블로그 플랫폼의 Go Fiber 기반 RESTful API 서버.
프론트엔드는 [tolelog](https://github.com/tolelom/tolelog) 참고.

## Tech Stack

- **Framework**: Go Fiber v2
- **Database**: MySQL + GORM
- **Cache**: Redis (선택, 미연결 시 캐시 없이 동작)
- **Auth**: JWT (Access 15분 + Refresh 7일)
- **Docs**: Swagger (개발 환경에서만 노출)
- **Deploy**: GitHub Actions → Docker → GHCR → SSH 배포

## Features

- 글 CRUD, 검색, 페이지네이션, 태그 필터링
- 임시저장 (drafts)
- 댓글 / 대댓글 (트리 구조)
- 좋아요
- 시리즈 (글 모음, 순서 변경, 네비게이션)
- 이미지 업로드 (MIME 검증, 5MB 제한)
- RSS 피드, XML 사이트맵
- Redis 캐싱, Rate Limiting, 보안 헤더, CORS
- gzip 압축, 구조화된 로깅 (slog), Graceful Shutdown

## Getting Started

```bash
git clone https://github.com/tolelom/tolelog-backend.git
cd tolelog-backend

cp .env_example .env
# .env 파일에 DB, JWT_SECRET 등 설정

go run cmd/server/main.go
```

## Development

```bash
# Hot reload
go install github.com/cosmtrek/air@latest
air

# Swagger 생성
swag init -g cmd/server/main.go

# 테스트
go test -v -cover ./...

# 린트
golangci-lint run
```

API 상세는 개발 서버의 `/swagger/` 참고.

## License

Personal and educational use.
