// Package config provides utilities for loading and validating provided configuration to the plugin
package config

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"
)

// define constants for enum validation.
const (
	BotActionPass  = "PASS"
	BotActionLog   = "LOG"
	BotActionBlock = "BLOCK"
	BotActionProxy = "PROXY"

	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"

	defaultMaxCacheSize = 500
)

// default robots.txt template that will be rendered.
const (
	RobotsTxtDefault = `
{{ range $agent := .UserAgentList }}
User-agent: {{ $agent }}
{{- end }}
Disallow: /
`
)

// robots.txt template that will disallow all user-agents.
const (
	RobotsTxtDisallowAll = `
User-agent: *
Disallow: /
`
)

// Config the plugin configuration.
type Config struct {
	Enabled              string `json:"enabled,omitempty"`
	BotAction            string `json:"botAction,omitempty"`
	BotBlockHTTPCode     int    `json:"botBlockHttpCode,omitempty"`
	BotBlockHTTPResponse string `json:"botBlockHttpResponse,omitempty"`
	BotProxyURL          string `json:"botProxyUrl,omitempty"`
	CacheSize            int    `json:"cacheSize,omitempty"`
	CacheUpdateInterval  string `json:"cacheUpdateInterval,omitempty"`
	LogLevel             string `json:"logLevel,omitempty"`
	RobotsTXTFilePath    string `json:"robotsTxtFilePath,omitempty"`
	RobotsTXTDisallowAll bool   `json:"robotsTxtDisallowAll,omitempty"`
	RobotsSourceURL      string `json:"robotsSourceUrl,omitempty"`
	UseFastMatch         bool   `json:"useFastMatch,omitempty"`
}

// New creates the default plugin configuration.
func New() *Config {
	return &Config{
		Enabled:              "true",
		BotAction:            "LOG",
		BotBlockHTTPCode:     http.StatusForbidden,
		BotBlockHTTPResponse: "Your user agent is associated with a large language model (LLM) and is blocked from accessing this resource",
		BotProxyURL:          "",
		CacheSize:            defaultMaxCacheSize,
		CacheUpdateInterval:  "24h",
		LogLevel:             "INFO",
		RobotsTXTFilePath:    "",
		RobotsTXTDisallowAll: false,
		RobotsSourceURL:      "https://cdn.jsdelivr.net/gh/ai-robots-txt/ai.robots.txt@v1.42/robots.json",
		UseFastMatch:         true,
	}
}

// ValidateConfig provides a way to validate an initialized Config instance.
func (c *Config) ValidateConfig() error {
	// Enabled
	_, err := strconv.ParseBool(c.Enabled)
	if err != nil {
		return err
	}
	// LogLevel
	if !slices.Contains([]string{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}, c.LogLevel) {
		return fmt.Errorf("ValidateConfig: LogLevel must be one of '%s', '%s', '%s', '%s'. Got '%s'", LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, c.LogLevel)
	}
	// BotAction
	if !slices.Contains([]string{BotActionPass, BotActionLog, BotActionBlock, BotActionProxy}, c.BotAction) {
		return fmt.Errorf("ValidateConfig: BotAction must be one of '%s', '%s', '%s', '%s'. Got '%s'", BotActionPass, BotActionLog, BotActionBlock, BotActionProxy, c.BotAction)
	}
	// BotBlockHttpCode
	if http.StatusText(c.BotBlockHTTPCode) == "" {
		return fmt.Errorf("ValidateConfig: BotBlockHTTPCode must be a valid HTTP response code. Got '%d'", c.BotBlockHTTPCode)
	}
	// BotBlockHttpResponse
	// no validation. We'll allow any string to be specified here.
	// BotProxyURL
	if c.BotProxyURL != "" {
		_, err = url.ParseRequestURI(c.BotProxyURL)
		if err != nil {
			return fmt.Errorf("ValidateConfig: BotProxyURL must be a valid URL. Got '%s'", c.BotProxyURL)
		}
	}
	// RobotsSourceURL
	_, err = url.ParseRequestURI(c.RobotsSourceURL)
	if err != nil {
		return fmt.Errorf("ValidateConfig: RobotsSourceURL must be a valid URL. Got '%s'", c.RobotsSourceURL)
	}
	// CacheUpdateInterval
	_, err = time.ParseDuration(c.CacheUpdateInterval)
	if err != nil {
		return fmt.Errorf("ValidateConfig: CacheUpdateInterval must be a time duration string. Got '%s'", c.CacheUpdateInterval)
	}
	// CacheSize
	if c.CacheSize <= 0 {
		return fmt.Errorf("ValidateConfig: CacheSize must be a positive integer. Got '%d'", c.CacheSize)
	}
	// UseFastMatch
	// no validation since boolean

	return nil
}
