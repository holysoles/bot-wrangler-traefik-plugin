// Package bot_wrangler_traefik_plugin a plugin for managing bot traffic with automatically updating robots.txt and remediation actions for violations.
package bot_wrangler_traefik_plugin //nolint:revive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/botmanager"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/config"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/proxy"
)

// Wrangler used to manage a instance of the plugin.
type Wrangler struct {
	next http.Handler
	name string

	enabled              bool
	botAction            string
	botBlockHTTPCode     int
	botBlockHTTPResponse string
	botUAManager         *botmanager.BotUAManager
	log                  *logger.Log
	proxy                *proxy.BotProxy
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *config.Config {
	return config.New()
}

// New creates a new plugin instance.
func New(_ context.Context, next http.Handler, c *config.Config, name string) (http.Handler, error) {
	log := logger.New(c.LogLevel)
	c.BotAction = strings.ToUpper(c.BotAction)

	err := c.ValidateConfig()
	if err != nil {
		log.Error("New: unable to load configuration properly. " + err.Error())
		return nil, err
	}

	uAMan, err := botmanager.New(c.RobotsSourceURL, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath)
	if err != nil {
		log.Error("New: Unable to initialize bot user agent list manager. " + err.Error())
		return nil, err
	}
	var bP *proxy.BotProxy
	if c.BotProxyURL != "" {
		bP = proxy.New(c.BotProxyURL)
	}

	enable, _ := strconv.ParseBool(c.Enabled)
	return &Wrangler{
		next: next,
		name: name,

		enabled:              enable,
		botAction:            c.BotAction,
		botUAManager:         uAMan,
		botBlockHTTPCode:     c.BotBlockHTTPCode,
		botBlockHTTPResponse: c.BotBlockHTTPResponse,
		log:                  log,
		proxy:                bP,
	}, nil
}

func (w *Wrangler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// make sure we should process the request.
	if !w.enabled {
		w.log.Debug("ServeHTTP: Plugin is not enabled.")
		w.next.ServeHTTP(rw, req)
		return
	}

	uA := req.Header.Get("User-Agent")
	// if they are checking robots.txt, give them our list
	rPath := req.URL.Path
	if rPath == "/robots.txt" {
		w.log.Debug("ServeHTTP: /robots.txt requested, rendering with live block list", "userAgent", uA)
		err := w.botUAManager.RenderRobotsTxt(rw)
		if err != nil {
			w.log.Error("ServeHTTP: Error rendering robots.txt template. " + err.Error())
		}
		return
	}

	// if its a normal request, see if they're on the bad robots list
	w.log.Debug("ServeHTTP: Got a request to evaluate", "userAgent", uA)
	botName, err := w.botUAManager.Search(uA)
	if err != nil {
		w.log.Error("ServeHTTP: Unable to search cache. " + err.Error())
		w.next.ServeHTTP(rw, req)
		return
	}
	if botName == "" {
		w.log.Debug("ServeHTTP: User agent did not match block list, passing traffic", "userAgent", uA)
		w.next.ServeHTTP(rw, req)
		return
	}
	w.log.Debug("ServeHTTP: Found bot name match of '"+botName+"'", "userAgent", uA)

	if w.botAction != config.BotActionPass {
		uALogMsg := fmt.Sprintf("ServeHTTP: User agent '%s' considered AI Robot.", uA)
		uAMetadata := w.botUAManager.GetInfo(botName).JSONMetadata
		w.log.Info(uALogMsg, "userAgent", uA, "sourceIP", req.RemoteAddr, "requestedPath",
			rPath, "remediationAction", w.botAction, "operator", uAMetadata.Operator, "respectsRobotsTxt",
			uAMetadata.Respect, "function", uAMetadata.Function, "description", uAMetadata.Description,
		)
	}
	// handle outcome of the request for the bot.
	w.handleOutcome(rw, req)
}

// handleOutcome applies the appropriate remediation actions to the request based on the config's BotAction.
func (w *Wrangler) handleOutcome(rw http.ResponseWriter, req *http.Request) {
	switch w.botAction {
	case config.BotActionLog:
		fallthrough
	case config.BotActionPass:
		w.handleOutcomePass(rw, req)
	case config.BotActionBlock:
		w.handleOutcomeBlock(rw, req)
	case config.BotActionProxy:
		w.handleOutcomeProxy(rw, req)
	}
}

// handleOutcomePass processes tasks if the bot request should be passed.
func (w *Wrangler) handleOutcomePass(rw http.ResponseWriter, req *http.Request) {
	w.next.ServeHTTP(rw, req)
}

// handleOutcomeBlock processes tasks if the bot request should be blocked.
func (w *Wrangler) handleOutcomeBlock(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(w.botBlockHTTPCode)
	if w.botBlockHTTPResponse != "" {
		statusText := http.StatusText(w.botBlockHTTPCode)
		response := map[string]string{
			"error":   statusText,
			"message": w.botBlockHTTPResponse,
		}
		err := json.NewEncoder(rw).Encode(response)
		if err != nil {
			w.log.Error("ServeHTTP: Error when rendering JSON for block response. Sending no content in reply. Error: " + err.Error())
			return
		}
	}
}

// handleOutcomeProxy processes tasks if the bot request should be proxied.
func (w *Wrangler) handleOutcomeProxy(rw http.ResponseWriter, req *http.Request) {
	w.log.Debug("ServeHTTP: Starting proxying request from bot")
	if w.proxy == nil {
		w.log.Error("ServeHTTP: cannot proxy request, proxy failed to initialize during setup. Falling back to BLOCK")
		w.handleOutcomeBlock(rw, req)
		return
	}
	w.proxy.ServeHTTP(rw, req)
	w.log.Debug("ServeHTTP: finished proxying request")
}
