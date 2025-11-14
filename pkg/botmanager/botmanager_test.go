package botmanager

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
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
	b, err := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	if err != nil {
		t.Error("unexpected error when initializing default bot manager: " + err.Error())
	}
	if tStart.Compare(b.nextUpdate) >= 0 {
		t.Error("BotUAManager's nextUpdate property was not updated as expected")
	}
	if len(b.botIndex) == 0 {
		t.Error("robots.txt index was not successfully retrieved")
	}
}

// TestNewBotManager calls botmanager.New() with the RobotsTXTDisallowAll config value set to true
func TestNewBotManagerDisallowAll(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	c.RobotsTXTDisallowAll = true
	_, err := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	if err != nil {
		t.Error("unexpected error when initializing bot manager with RobotsTXTDisallowAll: " + err.Error())
	}
}

// TestBotManagerBadURL tests exceptions inside BotUAManager.update() are properly handled and returned. Also validates that custom RobotsSourceURL config is respected.
func TestBotManagerBadURL(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()

	urls := []string{
		"%%",                                 // unparsable
		"https://somerandomhost.example.com", // dns failure
		"https://httpbin.io/json",            // malformed data
	}

	for _, u := range urls {
		t.Run(u, func(t *testing.T) {
			_, err := New(u, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
			if err == nil {
				t.Error("problematic RobotsSourceURL did not return an error when initializing BotUAManager: " + u)
			}
		})
	}
}

// TestGetBotIndex tests that a default configed BotUAManager can retrieve an index of robots
func TestGetBotIndex(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	b, _ := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	_ = b.refreshBotIndex()
	if len(b.botIndex) == 0 {
		t.Error("robots index with default configuration was empty")
	}

	// test retrieving a bot from the index
	want := "GPTBot"
	_, bInList := b.botIndex[want]
	if !bInList {
		t.Errorf("retrieved default robots index does not contain %s", want)
	}
}

// TestGetBotIndexMulti tests that a default configed BotUAManager can retrieve and merge multiple indexes
func TestGetBotIndexMulti(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	u := "https://cdn.jsdelivr.net/gh/ai-robots-txt/ai.robots.txt@latest/robots.json" + "," + "https://cdn.jsdelivr.net/gh/mitchellkrogza/nginx-ultimate-bad-bot-blocker@latest/robots.txt/robots.txt"

	b, _ := New(u, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	_ = b.refreshBotIndex()
	gotL := len(b.botIndex)
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
	b, _ := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	_ = b.refreshBotIndex()
	firstUpdate := b.nextUpdate

	time.Sleep(b.cacheUpdateInterval)
	_ = b.refreshBotIndex()
	secondUpdate := b.nextUpdate
	if secondUpdate.Compare(firstUpdate) != 1 {
		t.Error("BotUAManager cache refresh did not occur during GetBotIndex() call when expired")
	}
}

// TestBotIndexBadUpdate tests that a cache refresh attempt does not update the index if query returns invalid data
func TestBotIndexBadUpdate(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	c.CacheUpdateInterval = "5ns"
	b, _ := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	_ = b.refreshBotIndex()
	firstIndex := &b.botIndex

	b.sources = []parser.Source{{URL: "https://httpbin.org/json"}}
	time.Sleep(b.cacheUpdateInterval)
	_ = b.refreshBotIndex()
	secondIndex := &b.botIndex

	if firstIndex != secondIndex {
		t.Error("BotUAManager updated the cache with invalid values during a refresh")
	}
}

// TestBotIndexBadUpdateRetryInterval tests that the robotsSourceRetryInterval setting is used to rate limit source updates when an update request fails
func TestRobotSourceRetryInterval(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	c.CacheUpdateInterval = "5ns"
	c.RobotsSourceRetryInterval = "10ms"
	requestCount := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
		if r.URL.Path == "/robots.txt" {
			sampleTxt := `
			User-agent: GPTBot
			Disallow: /
			`
			_, _ = w.Write([]byte(sampleTxt))
			w.WriteHeader(http.StatusOK)
		}
	}))

	b, _ := New(s.URL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	attempts := 3
	// yaegi doesn't like a range over int loop
	// https://github.com/traefik/yaegi/issues/1701
	for i := 0; i < attempts; i++ { //nolint:intrange,modernize
		time.Sleep(b.cacheUpdateInterval)
		_ = b.refreshBotIndex()
		if requestCount != 1 {
			t.Error("BotUAManager attempted to retry a failed source update too soon")
		}
		if len(b.botIndex) != 0 {
			t.Error("BotUAManager unexpectedly populated botindex from invalid source")
		}
	}
	time.Sleep(b.sourceRetryInterval)
	b.sources = []parser.Source{{URL: s.URL + "/robots.txt"}}
	_ = b.refreshBotIndex()
	if requestCount != 2 {
		t.Error("BotUAManager did not retry requesting a source update after robotsSourceRetryInterval")
	}
	if len(b.botIndex) > 0 {
		t.Error("BotUAManager did not have a successful refresh after source became available")
	}
}

// TestBotIndexSearchCache tests that the bot index will retrieve previous results directly from the cache
func TestBotIndexSearchCache(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	bM, _ := New(exampleSource, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	botName, _, err := bM.Search(exampleLongString)
	if err != nil {
		t.Errorf("unexpected error when performing a search for '%s': %s", exampleLongString, err.Error())
	}

	newName := "foobar"
	bM.cache.set(exampleLongString, newName)

	updatedName, _, err := bM.Search(exampleLongString)
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
	c.CacheSize = 1
	bM, _ := New(exampleSource, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)

	bM.cache.set(exampleLongString, "")
	bM.cache.set(exampleShortString, "")
	_, ok := bM.cache.get(exampleLongString)

	if ok {
		t.Errorf("expected cache to be rolled over, but was not")
	}
}

// TestBotIndexSearchBadRefresh tests that an error is returned when an error is encountered refreshing bot sources when searching the bot index
func TestBotIndexSearchBadRefresh(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	c.CacheUpdateInterval = "1ns"
	b, err := New(exampleSource, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	if err != nil {
		t.Fatal("unexpected error constructing botmanager instance")
	}

	time.Sleep(b.cacheUpdateInterval)
	b.sources = []parser.Source{{URL: "http://localhost"}}
	_, _, err = b.Search(exampleLongString)

	if err == nil {
		t.Error("Search() did not return an error when a source refresh failed prior to the search")
	}
}

// TestBotIndexSearchSlow tests that the bot index can be searched via simple matching
func TestBotIndexSearchSlow(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	c.UseFastMatch = false
	bM, _ := New(exampleSource, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	botName, _, err := bM.Search(exampleLongString)
	if err != nil {
		t.Errorf("unexpected error when performing a slow search for '%s': %s", exampleLongString, err.Error())
	}
	if botName == "" {
		t.Errorf("slow search method did not return a match for '%s'", exampleLongString)
	}
}

// TestBotIndexSearchFast tests that the bot index can be searched via simple matching
func TestBotIndexSearchFast(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	bM, _ := New(exampleSource, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	bM.ahoCorasick = ahocorasick.NewFromIndex(bM.botIndex)
	botName, _, err := bM.Search(exampleLongString)
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
	_, _, err := bM.Search(exampleLongString)
	if err == nil {
		t.Error("expected an error when performing a search without first initializing the BotManager")
	}
}

// TestInitBadRobotsTxt tests that an error is returned by BotManager.New() when the robots.txt template file cannot be found
func TestInitBadRobotsTxt(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	c.RobotsTXTFilePath = "filenotexist.txt"
	_, err := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	if err == nil {
		t.Error("New() did not return an error when provided invalid robots.txt file")
	}
}

// badResponseWriter acts as a mock to force writing response content to fail
type badResponseWriter struct {
	io.Writer
}

func (f *badResponseWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

// TestInitBadRobotsTemplate tests that an error is returned when the robots.txt template file cannot be rendered
func TestInitBadRobotsTemplate(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	b, err := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	if err != nil {
		t.Fatal("unexpected error constructing botmanager instance")
	}

	w := &badResponseWriter{}
	err = b.RenderRobotsTxt(w, true)

	if err == nil {
		t.Error("RenderRobotsTxt() did not return an error when provided bad writer to write template content into")
	}
}

// TestRenderRobotsTxtBadRefresh tests that an error is returned when an error is encountered refreshing bot sources when rending the robots.txt template
func TestRenderRobotsTxtBadRefresh(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	c.CacheUpdateInterval = "1ns"
	b, err := New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
	if err != nil {
		t.Fatal("unexpected error constructing botmanager instance")
	}

	time.Sleep(b.cacheUpdateInterval)
	b.sources = []parser.Source{{URL: "http://localhost"}}
	w := &bytes.Buffer{}
	err = b.RenderRobotsTxt(w, true)

	if err == nil {
		t.Error("RenderRobotsTxt() did not return an error when a source refresh failed during render")
	}
}

// TestRenderRobotsTxt tests that RenderRobotsTxt() returns a cached instance of a valid robots.txt file
func TestRenderRobotsTxt(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sampleTxt := `
		User-agent: GPTBot
		Allow: /robots.txt
		Disallow: /
		`
		_, _ = w.Write([]byte(sampleTxt))
	}))
	bM, _ := New(s.URL+"/robots.txt", c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)

	w := &bytes.Buffer{}
	err := bM.RenderRobotsTxt(w, true)
	if err != nil {
		t.Error("unexpected error returned from RenderRobotsTxt: " + err.Error())
	}

	rendered := w.String()
	cached := bM.templateCache.String()
	hasUserAgent := strings.Contains(rendered, "User-agent: GPTBot")
	hasRule := strings.Contains(rendered, "Disallow: /")

	if !hasUserAgent || !hasRule {
		t.Error("RenderRobotsTxt() did not return a rendered template with expected content.")
	}
	if rendered != cached {
		t.Error("RenderRobotsTxt() did not return the same string that is stored in the cache. Expected: '" + cached + "', Got: '" + rendered + "'")
	}
}

// TestRenderRobotsTxtNoCache tests that RenderRobotsTxt() does not return a cached robots.txt
func TestRenderRobotsTxtNoCache(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sampleTxt := `
		User-agent: GPTBot
		Allow: /robots.txt
		Disallow: /
		`
		_, _ = w.Write([]byte(sampleTxt))
	}))
	bM, _ := New(s.URL+"/robots.txt", c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)

	bM.templateCache = &bytes.Buffer{}
	w := &bytes.Buffer{}
	err := bM.RenderRobotsTxt(w, false)
	if err != nil {
		t.Error("unexpected error returned from RenderRobotsTxt: " + err.Error())
	}

	rendered := w.String()
	cached := bM.templateCache.String()
	hasUserAgent := strings.Contains(rendered, "User-agent: GPTBot")
	hasRule := strings.Contains(rendered, "Disallow: /")

	if !hasUserAgent || !hasRule {
		t.Error("RenderRobotsTxt() did not return a rendered template with expected content. Got: " + rendered)
	}
	if rendered == cached {
		t.Error("RenderRobotsTxt() returned the cached robots.txt copy. Expected: '" + cached + "', Got: '" + rendered + "'")
	}
}

// TestRobotsTxtCacheCleared tests that the robots.txt rendered after a source refresh without changes matches the previous value
func TestRobotsTxtCacheCleared(t *testing.T) {
	log := logger.NewFromWriter("DEBUG", &testLogOut)
	c := config.New()
	c.CacheUpdateInterval = "5ns"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sampleTxt := `
User-agent: GPTBot
User-agent: TestBot
Allow: /robots.txt
Disallow: /
`
		_, _ = w.Write([]byte(sampleTxt))
	}))
	bM, _ := New(s.URL+"/robots.txt", c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)

	w1 := &bytes.Buffer{}
	err := bM.RenderRobotsTxt(w1, true)
	if err != nil {
		t.Error("unexpected error returned from RenderRobotsTxt: " + err.Error())
	}
	w1String := w1.String()

	_ = bM.refreshBotIndex()

	w2 := &bytes.Buffer{}
	err = bM.RenderRobotsTxt(w2, true)
	if err != nil {
		t.Error("unexpected error returned from RenderRobotsTxt: " + err.Error())
	}
	w2String := w2.String()

	w1Values := strings.Split(w1String, "\n")
	w2Values := strings.Split(w2String, "\n")
	for _, v := range w1Values {
		if !slices.Contains(w2Values, v) {
			t.Errorf("cached robots.txt did not match after a same-source refresh. First: '" + w1String + "', Second: '" + w2String + "'")
		}
	}
}
