package utils

import (
	"log/slog"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

var sentryInitialized bool

// InitSentry 는 SENTRY_DSN 환경변수가 설정되어 있을 때만 Sentry 클라이언트를 초기화한다.
// DSN 이 없으면 무동작 — CaptureException 은 slog.Error 폴백으로 동작.
// 호출 측은 프로그램 종료 직전 utils.FlushSentry() 를 호출해 미전송 이벤트를 비울 수 있다.
func InitSentry() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		slog.Info("SENTRY_DSN 미설정 — 에러 리포팅은 로컬 로그로만 기록")
		return
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		TracesSampleRate: 0,
		EnableTracing:    false,
	})
	if err != nil {
		slog.Error("Sentry 초기화 실패", "error", err)
		return
	}
	sentryInitialized = true
	slog.Info("Sentry 초기화 완료", "environment", env)
}

// CaptureException 은 에러를 Sentry 로 보내거나, DSN 미설정 시 슬로그에 기록한다.
func CaptureException(err error, tags map[string]string) {
	if err == nil {
		return
	}
	if !sentryInitialized {
		slog.Error("captured error", "error", err, "tags", tags)
		return
	}
	hub := sentry.CurrentHub().Clone()
	if len(tags) > 0 {
		hub.ConfigureScope(func(s *sentry.Scope) {
			for k, v := range tags {
				s.SetTag(k, v)
			}
		})
	}
	hub.CaptureException(err)
}

// FlushSentry 는 종료 시 큐 비우기 (graceful shutdown 마지막에 호출).
func FlushSentry() {
	if !sentryInitialized {
		return
	}
	sentry.Flush(2 * time.Second)
}
