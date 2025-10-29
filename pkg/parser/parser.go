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
	regexUserAgent    = `(?i:^user-agent\s?:\s?)(.*)`
	regexAllowRule    = `(?i:^allow\s?:\s?)(.*)`
	regexDisallowRule = `(?i:^disallow\s?:\s?)(.*)`
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

func getSourceContentType(r *http.Response) (*bufio.Reader, string) {
	cT := "txt"
	bR := bufio.NewReader(r.Body)

	sniff := r.Header.Get("X-Content-Type-Options") != "nosniff"
	u := r.Request.URL.String()
	if r.Header.Get("Content-Type") == mime.TypeByExtension(".json") || strings.HasSuffix(u, ".json") {
		cT = "json"
	} else if sniff {
		firstC, err := bR.Peek(1)
		if err == nil {
			if string(firstC) == "{" {
				cT = "json"
			}
		}
	}

	return bR, cT
}

func getIndexFromContent(r *http.Response) (RobotsIndex, error) {
	var rIndex RobotsIndex
	var err error
	bR, cT := getSourceContentType(r)

	switch cT {
	case "json":
		rIndex, err = robotsJSONParse(bR)
	case "txt":
		rIndex = robotsTxtParse(bR)
	}

	return rIndex, err
}

// GetIndexFromSources manages retrieving robots source from slice of URLs, and parses it accordingly to a merged RobotsIndex.
func GetIndexFromSources(s []string) (RobotsIndex, error) {
	i := make(RobotsIndex)
	for _, u := range s {
		r, err := getSourceContent(u)
		if err != nil {
			return i, err
		}
		defer func() { err = r.Body.Close() }()
		if r.StatusCode != http.StatusOK {
			return i, fmt.Errorf("error retrieving source data from '%s'. Status: %s", u, r.Status)
		}
		n, err := getIndexFromContent(r)
		if err != nil {
			return i, err
		}

		// could use golang.org/x/exp/maps, but this saves us a dep
		//nolint:modernize
		for k, v := range n {
			i[k] = v
		}
	}
	return i, nil
}

func robotsTxtParse(r *bufio.Reader) RobotsIndex {
	s := bufio.NewScanner(r)
	rIndex := make(RobotsIndex)

	// rfc9309. user-agent statement(s) precede any amount of rules, before starting another entry
	var e batchEntry
	ua := false
	rule := false
	for s.Scan() {
		l := s.Text()
		reUa := regexp.MustCompile(regexUserAgent)
		reAllow := regexp.MustCompile(regexAllowRule)
		reDisallow := regexp.MustCompile(regexDisallowRule)
		switch {
		case (ua || rule) && reAllow.MatchString(l):
			ua = false
			rule = true
			m := reAllow.FindStringSubmatch(l)
			e.allow = append(e.allow, m[1])
		case (ua || rule) && reDisallow.MatchString(l):
			ua = false
			rule = true
			m := reDisallow.FindStringSubmatch(l)
			e.disallow = append(e.disallow, m[1])
		default:
			if rule {
				rIndex.addTxtRule(e)
				e = batchEntry{}
			}
			uM := reUa.FindStringSubmatch(l)
			if len(uM) > 0 {
				ua = true
				rule = false
				e.ua = append(e.ua, uM[1])
			} else {
				ua = false
				rule = false
			}
		}
	}
	if rule {
		rIndex.addTxtRule(e)
	}

	return rIndex
}

type jsonBotUserAgentIndex map[string]BotMetadata

// Validate checks that the json bot source has all required values.
func (m *jsonBotUserAgentIndex) validate() error {
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
	err = jsonIndex.validate()
	if err != nil {
		return rIndex, err
	}

	for u, m := range jsonIndex {
		e := BotUserAgent{JSONMetadata: m}
		rIndex[u] = e
	}

	return rIndex, err
}
