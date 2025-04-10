// Package plugbot_wrangler_traefik_plugin a plugin for managing bot traffic
package bot_wrangler_traefik_plugin

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"encoding/json"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/botmanager"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/config"
)

type Wrangler struct {
	next               http.Handler
	name               string

	botAction          string
	botUAManager       botmanager.BotUAManager
	log                *logger.Log
	template           *template.Template
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *config.Config {
	return config.New()
}

// New creates a new plugin instance
func New(ctx context.Context, next http.Handler, config *config.Config, name string) (http.Handler, error) {
	config.LogLevel = strings.ToUpper(config.LogLevel)
	log := logger.New(config.LogLevel)

	config.BotAction = strings.ToUpper(config.BotAction)

	err := config.ValidateConfig()
	if err != nil {
		log.Error("New: unable to load configuration properly - " + err.Error())
		return nil, err
	}

	loadedTemplate, err := template.ParseFiles(config.RobotsTXTFilePath)
	if err != nil {
		log.Error("New: Unable to load robots.txt template - " + err.Error())
	}

	uAMan, err := botmanager.New(config.RobotsSourceURL, config.CacheUpdateInterval)
	if err != nil {
		log.Error("New: Unable to initialize bot user agent list manager - " + err.Error())
	}

	return &Wrangler{
		next: next,
		name: name,

		botAction: config.BotAction,
		botUAManager: uAMan,
		log:       log,
		template:  loadedTemplate,
	}, nil
}

func (w *Wrangler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log := w.log
	// get the current list of bad robots
	badUAMap, err := w.botUAManager.GetBotMap(log)
	if err != nil || len(badUAMap) == 0 {
		// fallback to just letting the request pass
		eStr := "ServeHTTP: Unable to retrieve list of bot useragents"
		if err != nil {
			eStr += " - " + err.Error()
		}
		log.Error(eStr)
		w.next.ServeHTTP(rw, req)
		return
	}

	// if they are checking robots.txt, give them our list
	rPath := req.URL.Path
	if rPath == "/robots.txt" {
		log.Debug("ServeHTTP: /robots.txt requested, rendering with live block list")
		uAList := make([]string, len(badUAMap))
		i := 0
		for k := range badUAMap {
			uAList[i] = k
			i++
		}
		w.template.Execute(rw, map[string][]string{
			"UserAgentList": uAList,
		})
		return
	}

	// if its a normal request, see if they're on the bad robots list
	uA := req.Header.Get("User-Agent")
	uAInfo, uAInList := badUAMap[uA]
	log.Debug("ServeHTTP: Got a request from user agent: '" + uA + "'")
	if !uAInList {
		log.Debug("ServeHTTP: User agent not in ban list, passing traffic")
		w.next.ServeHTTP(rw, req)
		return
	}

	// handle outcome of the request for the bot
	uALogStr := fmt.Sprintf("ServeHTTP: User agent '%s' considered AI Robot. Operator: %s, RespectsRobotsTxt: %s, Function: %s, Description: %s, RequestedPath: %s", uA, uAInfo["operator"], uAInfo["respect"], uAInfo["function"], uAInfo["description"], rPath)
	switch w.botAction {
	case config.BotActionLog:
		log.Info(uALogStr)
		fallthrough
	case config.BotActionPass:
		w.next.ServeHTTP(rw, req)
	case config.BotActionBlock:
		log.Info(uALogStr)
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusForbidden)
		response := map[string]string{
			"error": "Forbidden",
			"message": "Your user agent is associated with a large language model (LLM) and is banned from accessing this resource due to scraping activities.",
		}
		json.NewEncoder(rw).Encode(response)
	}
}