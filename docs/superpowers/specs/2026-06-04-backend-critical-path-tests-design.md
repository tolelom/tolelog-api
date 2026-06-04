# Backend Critical Path Tests — Design Spec (Final)

**Date**: 2026-06-04
**Status**: ✅ 확정 — WIP(2026-04-25) 브레인스토밍 재개 후 전체 결정 완료
**Scope**: `fiber_api_server/` — 서비스 층 권한/보안 크리티컬 경로 테스트
**Supersedes**: `2026-04-25-backend-critical-path-tests-design-WIP.md`

---

## 1. Context

서비스 층의 비즈니스 로직(GORM 직접 호출 코드)이 거의 미테스트 상태.
핸들러는 mock Service 기반 단위 테스트로 덮여 있으나, 권한 체크 같은
**회귀하면 보안 구멍이 되는 로직**은 실제 DB 동작과 함께 검증이 필요하다.

현재 커버리지 (2026-06-04 측정): post 40.7%, user 47.5%, comment 56.0%

## 2. 확정된 결정

| Topic | Decision | Reason |
|---|---|---|
| **스코프** | 크리티컬 경로 12개만 (권한/보안) | 전체 커버리지 목표 X |
| **테스트 DB** | SQLite in-memory, **pure-Go 드라이버 `github.com/glebarez/sqlite`** | CGO 불필요 → Windows 로컬 개발 + CI 모두 추가 설정 0. (WIP 의 `gorm.io/driver/sqlite` 는 CGO 필요라 변경) |
| **Redis 전략** | `nil` cache 주입 | 서비스가 `if s.cache != nil` 가드 전부 보유 |
| **헬퍼 구조** | `internal/testutil/` 공유 패키지 | post/comment/user 3개 도메인이 동일한 setupDB/fixture 사용 → DRY |
| **DB 세팅** | `testutil.SetupDB(t)` — 테스트마다 fresh in-memory DB + AutoMigrate | 격리 명확, in-memory 라 ms 단위 비용 |
| **Fixture** | factory 함수: `testutil.MakeUser(t, db, username)`, `testutil.MakePost(t, db, userID, opts...)` | 테이블 드리븐 픽스처보다 호출부 가독성 우수 |
| **파일 분리** | 기존 `service_test.go`(pure 함수)와 별도로 `service_db_test.go` 신설 | 의도 구분: pure vs DB 통합 |

## 3. 크리티컬 경로 12개 (사용자 확정: 이대로)

해피 + 차단 케이스 각 1개 이상 = 약 24개 테스트.

**Post 서비스 (7):**
1. `GetPostByID` — `is_public=false` 글을 비로그인/타인이 조회 시 차단
2. `GetPostByID` — 본인은 자기 비공개 글 조회 가능
3. `UpdatePost` — 본인 글만 수정 가능 (타인 수정 시 에러)
4. `DeletePost` — 본인 글만 삭제 가능
5. `GetUserPosts` — 타인 프로필 조회 시 `is_public=true`만, 본인은 전부
6. `GetDrafts` — 본인 초안만 조회
7. `ToggleLike` — 같은 유저 두 번 토글 시 liked 상태/count 증감 정상

**Comment 서비스 (3):**
8. `CreateComment` — 비공개 글에 타인 댓글 차단
9. `DeleteComment` — 댓글 작성자 본인만 삭제 가능 (현재 구현·Swagger 문서 기준. "글 작성자도 삭제 가능"은 미구현 — 도입 여부는 별도 제품 결정)
10. `UpdateComment` — 작성자 본인만 수정

**User 서비스 (2):**
11. `Login` — 잘못된 비밀번호 차단 (bcrypt 해시 검증 경유)
12. `Register` — 중복 username 거부 (User 모델에 email 필드 없음)

## 4. 구현 구조

```
internal/testutil/
  db.go        — SetupDB(t *testing.T) *gorm.DB  (in-memory + AutoMigrate 전 모델)
  fixtures.go  — MakeUser / MakePost / MakeComment factory

internal/post/service_db_test.go     — 경로 1~7
internal/comment/service_db_test.go  — 경로 8~10
internal/user/service_db_test.go     — 경로 11~12
```

- `go.mod` 에 `github.com/glebarez/sqlite` 추가 (test-only 의존성)
- 모델의 MySQL 전용 타입/태그가 SQLite AutoMigrate 에서 깨지면 해당 지점만 보고 후 조정

## 5. Non-goals (WIP 에서 유지)

- 전체 서비스 커버리지 목표 X
- 핸들러 층 추가 커버리지 X (이미 mock 기반 존재)
- Redis 동작 테스트 X / testcontainers X / MySQL-specific 쿼리 검증 X / 성능 테스트 X
