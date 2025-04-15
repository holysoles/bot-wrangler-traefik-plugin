// Package botmanager provides the BotUAManager type which can be used for storing, refreshing, and checking a robots.txt index.
package botmanager

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"reflect"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
)

// BotUserAgent is the expected info to receive per bot from the index source. These are pointers so that we can validate if the source actually returned the expected fields.
type BotUserAgent struct {
	Operator    *string `json:"operator"`
	Respect     *string `json:"respect"`
	Function    *string `json:"function"`
	Frequency   *string `json:"frequency"`
	Description *string `json:"description"`
}

// BotUserAgentIndex is the hashmap/dict to receive from the index source of bot_name: {info}.
type BotUserAgentIndex map[string]BotUserAgent

// BotUAManager acts as a management layer around checking the current bot index, querying the index source, and refreshing the cache.
type BotUAManager struct {
	cacheUpdateInterval time.Duration
	url                 string
	lastUpdate          time.Time
	botIndex            BotUserAgentIndex
	log                 *logger.Log
}

// Validate checks that the generated BotUserAgentIndex has all required values.
func (m *BotUserAgentIndex) Validate() error {
	for _, bInfo := range *m {
		r := reflect.ValueOf(bInfo)
		// it'd be better to range over r.NumField(), but yaegi is panicking when loading the plugin when we use that
		for i, fN:= range []string{"Operator", "Respect", "Function", "Frequency", "Description"} {
			if r.Field(i).IsNil() {
				return fmt.Errorf("missing required field '%s' on retrieved bot index entry", fN)
			}
		}
	}
	return nil
}

// New initializes a BotUAManager instance.
func New(u string, i string, l *logger.Log) (*BotUAManager, error) {
	// we validated the time duration earlier, so ignore any error now
	iDur, _ := time.ParseDuration(i)

	uAMan := BotUAManager{
		url:                 u,
		cacheUpdateInterval: iDur,
		log:                 l,
	}
	err := uAMan.update()
	return &uAMan, err
}

// update fetches the latest robots.txt index from the configured source, stores it, and updates the timestamp.
func (b *BotUAManager) update() error {
	var blockedUA BotUserAgentIndex

	req, err := http.NewRequest(http.MethodGet, b.url, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	resBody, err := io.ReadAll(res.Body)
	// this probably needs a refactor to unit test
	if err != nil {
		return err
	}

	err = json.Unmarshal(resBody, &blockedUA)
	if err != nil {
		return err
	}
	err = blockedUA.Validate()
	if err != nil {
		return err
	}
	b.botIndex = blockedUA
	b.lastUpdate = time.Now()
	return nil
}

// GetBotIndex is an exported function to retrieve the current robots.txt index. It will refreshed the cached copy if necessary.
func (b *BotUAManager) GetBotIndex() (BotUserAgentIndex, error) {
	var err error

	b.log.Debug("GetBotIndex: blocklist last updated at " + b.lastUpdate.Format(time.RFC1123))

	nextUpdate := b.lastUpdate.Add(b.cacheUpdateInterval)
	if time.Now().Compare(nextUpdate) >= 0 {
		b.log.Info("GetBotIndex: cache expired, updating")
		err = b.update()
	} else {
		b.log.Debug("GetBotIndex: cache has not expired. Next update due " + nextUpdate.Format(time.RFC1123))
	}

	return b.botIndex, err
}
