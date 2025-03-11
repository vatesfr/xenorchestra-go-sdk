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

// New returns a new Config with sensible defaults.
//
// The following environment variables are honored:
//
// - XOA_URL: the base URL of the Xen Orchestra API.
// - XOA_USER: the username to use when connecting to the API.
// - XOA_PASSWORD: the password to use when connecting to the API.
// - XOA_TOKEN: the authentication token to use when connecting to the API.
// - XOA_INSECURE: whether to skip verifying the server's TLS certificate.
// - XOA_DEVELOPMENT: whether to enable development mode.
// - XOA_RETRY_MODE: the retry mode to use. Defaults to "none". Valid values are "none", "backoff".
// - XOA_RETRY_MAX_TIME: the maximum time to wait between retries. Defaults to 5 minutes.
//
// If any of the required environment variables are not set, New will return an error.
func New() (*Config, error) {
	if os.Getenv("XOA_URL") == "" {
		return nil, fmt.Errorf("XOA_URL is not set, please set it to the Xen Orchestra URL")
	}
	if os.Getenv("XOA_USER") == "" {
		return nil, fmt.Errorf("XOA_USER is not set, please set it to the Xen Orchestra username")
	}
	if os.Getenv("XOA_PASSWORD") == "" {
		return nil, fmt.Errorf("XOA_PASSWORD is not set, please set it to the Xen Orchestra password")
	}

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
		Url:                os.Getenv("XOA_URL"),
		Username:           os.Getenv("XOA_USER"),
		Password:           os.Getenv("XOA_PASSWORD"),
		Token:              os.Getenv("XOA_TOKEN"),
		InsecureSkipVerify: insecure,
		Development:        development,
		RetryMode:          retryMode,
		RetryMaxTime:       retryMaxTime,
	}, nil
}
