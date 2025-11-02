package ahocorasick

import (
	"testing"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/parser"
)

const (
	exampleUserAgent = "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; GPTBot/1.0; +https://openai.com/gptbot)"
)

var (
	simpleIndex = parser.RobotsIndex{
		"a":   parser.BotUserAgent{},
		"ab":  parser.BotUserAgent{},
		"bab": parser.BotUserAgent{},
		"caa": parser.BotUserAgent{},
	}
	exampleIndex = parser.RobotsIndex{
		"GPTBot":   parser.BotUserAgent{},
		"TestBot":  parser.BotUserAgent{},
		"Test-Bot": parser.BotUserAgent{},
	}
)

// TestNewFromIndex constructs a new Aho-Corasick automaton and inspects its structure
func TestNewFromIndex(t *testing.T) {
	a := NewFromIndex(simpleIndex)
	check := 'b'
	a, ok := a.next[check]
	if !ok {
		t.Errorf("expected to find '%c' as child of root node", check)
	}
	check = 'a'
	b, ok := a.next[check]
	if !ok {
		t.Errorf("expected to find '%c' as child of root node", check)
	}
	check = 'b'
	c, ok := b.next[check]
	if !ok {
		t.Errorf("expected to find '%c' as child of root node", check)
	}
	if !c.output {
		t.Errorf("expected '%c' to be marked as exit node", check)
	}
	checkStr := "bab"
	if c.endsHere != checkStr {
		t.Errorf("expected '%c' to be have word '%s' set as endsHere, got '%s'", check, checkStr, c.endsHere)
	}
}

func TestSuffixLinks(t *testing.T) {
	// Create the automaton
	root := NewFromIndex(simpleIndex)

	// Manually validate suffix links
	tests := []struct {
		node     *Node
		expected *Node
	}{
		// Root node should link to itself
		{root, root},

		// 'a' node should link to root
		{root.next['a'], root},

		// 'ab' node should link to 'b' node
		{root.next['a'].next['b'], root.next['b']},

		// 'b'; node should link to root
		{root.next['b'], root},

		// 'ba' node should link to 'a' node
		{root.next['b'].next['a'], root.next['a']},

		// 'bab' node should link to 'ab' node
		{root.next['b'].next['a'].next['b'], root.next['a'].next['b']},

		// 'c' node should link to root
		{root.next['c'], root},

		// 'ca' node should link to 'a' node'
		{root.next['c'].next['a'], root.next['a']},

		// 'caa' node should link to 'a' node
		{root.next['c'].next['a'].next['a'], root.next['a']},
	}

	for _, tt := range tests {
		t.Run(string(tt.node.letter), func(t *testing.T) {
			if tt.node.suffixLink != tt.expected {
				t.Errorf("expected suffix link to be '%c' (%p), got '%c' (%p)", tt.expected.letter, tt.expected, tt.node.suffixLink.letter, tt.node.suffixLink)
			}
		})
	}
}

// TestSearchExactMatch constructs a new Aho-Corasick automaton and runs a search for a string with an exact match.
func TestSearchExactMatch(t *testing.T) {
	a := NewFromIndex(exampleIndex)
	check := "GPTBot"
	_, match := a.Search(check)
	if !match {
		t.Errorf("expected match for '%s', did not find match", check)
	}
}

// TestSearchPrefixMatch constructs a new Aho-Corasick automaton and runs a search for a string with a leading match.
func TestSearchPrefixMatch(t *testing.T) {
	a := NewFromIndex(exampleIndex)
	check := "GPTBot followed by words"
	_, match := a.Search(check)
	if !match {
		t.Errorf("expected match for '%s', did not find match", check)
	}
}

// TestSearchPrefixMatch constructs a new Aho-Corasick automaton and runs a search for a string with a tailing match.
func TestSearchSuffixMatch(t *testing.T) {
	a := NewFromIndex(exampleIndex)
	check := "some words followed by GPTBot"
	_, match := a.Search(check)
	if !match {
		t.Errorf("expected match for '%s', did not find match", check)
	}
}

// TestSearchPrefixMatch constructs a new Aho-Corasick automaton and runs a search for a string with no match.
func TestSearchNoMatch(t *testing.T) {
	a := NewFromIndex(exampleIndex)
	check := "just some words"
	matchStr, match := a.Search(check)
	if match {
		t.Errorf("expected no match for '%s', but found a match under '%s'", check, matchStr)
	}
}

// TestSearchPrefixMatch constructs a new Aho-Corasick automaton from a large dataset and runs searches for both a match and no match
func TestSearchLargeIndex(t *testing.T) {
	u := parser.Source{URL: "https://cdn.jsdelivr.net/gh/ai-robots-txt/ai.robots.txt/robots.json"}
	r, _ := u.GetIndex()
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
