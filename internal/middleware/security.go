package middleware

import "github.com/gofiber/fiber/v2"

// 응답은 JSON 뿐이라 모든 출처를 막아, HTML 로 해석되거나 iframe 에 실리는 경우를 차단한다.
const apiCSP = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"

// 엣지 프록시를 거치지 않은 호출에도 HSTS 가 붙도록 백엔드에서도 건다.
const hsts = "max-age=31536000; includeSubDomains"

func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Set("Content-Security-Policy", apiCSP)
		c.Set("Strict-Transport-Security", hsts)
		return c.Next()
	}
}
