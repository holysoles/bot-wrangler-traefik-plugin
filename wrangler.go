// Package plugbot_wrangler_traefik_plugin a plugin for managing bot traffic
package bot_wrangler_traefik_plugin

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/useragent"
)

// Config the plugin configuration.
type Config struct {
	LogLevel           string    `json:"logLevel,omitempty"`
	BotAction          BotAction `json:"botAction,omitempty"`
	RobotsTXTFilePath  string    `json:"robotsTxtFilePath,omitempty"`
	UserAgentSourceURL string    `json:"userAgentSourceUrl,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		BotAction:          ActionLog,
		LogLevel:           "DEBUG", //TODO set INFO
		RobotsTXTFilePath:  "robots.txt",
		UserAgentSourceURL: "https://raw.githubusercontent.com/ai-robots-txt/ai.robots.txt/refs/heads/main/robots.json",

	}
}

type BotAction int

const (
	ActionPass BotAction = iota
	ActionLog
	ActionBlock
)

var botActionName = map[BotAction]string{
	ActionPass:  "pass",
	ActionLog:   "log",
	ActionBlock: "block",
}

func (b BotAction) String() string {
	return botActionName[b]
}

type Wrangler struct {
	next               http.Handler
	name               string

	botAction          BotAction
	log                *logger.Log
	template           *template.Template
	userAgentSourceURL string
}

// New creates a new plugin instance
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	config.LogLevel = strings.ToUpper(config.LogLevel)
	log := logger.New(config.LogLevel)

	// TODO validate config

	loadedTemplate, err := template.ParseFiles(config.RobotsTXTFilePath)
	if err != nil {
		log.Error("Unable to load robots.txt template: " + err.Error())
	}
	return &Wrangler{
		next: next,
		name: name,

		botAction: config.BotAction,
		log:       log,
		template:  loadedTemplate,
		userAgentSourceURL: config.UserAgentSourceURL,
	}, nil
}

func (w *Wrangler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log := w.log
	// get the current list of bad robots
	badUAMap, err := useragent.GetBanned(w.userAgentSourceURL, log)
	if err != nil {
		log.Error("Unable to retrieve list of user agents: " + err.Error())
		// fallback to just letting the request pass
		w.next.ServeHTTP(rw, req)
		return
	}

	// if they are checking robots.txt, give them our list
	path := req.URL.Path
	log.Debug("requested Path:" + path)
	if path == "/robots.txt" {
		log.Debug("/robots.txt requested, rendering robots.txt with current ban list")
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
	log.Debug("Got a request from user agent: '" + uA + "'")
	if !uAInList {
		log.Debug("User agent not in ban list, passing traffic")
		w.next.ServeHTTP(rw, req)
		return
	}

	// handle outcome of the request for the bot
	uALogStr := fmt.Sprintf("User agent '%s' considered AI Robot. Operator: %s, Respects Robots.txt?: %s, Function: %s, Description: %s", uA, uAInfo["operator"], uAInfo["respect"], uAInfo["function"], uAInfo["description"])
	switch w.botAction {
	case ActionLog:
		log.Info(uALogStr)
		fallthrough
	case ActionPass:
		w.next.ServeHTTP(rw, req)
	case ActionBlock:
		log.Info(uALogStr)
		// TODO provide any body/content indicating the ban?
		rw.WriteHeader(http.StatusForbidden)
	}
}