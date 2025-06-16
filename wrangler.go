// Package bot_wrangler_traefik_plugin a plugin for managing bot traffic with automatically updating robots.txt and remediation actions for violations.
package bot_wrangler_traefik_plugin //nolint:revive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"

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
	template             *template.Template
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
	t := template.New("tmp")
	var loadedT *template.Template
	if c.RobotsTXTFilePath == "" {
		loadedT, err = t.Parse(config.RobotsTxtDefault)
	} else {
		log.Info("New: Custom robots.txt template file '" + c.RobotsTXTFilePath + "' specified, parsing..")
		loadedT, err = t.ParseFiles(c.RobotsTXTFilePath)
	}
	if err != nil {
		log.Error("New: Unable to load robots.txt template. " + err.Error())
		return nil, err
	}
	uAMan, err := botmanager.New(c.RobotsSourceURL, c.CacheUpdateInterval, log)
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
		template:             loadedT,
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

	botUAIndex, err := w.botUAManager.GetBotIndex()
	// this condition is unexpected. If the index source is permanently bad, we would've failed initialization, and if temporarily bad,
	// we wouldn't update the index cache. Testing this would require unecessarily exposing the raw index, or using unsafe reflection
	if err != nil || len(botUAIndex) == 0 {
		w.log.Error("ServeHTTP: Unable to retrieve list of bot useragents. " + err.Error())
		w.next.ServeHTTP(rw, req)
		return
	}

	uA := req.Header.Get("User-Agent")
	// if they are checking robots.txt, give them our list
	rPath := req.URL.Path
	if rPath == "/robots.txt" {
		w.log.Debug("ServeHTTP: /robots.txt requested, rendering with live block list", "userAgent", uA)
		w.renderRobotsTxt(botUAIndex, rw)
		return
	}

	// if its a normal request, see if they're on the bad robots list
	uAInfo, uAInList := botUAIndex[uA]
	w.log.Debug("ServeHTTP: Got a request to evaluate", "userAgent", uA)
	if !uAInList {
		w.log.Debug("ServeHTTP: User agent not in block list, passing traffic", "userAgent", uA)
		w.next.ServeHTTP(rw, req)
		return
	}

	if w.botAction != config.BotActionPass {
		uALogMsg := fmt.Sprintf("ServeHTTP: User agent '%s' considered AI Robot.", uA)
		w.log.Info(uALogMsg, "userAgent", uA, "sourceIP", req.RemoteAddr, "requestedPath",
			rPath, "remediationAction", w.botAction, "operator", *uAInfo.Operator, "respectsRobotsTxt",
			*uAInfo.Respect, "function", *uAInfo.Function, "description", *uAInfo.Description,
		)
	}
	// handle outcome of the request for the bot.
	w.handleOutcome(rw, req)
}

// renderRobotsTxt renders and writes the current Robots Exclusion list into the request's response.
func (w *Wrangler) renderRobotsTxt(bIndex botmanager.BotUserAgentIndex, rw http.ResponseWriter) {
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
