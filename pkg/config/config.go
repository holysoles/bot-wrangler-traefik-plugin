package config

import (
	"fmt"
	"slices"
	"net/url"
)

// define constants for enum validation
const (
	BotActionPass  = "PASS"
	BotActionLog   = "LOG"
	BotActionBlock = "BLOCK"

	LogLevelDebug  = "DEBUG"
	LogLevelInfo   = "INFO"
	LogLevelWarn   = "WARN"
	LogLevelError  = "ERROR"
)

// Config the plugin configuration.
type Config struct {
	LogLevel           string `json:"logLevel,omitempty"`
	BotAction          string `json:"botAction,omitempty"`
	RobotsTXTFilePath  string `json:"robotsTxtFilePath,omitempty"`
	UserAgentSourceURL string `json:"userAgentSourceUrl,omitempty"`
}

// New creates the default plugin configuration.
func New() *Config {
	return &Config{
		BotAction:          BotActionPass,
		LogLevel:           "INFO",
		RobotsTXTFilePath:  "robots.txt",
		UserAgentSourceURL: "https://raw.githubusercontent.com/ai-robots-txt/ai.robots.txt/refs/heads/main/robots.json",
	}
}

func (c *Config) ValidateConfig() error {
	// LogLevel
	if !slices.Contains([]string{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}, c.LogLevel) {
		return fmt.Errorf("ValidateConfig: LogLevel must be one of '%s', '%s', '%s', '%s'. Got '%s'", LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, c.LogLevel)
	}
	// BotAction
	if !slices.Contains([]string{BotActionPass, BotActionBlock, BotActionPass}, c.BotAction) {
		return fmt.Errorf("ValidateConfig: BotAction must be one of '%s', '%s', '%s'. Got '%s'", BotActionPass, BotActionBlock, BotActionPass, c.BotAction)
	}
	// UserAgentSourceURL
	_, err := url.Parse(c.UserAgentSourceURL)
	if err != nil {
		return fmt.Errorf("ValidateConfig: UserAgentSourceURL must be a valid URL. Got '%s'", c.UserAgentSourceURL)
	}

	return nil
}