package botmanager

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
)

type BlockUserAgentMap map[string]map[string]string

type BotUAManager struct {
	cacheUpdateInterval time.Duration
	url                 string
	lastUpdate          time.Time
	blockListMap        BlockUserAgentMap
}

func New(u string, i string) (BotUAManager, error) {
	// we validated the time duration earlier, so ignore any error now
	iDur, _ := time.ParseDuration(i)

	uAMan := BotUAManager{
		url: u,
		cacheUpdateInterval: iDur,
	}
	err := uAMan.update()
	return uAMan, err
}


func (b *BotUAManager) update() error {
	var bannedUA BlockUserAgentMap

	req, err := http.NewRequest(http.MethodGet, b.url, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(resBody, &bannedUA)
	b.blockListMap = bannedUA
	b.lastUpdate = time.Now()
	return err
}

func (b *BotUAManager) GetBotMap(log *logger.Log) (BlockUserAgentMap, error) {
	var err error

	log.Debug("GetBotMap: blocklist last updated at " + b.lastUpdate.Format(time.RFC1123))

	nextUpdate := b.lastUpdate.Add(b.cacheUpdateInterval)
	if time.Now().Compare(nextUpdate) >= 0 {
		log.Info("GetBotMap: cache expired, updating")
		err = b.update()
	} else {
		log.Debug("GetBotMap: cache has not expired. Next update due " + nextUpdate.Format(time.RFC1123))
	}

	return b.blockListMap, err
}