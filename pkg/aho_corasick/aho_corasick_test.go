package aho_corasick

import (
	"testing"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/parser"
)

const (
	exampleUserAgent = "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; GPTBot/1.0; +https://openai.com/gptbot)"
)

var (
	r = parser.RobotsIndex{
		"Go":                         parser.BotUserAgent{},
		"GPTBot":                     parser.BotUserAgent{},
		"GPTOther":                   parser.BotUserAgent{},
		"CustomBot":                  parser.BotUserAgent{},
		"TestBotIsAreallyLongString": parser.BotUserAgent{},
	}
)

// TestNewFromIndex constructs a new Aho-Corasick automaton and inspects its structure
func TestNewFromIndex(t *testing.T) {
	a := NewFromIndex(r)
	// TODO check complexity?
	check := 'G'
	g, ok := a.next[check]
	if !ok {
		t.Errorf("expected to find '%c' as child of root node", check)
	}
	check = 'o'
	o, ok := g.next[check]
	if !ok {
		t.Errorf("expected to find '%c' as child of root node", check)
	}
	if !o.output {
		t.Errorf("expected '%c' to be marked as exit node", check)
	}
	checkStr := "Go"
	if o.endsHere != checkStr {
		t.Errorf("expected '%c' to be have word '%s' set as endsHere, got '%s'", check, checkStr, o.endsHere)
	}

}

func TestSearchExactMatch(t *testing.T) {
	a := NewFromIndex(r)
	check := "GPTBot"
	_, match := a.Search(check)
	if !match {
		t.Errorf("expected match for '%s', did not find match", check)
	}
}

func TestSearchPrefixMatch(t *testing.T) {
	a := NewFromIndex(r)
	check := "GPTBot followed by words"
	_, match := a.Search(check)
	if !match {
		t.Errorf("expected match for '%s', did not find match", check)
	}
}
func TestSearchSuffixMatch(t *testing.T) {
	a := NewFromIndex(r)
	check := "some words followed by GPTBot"
	_, match := a.Search(check)
	if !match {
		t.Errorf("expected match for '%s', did not find match", check)
	}
}
func TestSearchNoMatch(t *testing.T) {
	a := NewFromIndex(r)
	check := "just some words"
	matchStr, match := a.Search(check)
	if match {
		t.Errorf("expected no match for '%s', but found a match under '%s'", check, matchStr)
	}
}

func TestSearchLargeIndex(t *testing.T) {
	u := []parser.Source{{URL: "https://cdn.jsdelivr.net/gh/ai-robots-txt/ai.robots.txt/robots.json"}}
	r, _ := parser.GetIndexFromSources(u)
	a := NewFromIndex(r)

	t.Run("NoMatch", func(t *testing.T) {
		check := "just some words"
		matchStr, match := a.Search(check)
		if match {
			t.Errorf("expected no match for '%s', but found a match under '%s'", check, matchStr)
		}
	})

	t.Run("Match", func(t *testing.T) {
		check := exampleUserAgent
		_, match := a.Search(check)
		if !match {
			t.Errorf("expected match for '%s', but found no match", check)
		}
	})
}

// TODO benchmarks? node sizes?
