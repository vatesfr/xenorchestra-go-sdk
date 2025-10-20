package config

import (
	"errors"
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

func ToRetryMode(mode string) core.RetryMode {
	retry, ok := retryModeMap[mode]
	if !ok {
		return core.None
	}
	return retry
}

// NOTE: Same as the shared types or constants, we could have in the internal package,
// errors message declared to be used in the different v2 packages. (OPTIONAL)
const (
	// #nosec G101 -- Not actual credentials, just environment variable names
	errMissingAuthInfo = `authentication information not provided. Please set XOA_TOKEN or both XOA_USER and XOA_PASSWORD`
	errMissingUrl      = `XOA_URL is not set, please set it`
)

// New returns a new Config with sensible defaults.
//
// The following environment variables are honored:
//
// Note that either XOA_TOKEN or XOA_USER and XOA_PASSWORD must be set.
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
	url := os.Getenv("XOA_URL")
	token := os.Getenv("XOA_TOKEN")
	username := os.Getenv("XOA_USER")
	password := os.Getenv("XOA_PASSWORD")
	if url == "" {
		return nil, errors.New(errMissingUrl)
	}
	if token == "" && (username == "" || password == "") {
		return nil, errors.New(errMissingAuthInfo)
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
		Url:                url,
		Username:           username,
		Password:           password,
		Token:              token,
		InsecureSkipVerify: insecure,
		Development:        development,
		RetryMode:          retryMode,
		RetryMaxTime:       retryMaxTime,
	}, nil
}

// NewWithValues returns a new Config with the values provided.
//
// The purpose of this function is to allow the user to use the SDK without
// having to set the environment variables, for example in the terraform
// provider where the variables are part of the config files rather than
// the environment variables.
//
// The following fields are required:
// - Url
// - Token or Username and Password
func NewWithValues(config *Config) (*Config, error) {

	if config.Url == "" {
		return nil, errors.New(errMissingUrl)
	}

	if config.Token == "" && (config.Username == "" || config.Password == "") {
		return nil, errors.New(errMissingAuthInfo)
	}

	return &Config{
		Url:                config.Url,
		Username:           config.Username,
		Password:           config.Password,
		Token:              config.Token,
		InsecureSkipVerify: config.InsecureSkipVerify,
		RetryMode:          config.RetryMode,
		RetryMaxTime:       config.RetryMaxTime,
		Development:        config.Development,
	}, nil
}
