package utils

import (
	"log/slog"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

var sentryInitialized bool

// InitSentry 는 SENTRY_DSN 이 없으면 아무것도 하지 않는다. 그때 CaptureException 은 로그로만 남긴다.
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

func FlushSentry() {
	if !sentryInitialized {
		return
	}
	sentry.Flush(2 * time.Second)
}
