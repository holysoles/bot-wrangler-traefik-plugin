package bot_wrangler_traefik_plugin //nolint:revive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/config"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
)

// most common user agent as of 3/31/2025 from https://microlink.io/user-agents
const (
	RealUserAgent = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36`
	BotUserAgent  = `Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; GPTBot/1.0; +https://openai.com/gptbot)`
)

// We need to suppress logging, and in some cases validate that logs were written
// init sets up the testing environment and helpers
var testLogOut bytes.Buffer

// TestWranglerInit tests that the plugin can be initialized (along with config), and can process a simple request cleanly
func TestWranglerInit(t *testing.T) {
	cfg := CreateConfig()

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	h, err := New(ctx, next, cfg, "wrangler")
	if err != nil {
		t.Fatal(err)
	}
	w, ok := h.(*Wrangler)
	if !ok {
		t.Error("unable to assert handler as type Wrangler")
	}
	w.log = logger.NewFromWriter(config.LogLevelDebug, &testLogOut)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(recorder, req)
}

// TestWranglerInitBadConfig tests plugin behavior when invalid configuration is provided at startup
func TestWranglerInitBadConfig(t *testing.T) {
	cfg := CreateConfig()
	cfg.BotAction = "NOOP"

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	_, err := New(ctx, next, cfg, "wrangler")
	if err == nil {
		t.Error("New() did not return an error when provided invalid config")
	}
}

// TestWranglerInitBadBotProxyURL tests plugin behavior when the BotProxy URL provided is invalid
func TestWranglerInitBadBotProxyURL(t *testing.T) {
	cfg := CreateConfig()
	cfg.BotProxyURL = "%%"

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	_, err := New(ctx, next, cfg, "wrangler")
	if err == nil {
		t.Error("New() did not return an error when provided invalid bot proxy URL")
	}
}

// badResponseWriter acts as a mock to force writing response content to fail
type badResponseWriter struct {
	http.ResponseWriter
}

func (f *badResponseWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

// TestWranglerInitBadRobotsTemplate tests plugin behavior when the robots.txt template file cannot be rendered
func TestWranglerInitBadRobotsTemplate(t *testing.T) {
	testLogOut.Reset()
	cfg := CreateConfig()

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	h, err := New(ctx, next, cfg, "wrangler")
	if err != nil {
		t.Fatal(err)
	}

	w, ok := h.(*Wrangler)
	if !ok {
		t.Error("unable to assert handler as type Wrangler")
	}
	w.log = logger.NewFromWriter(config.LogLevelError, &testLogOut)

	recorder := &badResponseWriter{ResponseWriter: httptest.NewRecorder()}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/robots.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(recorder, req)

	msg := "ServeHTTP: Error rendering robots.txt template."
	want := regexp.MustCompile(`.* level=ERROR msg="` + msg + `.+".*`)
	got := testLogOut.String()
	if !want.MatchString(got) {
		t.Errorf("rendering invalid template file during request did not write the expected error message. Wanted: msg=\"%s\". Got: %s", msg, got)
	}
}

// TestWranglerBadBlockResponse tests the plugin behavior when a block response cannot be properly encoded to JSON
func TestWranglerBadBlockResponse(t *testing.T) {
	testLogOut.Reset()
	cfg := CreateConfig()
	cfg.BotAction = config.BotActionBlock
	ua := BotUserAgent

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	h, err := New(ctx, next, cfg, "wrangler")
	if err != nil {
		t.Fatal(err)
	}
	w, ok := h.(*Wrangler)
	if !ok {
		t.Error("unable to assert handler as type Wrangler")
	}
	w.log = logger.NewFromWriter(config.LogLevelError, &testLogOut)

	recorder := &badResponseWriter{ResponseWriter: httptest.NewRecorder()}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", ua)
	h.ServeHTTP(recorder, req)

	msg := "ServeHTTP: Error when rendering JSON for block response. Sending no content in reply. Error:"
	want := regexp.MustCompile(".*level=ERROR msg=\"" + msg + "(.+)?\".*")
	got := testLogOut.String()
	if !want.MatchString(got) {
		t.Errorf("failing to render block response JSON did not write the expected error. Wanted: '%s' Got: %s", msg, got)
	}
}

// TestWranglerInitBadRobotsIndex tests plugin behavior when an invalid robots index is supplied
func TestWranglerInitBadRobotsIndex(t *testing.T) {
	cfg := CreateConfig()
	cfg.RobotsSourceURL = "https://httpbin.io/status/404"

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	_, err := New(ctx, next, cfg, "wrangler")
	if err == nil {
		t.Error("New() did not return an error when provided invalid source to load robots index")
	}
}

// getWrangler is a helper function to initialize a Wrangler instance
func getWrangler(t *testing.T, bA string, disable bool, disallowAll bool) *Wrangler {
	t.Helper()
	cfg := CreateConfig()
	if bA != "" {
		cfg.BotAction = bA
	}
	if disable {
		cfg.Enabled = "false"
	}
	if disallowAll {
		cfg.RobotsTXTDisallowAll = true
	}

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	h, err := New(ctx, next, cfg, "wrangler")
	if err != nil {
		t.Fatal(err)
	}
	w, ok := h.(*Wrangler)
	if !ok {
		t.Error("unable to assert handler as type Wrangler")
	}
	w.log = logger.NewFromWriter(config.LogLevelInfo, &testLogOut)

	return w
}

// getWranglerResponse is a helper function to setup a context, plugin, responsewriter, etc to generate a response. UserAgent, Botaction, and request URL can be specified
func getWranglerResponse(t *testing.T, w *Wrangler, url string, uA string) *http.Response {
	t.Helper()
	if url == "" {
		url = "http://localhost"
	}

	ctx := context.Background()

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}

	if uA != "" {
		req.Header.Set("User-Agent", uA)
	}
	w.ServeHTTP(recorder, req)
	res := recorder.Result()
	return res
}

// TestWranglerDisabled tests that the plugin simply returns and exits early
func TestWranglerDisabled(t *testing.T) {
	w := getWrangler(t, "", false, false)
	res := getWranglerResponse(t, w, "http://localhost/robots.txt", "")
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

// TestWranglerRobotsTxt tests that the plugin renders a valid robots.txt exclusions file when requested
func TestWranglerRobotsTxt(t *testing.T) {
	w := getWrangler(t, "", true, false)
	res := getWranglerResponse(t, w, "http://localhost/robots.txt", "")
	if res.StatusCode != http.StatusOK {
		t.Errorf("disabled plugin request returned non-200 unexpectedly. Got: %d", res.StatusCode)
	}
}

// TestWranglerRobotsTxtDisallowAll tests that the plugin renders a robots.txt with all user-agents disallowed when the config flag is specified
func TestWranglerRobotsTxtDisallowAll(t *testing.T) {
	w := getWrangler(t, "", false, true)
	res := getWranglerResponse(t, w, "http://localhost/robots.txt", "")
	if res.StatusCode != http.StatusOK {
		t.Errorf("robots.txt page returned non-200 unexpectedly. Got: %d", res.StatusCode)
	}
	resBodyB, _ := io.ReadAll(res.Body)
	resBody := string(resBodyB)
	want := regexp.MustCompile("User-agent: \\*\nDisallow: \\/")
	if !want.MatchString(resBody) {
		t.Errorf("robots.txt did not contain a wildcard for disallowed user-agents. Got: %s", resBody)
	}
}

// TestWranglerPassActions tests scenarios where a request (with User-Agent provided) is expected to pass
func TestWranglerPassActions(t *testing.T) {
	type scenario struct {
		userAgent         string
		botAction         string
		expectHeaderCount int
	}
	passScenarios := []scenario{
		{
			userAgent:         RealUserAgent,
			botAction:         "PASS",
			expectHeaderCount: 0,
		},
		{
			userAgent:         RealUserAgent,
			botAction:         "LOG",
			expectHeaderCount: 0,
		},
		{
			userAgent:         RealUserAgent,
			botAction:         "BLOCK",
			expectHeaderCount: 0,
		},
		{
			userAgent:         BotUserAgent,
			botAction:         "PASS",
			expectHeaderCount: 1,
		},
		{
			userAgent:         BotUserAgent,
			botAction:         "LOG",
			expectHeaderCount: 1,
		},
	}

	for _, s := range passScenarios {
		scenarioName := fmt.Sprintf("BotAction:%s,UA:%s", s.botAction, s.userAgent)
		t.Run(scenarioName, func(t *testing.T) {
			w := getWrangler(t, s.botAction, false, false)
			res := getWranglerResponse(t, w, "", s.userAgent)
			resBody, _ := io.ReadAll(res.Body)
			resUnmodified := res.StatusCode == http.StatusOK && len(res.Header) == s.expectHeaderCount && len(resBody) == 0
			if !resUnmodified {
				t.Errorf("request passed to plugin had response unexpectedly modified")
			}
		})
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
	w := getWrangler(t, action, false, false)
	res := getWranglerResponse(t, w, "", ua)
	resBody, _ := io.ReadAll(res.Body)
	err := json.Unmarshal(resBody, &blockedBody)
	if err != nil {
		t.Fatal(err)
	}
	want := res.StatusCode == http.StatusForbidden && res.Header.Get("Content-Type") == "application/json" && blockedBody.Error == "Forbidden" &&
		blockedBody.Message == "Your user agent is associated with a large language model (LLM) and is blocked from accessing this resource"
	if !want {
		t.Errorf("request passed to plugin with BotAction '%s' from User-Agent '%s' did not match expected response", action, BotUserAgent)
	}
	uaEscape := regexp.QuoteMeta(ua)
	wantLog := regexp.MustCompile(`.* level=INFO msg="ServeHTTP: User agent '` + uaEscape + `' considered AI Robot." pluginName=bot-wrangler-traefik-plugin userAgent="?` + uaEscape +
		`"? sourceIP="?.*"? requestedPath="?.*"? remediationAction=BLOCK operator="?.+"? respectsRobotsTxt="?.+"? function="?.+"? description="?.+"?` + "\n")
	got := testLogOut.String()
	if !wantLog.MatchString(got) {
		t.Error("blocked bot request did not log expected info. Got: " + got)
	}
}

// TestWranglerCacheActions tests that plugin behavior is consistent before and after caching the user-agent.
func TestWranglerCacheActions(t *testing.T) {
	type scenario struct {
		userAgent string
		outcome   int
	}
	action := "BLOCK"
	passScenarios := []scenario{
		{
			userAgent: RealUserAgent,
			outcome:   http.StatusOK,
		},
		{
			userAgent: RealUserAgent,
			outcome:   http.StatusOK,
		},
		{
			userAgent: BotUserAgent,
			outcome:   http.StatusForbidden,
		},
		{
			userAgent: BotUserAgent,
			outcome:   http.StatusForbidden,
		},
	}

	w := getWrangler(t, action, false, false)
	for _, s := range passScenarios {
		t.Run(s.userAgent, func(t *testing.T) {
			res := getWranglerResponse(t, w, "", s.userAgent)
			got := res.StatusCode
			want := got == s.outcome
			if !want {
				t.Errorf("plugin response did not return expected status %d, got %d", s.outcome, got)
			}
		})
	}
}

// TestWranglerProxyAction tests that the plugin proxies bot requests when specified via config, to the specified backend server
func TestWranglerProxyAction(t *testing.T) {
	want := "the backend server has been reached by the reverse proxy"
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Helper()
		_, err := fmt.Fprint(w, want)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer backendServer.Close()

	cfg := CreateConfig()
	cfg.BotProxyURL = backendServer.URL
	cfg.BotAction = config.BotActionProxy
	ua := BotUserAgent

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	h, err := New(ctx, next, cfg, "wrangler")
	if err != nil {
		t.Fatal(err)
	}
	w, ok := h.(*Wrangler)
	if !ok {
		t.Error("unable to assert handler as type Wrangler")
	}
	w.log = logger.NewFromWriter(config.LogLevelDebug, &testLogOut)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", ua)
	h.ServeHTTP(recorder, req)

	got := recorder.Body.String()
	if got != want {
		t.Error("the BotProxy did not forward the response to the backend server")
	}
}

// TestWranglerProxyActionNoInit tests that the plugin yields blocked responses when a request should be proxied but the proxy wasnt initialized properly
func TestWranglerProxyActionNoInit(t *testing.T) {
	type jsonBody struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	var blockedBody jsonBody
	cfg := CreateConfig()
	cfg.BotAction = config.BotActionProxy
	ua := BotUserAgent

	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})

	h, err := New(ctx, next, cfg, "wrangler")
	if err != nil {
		t.Fatal(err)
	}
	w, ok := h.(*Wrangler)
	if !ok {
		t.Error("unable to assert handler as type Wrangler")
	}
	w.log = logger.NewFromWriter(config.LogLevelDebug, &testLogOut)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", ua)
	h.ServeHTTP(recorder, req)

	res := recorder.Result()
	resBody, _ := io.ReadAll(res.Body)
	err = json.Unmarshal(resBody, &blockedBody)
	if err != nil {
		t.Fatal(err)
	}
	want := res.StatusCode == http.StatusForbidden && res.Header.Get("Content-Type") == "application/json" && blockedBody.Error == "Forbidden" && blockedBody.Message == "Your user agent is associated with a large language model (LLM) and is blocked from accessing this resource"
	if !want {
		t.Errorf("request from bot that should've been proxied and failed did not return a blocked fallback response")
	}
}

func TestWranglerConcurrentRequests(t *testing.T) {
	// Create a test server
	ctx := context.Background()
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})
	cfg := CreateConfig()
	cfg.BotAction = config.BotActionBlock
	cfg.CacheSize = 1
	h, err := New(ctx, next, cfg, "wrangler")
	if err != nil {
		t.Fatal(err)
	}

	w, ok := h.(*Wrangler)
	if !ok {
		t.Error("unable to assert handler as type Wrangler")
	}
	w.log = logger.NewFromWriter(config.LogLevelDebug, &testLogOut)

	server := httptest.NewServer(h)
	defer server.Close()

	var wg sync.WaitGroup
	requests := 50 // Number of concurrent requests
	responses := make(chan *http.Response, requests)

	// yaegi doesn't like a range over int loop here
	// https://github.com/traefik/yaegi/issues/1701
	for i := 0; i < requests; i++ { //nolint:intrange,modernize
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
			if i%2 == 0 {
				req.Header.Set("User-Agent", RealUserAgent)
			} else {
				req.Header.Set("User-Agent", BotUserAgent)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			responses <- resp
		}()
	}

	wg.Wait()
	close(responses)

	for resp := range responses {
		want := http.StatusOK
		uaType := "regular"
		if resp.Request.Header.Get("User-Agent") == BotUserAgent {
			want = http.StatusForbidden
			uaType = "bot"
		}
		if resp.StatusCode != want {
			t.Errorf("Expected status %d for %s user-agent, got %d", want, uaType, resp.StatusCode)
		}
	}
}

// TestWranglerNoArchiveEnabled tests that the X-Robots-Tag is appropriately set/unset based on configuration and user-agent
func TestWranglerNoArchive(t *testing.T) {
	w := getWrangler(t, "", false, false)
	type scenario struct {
		ua            string
		headerEnabled bool
		expectHeader  bool
	}
	scenarios := []scenario{
		{ua: RealUserAgent, headerEnabled: true, expectHeader: false},
		{ua: BotUserAgent, headerEnabled: true, expectHeader: true},
		{ua: RealUserAgent, headerEnabled: false, expectHeader: false},
		{ua: BotUserAgent, headerEnabled: false, expectHeader: false},
	}

	for _, s := range scenarios {
		scenarioName := fmt.Sprintf("SetHeader:%v,UA:%s", s.headerEnabled, s.ua)
		t.Run(scenarioName, func(t *testing.T) {
			w.setNoArchiveHeader = s.headerEnabled
			res := getWranglerResponse(t, w, "http://localhost/", s.ua)

			headerVal := res.Header.Get("X-Robots-Tag")
			got := headerVal != ""
			if s.expectHeader != got {
				t.Errorf("expected presence of 'X-Robots-Tag' header to be %v, but got %v", s.expectHeader, got)
			}
			if got && headerVal != "noarchive" {
				t.Errorf("expected value of 'X-Robots-Tag' header to be 'noarchive', but got '%s'", headerVal)
			}
		})
	}
}
