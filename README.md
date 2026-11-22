# Tolelog API

> 개인 블로그 플랫폼의 REST API 서버

Go 1.26 · Fiber v2 · GORM · MySQL · 서비스 https://blog.tolelom.xyz

웹 프론트엔드는 [tolelog-web](https://github.com/tolelom/tolelog-web)입니다.

## 기능

- 마크다운 글 작성·수정·삭제, 검색, 페이지 나눔, 태그 필터
- 공개 여부와 상태(발행, 임시저장)에 따른 조회 범위. 비공개 글과 임시저장 글은 작성자만 조회
- 댓글과 대댓글, 좋아요
- 시리즈: 글 묶기, 순서 변경, 이전/다음 이동
- 이미지와 아바타 업로드
- 회원가입, 로그인, 로그아웃, 비밀번호 변경(기존 세션 무효화), 계정 삭제
- RSS 피드와 XML 사이트맵

## 시작하기

Go 1.26과 MySQL이 필요합니다.

```bash
git clone https://github.com/tolelom/tolelog-api.git
cd tolelog-api
cp .env_example .env
go run ./cmd/server
```

| 접속 | 주소 |
|---|---|
| API | http://localhost:8080/api/v1 |
| Swagger UI(개발 환경) | http://localhost:8080/swagger/ |
| 헬스 체크 | http://localhost:8080/health |
| RSS 피드 | http://localhost:8080/feed |
| 사이트맵 | http://localhost:8080/sitemap.xml |

설정은 `.env`에서 읽습니다. 값을 비우면 localhost:3306 MySQL의 `blog` DB에 붙고, 시작할 때 테이블을 자동으로 만듭니다. 로컬 tolelog-web은 API를 localhost:3000으로 보므로 함께 띄울 때는 `PORT=3000`으로 둡니다.

## 문서

| 문서 | 내용 |
|---|---|
| `ARCHITECTURE.md` | 시스템 구성, 패키지 구성과 지켜야 할 규칙 |
| `docs/swagger.json` | API 계약(Swagger 2.0, swag 생성) |

## 기여

개발 명령, API 계약 변경 절차, 배포 흐름은 `CONTRIBUTING.md`에 있습니다.
