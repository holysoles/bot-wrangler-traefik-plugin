package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func getPtr(s string) *string { return &s }
func sliceMatch(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, e := range a {
		if e != b[i] {
			return false
		}
	}
	return true
}
func indexMatchSource(r RobotsIndex, s map[string]botMetadata) bool {
	for k, v := range r {
		getV, ok := s[k]
		if !ok {
			return false
		}
		if v.JSONMetadata == getV {
			return false
		}
	}
	return true
}

// newJSONServer is a helper function to return a test server that will return example JSON
func newJSONServer(t *testing.T, cT string) (*httptest.Server, []byte) {
	t.Helper()
	b, err := json.Marshal(sourceRobotsJSON)
	if err != nil {
		t.Error("unexpected error marshaling example JSON: " + err.Error())
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if cT != "" {
			w.Header().Add("Content-Type", cT)
		}
		_, err := w.Write(b)
		if err != nil {
			t.Error("unexpected error writing example JSON: " + err.Error())
		}
	}))
	return s, b
}

const (
	exampleSourceRobotsTxt = `
user-agent: MyBot
disallow: /
allow: /sitemap.xml`
	exampleSourceRobotsTxtMulti = `user-agent: MyBot
disallow: /
allow: /sitemap.xml
user-agent: MyBot2
allow: /index.html
disallow: /
allow: /sitemap.xml
# some comment`
)

var (
	exampleSourceRobotsTxtMap = map[string]BotUserAgent{
		"MyBot": {
			AllowPath:    []string{"/sitemap.xml"},
			DisallowPath: []string{"/"},
		},
		"MyBot2": {
			AllowPath:    []string{"/index.html", "/sitemap.xml"},
			DisallowPath: []string{"/"},
		},
	}
	sourceRobotsMetadata = botMetadata{
		Operator:    getPtr("MyBot.lan"),
		Respect:     getPtr("Yes"),
		Function:    getPtr("golang unit tests"),
		Frequency:   getPtr("n/a"),
		Description: getPtr("used for this package's unit tests"),
	}
	sourceRobotsJSON        = map[string]botMetadata{"MyBot": sourceRobotsMetadata}
	sourceRobotsMetadataBad = botMetadata{
		Operator: getPtr("MyBot.lan"),
	}
	sourceRobotsJSONBad = map[string]botMetadata{"MyBadBot": sourceRobotsMetadataBad}
)

func TestAddTxtRule(t *testing.T) {
	i := make(RobotsIndex)
	testUa := "MyBot"
	testAllow := []string{"/sitemap.xml"}
	testDisallow := []string{"/"}
	e := batchEntry{ua: []string{testUa}, allow: testAllow, disallow: testDisallow}

	i.addTxtRule(e)
	v, ok := i[testUa]
	if !ok {
		t.Error("User Agent from Batch Entry not a key in RobotsIndex")
	}
	if len(v.AllowPath) < 1 || v.AllowPath[0] != testAllow[0] {
		t.Error("Allowed paths from batch entry not preserved in RobotsIndex")
	}
	if len(v.DisallowPath) < 1 || v.DisallowPath[0] != testDisallow[0] {
		t.Error("Disallowed paths from batch entry not preserved in RobotsIndex")
	}
}

type errorReader struct{}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("simulated read error")
}

// TestRobotsJSONParse tests that an error is raised if the bufio.Reader cannot be read from
func TestRobotsJSONParseClosedReader(t *testing.T) {
	r := &errorReader{}
	bR := bufio.NewReader(r)
	_, err := robotsJSONParse(bR)
	if err == nil {
		t.Error("passing an invalid reader did not return an error")
	}
}

// TestGetSourceContent tests retrieving a http.response for a valid URL
func TestGetSourceContent(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
		_, err := fmt.Fprintln(w, "foo bar")
		if err != nil {
			t.Error("unexpected error writing response body: " + err.Error())
		}
	}))
	defer s.Close()

	err := (&Source{URL: s.URL}).getContent()
	if err != nil {
		t.Error("unexpected error when requesting source: " + err.Error())
	}
}

// TestGetIndexFromSourcesBadUrl tests retrieving a http.response for a invalid URL returns an error
func TestGetIndexFromSourcesBadUrl(t *testing.T) {
	s := &Source{URL: "%%"}
	_, err := s.GetIndex()
	if err == nil {
		t.Error("Malformed source URL did not return an error when requesting content")
	}
}

// TestGetIndexFromSourcesHttpErr tests an error is raised if an unexpected HTTP status code is returned when fetching content
func TestGetIndexFromSourcesHttpErr(t *testing.T) {
	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer serv.Close()

	s := &Source{URL: serv.URL}
	_, err := s.GetIndex()
	if err == nil {
		t.Error("Malformed source URL did not return an error when requesting content")
	}
}

// TestGetSourceContentJSON tests that the correct content type is determined for a JSON source
func TestGetSourceContentTypeJSON(t *testing.T) {
	serv, _ := newJSONServer(t, "application/json")
	defer serv.Close()

	s := &Source{URL: serv.URL}
	err := s.getContent()
	if err != nil {
		t.Error("unexpected error when requesting source: " + err.Error())
	}
	_, err = s.getContentType()
	if err != nil {
		t.Error("unexpected error when detecting content-type of source: " + err.Error())
	}
	if s.contentType != contentRobotsJSON {
		t.Errorf("expected content type '%s', got '%s'", contentRobotsJSON, s.contentType)
	}
}

// TestGetSourceContentJSONSniff tests that the correct content type is determined for a JSON source when mime-type sniffing is used
func TestGetSourceContentTypeJSONSniff(t *testing.T) {
	serv, _ := newJSONServer(t, "")
	defer serv.Close()

	s := &Source{URL: serv.URL}
	err := s.getContent()
	if err != nil {
		t.Error("unexpected error when requesting source: " + err.Error())
	}
	_, err = s.getContentType()
	if err != nil {
		t.Error("unexpected error when detecting content-type of source: " + err.Error())
	}
	if s.contentType != contentRobotsJSON {
		t.Errorf("expected content type '%s', got '%s'", contentRobotsJSON, s.contentType)
	}
}

// TestGetIndexFromContentBadReader tests an error is raised if the content type detection encounters a bad reader
func TestGetIndexFromContentBadReader(t *testing.T) {
	serv, _ := newJSONServer(t, "")
	defer serv.Close()

	s := &Source{URL: serv.URL}
	err := s.getContent()
	if err != nil {
		t.Error("unexpected error when requesting source: " + err.Error())
	}
	emptyR := bytes.NewReader([]byte{})
	emptyRC := io.NopCloser(emptyR)
	s.response.Body = emptyRC
	_, err = s.getIndexFromContent()
	if err == nil {
		t.Error("expected error when trying to detect content type without valid reader")
	}
}

// TestGetIndexJSONMalformed tests that an error is raised if malformed JSON is retrieved and attempted to be parsed
func TestGetIndexJSONMalformed(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		_, err := fmt.Fprint(w, "{{{}{}{{")
		if err != nil {
			t.Error("unexpected error writing malformed JSON: " + err.Error())
		}
	}))
	defer s.Close()

	src := Source{URL: s.URL}
	_, err := src.GetIndex()
	if err == nil {
		t.Error("source URL providing malformed JSON did not return an error when parsing content")
	}
}

// TestGetSourceContentTxt tests that the correct content type is determined for a robots.txt source
func TestGetSourceContentTypeTxt(t *testing.T) {
	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := fmt.Fprintln(w, exampleSourceRobotsTxt)
		if err != nil {
			t.Error("unexpected error writing response body: " + err.Error())
		}
	}))
	defer serv.Close()

	s := &Source{URL: serv.URL}
	err := s.getContent()
	if err != nil {
		t.Error("unexpected error when requesting source: " + err.Error())
	}
	_, err = s.getContentType()
	if err != nil {
		t.Error("unexpected error when detecting content-type of source: " + err.Error())
	}
	if s.contentType != contentRobotsTxt {
		t.Errorf("expected content type '%s', got '%s'", contentRobotsTxt, s.contentType)
	}
}

// TestRobotsSourceUpdateTxt tests updating a bot index from a single robots.txt source
func TestRobotsSourceUpdateTxtSingle(t *testing.T) {
	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := fmt.Fprintln(w, exampleSourceRobotsTxt)
		if err != nil {
			t.Error("unexpected error writing response body: " + err.Error())
		}
	}))
	defer serv.Close()

	src := Source{URL: serv.URL}
	r, err := src.GetIndex()
	if err != nil {
		t.Error("unexpected error when parsing robots.txt source: " + err.Error())
	}
	rL := len(r)
	getL := 1
	if rL != getL {
		t.Errorf("expected %d bot entries, got %d", getL, rL)
	}
	for k, v := range r {
		getV, ok := exampleSourceRobotsTxtMap[k]
		if !ok {
			t.Errorf("expected User-Agent '%s' to be retrieved", k)
		}
		aOk := sliceMatch(v.AllowPath, getV.AllowPath)
		if !aOk {
			t.Errorf("expected Allow: '%s', got '%s'", strings.Join(getV.AllowPath, ","), strings.Join(v.AllowPath, ","))
		}
		dOk := sliceMatch(v.DisallowPath, getV.DisallowPath)
		if !dOk {
			t.Errorf("expected Disallow: '%s', got '%s'", strings.Join(getV.DisallowPath, ","), strings.Join(v.DisallowPath, ","))
		}
	}
}

// TestGetIndexFromContentRefresh tests that if a content type for a source is already known, we parse the list correctly
func TestGetIndexFromContentRefresh(t *testing.T) {
	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := fmt.Fprintln(w, exampleSourceRobotsTxt)
		if err != nil {
			t.Error("unexpected error writing response body: " + err.Error())
		}
	}))
	defer serv.Close()

	src := Source{URL: serv.URL, contentType: contentRobotsTxt}
	_, err := src.GetIndex()
	if err != nil {
		t.Error("unexpected error when requesting source: " + err.Error())
	}
}

// TestRobotsSourceUpdateTxtMulti tests updating a bot index from a single robots.txt source with multiple bot entries and checks its fields
func TestRobotsSourceUpdateTxtMulti(t *testing.T) {
	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := fmt.Fprintln(w, exampleSourceRobotsTxtMulti)
		if err != nil {
			t.Error("unexpected error writing response body: " + err.Error())
		}
	}))
	defer serv.Close()

	src := Source{URL: serv.URL}
	r, err := src.GetIndex()
	if err != nil {
		t.Error("unexpected error when parsing robots.txt source: " + err.Error())
	}
	rL := len(r)
	getL := 2
	if rL != getL {
		t.Errorf("expected %d bot entries, got %d", getL, rL)
	}
	for k, v := range r {
		getV, ok := exampleSourceRobotsTxtMap[k]
		if !ok {
			t.Errorf("expected User-Agent '%s' to be retrieved", k)
		}
		aOk := sliceMatch(v.AllowPath, getV.AllowPath)
		if !aOk {
			t.Errorf("expected Allow: '%s', got '%s'", strings.Join(getV.AllowPath, ","), strings.Join(v.AllowPath, ","))
		}
		dOk := sliceMatch(v.DisallowPath, getV.DisallowPath)
		if !dOk {
			t.Errorf("expected Disallow: '%s', got '%s'", strings.Join(getV.DisallowPath, ","), strings.Join(v.DisallowPath, ","))
		}
	}
}

// TestRobotsSourceUpdateJSONSingle tests updating a bot index from a single json source with a single bot entry, and checks its fields
func TestRobotsSourceUpdateJSONSingle(t *testing.T) {
	serv, b := newJSONServer(t, "application/json")
	defer serv.Close()

	src := Source{URL: serv.URL}
	r, err := src.GetIndex()
	if err != nil {
		t.Error("unexpected error when parsing robots json source: " + err.Error())
	}
	rL := len(r)
	getL := len(sourceRobotsJSON)
	if len(r) != 1 {
		t.Errorf("expected %d bot entries, got %d", getL, rL)
	}
	if !indexMatchSource(r, sourceRobotsJSON) {
		rB, err := json.Marshal(r)
		if err != nil {
			t.Fatal("unable to marshall generated RobotsIndex to JSON")
		}
		t.Errorf("retrieved RobotsIndex does not match source data. expected: '%s', got: '%s'", b, rB)
	}
}

// TestRobotsSourceUpdateJSONSingleInvalid tests updating a bot index from a single json source with a single bot entry that is missing required fields
func TestRobotsSourceUpdateJSONSingleInvalid(t *testing.T) {
	serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		b, err := json.Marshal(sourceRobotsJSONBad)
		if err != nil {
			t.Error("unexpected error marshaling example JSON: " + err.Error())
		}
		w.Header().Add("Content-Type", "application/json")
		_, err = w.Write(b)
		if err != nil {
			t.Error("unexpected error writing example JSON: " + err.Error())
		}
	}))
	defer serv.Close()

	src := Source{URL: serv.URL}
	_, err := src.GetIndex()
	if err == nil {
		t.Error("expected error to be raised when passed source that returns JSON with missing fields")
	}
}

// TestRobotsSourceUpdatePlaintext tests updating a bot index from a single plaintext source
func TestRobotsSourceUpdatePlaintext(t *testing.T) {
	src := Source{URL: "https://cdn.jsdelivr.net/gh/ai-robots-txt/ai.robots.txt@latest/haproxy-block-ai-bots.txt"}
	r, err := src.GetIndex()
	if err != nil {
		t.Error("unexpected error when parsing multiple mixed robots sources: " + err.Error())
	}
	rL := len(r)
	// approximate ai robots plaintext list at > 100 entries
	getL := 100
	if len(r) < getL {
		t.Errorf("expected at least %d bot entries, got %d", getL, rL)
	}
}
