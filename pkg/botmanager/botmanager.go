// Package botmanager provides the BotUAManager type which can be used for storing, refreshing, and checking a robots.txt index.
package botmanager

import (
	"errors"
	"io"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/ahocorasick"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/config"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/parser"
)

var (
	errBotManagerNoInit = errors.New("attempted to search uninitialized BotManager. Ensure it is created with the New() constructor")
)

type userAgentCache struct {
	cursor int
	data   map[string]string
	keys   []*string
	limit  int
	lock   sync.RWMutex
}

func newUserAgentCache(s int) *userAgentCache {
	return &userAgentCache{
		data:  make(map[string]string, s),
		keys:  make([]*string, s),
		limit: s,
	}
}

func (c *userAgentCache) get(k string) (string, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	v, ok := c.data[k]
	return v, ok
}

func (c *userAgentCache) set(k string, v string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	// rollover
	if c.cursor >= c.limit {
		c.cursor = 0
	}

	// free up a slot if we need it
	p := c.keys[c.cursor]
	if p != nil {
		delete(c.data, *p)
	}

	c.data[k] = v
	c.keys[c.cursor] = &k
	c.cursor++
}

// BotUAManager acts as a management layer around checking the current bot index, querying the index source, and refreshing the cache.
type BotUAManager struct {
	ahoCorasick         *ahocorasick.Node
	botIndex            parser.RobotsIndex
	cache               *userAgentCache
	cacheUpdateInterval time.Duration
	lastUpdate          time.Time
	log                 *logger.Log
	searchFast          bool
	sources             []parser.Source
	template            *template.Template
}

func loadTemplate(disallowAll bool, templatePath string, log *logger.Log) (*template.Template, error) {
	t := template.New("tmp")
	var loadedT *template.Template
	var err error
	switch {
	case disallowAll:
		log.Info("New: robotsTxtDisallowAll specified, robots.txt will disallow all user-agents")
		loadedT, err = t.Parse(config.RobotsTxtDisallowAll)
	case templatePath == "":
		loadedT, err = t.Parse(config.RobotsTxtDefault)
	default:
		log.Info("New: Custom robots.txt template file '" + templatePath + "' specified, parsing..")
		loadedT, err = t.ParseFiles(templatePath)
	}
	return loadedT, err
}

// New initializes a BotUAManager instance.
func New(source string, i string, l *logger.Log, cS int, sF bool, disallowAll bool, templatePath string) (*BotUAManager, error) {
	// we validated the time duration earlier, so ignore any error now
	iDur, _ := time.ParseDuration(i)
	uL := strings.Split(source, ",")
	sources := make([]parser.Source, len(uL))
	for i, u := range uL {
		sources[i] = parser.Source{URL: u}
	}
	t, err := loadTemplate(disallowAll, templatePath, l)
	if err != nil {
		return nil, err
	}
	bI := make(parser.RobotsIndex)

	uAMan := BotUAManager{
		botIndex:            bI,
		cache:               newUserAgentCache(cS),
		cacheUpdateInterval: iDur,
		log:                 l,
		sources:             sources,
		searchFast:          sF,
		template:            t,
	}
	err = uAMan.update()
	return &uAMan, err
}

// RenderRobotsTxt renders and writes the current Robots Exclusion list into the request's response.
func (b *BotUAManager) RenderRobotsTxt(w io.Writer) error {
	var err error
	// TODO err check
	// TODO
	bIndex, _ := b.getBotIndex()
	uAList := make([]string, len(bIndex))
	i := 0
	for k := range bIndex {
		uAList[i] = k
		i++
	}
	err = b.template.Execute(w, map[string][]string{
		"UserAgentList": uAList,
	})
	return err
}

// GetInfo retrieves the metadata associated with a specific botName from the Bot Index.
func (b *BotUAManager) GetInfo(botName string) parser.BotUserAgent {
	// TODO i think this change is not thread safe
	return b.botIndex[botName]
}

// Search checks if the provided user-agent has a (partial) match in the botIndex.
func (b *BotUAManager) Search(u string) (string, error) {
	var botName string
	if b.cache == nil {
		return botName, errBotManagerNoInit
	}

	// TODO cleanup
	_, _ = b.getBotIndex()

	botName, hit := b.cache.get(u)
	if hit {
		b.log.Debug("Search: cache hit, got '"+botName+"'", "userAgent", u)
	} else {
		b.log.Debug("Search: cache miss", "userAgent", u)
		if b.searchFast {
			botName = b.fastSearch(u)
		} else {
			botName = b.slowSearch(u)
		}
		b.cache.set(u, botName)
	}
	return botName, nil
}

// getBotIndex retrieves the current, merged robots.txt index. It will refreshed the cached copy if necessary.
func (b *BotUAManager) getBotIndex() (parser.RobotsIndex, error) {
	var err error

	b.log.Debug("GetBotIndex: sources last updated at " + b.lastUpdate.Format(time.RFC1123))

	nextUpdate := b.lastUpdate.Add(b.cacheUpdateInterval)
	if time.Now().Compare(nextUpdate) >= 0 {
		b.log.Info("GetBotIndex: cache expired, updating")
		err = b.update()
	} else {
		b.log.Debug("GetBotIndex: cache has not expired. Next update due " + nextUpdate.Format(time.RFC1123))
	}

	return b.botIndex, err
}

// slowSearch runs a substring search in a simple for loop.
func (b *BotUAManager) slowSearch(u string) string {
	var match bool
	var nameMatch string
	for name := range b.botIndex {
		match = strings.Contains(u, name)
		if match {
			nameMatch = name
			break
		}
	}
	return nameMatch
}

// fastSearch runs a match search using a Aho-Corasick automaton.
func (b *BotUAManager) fastSearch(u string) string {
	s, _ := b.ahoCorasick.Search(u)
	return s
}

// update fetches the latest robots.txt index from each configured source, merges them, stores it, and updates the timestamp.
func (b *BotUAManager) update() error {
	newI := parser.RobotsIndex{}
	for _, s := range b.sources {
		n, err := s.GetIndex()
		if err != nil {
			return err
		}
		// could use golang.org/x/exp/maps, but this saves us a dep
		//nolint:modernize
		for k, v := range n {
			newI[k] = v
		}
	}
	b.botIndex = newI
	if b.searchFast {
		b.ahoCorasick = ahocorasick.NewFromIndex(b.botIndex)
	}
	b.cache = newUserAgentCache(b.cache.limit)
	b.lastUpdate = time.Now()
	return nil
}
