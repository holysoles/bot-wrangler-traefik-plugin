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
		t.Error("ValidateConfig() did not pass a custom configuration that should be considered valid. " + err.Error())
	}
}

// TestConfigBadEnabled overrides a default config with an invalid Enabled value and checks that an error is raised by ValidateConfig().
func TestConfigBadEnabled(t *testing.T) {
	c := New()
	c.Enabled = "_____"
	err := c.ValidateConfig()
	if err == nil {
		t.Error("ValidateConfig didn't fail an invalid Enabled value.")
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

// TestConfigBadBotBlockHTTPCode overrides a default config with an invalid BotBlockHTTPCode and checks that an error is raised by ValidateConfig().
func TestConfigBadBotBlockHTTPCode(t *testing.T) {
	c := New()
	c.BotBlockHTTPCode = 999
	err := c.ValidateConfig()
	if err == nil {
		t.Error("ValidateConfig didn't fail an invalid BotBlockHTTPCode.")
	}
}

// TestConfigBadBotProxyURL overrides a default config with an invalid BotProxyURL and checks that an error is raised by ValidateConfig().
func TestConfigBadBotProxyURL(t *testing.T) {
	c := New()
	c.BotProxyURL = "this is not a URL"
	err := c.ValidateConfig()
	if err == nil {
		t.Error("ValidateConfig didn't fail an invalid BotProxyURL.")
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

// TestConfigBadCacheSize overrides a default config with an invalid CacheSize and checks that an error is raised by ValidateConfig().
func TestConfigBadCacheSize(t *testing.T) {
	c := New()
	c.CacheSize = -5
	err := c.ValidateConfig()
	if err == nil {
		t.Error("ValidateConfig didn't fail an invalid CacheSize.")
	}
}

// TestConfigBadSourceRetryInterval overrides a default config with an invalid RobotsSourceRetryInterval and checks that an error is raised by ValidateConfig().
func TestConfigBadSourceRetryInterval(t *testing.T) {
	c := New()
	c.RobotsSourceRetryInterval = "something time.ParseDuration can't parse"
	err := c.ValidateConfig()
	if err == nil {
		t.Error("ValidateConfig didn't fail an invalid RobotsSourceRetryInterval.")
	}
}
