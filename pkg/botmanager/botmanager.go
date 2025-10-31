// Package botmanager provides the BotUAManager type which can be used for storing, refreshing, and checking a robots.txt index.
package botmanager

import (
	"strings"
	"time"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/parser"
)

type userAgentCache map[string]string

// BotUAManager acts as a management layer around checking the current bot index, querying the index source, and refreshing the cache.
type BotUAManager struct {
	cache               userAgentCache
	cacheUpdateInterval time.Duration
	sources             []parser.Source
	lastUpdate          time.Time
	botIndex            parser.RobotsIndex
	log                 *logger.Log
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
	var uAInfo parser.BotUserAgent
	var inList bool
	botName, isCached := b.cache[u]
	if isCached {
		uAInfo, inList = b.botIndex[botName]
	} else {
		// TODO
		botName = u
		uAInfo, inList = b.botIndex[botName]
		b.cache[u] = botName
	}
	return uAInfo, inList
}

// update fetches the latest robots.txt index from each configured source, merges them, stores it, and updates the timestamp.
func (b *BotUAManager) update() error {
	var err error
	b.botIndex, err = parser.GetIndexFromSources(b.sources)
	if err != nil {
		return err
	}
	b.lastUpdate = time.Now()
	return nil
}
