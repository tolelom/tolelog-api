# 기여

```bash
go test ./...             # 테스트(DB 테스트는 인메모리 SQLite)
golangci-lint run         # 린트(.golangci.yml)
swag init -g cmd/server/main.go --requiredByDefault   # Swagger 문서 생성
```

패키지 구성과 지켜야 할 규칙은 `ARCHITECTURE.md`에 있습니다.

## API 계약

`docs/swagger.json`이 API 계약이고, tolelog-web이 여기서 TypeScript 타입을 생성합니다.

- handler 주석이나 DTO를 바꾸면 `swag init`으로 `docs/`를 다시 만들어 함께 커밋합니다. CI가 다시 생성한 결과와 `docs/`가 다르면 실패합니다.
- 성공 응답 주석은 `dto.SuccessResponse{data=dto.X}`로 씁니다. 맨 `dto.X`나 타입 없는 `fiber.Map`은 쓰지 않습니다.
- `--requiredByDefault` 때문에 json 태그에 `omitempty`가 없는 칸은 모두 필수가 됩니다. 선택 요청 칸에는 `omitempty`를 붙입니다.
- `omitempty` 없이 `null`로 나가는 칸(nil 포인터, nil 슬라이스)에는 `extensions:"x-nullable"`을 붙입니다.
- 응답이 바뀐 뒤에는 tolelog-web에서 `npm run contract:sync`로 사본과 타입을 갱신합니다.

## 배포

`main`에 push하면 GitHub Actions(`.github/workflows/ci.yml`)가 lint, 테스트, linux/arm64 빌드를 거쳐 이미지 `ghcr.io/tolelom/tolelom-api`를 `latest`, `main`, 커밋 SHA 태그로 올립니다. workflow는 이미지 push에서 끝나고, Renovate가 `server-config` 저장소의 이미지 digest를 올리면 git-sync가 서버에 반영합니다.

- 운영(`ENVIRONMENT=production`)에서는 `JWT_SECRET`이 없으면 서버가 시작하지 않고, Swagger UI를 열지 않습니다.
- 이미지는 8000 포트를 열고 `/health`로 헬스 체크를 합니다.
- 업로드 이미지는 `/app/uploads` 볼륨에 있습니다.
- `SENTRY_DSN`을 주면 오류를 Sentry로 보냅니다.
- `TRUSTED_PROXIES` 대역에서 온 요청만 `CF-Connecting-IP` 헤더를 클라이언트 IP로 씁니다.
