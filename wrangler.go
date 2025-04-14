// Package bot_wrangler_traefik_plugin a plugin for managing bot traffic with automatically updating robots.txt and remediation actions for violations.
package bot_wrangler_traefik_plugin //nolint:revive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"strconv"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/botmanager"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/proxy"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/config"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
)

// Wrangler used to manage a instance of the plugin.
type Wrangler struct {
	next http.Handler
	name string

	enabled      bool
	botAction    string
	botUAManager *botmanager.BotUAManager
	log          *logger.Log
	template     *template.Template
	proxy        *proxy.BotProxy
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *config.Config {
	return config.New()
}

// New creates a new plugin instance.
func New(_ context.Context, next http.Handler, config *config.Config, name string) (http.Handler, error) {
	config.LogLevel = strings.ToUpper(config.LogLevel)
	log := logger.New(config.LogLevel)

	config.BotAction = strings.ToUpper(config.BotAction)

	err := config.ValidateConfig()
	if err != nil {
		log.Error("New: unable to load configuration properly. " + err.Error())
		return nil, err
	}
	loadedTemplate, err := template.ParseFiles(config.RobotsTXTFilePath)
	if err != nil {
		log.Error("New: Unable to load robots.txt template. " + err.Error())
		return nil, err
	}
	uAMan, err := botmanager.New(config.RobotsSourceURL, config.CacheUpdateInterval, log)
	if err != nil {
		log.Error("New: Unable to initialize bot user agent list manager. " + err.Error())
		return nil, err
	}
	var bP *proxy.BotProxy
	if config.BotProxyURL != "" {
		bP = proxy.New(config.BotProxyURL)
	}

	enable, _ := strconv.ParseBool(config.Enabled)
	return &Wrangler{
		next: next,
		name: name,

		enabled:      enable,
		botAction:    config.BotAction,
		botUAManager: uAMan,
		log:          log,
		template:     loadedTemplate,
		proxy:        bP,
	}, nil
}

func (w *Wrangler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// make sure we should process the request.
	if !w.enabled {
		w.log.Debug("ServeHTTP: Plugin is not enabled.")
		w.next.ServeHTTP(rw, req)
		return
	}

	botUAIndex, err := w.botUAManager.GetBotIndex()
	// this condition is unexpected. If the index source is permanently bad, we would've failed initialization, and if temporarily bad,
	// we wouldn't update the index cache. Would use assert if we could have an external pkg
	if err != nil || len(botUAIndex) == 0 {
		w.log.Error("ServeHTTP: Unable to retrieve list of bot useragents. " + err.Error())
		w.next.ServeHTTP(rw, req)
		return
	}

	// if they are checking robots.txt, give them our list
	rPath := req.URL.Path
	if rPath == "/robots.txt" {
		w.renderRobotsTxt(botUAIndex, rw)
		return
	}

	// if its a normal request, see if they're on the bad robots list
	uA := req.Header.Get("User-Agent")
	uAInfo, uAInList := botUAIndex[uA]
	w.log.Debug("ServeHTTP: Got a request from user agent: '" + uA + "'")
	if !uAInList {
		w.log.Debug("ServeHTTP: User agent not in block list, passing traffic")
		w.next.ServeHTTP(rw, req)
		return
	}

	// handle outcome of the request for the bot.
	uALogStr := fmt.Sprintf("ServeHTTP: User agent '%s' considered AI Robot. SourceIP: '%s', Operator: '%s', RespectsRobotsTxt: '%s', Function: '%s', Description: '%s', RequestedPath: '%s'", uA, req.RemoteAddr, *uAInfo.Operator, *uAInfo.Respect, *uAInfo.Function, *uAInfo.Description, rPath)
	w.handleOutcome(rw, req, uALogStr)
}

// renderRobotsTxt renders and writes the current Robots Exclusion list into the request's response
func (w *Wrangler) renderRobotsTxt(bIndex botmanager.BotUserAgentIndex, rw http.ResponseWriter) {
	w.log.Debug("ServeHTTP: /robots.txt requested, rendering with live block list")
	uAList := make([]string, len(bIndex))
	i := 0
	for k := range bIndex {
		uAList[i] = k
		i++
	}
	err := w.template.Execute(rw, map[string][]string{
		"UserAgentList": uAList,
	})
	if err != nil {
		w.log.Error("ServeHTTP: Error rendering robots.txt template. " + err.Error())
	}
}

// handleOutcome applies the appropriate remediation actions to the request based on the config's BotAction
func (w *Wrangler) handleOutcome(rw http.ResponseWriter, req *http.Request, uALogStr string) {
	w.log.Debug("taking " + w.botAction + " remediation action")
	switch w.botAction {
	case config.BotActionPass:
		w.handleOutcomePass(rw, req)
	case config.BotActionLog:
		w.log.Info(uALogStr)
		w.handleOutcomePass(rw, req)
	case config.BotActionBlock:
		w.log.Info(uALogStr)
		w.handleOutcomeBlock(rw, req)
	case config.BotActionProxy:
		w.log.Info(uALogStr)
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
	rw.WriteHeader(http.StatusForbidden)
	response := map[string]string{
		"error":   "Forbidden",
		"message": "Your user agent is associated with a large language model (LLM) and is blocked from accessing this resource due to scraping activities.",
	}
	err := json.NewEncoder(rw).Encode(response)
	if err != nil {
		w.log.Error("ServeHTTP: Error when rendering JSON for block response. Sending no content in reply. Error: " + err.Error())
		return
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