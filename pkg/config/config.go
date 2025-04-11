// Package config provides utilities for loading and validating provided configuration to the plugin
package config

import (
	"fmt"
	"net/url"
	"slices"
	"time"
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
	BotAction           string `json:"botAction,omitempty"`
	CacheUpdateInterval string `json:"cacheUpdateInterval,omitempty"`
	LogLevel            string `json:"logLevel,omitempty"`
	RobotsTXTFilePath   string `json:"robotsTxtFilePath,omitempty"`
	RobotsSourceURL     string `json:"robotsSourceUrl,omitempty"`
}

// New creates the default plugin configuration.
func New() *Config {
	return &Config{
		BotAction:           "LOG",
		CacheUpdateInterval: "1m",
		LogLevel:            "INFO",
		RobotsTXTFilePath:   "robots.txt",
		RobotsSourceURL:     "https://raw.githubusercontent.com/ai-robots-txt/ai.robots.txt/refs/heads/main/robots.json",
	}
}

// ValidateConfig provides a way to validate an initialized Config instance
func (c *Config) ValidateConfig() error {
	// LogLevel
	if !slices.Contains([]string{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}, c.LogLevel) {
		return fmt.Errorf("ValidateConfig: LogLevel must be one of '%s', '%s', '%s', '%s'. Got '%s'", LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, c.LogLevel)
	}
	// BotAction
	if !slices.Contains([]string{BotActionPass, BotActionLog, BotActionBlock}, c.BotAction) {
		return fmt.Errorf("ValidateConfig: BotAction must be one of '%s', '%s', '%s'. Got '%s'", BotActionPass, BotActionLog, BotActionBlock, c.BotAction)
	}
	// RobotsSourceURL
	_, err := url.ParseRequestURI(c.RobotsSourceURL)
	if err != nil {
		return fmt.Errorf("ValidateConfig: RobotsSourceURL must be a valid URL. Got '%s'", c.RobotsSourceURL)
	}
	//CacheUpdateInterval
	_, err = time.ParseDuration(c.CacheUpdateInterval)
	if err != nil {
		return fmt.Errorf("ValidateConfig: CacheUpdateInterval must be a time duration string. Got '%s'", c.CacheUpdateInterval)
	}

	return nil
}
