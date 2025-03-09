package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/core"
)

type Config struct {
	Url                string
	Username           string
	Password           string
	Token              string
	InsecureSkipVerify bool
	// Mostly used for log level.
	Development  bool
	RetryMode    core.RetryMode
	RetryMaxTime time.Duration
}

var (
	retryModeMap = map[string]core.RetryMode{
		"none":    core.None,
		"backoff": core.Backoff,
	}
)

func isZero[T comparable](value T) bool {
	var zero T
	return value == zero
}

func valueOrFallback[T comparable](value, fallback T) T {
	if isZero(value) {
		return fallback
	}
	return value
}

func New() *Config {
	retryMode := core.None
	retryMaxTime := 5 * time.Minute

	if v := os.Getenv("XOA_RETRY_MODE"); v != "" {
		retry, ok := retryModeMap[v]
		if !ok {
			fmt.Println("[ERROR] failed to set retry mode, disabling retries")
		} else {
			retryMode = retry
		}
	}

	if v := os.Getenv("XOA_RETRY_MAX_TIME"); v != "" {
		duration, err := time.ParseDuration(v)
		if err == nil {
			retryMaxTime = duration
		} else {
			fmt.Println("[ERROR] failed to set retry mode, disabling retries")
		}
	}

	insecureStr := os.Getenv("XOA_INSECURE")
	insecure := false
	if insecureStr != "" {
		insecure, _ = strconv.ParseBool(insecureStr)
	}

	development := false
	if v := os.Getenv("XOA_DEVELOPMENT"); v != "" {
		development, _ = strconv.ParseBool(v)
	}

	return &Config{
		Url:                valueOrFallback(os.Getenv("XOA_URL"), "http://localhost:80"),
		Username:           valueOrFallback(os.Getenv("XOA_USER"), "admin@admin.net"),
		Password:           valueOrFallback(os.Getenv("XOA_PASSWORD"), "admin"),
		Token:              valueOrFallback(os.Getenv("XOA_TOKEN"), ""),
		InsecureSkipVerify: insecure,
		Development:        development,
		RetryMode:          retryMode,
		RetryMaxTime:       retryMaxTime,
	}
}
