package botmanager

import (
	"testing"
	"time"
	"bytes"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/config"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
)

var testLogOut bytes.Buffer //nolint:gochecknoglobals

// TestNewBotManager calls botmanager.New() with default configuration and validates its properties
func TestNewBotManager(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	tStart := time.Now()
	c := config.New()
	b, err := New(c.RobotsSourceURL, c.CacheUpdateInterval, log)
	if err != nil {
		t.Error("unexpected error when initializing default bot manager: " + err.Error())
	}
	if tStart.Compare(b.lastUpdate) >= 0 {
		t.Error("BotUAManager's lastUpdate property was not updated as expected")
	}
	if len(b.botIndex) == 0 {
		t.Error("robots.txt index was not successfully retrieved")
	}
}

// TestBotManagerBadURL tests exceptions inside BotUAManager.update() are properly handled and returned. Also validates that custom RobotsSourceURL config is respected.
func TestBotManagerBadURL(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()

	c.RobotsSourceURL = "%%"
	_, err := New(c.RobotsSourceURL, c.CacheUpdateInterval, log)
	if err == nil {
		t.Error("Malformed RobotsSourceURL did not return an error when initializing BotUAManager")
	}

	c.RobotsSourceURL = "https://somerandomhost.example.com"
	_, err = New(c.RobotsSourceURL, c.CacheUpdateInterval, log)
	if err == nil {
		t.Error("Unreachable RobotsSourceURL did not return an error when initializing BotUAManager")
	}

	c.RobotsSourceURL = "https://httpbin.org/json"
	_, err = New(c.RobotsSourceURL, c.CacheUpdateInterval, log)
	if err == nil {
		t.Error("RobotsSourceURL that returns invalid data did not return an error when initializing BotUAManager")
	}
}

// TestGetBotIndex tests that a default configed BotUAManager can retrieve an index of robots
func TestGetBotIndex(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	b, _ := New(c.RobotsSourceURL, c.CacheUpdateInterval, log)
	botI, err := b.GetBotIndex()
	if err != nil {
		t.Error("Unable to get robots index with default configuration. " + err.Error())
	}
	if len(botI) == 0 {
		t.Error("robots index with default configuration was empty")
	}

	// test retrieving a bot from the index
	want := "GPTBot"
	_, bInList := botI[want]
	if !bInList {
		t.Errorf("retrieved default robots index does not contain %s", want)
	}
}

// TestBoxIndexCacheRefresh tests that a call to GetBotIndex() triggers a cache refresh if the cache is considered expired
func TestBotIndexCacheRefresh(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	c.CacheUpdateInterval = "5ns"
	b, _ := New(c.RobotsSourceURL, c.CacheUpdateInterval, log)
	_, _ = b.GetBotIndex()
	firstUpdate := b.lastUpdate
	time.Sleep(b.cacheUpdateInterval)
	_, _ = b.GetBotIndex()
	secondUpdate := b.lastUpdate
	if secondUpdate.Compare(firstUpdate) != 1 {
		t.Error("BotUAManager cache refresh did not occur during GetBotIndex() call when expired")
	}
}

// TestBotIndexBadUpdate tests that a cache refresh attempt does not update the index if query returns invalid data
func TestBotIndexBadUpdate(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	c.CacheUpdateInterval = "5ns"
	b, _ := New(c.RobotsSourceURL, c.CacheUpdateInterval, log)
	_, _ = b.GetBotIndex()
	firstUpdate := b.lastUpdate
	b.url = "https://httpbin.org/json"
	time.Sleep(b.cacheUpdateInterval)
	_, err := b.GetBotIndex()
	if b.lastUpdate != firstUpdate {
		t.Error("BotUAManager updated the cache with invalid values during a refresh")
	}
	if err == nil {
		t.Error("BotUAManager.GetBotIndex() did not return an error during a problematic refresh")
	}
}
