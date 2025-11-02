// Package botmanager provides the BotUAManager type which can be used for storing, refreshing, and checking a robots.txt index.
package botmanager

import (
	"regexp"
	"strings"
	"time"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/aho_corasick"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/parser"
)

// TODO max size
type userAgentCache map[string]string

// BotUAManager acts as a management layer around checking the current bot index, querying the index source, and refreshing the cache.
type BotUAManager struct {
	ahoCorasick         *aho_corasick.Node
	botIndex            parser.RobotsIndex
	cache               userAgentCache
	cacheUpdateInterval time.Duration
	lastUpdate          time.Time
	log                 *logger.Log
	searchFast          bool
	sources             []parser.Source
}

// New initializes a BotUAManager instance.
func New(s string, i string, l *logger.Log) (*BotUAManager, error) {
	// we validated the time duration earlier, so ignore any error now
	iDur, _ := time.ParseDuration(i)
	uL := strings.Split(s, ",")
	sources := make([]parser.Source, len(uL))
	for i, u := range uL {
		sources[i] = parser.Source{URL: u}
	}
	bI := make(parser.RobotsIndex)

	uAMan := BotUAManager{
		sources:             sources,
		cache:               make(userAgentCache),
		cacheUpdateInterval: iDur,
		log:                 l,
		botIndex:            bI,
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

// TODO def
func (b *BotUAManager) Search(u string) (parser.BotUserAgent, bool) {
	botName, found := b.cache[u]
	if !found {
		if b.searchFast {
			botName, found = b.fastSearch(u)
			b.cache[u] = botName
		} else {
			botName, found = b.slowSearch(u)
			b.cache[u] = botName
		}
	}
	return b.botIndex[botName], found
}

func (b *BotUAManager) slowSearch(u string) (string, bool) {
	var match bool
	var nameMatch string
	for name := range b.botIndex {
		match, _ = regexp.MatchString(name, u)
		if match {
			nameMatch = name
			break
		}
	}
	return nameMatch, match
}

func (b *BotUAManager) fastSearch(u string) (string, bool) {
	// TODO return err if not configured
	return b.ahoCorasick.Search(u)
}

// update fetches the latest robots.txt index from each configured source, merges them, stores it, and updates the timestamp.
func (b *BotUAManager) update() error {
	var err error
	b.botIndex, err = parser.GetIndexFromSources(b.sources)
	if err != nil {
		return err
	}
	b.ahoCorasick = aho_corasick.NewFromIndex(b.botIndex)
	b.lastUpdate = time.Now()
	return nil
}
