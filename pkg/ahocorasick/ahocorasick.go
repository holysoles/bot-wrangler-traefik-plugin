// Package ahocorasick provides an implementation of the Aho-Corasick string-search algorithm to search a RobotsIndex.
package ahocorasick

import (
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/parser"
)

// Node represents a single node in the linked Aho-Corasick automaton.
type Node struct {
	letter     rune
	next       map[rune]*Node
	endsHere   string
	output     bool
	suffixLink *Node
}

// NewFromIndex is a constructor that returns an automaton based on the provided RobotsIndex.
func NewFromIndex(m parser.RobotsIndex) *Node {
	arr := make([]string, len(m))
	i := 0
	for k := range m {
		arr[i] = k
		i++
	}

	start := &Node{next: map[rune]*Node{}}

	// construct Trie
	for _, word := range arr {
		this := start
		for _, l := range word {
			exist := false
			for r, n := range this.next {
				if r == l {
					this = n
					exist = true
					break
				}
			}
			if !exist {
				newN := &Node{letter: l, next: map[rune]*Node{}}
				this.next[l] = newN
				this = newN
			}
		}
		this.endsHere = word
		this.output = true
	}

	start.buildLinks()

	return start
}

// Search searches the provided string against the constructed automaton's dictionary for a match.
func (a *Node) Search(s string) (string, bool) {
	curr := a
	match := false
	for _, l := range s {
		n, ok := curr.next[l]
		if ok {
			curr = n
		} else {
			curr = curr.suffixLink
		}
		if curr.output {
			match = true
			break
		}
	}
	return curr.endsHere, match
}
func (a *Node) buildLinks() {
	// BFS, recurse towards root to find longest suffix
	// root's suffixLink is itself
	a.suffixLink = a
	q := []*Node{a}
	for len(q) > 0 {
		curr := q[0]
		q = q[1:]
		for _, n := range curr.next {
			n.setSuffixLink(curr)
			q = append(q, n)
		}
	}
}

func (a *Node) setSuffixLink(p *Node) {
	for {
		p = p.suffixLink

		link, childMatch := p.next[a.letter]
		if childMatch && a != link {
			a.suffixLink = link
			break
		}

		// when true, p is the root node
		if p == p.suffixLink {
			a.suffixLink = p
			break
		}
	}
}
