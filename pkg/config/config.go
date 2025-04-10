package config

import (
	"fmt"
	"slices"
)

// define constants for enum validation
const (
	BotActionPass  = "PASS"
	BotActionLog   = "LOG"
	BotActionBlock = "BLOCK"

	LogLevelDebug = "DEBUG"
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
		LogLevel:           "INFO", //TODO info
		RobotsTXTFilePath:  "robots.txt",
		UserAgentSourceURL: "https://raw.githubusercontent.com/ai-robots-txt/ai.robots.txt/refs/heads/main/robots.json",
	}
}

func (c *Config) ValidateConfig() error {
	//TODO validate loglevel

	if !slices.Contains([]string{BotActionPass, BotActionBlock, BotActionPass}, c.BotAction) {
		return fmt.Errorf("CaptchaProvider: must be one of '%s', '%s', '%s'", BotActionPass, BotActionBlock, BotActionPass)
	}

	// TODO validate useragentsourceurl

	return nil
}