// Package botmanager provides the BotUAManager type which can be used for storing, refreshing, and checking a robots.txt index.
package botmanager

import (
	"errors"
	"strings"
	"time"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/ahocorasick"
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
}

func newUserAgentCache(s int) *userAgentCache {
	return &userAgentCache{
		data:  make(map[string]string, s),
		keys:  make([]*string, s),
		limit: s,
	}
}

func (c *userAgentCache) get(k string) (string, bool) {
	v, ok := c.data[k]
	return v, ok
}

func (c *userAgentCache) set(k string, v string) {
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
}

// New initializes a BotUAManager instance.
func New(s string, i string, l *logger.Log, cS int, sF bool) (*BotUAManager, error) {
	// we validated the time duration earlier, so ignore any error now
	iDur, _ := time.ParseDuration(i)
	uL := strings.Split(s, ",")
	sources := make([]parser.Source, len(uL))
	for i, u := range uL {
		sources[i] = parser.Source{URL: u}
	}
	bI := make(parser.RobotsIndex)

	uAMan := BotUAManager{
		botIndex:            bI,
		cache:               newUserAgentCache(cS),
		cacheUpdateInterval: iDur,
		log:                 l,
		sources:             sources,
		searchFast:          sF,
	}
	err := uAMan.update()
	return &uAMan, err
}

// GetBotIndex is an exported function to retrieve the current, merged robots.txt index. It will refreshed the cached copy if necessary.
func (b *BotUAManager) GetBotIndex() (parser.RobotsIndex, error) {
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

// Search checks if the provided user-agent has a (partial) match in the botIndex.
func (b *BotUAManager) Search(u string) (string, error) {
	var botName string
	if b.cache == nil {
		return botName, errBotManagerNoInit
	}
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
