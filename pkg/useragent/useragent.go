package useragent

import (
	"encoding/json"
	"strconv"
	"io"
	"net/http"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
)

type BannedUserAgents map[string]map[string]string

func GetBanned(listUrl string, log *logger.Log) (BannedUserAgents, error) {
	// TODO cache
	var bannedUA BannedUserAgents

	req, err := http.NewRequest(http.MethodGet, listUrl, nil)
	if err != nil {
		log.Error("GetBanned - could not create request to retrieve user agent list: " + err.Error())
		return bannedUA, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("GetBanned - error retrieving user agent list: " + err.Error())
		return bannedUA, err
	}

	log.Debug("GetBanned - retrieving list yielded status code: " + strconv.Itoa(res.StatusCode))

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("GetBanned - could not read user agent list response body: " + err.Error())
		return bannedUA, err
	}

	err = json.Unmarshal(resBody, &bannedUA)
	return bannedUA, err
}
