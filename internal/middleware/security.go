package middleware

import "github.com/gofiber/fiber/v2"

// API 응답은 JSON이므로 default-src 'none' + frame-ancestors 'none' 으로
// 혹시라도 HTML로 해석되거나 iframe에 임베드되는 경우를 차단한다.
const apiCSP = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"

// HSTS — 프록시/CDN이 헤더를 제거하지 않는 한 1년 유지, 서브도메인 포함, preload 신청 가능.
// 백엔드에서도 설정하여 엣지(nginx) 우회 호출 시에도 적용되도록 한다 (defense in depth).
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
