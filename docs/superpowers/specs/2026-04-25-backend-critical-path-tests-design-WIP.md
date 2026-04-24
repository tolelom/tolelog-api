# Backend Critical Path Tests — Design Spec (WIP)

**Date**: 2026-04-25
**Status**: 🚧 WIP — brainstorming paused midway
**Scope**: `fiber_api_server/` — Go Fiber backend service-layer tests
**Resume**: 이 문서 읽고 "Pending Decisions" 섹션의 질문부터 재개

---

## 1. Context

현재 `fiber_api_server` 테스트 상태 (coverage %):

| 패키지 | 커버리지 | 상태 |
|---|---|---|
| `feed` | 93.3% | 양호 |
| `image` | 93.8% | 양호 |
| `middleware` | 80.0% | 양호 |
| `utils` | 84.0% | 양호 |
| `sitemap` | 68.0% | 양호 |
| `series` | 50.6% | 중간 |
| `dto` | 43.1% | 중간 |
| `comment` | 38.8% | **낮음** |
| `user` | 38.1% | **낮음** |
| `post` | 24.3% | **가장 낮음** |

**진짜 갭**: 핸들러는 mock Service 기반 단위 테스트가 대부분 있음. **서비스 층의 비즈니스 로직(GORM 직접 호출 코드)이 거의 전부 0%**. 이유: DB 필요해서 아무도 안 씀.

---

## 2. Decisions Made (확정)

| Topic | Decision | Reason |
|---|---|---|
| **스코프** | C — 크리티컬 경로만 (권한/보안 관련 서비스 로직) | 전체 커버리지 욕심 X, 회귀하면 보안 구멍 되는 곳만 |
| **테스트 DB** | A — SQLite in-memory (`gorm.io/driver/sqlite`) | 외부 의존성 0, ms 단위 빠름, CI 설정 변경 불필요. 나중에 dialect 문제 생기면 해당 테스트만 testcontainers로 격상 가능 |
| **Redis 전략** | `nil` cache 넘기기 | 서비스가 `if s.cache != nil` 가드 전부 보유 → mock 불필요 |

### 선택지에서 뺀 것들

- **B 스코프 (서비스 풀 커버리지)**: tag sync, 캐시 무효화까지 가면 Redis mock 파고들어야 함 → 스코프 폭발
- **testcontainers MySQL**: Docker 세팅 필요, CI yaml 수정 필요, 느림 (10~20초 오버헤드)
- **sqlmock**: GORM 생성 SQL에 커플링 → 유지보수 지옥
- **A 스코프 (핸들러 갭만)**: mock Service 기반이라 진짜 로직 검증 X. 의미 없음

---

## 3. Pending Decisions (다음에 재개할 지점)

### 3.1 크리티컬 경로 리스트 확정 ⬅ **재개는 여기부터**

제가 식별한 **12개 경로** (해피 + 차단 케이스 각 1개씩 잡으면 ~24 테스트):

**Post 서비스 (7):**
1. `GetPostByID` — `is_public=false` 글을 비로그인/타인이 조회 시 차단
2. `GetPostByID` — 본인은 자기 비공개 글 조회 가능
3. `UpdatePost` — 본인 글만 수정 가능 (타인 수정 시 403)
4. `DeletePost` — 본인 글만 삭제 가능
5. `GetUserPosts` — 타인 프로필 조회 시 `is_public=true`만 보임, 본인은 전부
6. `GetDrafts` — 본인 초안만 조회
7. `ToggleLike` — 같은 유저 두 번 누르면 toggle 정상 작동 + count 증감

**Comment 서비스 (3):**
8. `CreateComment` — 비공개 글에 댓글 차단 조건 확인
9. `DeleteComment` — 댓글 작성자 본인 or 글 작성자만 삭제 가능
10. `UpdateComment` — 작성자 본인만 수정

**User 서비스 (2):**
11. `Login` — 잘못된 비밀번호 차단, bcrypt 해시 검증
12. `Register` — 중복 username/email 거부

**사용자에게 물어볼 질문:**
> 이 리스트 그대로 갈까요, 아니면 빼거나 더 넣을 거 있나요? (예: "좋아요 토글은 권한 이슈 아니니 빼자", "tag sync도 넣자" 등)

### 3.2 이후 결정 필요한 것

- **테스트 헬퍼 구조**: `internal/testutil/` 패키지를 만들지, 각 서비스 `_test.go` 내부에 helper 둘지
- **DB 세팅 방식**: `setupDB(t *testing.T)` 함수로 매 테스트마다 fresh DB vs. 서브테스트로 공유 DB + truncate
- **사용자 fixture 방식**: `makeUser(t, username)` 같은 factory vs. 테이블 드리븐 픽스처
- **기존 service_test.go와 공존**: 현재 `post/service_test.go`는 `SanitizeTag` 같은 pure 함수만 테스트. 새 DB 기반 테스트를 같은 파일에 넣을지, `service_db_test.go`로 분리할지

---

## 4. Resume Checklist

다음에 재개할 때 순서:

1. **이 문서 읽기** — 위 섹션 1~3 훑어서 컨텍스트 복원
2. **사용자에게 3.1의 12개 경로 리스트 재확인** — 빼거나 추가할 거 물어봄
3. **3.2의 나머지 디자인 결정 사항 하나씩 마무리** — 헬퍼 구조 / DB 세팅 / fixture / 파일 분리
4. **스펙 문서 정식 버전 작성** — 이 WIP 문서를 정리해서 `2026-XX-XX-backend-critical-path-tests-design.md`로 최종본 커밋
5. **writing-plans 스킬 호출** — 구현 계획 작성
6. **subagent-driven-development로 실행** — 경로 하나씩 구현

---

## 5. Key Code References (재개 시 참고)

- `internal/post/service.go:85` — `NewService(db, cache)` — cache는 nil 허용
- `internal/post/service.go:117` — `invalidatePostCaches` — `s.cache == nil` 체크 있음
- `internal/post/service.go:167` — `GetPostByID` — is_public 체크 로직
- `internal/post/service_test.go` — 기존은 `SanitizeTag` 같은 pure 함수만
- `internal/cache/cache.go` — Cache 구조체 (인터페이스 아님, nil 처리 서비스에서)
- `go.mod` — `gorm.io/driver/sqlite` 추가 필요 (SQLite in-memory용)

---

## 6. Non-goals (확정)

- 전체 서비스 커버리지 목표 X (권한/보안 외는 손 안 댐)
- 핸들러 층 추가 커버리지 X (이미 mock으로 덮여있음)
- Redis 동작 테스트 X (nil 전략)
- testcontainers / Docker 환경 X
- MySQL-specific 쿼리 검증 X (SQLite에서 돌아가면 OK)
- 성능/부하 테스트 X
