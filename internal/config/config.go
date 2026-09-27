package config

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"mio9/mtx-monitor/internal/constants"
)

// ApiAuth represents authentication credentials for the MediaMTX API.
type ApiAuth struct {
	Scheme   string // "basic" or "bearer"
	Username string
	Password string
	Token    string
}

// Config holds the application configuration.
type Config struct {
	APIURL           string
	RTSPURL          string
	WatchPlayer      string
	PollIntervalMs   int
	MaxBitrateBps    int
	ApiAuth          *ApiAuth
	PathIncludeRegex *regexp.Regexp
}

// LoadConfig loads configuration from environment variables and .env file.
func LoadConfig() (*Config, error) {
	// Load .env file if it exists (relative to current working directory).
	_ = godotenv.Load() // best-effort; missing .env is OK

	pollIntervalSec := parsePositiveNumber(
		os.Getenv("POLL_INTERVAL_SEC"),
		"POLL_INTERVAL_SEC",
		constants.DefaultPollIntervalSec,
	)

	maxBitrateKbps := parsePositiveNumber(
		os.Getenv("MAX_BITRATE_KBPS"),
		"MAX_BITRATE_KBPS",
		constants.DefaultMaxBitrateKbps,
	)

	apiAuth, err := loadApiAuth()
	if err != nil {
		return nil, fmt.Errorf("load auth: %w", err)
	}

	apiURL := os.Getenv("MTX_API_URL")
	if apiURL == "" {
		apiURL = constants.DefaultAPIURL
	}
	apiURL = strings.TrimRight(apiURL, "/")

	rtspURL := os.Getenv("MTX_RTSP_URL")
	if rtspURL == "" {
		// Construct from API URL.
		parsed := parseHostname(apiURL)
		rtspURL = fmt.Sprintf("rtsp://%s:%d", parsed, constants.DefaultRTSPPort)
	}
	rtspURL = strings.TrimRight(rtspURL, "/")

	watchPlayer := strings.TrimSpace(os.Getenv("WATCH_PLAYER"))
	if watchPlayer == "" {
		watchPlayer = constants.DefaultWatchPlayer
	}

	pathIncludeRegex, err := loadPathIncludeRegex()
	if err != nil {
		return nil, fmt.Errorf("load path regex: %w", err)
	}

	return &Config{
		APIURL:           apiURL,
		RTSPURL:          rtspURL,
		WatchPlayer:      watchPlayer,
		PollIntervalMs:   pollIntervalSec * 1000,
		MaxBitrateBps:    maxBitrateKbps * 1000,
		ApiAuth:          apiAuth,
		PathIncludeRegex: pathIncludeRegex,
	}, nil
}

func parsePositiveNumber(value string, name string, fallback int) int {
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 || parsed > math.MaxInt {
		panic(fmt.Sprintf("%s must be a positive number, got %q", name, value))
	}

	return parsed
}

func loadApiAuth() (*ApiAuth, error) {
	token := strings.TrimSpace(os.Getenv("MTX_API_TOKEN"))
	if token != "" {
		return &ApiAuth{Scheme: "bearer", Token: token}, nil
	}

	username := strings.TrimSpace(os.Getenv("MTX_API_USER"))
	password := os.Getenv("MTX_API_PASSWORD")

	if username == "" {
		if password != "" {
			return nil, fmt.Errorf("MTX_API_PASSWORD set without MTX_API_USER")
		}
		return nil, nil
	}

	return &ApiAuth{Scheme: "basic", Username: username, Password: password}, nil
}

func loadPathIncludeRegex() (*regexp.Regexp, error) {
	pattern := strings.TrimSpace(os.Getenv("PATH_INCLUDE_REGEX"))
	if pattern == "" {
		return nil, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("PATH_INCLUDE_REGEX is not valid: %q", pattern)
	}

	return re, nil
}

func parseHostname(url string) string {
	// Strip scheme.
	schemeEnd := strings.Index(url, "://")
	if schemeEnd != -1 {
		url = url[schemeEnd+3:]
	}
	// Strip port and path.
	if idx := strings.Index(url, ":"); idx != -1 {
		url = url[:idx]
	}
	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}
	return url
}
