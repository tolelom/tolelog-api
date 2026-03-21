# Fiber API Server

Tolelog 블로그 플랫폼의 Go Fiber 기반 RESTful API 서버입니다. JWT 인증, GORM/MySQL, Redis 캐싱, Swagger API 문서화를 지원합니다.

## Tech Stack

- **Framework**: Go Fiber v2
- **Database**: MySQL with GORM (soft deletes, auto-migration)
- **Cache**: Redis (선택 사항, 미연결 시 캐시 없이 동작)
- **Authentication**: JWT (golang-jwt/jwt) — Access + Refresh Token
- **Validation**: go-playground/validator
- **Documentation**: Swagger (swaggo)
- **Logging**: log/slog (구조화된 로깅)
- **Configuration**: godotenv
- **Compression**: gzip 응답 압축

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go                 # 엔트리포인트, Swagger 메타 주석
├── internal/
│   ├── cache/                      # Redis 캐시 래퍼 (JSON 직렬화, TTL, 패턴 삭제)
│   ├── comment/                    # 댓글 도메인 (대댓글 지원, 트리 구조)
│   ├── config/                     # 환경변수 로딩, DB 초기화, auto-migration
│   ├── dto/                        # 요청/응답 DTO, 유효성 검증 규칙
│   ├── feed/                       # RSS 2.0 피드 생성
│   ├── image/                      # 이미지 업로드 (multipart, MIME 검증)
│   ├── middleware/                  # 인증, 보안 헤더, 요청 로깅, 에러 핸들러
│   ├── model/                      # GORM 엔티티 (User, Post, Comment, Tag, Series, Like)
│   ├── post/                       # 글 도메인 (CRUD, 검색, 페이지네이션, 태그 필터링, 좋아요)
│   ├── router/                     # 라우트 등록, DI 설정, 미들웨어 스택
│   ├── series/                     # 시리즈 도메인 (CRUD, 글 추가/제거/순서 변경, 네비게이션)
│   ├── sitemap/                    # XML 사이트맵 생성 (sitemaps.org 프로토콜)
│   ├── tag/                        # 태그 도메인 (사용 횟수 기반 목록)
│   ├── upload/                     # 파일 타입 검증, UUID 파일명 생성 (5MB 제한)
│   ├── user/                       # 사용자/인증 도메인 (회원가입, 로그인, 토큰 갱신, 프로필, 비밀번호 변경)
│   ├── utils/                      # JWT 토큰 생성/검증, bcrypt 비밀번호 유틸
│   └── validate/                   # 공유 validator 인스턴스, 커스텀 규칙
├── docs/                           # Swagger 자동생성 파일 (직접 편집 금지)
├── .env_example                    # 환경변수 템플릿
└── go.mod                          # Go 모듈 의존성
```

## Features

- **인증**: 회원가입, 로그인, JWT 토큰 갱신 (Access 15분 + Refresh 7일), 비밀번호 변경
- **글 관리**: CRUD, 공개/비공개 설정, 소유권 검증
- **검색**: 제목/본문 키워드 검색 (2~100자)
- **태그**: 다대다 관계 (정규화된 태그 테이블), 태그별 필터링, 인기순/이름순 태그 목록
- **댓글**: 작성/수정/삭제, 대댓글 (트리 구조)
- **좋아요**: 토글 방식, 좋아요 상태/수 조회
- **시리즈**: 글 모음 관리 (CRUD, 글 추가/제거, 순서 변경, 시리즈 네비게이션)
- **이미지 업로드**: MIME 검증 (jpeg/png/gif/webp), UUID 파일명, 5MB 제한
- **RSS 피드**: 최근 공개 글 20개 RSS 2.0 피드
- **사이트맵**: 공개 글 + 시리즈 XML 사이트맵
- **캐싱**: Redis 기반 글 캐싱 (TTL, 패턴 삭제로 무효화)
- **보안**: 보안 헤더 (X-Content-Type-Options, X-Frame-Options, CSP 등), Rate Limiting (인증 10req/min), CORS
- **Swagger**: 전체 API 문서 자동 생성 (개발 환경에서만 노출)
- **운영**: gzip 압축, 구조화된 로깅 (slog), Graceful Shutdown

## Prerequisites

- Go 1.25 or higher
- MySQL database
- Redis (선택 — 미연결 시 캐시 없이 정상 동작)

## Installation

1. Clone the repository

```bash
git clone https://github.com/tolelom/fiber_api_server.git
cd fiber_api_server
```

2. Install dependencies

```bash
go mod download
```

3. Set up environment variables

```bash
cp .env_example .env
```

Edit `.env` file with your configuration:

```env
PORT=8080
DB_USER=your_db_user
DB_PASSWORD=your_db_password
DB_HOST=localhost
DB_PORT=3306
DB_NAME=your_db_name
JWT_SECRET=your_jwt_secret_key
ENVIRONMENT=development
REDIS_ADDR=localhost:6379
UPLOAD_DIR=./uploads/images
```

| Variable | Default | Notes |
|----------|---------|-------|
| `PORT` | 8080 | 서버 포트 |
| `DB_HOST` | localhost | MySQL 호스트 |
| `DB_PORT` | 3306 | MySQL 포트 |
| `DB_USER` | root | MySQL 사용자 |
| `DB_PASSWORD` | root | MySQL 비밀번호 |
| `DB_NAME` | blog | MySQL 데이터베이스명 |
| `JWT_SECRET` | (개발: 자동생성) | **프로덕션에서 반드시 설정** |
| `ENVIRONMENT` | development | `development` 또는 `production` |
| `REDIS_ADDR` | localhost:6379 | Redis 주소 (선택, 미연결 시 캐시 비활성화) |
| `UPLOAD_DIR` | ./uploads/images | 이미지 저장 경로 |

4. Run the server

```bash
go run cmd/server/main.go
```

The server will start on the port specified in your `.env` file.

## API Endpoints

All API routes are under `/api/v1`.

### Authentication (Rate Limited: 10req/min)

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `POST` | `/auth/register` | 회원가입 | - |
| `POST` | `/auth/login` | 로그인 | - |
| `POST` | `/auth/refresh` | 토큰 갱신 | - |

### Posts

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `GET` | `/posts` | 공개 글 목록 (페이지네이션, 태그 필터링) | - |
| `GET` | `/posts/search?q=` | 글 검색 (2~100자) | - |
| `GET` | `/posts/:id` | 글 상세 조회 (비공개 글은 작성자만) | Optional |
| `POST` | `/posts` | 글 생성 | Required |
| `PUT` | `/posts/:id` | 글 수정 | Required |
| `PATCH` | `/posts/:id` | 글 수정 (부분) | Required |
| `DELETE` | `/posts/:id` | 글 삭제 | Required |
| `POST` | `/posts/:id/like` | 좋아요 토글 | Required |
| `GET` | `/posts/:id/like` | 좋아요 상태 조회 | Optional |

### Comments

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `GET` | `/posts/:id/comments` | 댓글 목록 (트리 구조) | - |
| `POST` | `/posts/:id/comments` | 댓글 작성 (대댓글: parent_id 지정) | Required |
| `PUT` | `/posts/:id/comments/:comment_id` | 댓글 수정 | Required |
| `DELETE` | `/posts/:id/comments/:comment_id` | 댓글 삭제 | Required |

### Series

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `GET` | `/series/:id` | 시리즈 상세 (글 목록 포함) | - |
| `POST` | `/series` | 시리즈 생성 | Required |
| `PUT` | `/series/:id` | 시리즈 수정 | Required |
| `DELETE` | `/series/:id` | 시리즈 삭제 | Required |
| `POST` | `/series/:id/posts` | 시리즈에 글 추가 | Required |
| `DELETE` | `/series/:id/posts/:post_id` | 시리즈에서 글 제거 | Required |
| `PUT` | `/series/:id/reorder` | 시리즈 글 순서 변경 | Required |
| `GET` | `/posts/:id/series-nav` | 시리즈 이전/다음 글 네비게이션 | - |

### Tags

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `GET` | `/tags` | 태그 목록 (sort=popular/name, limit) | - |

### Users

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `GET` | `/users/:user_id` | 사용자 프로필 | - |
| `GET` | `/users/:user_id/posts` | 사용자 글 목록 (비공개 포함 여부) | Optional |
| `GET` | `/users/:user_id/series` | 사용자 시리즈 목록 | - |
| `PUT` | `/users/avatar` | 프로필 이미지 업로드 | Required |
| `PUT` | `/users/password` | 비밀번호 변경 | Required |

### Upload

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| `POST` | `/upload` | 이미지 업로드 (5MB, jpeg/png/gif/webp) | Required |

### Other

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/feed` | RSS 2.0 피드 |
| `GET` | `/sitemap.xml` | XML 사이트맵 |
| `GET` | `/swagger/*` | Swagger UI (개발 환경만) |

## Authentication

Protected endpoints require a JWT token in the Authorization header:

```
Authorization: Bearer <your_jwt_token>
```

- Access Token: 15분 만료
- Refresh Token: 7일 만료
- 토큰 발급: `/api/v1/auth/login`
- 토큰 갱신: `/api/v1/auth/refresh`

## Development

### Run with hot reload

```bash
go install github.com/cosmtrek/air@latest
air
```

### Generate Swagger documentation

```bash
swag init -g cmd/server/main.go
```

### Run tests

```bash
go test -v ./...
go test -v -cover ./...
```

### Lint

```bash
golangci-lint run
```

## CI/CD

GitHub Actions (`.github/workflows/deploy_backend.yml`): push to main → lint → test → build (linux/arm64) → Docker image → GHCR → SSH deploy + docker-compose.

## License

This project is available for personal and educational use.
