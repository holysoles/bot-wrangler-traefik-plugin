package bot_wrangler_traefik_plugin_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/holysoles/bot-wrangler-traefik-plugin"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/config"
)

// most common user agent as of 3/31/2025 from https://microlink.io/user-agents
const RealUserAgent string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"
const BotUserAgent string = "GPTBot"

// TestWranglerInit tests that the plugin can be initialized (along with config), and can process a simple request cleanly
func TestWranglerInit(t *testing.T) {
	cfg := bot_wrangler_traefik_plugin.CreateConfig()

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	handler, err := bot_wrangler_traefik_plugin.New(ctx, next, cfg, "wrangler")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)
}

// TestWranglerInitBadConfig tests plugin behavior when invalid configuration is provided at startup
func TestWranglerInitBadConfig(t *testing.T) {
	cfg := bot_wrangler_traefik_plugin.CreateConfig()
	cfg.BotAction = "NOOP"

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	_, err := bot_wrangler_traefik_plugin.New(ctx, next, cfg, "wrangler")
	if err == nil {
		t.Error("New() did not return an error when provided invalid config")
	}
}

// TestWranglerInitBadRobotsTxt tests plugin behavior when the robots.txt template file cannot be found at startup
func TestWranglerInitBadRobotsTxt(t *testing.T) {
	cfg := bot_wrangler_traefik_plugin.CreateConfig()
	cfg.RobotsTXTFilePath = "filenotexist.txt"

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	_, err := bot_wrangler_traefik_plugin.New(ctx, next, cfg, "wrangler")
	if err == nil {
		t.Error("New() did not return an error when provided invalid robots.txt file")
	}
}

// TestWranglerInitBadRobotsIndex tests plugin behavior when an invalid robots index is supplied
func TestWranglerInitBadRobotsIndex(t *testing.T) {
	cfg := bot_wrangler_traefik_plugin.CreateConfig()
	cfg.RobotsSourceURL = "https://httpbin.org/status/404"

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	_, err := bot_wrangler_traefik_plugin.New(ctx, next, cfg, "wrangler")
	if err == nil {
		t.Error("New() did not return an error when provided invalid source to load robots index")
	}
}


// TestWranglerRobotsTxt tests that the plugin renders a valid robots.txt exclusions file when requested
func TestWranglerRobotsTxt(t *testing.T) {
	res := getWranglerResponse(t, "", "", "http://localhost/robots.txt")
	if res.StatusCode != http.StatusOK {
		t.Errorf("robots.txt page returned non-200 unexpectedly. Got: %d", res.StatusCode)
	}
	resBodyB, _ := io.ReadAll(res.Body)
	resBody := string(resBodyB)
	want := regexp.MustCompile("(User-agent: .+)+\nDisallow: /")
	if !want.MatchString(resBody) {
		t.Errorf("robots.txt page does not match expected format. Got: %s", resBody)
	}
}

// TestWranglerPassActions tests scenarios where a request (with User-Agent provided) is expected to pass
func TestWranglerPassActions(t *testing.T) {
	type scenario struct{
		userAgent string
		botAction string
	}
	passScenarios := []scenario{
		{
			userAgent: RealUserAgent,
			botAction: "PASS",
		},
		{
			userAgent: RealUserAgent,
			botAction: "LOG",
		},
		{
			userAgent: RealUserAgent,
			botAction: "BLOCK",
		},
		{
			userAgent: BotUserAgent,
			botAction: "PASS",
		},
		{
			userAgent: BotUserAgent,
			botAction: "LOG",
		},
	}

	for _, s := range passScenarios{
		res := getWranglerResponse(t, s.userAgent, s.botAction, "")
		resBody, _ := io.ReadAll(res.Body)
		resUnmodified := res.StatusCode == http.StatusOK && len(res.Header) == 0 && len(resBody) == 0
		if !resUnmodified {
			t.Errorf("request passed to plugin with BotAction '%s' from User-Agent '%s' had response unexpectedly modified", s.botAction, s.userAgent)
		}
	}
}

// TestWranglerBlockAction tests the plugin behavior when a request, based on User-Agent, should be blocked
func TestWranglerBlockAction(t *testing.T) {
	type jsonBody struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	var blockedBody jsonBody
	ua := BotUserAgent
	action := config.BotActionBlock
	res := getWranglerResponse(t, ua, action, "")
	resBody, _ := io.ReadAll(res.Body)
	err := json.Unmarshal(resBody, &blockedBody)
	if err != nil {
		t.Fatal(err)
	}
	want := res.StatusCode == http.StatusForbidden && res.Header.Get("Content-Type") == "application/json" && blockedBody.Error == "Forbidden" && blockedBody.Message == "Your user agent is associated with a large language model (LLM) and is banned from accessing this resource due to scraping activities."
	if !want {
		t.Errorf("request passed to plugin with BotAction '%s' from User-Agent '%s' did not match expected response", action, BotUserAgent)
	}
}

// getWranglerResponse is a helper function to setup a context, plugin, responsewriter, etc to generate a response. UserAgent, Botaction, and request URL can be specified
func getWranglerResponse(t *testing.T, uA string, bA string, url string) *http.Response {
	t.Helper()
	if url == "" {
		url = "http://localhost"
	}
	cfg := bot_wrangler_traefik_plugin.CreateConfig()
	if bA != "" {
		cfg.BotAction = bA
	}

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	handler, err := bot_wrangler_traefik_plugin.New(ctx, next, cfg, "wrangler")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}

	if uA != "" {
		req.Header.Set("User-Agent", uA)
	}
	handler.ServeHTTP(recorder, req)
	res := recorder.Result()
	return res
}
