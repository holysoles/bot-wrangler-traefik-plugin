// Package parser provides parsing functionality for bot sources.
package parser

import (
	"bufio"
	"bytes"
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
	regexUserAgent    = `(?im)(?:^user-agent\s?:\s?)(.*)$`
	regexAllowRule    = `(?im)(?:^allow\s?:\s?)(.*)$`
	regexDisallowRule = `(?im)(?:^disallow\s?:\s?)(.*)$`
	// while RFC 9309 says only letters, _, and - are allowed, in the wild we see almost any non-newline characters.
	regexProductToken = `(?i)(^[^\n\r]+$)` //nolint:gosec

	contentRobotsJSON = "robots.json"
	contentRobotsTxt  = "robots.txt"
	contentPlaintext  = "plaintext"
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

// RobotsIndex is a hash of bot user agents and associated data with each.
type RobotsIndex map[string]BotUserAgent

// batchEntry represents a logical entry from a robots.txt file.
type batchEntry struct {
	ua       []string
	allow    []string
	disallow []string
}

// Source represents a location that content will be retrieved from to populate a RobotsIndex.
type Source struct {
	URL         string
	response    *http.Response
	contentType string
}

func (r *RobotsIndex) addTxtRule(e batchEntry) {
	for _, u := range e.ua {
		(*r)[u] = BotUserAgent{AllowPath: e.allow, DisallowPath: e.disallow}
	}
}

func (s *Source) getContent() error {
	req, err := http.NewRequest(http.MethodGet, s.URL, nil)
	if err != nil {
		return err
	}

	s.response, err = http.DefaultClient.Do(req)
	return err
}

func (s *Source) getContentType() (*bufio.Reader, error) {
	s.contentType = contentPlaintext
	bR := bufio.NewReader(s.response.Body)
	var err error

	sniff := s.response.Header.Get("X-Content-Type-Options") != "nosniff"
	u := s.response.Request.URL.String()
	if s.response.Header.Get("Content-Type") == mime.TypeByExtension(".json") || strings.HasSuffix(u, ".json") {
		s.contentType = contentRobotsJSON
		return bR, err
	}
	if sniff {
		var firstC []byte
		firstC, err = bR.Peek(1)
		if err == nil {
			if string(firstC) == "{" {
				s.contentType = contentRobotsJSON
				return bR, err
			}
		}
	}
	// look for user-agent directive as hint this is robots.txt
	buf := &bytes.Buffer{}
	tee := io.TeeReader(bR, buf)
	re := regexp.MustCompile(regexUserAgent)
	bT := bufio.NewReader(buf)
	bS := bufio.NewScanner(tee)
	for bS.Scan() {
		if re.MatchString(bS.Text()) {
			s.contentType = contentRobotsTxt
			break
		}
	}
	return bT, err
}

func (s *Source) getIndexFromContent() (RobotsIndex, error) {
	var rIndex RobotsIndex
	var bR *bufio.Reader
	var err error

	if s.contentType == "" {
		bR, err = s.getContentType()
		if err != nil {
			return rIndex, err
		}
	} else {
		bR = bufio.NewReader(s.response.Body)
	}

	switch s.contentType {
	case contentRobotsJSON:
		rIndex, err = robotsJSONParse(bR)
	case contentRobotsTxt:
		rIndex = robotsTxtParse(bR)
	case contentPlaintext:
		rIndex = robotsPlaintextParse(bR)
	}

	return rIndex, err
}

func (s *Source) getIndex() (RobotsIndex, error) {
	i := make(RobotsIndex)
	err := s.getContent()
	if err != nil {
		return i, err
	}
	defer func() { err = s.response.Body.Close() }()
	if s.response.StatusCode != http.StatusOK {
		return i, fmt.Errorf("error retrieving source data from '%s'. Status: %s", s.URL, s.response.Status)
	}
	i, err = s.getIndexFromContent()
	return i, err
}

// GetIndexFromSources manages retrieving robots source from slice of URLs, and parses it accordingly to a merged RobotsIndex.
// TODO move this into botmanager..
func GetIndexFromSources(l []Source) (RobotsIndex, error) {
	i := make(RobotsIndex)
	for _, s := range l {
		n, err := s.getIndex()
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
	reUa := regexp.MustCompile(regexUserAgent)
	reAllow := regexp.MustCompile(regexAllowRule)
	reDisallow := regexp.MustCompile(regexDisallowRule)
	for s.Scan() {
		l := s.Text()
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

func robotsPlaintextParse(r *bufio.Reader) RobotsIndex {
	s := bufio.NewScanner(r)
	rIndex := make(RobotsIndex)
	re := regexp.MustCompile(regexProductToken)
	for s.Scan() {
		l := s.Text()
		m := re.FindStringSubmatch(l)
		if len(m) > 0 {
			r := BotUserAgent{}
			rIndex[m[0]] = r
		}
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
