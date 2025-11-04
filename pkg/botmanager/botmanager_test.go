package botmanager

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/ahocorasick"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/config"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/parser"
)

var testLogOut bytes.Buffer //nolint:gochecknoglobals

// TestNewBotManager calls botmanager.New() with default configuration and validates its properties
func TestNewBotManager(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	tStart := time.Now()
	c := config.New()
	b, err := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch)
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
	_, err := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch)
	if err == nil {
		t.Error("Malformed RobotsSourceURL did not return an error when initializing BotUAManager")
	}

	c.RobotsSourceURL = "https://somerandomhost.example.com"
	_, err = New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch)
	if err == nil {
		t.Error("Unreachable RobotsSourceURL did not return an error when initializing BotUAManager")
	}

	c.RobotsSourceURL = "https://httpbin.io/json"
	_, err = New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch)
	if err == nil {
		t.Error("RobotsSourceURL that returns invalid data did not return an error when initializing BotUAManager")
	}
}

// TestGetBotIndex tests that a default configed BotUAManager can retrieve an index of robots
func TestGetBotIndex(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	b, _ := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch)
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

// TestGetBotIndexMulti tests that a default configed BotUAManager can retrieve and merge multiple indexes
func TestGetBotIndexMulti(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	u := "https://cdn.jsdelivr.net/gh/ai-robots-txt/ai.robots.txt@latest/robots.json" + "," + "https://cdn.jsdelivr.net/gh/mitchellkrogza/nginx-ultimate-bad-bot-blocker@latest/robots.txt/robots.txt"

	b, _ := New(u, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch)
	botI, err := b.GetBotIndex()
	if err != nil {
		t.Error("Unable to get robots index with default configuration. " + err.Error())
	}
	gotL := len(botI)
	// approximate ai robots json at > 100 entries, bad bots at 50+
	getL := 100 + 50
	if gotL < getL {
		t.Errorf("expected at least %d bot entries, got %d", getL, gotL)
	}
}

// TestBoxIndexCacheRefresh tests that a call to GetBotIndex() triggers a cache refresh if the cache is considered expired
func TestBotIndexCacheRefresh(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	c.CacheUpdateInterval = "5ns"
	b, _ := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch)
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
	b, _ := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch)
	_, _ = b.GetBotIndex()
	firstUpdate := b.lastUpdate
	b.sources = []parser.Source{{URL: "https://httpbin.org/json"}}
	time.Sleep(b.cacheUpdateInterval)
	_, err := b.GetBotIndex()
	if b.lastUpdate != firstUpdate {
		t.Error("BotUAManager updated the cache with invalid values during a refresh")
	}
	if err == nil {
		t.Error("BotUAManager.GetBotIndex() did not return an error during a problematic refresh")
	}
}

// TestBotIndexSearchCache tests that the bot index will retrieve previous results directly from the cache
func TestBotIndexSearchCache(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	bM, _ := New(exampleSource, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch)
	botName, err := bM.Search(exampleLongString)
	if err != nil {
		t.Errorf("unexpected error when performing a search for '%s': %s", exampleLongString, err.Error())
	}

	newName := "foobar"
	bM.cache.set(exampleLongString, newName)

	updatedName, err := bM.Search(exampleLongString)
	if err != nil {
		t.Errorf("unexpected error when performing a search for '%s': %s", exampleLongString, err.Error())
	}
	if botName == updatedName {
		t.Errorf("expected overwritten cache value '%s' to be returned, got '%s'", newName, updatedName)
	}
}

// TestBotIndexSearchCacheRollover tests that the bot index's user-agent cache rolls over when exceeded
func TestBotIndexSearchCacheRollover(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	bM, _ := New(exampleSource, c.CacheUpdateInterval, log, 1, c.UseFastMatch)

	bM.cache.set(exampleLongString, "")
	fmt.Printf("DEBUG: before %d\n", len(bM.cache.data))

	bM.cache.set(exampleShortString, "")
	fmt.Printf("DEBUG: after %d\n", len(bM.cache.data))
	_, ok := bM.cache.get(exampleLongString)

	if ok {
		t.Errorf("expected cache to be rolled over, but was not")
	}
}

// TestBotIndexSearchSlow tests that the bot index can be search via simple matching
func TestBotIndexSearchSlow(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	bM, _ := New(exampleSource, c.CacheUpdateInterval, log, c.CacheSize, false)
	botName, err := bM.Search(exampleLongString)
	if err != nil {
		t.Errorf("unexpected error when performing a slow search for '%s': %s", exampleLongString, err.Error())
	}
	if botName == "" {
		t.Errorf("slow search method did not return a match for '%s'", exampleLongString)
	}
}

// TestBotIndexSearchFast tests that the bot index can be search via simple matching
func TestBotIndexSearchFast(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	bM, _ := New(exampleSource, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch)
	bM.ahoCorasick = ahocorasick.NewFromIndex(bM.botIndex)
	botName, err := bM.Search(exampleLongString)
	if err != nil {
		t.Errorf("unexpected error when performing a fast search for '%s': %s", exampleLongString, err.Error())
	}
	if botName == "" {
		t.Errorf("slow search method did not return a match for '%s'", exampleLongString)
	}
}

// TestBotIndexSearchNoInit tests that an error is returned when attempting a search with an uninitialized bot manager
func TestBotIndexSearchNoInit(t *testing.T) {
	bM := BotUAManager{}
	bM.ahoCorasick = ahocorasick.NewFromIndex(bM.botIndex)
	_, err := bM.Search(exampleLongString)
	if err == nil {
		t.Error("expected an error when performing a search without first initializing the BotManager")
	}
}
