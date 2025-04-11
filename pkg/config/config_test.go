package config

import (
	"testing"
)

// TestNewDefaultConfig calls config.New() to generate default configuration and validates the configuration.
func TestNewDefaultConfig(t *testing.T) {
	c := New()
	err := c.ValidateConfig()

	if err != nil {
		t.Error("config.New() did not generate a valid default configuration. " + err.Error())
	}
}

// TestNewDefaultConfig calls config.New() to generate default configuration, overrides each property, and validates the configuration.
func TestNewCustomConfig(t *testing.T) {
	c := New()
	c.BotAction = "BLOCK"
	c.CacheUpdateInterval = "5m"
	c.LogLevel = "ERROR"
	c.RobotsSourceURL = "https://example.com"
	c.RobotsTXTFilePath = "custom-file-path.txt"
	err := c.ValidateConfig()

	if err != nil {
		t.Errorf("ValidateConfig() did not pass a custom configuration that should be considered valid. " + err.Error())
	}
}

// TestConfigBadLogLevel overrides a default config with an invalid LogLevel and checks that an error is raised by ValidateConfig().
func TestConfigBadLogLevel(t *testing.T) {
	c := New()
	c.LogLevel = "NONE"
	err := c.ValidateConfig()
	if err == nil {
		t.Error("ValidateConfig didn't fail an invalid LogLevel.")
	}
}

// TestConfigBadBotAction overrides a default config with an invalid BotAction and checks that an error is raised by ValidateConfig().
func TestConfigBadBotAction(t *testing.T) {
	c := New()
	c.BotAction = "Do a Flip"
	err := c.ValidateConfig()
	if err == nil {
		t.Error("ValidateConfig didn't fail an invalid BotAction.")
	}
}

// TestConfigBadRobotsSourceURL overrides a default config with an invalid RobotsSourceURL and checks that an error is raised by ValidateConfig().
func TestConfigBadRobotsSourceURL(t *testing.T) {
	c := New()
	c.RobotsSourceURL = "this is not a URL"
	err := c.ValidateConfig()
	if err == nil {
		t.Error("ValidateConfig didn't fail an invalid RobotsSourceURL.")
	}
}

// TestConfigBadCacheUpdateInterval overrides a default config with an invalid CacheUpdateInterval and checks that an error is raised by ValidateConfig().
func TestConfigBadCacheUpdateInterval(t *testing.T) {
	c := New()
	c.CacheUpdateInterval = "something time.ParseDuration can't parse"
	err := c.ValidateConfig()
	if err == nil {
		t.Error("ValidateConfig didn't fail an invalid CacheUpdateInterval.")
	}
}