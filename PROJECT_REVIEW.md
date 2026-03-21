# Tolelog 블로그 서비스 종합 코드 리뷰

> 최초 리뷰: 2026-03-05
> 현행화: 2026-03-21
> 대상: Backend (`fiber_api_server`) + Frontend (`tolelog`)

---

## 1. 종합 점수표

| 평가 항목 | Backend | Frontend | 종합 | 비고 |
|-----------|---------|----------|------|------|
| **아키텍처 & 구조** | 8/10 | 8/10 | **8/10** | 양쪽 모두 깔끔한 레이어 분리 |
| **코드 품질** | 7/10 | 7/10 | **7/10** | Go 관용구 준수, JSX 일관성 양호 |
| **보안** | 6/10 | 6/10 | **6/10** | JWT 시크릿 문제 잔존, 보안 헤더 적용됨 |
| **테스트** | 4/10 | 1/10 | **2.5/10** | Backend 단위 테스트 추가됨, Frontend 여전히 부재 |
| **API 설계** | 8/10 | - | **8/10** | RESTful 준수, 응답 포맷 통일됨 |
| **UI/UX** | - | 8/10 | **8/10** | 다크모드, 반응형, 블록 에디터 |
| **성능** | 7/10 | 6/10 | **6.5/10** | Redis 캐싱 도입, gzip 압축 적용 |
| **배포 & 운영** | 6/10 | 6/10 | **6/10** | CI/CD 존재, 모니터링 없음 |
| **문서화** | 7/10 | 5/10 | **6/10** | Swagger 전체 재생성 완료, CLAUDE.md 상세 |
| **확장성** | 7/10 | 7/10 | **7/10** | 태그 M:N 정규화 완료, TypeScript 전환 완료 |
| | | | | |
| **총점** | | | **6.7/10** | |

---

## 2. 아키텍처 평가

### 2.1 Backend: 도메인 기반 레이어드 아키텍처

```
Handler → Service Interface → GORM/DB
```

**잘한 점:**
- `Handler → Service(Interface) → DB` 계층이 명확하게 분리됨
- 인터페이스 기반 DI로 테스트 용이한 구조 (실제 테스트는 없지만)
- DTO와 Model 분리로 내부 구조 노출 방지
- `internal/` 패키지 활용으로 외부 접근 차단

**아쉬운 점:**
- Repository 계층 없이 Service에서 GORM 직접 호출 → Service에 DB 로직이 혼재
- 도메인 에러가 Service 파일에 선언 → 별도 errors 패키지로 분리 권장

### 2.2 Frontend: 컴포넌트 기반 + Context API

```
Pages → Components → Context/Hooks → API Utils
```

**잘한 점:**
- `pages/`, `components/`, `context/`, `hooks/`, `utils/` 역할 분리 명확
- Context API로 Auth/Theme 전역 상태 관리
- 커스텀 훅 (`useAutoSave`) 활용

**아쉬운 점:**
- ~~TypeScript 미사용으로 타입 안정성 부족~~ **해결됨** — TypeScript 전환 완료 (`.ts`/`.tsx`)
- `AuthForm` 컴포넌트가 존재하나 미사용, `LoginBox`/`RegisterBox`가 로직 중복
- ~~페이지 레벨 코드 스플리팅 (`React.lazy`) 미적용~~ **해결됨** — `React.lazy` + `Suspense` 적용

---

## 3. 보안 평가 (가장 시급)

### 3.1 심각도: Critical

| 이슈 | 설명 | 위치 |
|------|------|------|
| **JWT 시크릿 자동 생성** | `JWT_SECRET` 미설정 시 랜덤 생성 → 서버 재시작마다 모든 토큰 무효화 | `internal/config/config.go` |
| **테스트 부재** | 인증/인가 로직 검증 불가, 보안 회귀 감지 불가 | 전체 |

### 3.2 심각도: High

| 이슈 | 설명 | 위치 |
|------|------|------|
| ~~**보안 헤더 미적용**~~ | ~~`X-Content-Type-Options`, `X-Frame-Options`, `CSP` 헤더 없음~~ | **해결됨** — `middleware.SecurityHeaders()` 적용 |
| ~~**태그 파라미터 미검증**~~ | ~~`tags LIKE "%"+tag+"%"` 쿼리에 길이/문자 검증 없음~~ | **해결됨** — 태그 M:N 정규화, 입력 검증 추가 |
| **localStorage 토큰 저장** | XSS 공격 시 토큰 탈취 가능 → HttpOnly 쿠키 권장 | Frontend `AuthProvider.jsx` |
| **HTTPS 미강제** | HTTP→HTTPS 리다이렉트 없음, HSTS 헤더 없음 | Backend |

### 3.3 심각도: Medium

| 이슈 | 설명 |
|------|------|
| **비밀번호 정책 미흡** | 최소 6자만 요구, 복잡도 검증 없음 |
| ~~**토큰 갱신 메커니즘 없음**~~ | **해결됨** — Access Token 15분 + Refresh Token 7일, `/auth/refresh` 엔드포인트 구현 |
| **CSRF 보호 없음** | SameSite 쿠키나 CSRF 토큰 미적용 |
| **파일 업로드 보안** | 바이러스 스캔 없음, Content-Disposition 헤더 미설정 |

---

## 4. 코드 품질 상세 분석

### 4.1 Backend 코드 품질

**잘한 점:**
- Go 네이밍 컨벤션 준수 (`CamelCase` 함수, `camelCase` 변수)
- 에러 타입을 `errors.New()`로 정의하여 `errors.Is()` 비교 가능
- `golangci-lint` 설정 파일 존재
- Fiber의 `c.Locals()` 활용한 미들웨어 → 핸들러 데이터 전달
- Soft Delete 지원 (`gorm.DeletedAt`)

**아쉬운 점:**
```go
// 응답 포맷이 3가지로 비일관
// 1) 성공 응답
{"status": "success", "data": {...}}

// 2) 에러 핸들러
{"status": "error", "message": "..."}

// 3) Auth 에러
{"error": "...", "message": "..."}
```

- `main.go`의 goroutine 에러가 `slog.Error` + `os.Exit(1)`로 전체 프로세스 종료 가능
- ~~구조화된 로깅(structured logging) 없음~~ **해결됨** — `log/slog` 전환 완료
- PUT과 PATCH가 동일 핸들러 → RESTful 의미론 위반

### 4.2 Frontend 코드 품질

**잘한 점:**
- 커스텀 마크다운 파서 직접 구현 (KaTeX, highlight.js, 테이블, 각주 지원)
- `AbortController`로 요청 취소 → 경쟁 상태 방지
- `useAutoSave` 훅으로 임시 저장 구현 (1초 디바운스)
- 탭 간 로그아웃 동기화 (`storage` 이벤트 리스너)
- DOMPurify로 마크다운 HTML 새니타이징 (XSS 방지)

**아쉬운 점:**
- `BlockEditor.jsx`가 527줄 → 분할 필요 (에디터 로직, 렌더링, 이벤트 핸들링)
- `markdownParser.js`가 540줄 → 인라인 파싱/블록 파싱 분리 권장
- 코드 복사 버튼 로직이 BlockEditor와 PostDetailPage에 중복
- `useApi` 훅이 정의되어 있으나 실제 컴포넌트에서 미사용

---

## 5. 테스트 평가

### 현재 상태: Backend 단위 테스트 추가됨, Frontend 테스트 부재

**Backend - 필요한 테스트:**
| 우선순위 | 대상 | 테스트 유형 |
|----------|------|------------|
| P0 | JWT 생성/검증 | 단위 테스트 |
| P0 | 비밀번호 해싱/비교 | 단위 테스트 |
| P0 | 인증 미들웨어 | 단위 테스트 |
| P1 | Service 비즈니스 로직 | 단위 테스트 (Mock DB) |
| P1 | 소유권 검증 로직 | 단위 테스트 |
| P2 | API 엔드포인트 | 통합 테스트 |
| P2 | 페이지네이션 | 단위 테스트 |

**Frontend - 필요한 테스트:**
| 우선순위 | 대상 | 테스트 유형 |
|----------|------|------------|
| P0 | markdownParser | 단위 테스트 (Jest) |
| P0 | API 유틸 | 단위 테스트 (Mock fetch) |
| P1 | AuthProvider | 컴포넌트 테스트 (RTL) |
| P1 | BlockEditor | 컴포넌트 테스트 |
| P2 | 로그인 → 글쓰기 → 조회 플로우 | E2E 테스트 (Playwright) |

---

## 6. API 설계 평가

### 잘한 점:
- `/api/v1` 버전 프리픽스 사용
- HTTP 상태 코드 적절히 사용 (201, 400, 401, 403, 404, 409, 429)
- 페이지네이션 메타데이터 포함 (page, page_size, total, total_pages)
- Rate Limiting 적용 (인증 엔드포인트 10req/min)
- Optional Auth 미들웨어로 비공개 글 조건부 조회

### 아쉬운 점:
- ~~응답 포맷 비일관~~ **해결됨** — `dto.SuccessResponse`/`dto.ErrorResponse`로 통일
- PUT/PATCH 구분 없음 (동일 핸들러)
- ~~검색 API 없음~~ **해결됨** — `GET /posts/search?q=` 구현
- ~~댓글 기능 없음~~ **해결됨** — 댓글 CRUD + 대댓글 (트리 구조) 구현
- ~~사용자 정보 수정 API 없음~~ **부분 해결** — 비밀번호 변경, 아바타 업로드 구현
- Link 헤더 미포함 (HATEOAS)
- ETag/Last-Modified 캐싱 헤더 없음

---

## 7. 데이터베이스 설계 평가

### 현재 스키마:

```
User (id, username, password, avatar_url, created_at, updated_at)
  ├── Post (id, user_id FK, title, content, is_public, view_count, like_count, series_id FK, series_order, created_at, updated_at, deleted_at)
  │     ├── Tag (id, name) — M:N via post_tags junction table
  │     ├── Comment (id, post_id FK, user_id FK, content, parent_id FK, created_at, updated_at, deleted_at)
  │     └── Like (id, post_id FK, user_id FK, created_at)
  └── Series (id, user_id FK, title, description, created_at, updated_at, deleted_at)
```

### ~~문제점~~ 진행 상황:

1. ~~**태그가 문자열 필드**~~ **해결됨** — `Tag` 테이블 + `post_tags` 다대다 관계 테이블로 정규화 완료

2. **인덱스 부족** (일부 잔존)
   - `(user_id, is_public)` 복합 인덱스 검토 필요

3. ~~**확장성 한계**~~ **대부분 해결**
   - ~~조회수(`view_count`) 없음~~ **해결됨** — Redis 캐싱 기반 조회수 구현
   - ~~좋아요/댓글 테이블 없음~~ **해결됨** — Like, Comment 테이블 + CRUD 구현
   - ~~카테고리 분류 없음~~ — 시리즈(Series) 기능으로 글 모음 관리 가능

---

## 8. 성능 평가

### Backend:
| 항목 | 현재 상태 | 비고 |
|------|----------|------|
| DB 커넥션 풀 | 100 max, 10 idle | 적정 |
| 캐싱 | **Redis 도입 완료** | 글 조회 캐싱, TTL, 패턴 삭제 무효화 |
| N+1 쿼리 | Preload("User") 사용 | 양호 |
| 태그 검색 | **M:N 정규화 완료** | JOIN 기반 정확한 검색 |
| 응답 압축 | **gzip 적용 완료** | `compress.LevelDefault` |
| 이미지 서빙 | 정적 파일 직접 서빙 | CDN 도입 권장 |

### Frontend:
| 항목 | 현재 상태 | 비고 |
|------|----------|------|
| 코드 스플리팅 | **React.lazy + Suspense 적용** | 페이지별 lazy load |
| 이미지 최적화 | 업로드 시 압축 (1200px, 0.8 quality) | WebP 변환 추가 권장 |
| 마크다운 파싱 | 매 렌더링마다 재파싱 | useMemo로 캐싱 권장 |
| 번들 크기 | highlight.js 별도 청크 분리 | rollup-plugin-visualizer 추가 권장 |

---

## 9. UI/UX 평가

### 잘한 점 (8/10):
- **다크/라이트 모드**: CSS 변수 기반, 시스템 설정 연동, 부드러운 전환
- **블록 에디터**: Typora/Notion 스타일의 인라인 편집, 키보드 단축키 지원
- **이미지 리사이즈**: 드래그 핸들로 직관적 크기 조절
- **임시 저장/복원**: 작성 중 이탈 시 데이터 보존
- **반응형 디자인**: 모바일/태블릿/데스크톱 브레이크포인트
- **TOC (목차)**: 1200px 이상에서 사이드바 목차 자동 생성
- **OG 태그**: 소셜 미디어 공유 시 미리보기 지원

### 아쉬운 점:
- ~~글 검색 기능 없음~~ **해결됨** — 키워드 검색 구현
- ~~댓글/반응 기능 없음~~ **해결됨** — 댓글 (대댓글 포함) + 좋아요 구현
- 무한 스크롤 대신 페이지네이션만 제공
- 글 작성 시 미리보기가 별도 모드로만 가능 (분할 뷰 없음)
- 접근성(a11y) ARIA 속성 일부만 적용

---

## 10. 배포 & 운영 평가

### CI/CD 파이프라인:

```
Backend:  Push → Lint → Build(ARM64) → Docker Image → GHCR → SSH Deploy
Frontend: Push → Build(Vite) → Docker Image → GHCR → SSH Deploy
```

**잘한 점:**
- GitHub Actions로 자동 배포
- Docker 컨테이너화
- Alpine 기반 경량 이미지

**아쉬운 점:**
| 항목 | 상태 |
|------|------|
| **CI에서 테스트 실행** | lint → test → build (추가됨) |
| **스테이징 환경** | 없음 (프로덕션 직접 배포) |
| **보안 스캔** | SAST/SCA/컨테이너 스캔 없음 |
| **모니터링** | Prometheus/Grafana 없음 |
| **로그 수집** | 구조화된 로깅 없음 |
| **헬스 체크** | 엔드포인트 존재하나 Docker에서 미활용 |
| **롤백 전략** | 명시적 롤백 절차 없음 |

---

## 11. 프로젝트 강점 (잘한 점 종합)

1. **아키텍처 설계**: 백엔드 도메인별 레이어드 아키텍처 + 인터페이스 DI, 프론트 Context 패턴
2. **커스텀 마크다운 엔진**: KaTeX 수식, 코드 하이라이팅, 테이블, 각주까지 직접 구현
3. **블록 에디터**: Notion/Typora 스타일의 사용자 경험
4. **테마 시스템**: CSS 변수 기반 다크/라이트 모드
5. **보안**: bcrypt, JWT (Access+Refresh), DOMPurify, Rate Limiting, CORS, 보안 헤더
6. **임시 저장**: 자동 저장 + 복원 메커니즘
7. **탭 간 동기화**: storage 이벤트로 크로스탭 로그아웃
8. **이미지 처리**: 업로드 전 압축, MIME 검증, UUID 파일명
9. **Redis 캐싱**: 글 조회 캐싱, TTL, 패턴 삭제 기반 무효화
10. **시리즈 기능**: 글 모음 관리, 순서 변경, 네비게이션
11. **SEO**: RSS 피드, XML 사이트맵, OG 태그
12. **Swagger 문서화**: 전체 API 22개 엔드포인트 문서화 완료

---

## 12. 개선 로드맵

### Phase 1: 즉시 (안정성 & 보안)
- [ ] JWT_SECRET 프로덕션 필수 설정으로 변경
- [x] ~~보안 헤더 미들웨어 추가~~ (2026-03 완료)
- [x] ~~태그 파라미터 입력 검증 추가~~ (M:N 정규화로 해결)
- [x] ~~핵심 로직 단위 테스트 작성~~ (Backend 단위 테스트 추가됨)
- [x] ~~응답 포맷 통일~~ (dto.SuccessResponse/ErrorResponse)

### Phase 2: 단기 (1~2주)
- [ ] 테스트 커버리지 60% 이상 달성
- [x] ~~CI 파이프라인에 테스트 단계 추가~~ (lint → test → build)
- [x] ~~구조화된 로깅 도입~~ (log/slog 전환 완료)
- [ ] DB 인덱스 추가 (is_public, created_at)
- [x] ~~gzip 응답 압축 미들웨어 추가~~ (compress.LevelDefault)
- [x] ~~TypeScript 마이그레이션~~ (전환 완료, .ts/.tsx)

### Phase 3: 중기 (1~2개월)
- [x] ~~태그 테이블 정규화~~ (Tag + post_tags M:N 완료)
- [x] ~~검색 기능 구현~~ (GET /posts/search?q= 구현)
- [x] ~~댓글 시스템 추가~~ (CRUD + 대댓글 트리 구조)
- [x] ~~Refresh Token 도입~~ (15분 access + 7일 refresh)
- [x] ~~Redis 캐싱 레이어 추가~~ (글 조회 캐싱, 패턴 삭제 무효화)
- [x] ~~페이지별 코드 스플리팅~~ (React.lazy + Suspense)

### Phase 4: 장기
- [ ] 모니터링 시스템 구축 (Prometheus + Grafana)
- [ ] E2E 테스트 (Playwright)
- [ ] CDN 도입 (이미지/정적 파일)
- [ ] SSR 또는 SSG 검토 (SEO 강화)
- [ ] 알림 시스템

---

## 13. 최종 평가

### 총점: 6.7 / 10 (이전 5.9 → 현재 6.7)

**한 줄 평가:** Phase 1~3의 주요 항목이 대부분 해결되어 기능적으로 완성도가 높아졌으나, 테스트 커버리지 확대와 JWT 시크릿 프로덕션 필수화가 남은 과제.

**포지티브 관점:** 초기 리뷰 대비 보안 헤더 적용, 응답 포맷 통일, 태그 정규화, 검색/댓글/좋아요/시리즈 기능 구현, Redis 캐싱, Refresh Token, TypeScript 전환, React.lazy 적용, Swagger 전체 문서화 등 대대적 개선이 이뤄짐.

**크리티컬 관점:** JWT 시크릿 자동 생성 문제 잔존, Frontend 테스트 여전히 부재, Backend 테스트 커버리지 확대 필요.

**프로덕션 배포 판단:** 소규모~중규모 블로그 서비스로 운영 가능. JWT_SECRET 프로덕션 필수 설정과 테스트 커버리지 60% 이상 달성 시 안정적 서비스 운영 가능.
