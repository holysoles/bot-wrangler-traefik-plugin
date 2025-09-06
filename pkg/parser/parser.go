// Package parser provides parsing functionality for bot sources.
package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"regexp"
	"strings"
)

const (
	regexUserAgent    = "(?i:user-agent:\\s)(.*)"
	regexAllowRule    = "(?i:allow:\\s)(.*)"
	regexDisallowRule = "(?i:disallow:\\s)(.*)"
)

// BotMetadata holds metadata about a bot's user agent. Populated from a JSON source.
type BotMetadata struct {
	Operator    *string `json:"operator"`
	Respect     *string `json:"respect"`
	Function    *string `json:"function"`
	Frequency   *string `json:"frequency"`
	Description *string `json:"description"`
}

// BotUserAgent holds the fields associated with a bot's user agent.
type BotUserAgent struct {
	DisallowPath []string
	AllowPath    []string
	JSONMetadata BotMetadata
}
type batchEntry struct {
	ua       []string
	allow    []string
	disallow []string
}

// RobotsIndex is a hash of bot user agents and associated data with each.
type RobotsIndex map[string]BotUserAgent

func (r *RobotsIndex) addTxtRule(e batchEntry) {
	for _, u := range e.ua {
		(*r)[u] = BotUserAgent{AllowPath: e.allow, DisallowPath: e.disallow}
	}
}

func getSourceContent(u string) (*http.Response, error) {
	var res *http.Response

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return res, err
	}

	return http.DefaultClient.Do(req)
}

func getSourceContentType(r *http.Response) (*bufio.Reader, string, error) {
	cT := "txt"
	bR := bufio.NewReader(r.Body)

	firstC, err := bR.Peek(1)
	if err != nil {
		return bR, cT, err
	}
	sniff := r.Header.Get("X-Content-Type-Options") != "nosniff"

	u := r.Request.URL.String()
	if r.Header.Get("Content-Type") == mime.TypeByExtension(".json") || strings.HasSuffix(u, ".json") || (sniff && string(firstC) == "{") {
		cT = "json"
	}
	return bR, cT, err
}

// RobotsSourceUpdate manages retrieving robots source from a URL, and parses it accordingly to a RobotsIndex.
func RobotsSourceUpdate(u string) (RobotsIndex, error) {
	var rIndex RobotsIndex
	r, err := getSourceContent(u)
	if err != nil {
		return rIndex, err
	}
	defer func() { err = r.Body.Close() }()
	bR, cT, err := getSourceContentType(r)
	if err != nil {
		return rIndex, err
	}

	switch cT {
	case "json":
		rIndex, err = robotsJSONParse(bR)
	case "txt":
		rIndex = robotsTxtParse(bR)
	}

	return rIndex, err
}

func robotsTxtParse(r *bufio.Reader) RobotsIndex {
	s := bufio.NewScanner(r)
	var rIndex RobotsIndex

	// rfc9309. user-agent statement(s) precede any amount of rules, before starting another entry
	var e batchEntry
	uaStart := false
	rulesStart := false
	for s.Scan() {
		l := s.Text()
		reUa := regexp.MustCompile(regexUserAgent)
		reAllow := regexp.MustCompile(regexAllowRule)
		reDisallow := regexp.MustCompile(regexDisallowRule)
		switch {
		case reUa.MatchString(l):
			if rulesStart {
				rulesStart = false
				if uaStart {
					rIndex.addTxtRule(e)
					e = batchEntry{}
				}
			}
			uaStart = true
			m := reUa.FindAllString(l, -1)
			e.ua = append(e.ua, m[0])
		case reAllow.MatchString(l):
			if uaStart {
				uaStart = false
			}
			rulesStart = true
			m := reDisallow.FindAllString(l, -1)
			e.allow = append(e.allow, m[0])
		case reDisallow.MatchString(l):
			if uaStart {
				uaStart = false
			}
			rulesStart = true
			m := reDisallow.FindAllString(l, -1)
			e.disallow = append(e.disallow, m[0])
		}
	}

	return rIndex
}

type jsonBotUserAgentIndex map[string]BotMetadata

// Validate checks that the json bot source has all required values.
func (m *jsonBotUserAgentIndex) Validate() error {
	for _, bInfo := range *m {
		r := reflect.ValueOf(bInfo)
		// it'd be better to range over r.NumField(), but yaegi is panicking when loading the plugin when we use that
		for i, fN := range []string{"Operator", "Respect", "Function", "Frequency", "Description"} {
			if r.Field(i).IsNil() {
				return fmt.Errorf("missing required field '%s' on retrieved bot index entry", fN)
			}
		}
	}
	return nil
}

func robotsJSONParse(r *bufio.Reader) (RobotsIndex, error) {
	rIndex := make(RobotsIndex)

	var c []byte
	c, err := io.ReadAll(r)
	if err != nil {
		return rIndex, err
	}

	var jsonIndex jsonBotUserAgentIndex

	err = json.Unmarshal(c, &jsonIndex)
	if err != nil {
		return rIndex, err
	}
	err = jsonIndex.Validate()
	if err != nil {
		return rIndex, err
	}

	for u, m := range jsonIndex {
		e := BotUserAgent{JSONMetadata: m}
		rIndex[u] = e
	}

	return rIndex, err
}
