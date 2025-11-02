package aho_corasick

import (
	"fmt"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/parser"
)

type Node struct {
	letter rune
	// TODO could be array of 256 for all ASCII
	next       map[rune]*Node
	endsHere   string
	output     bool
	suffixLink *Node
}

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
	ancestor := p.suffixLink
	for {
		// root child
		if ancestor == ancestor.suffixLink {
			a.suffixLink = ancestor
			break
		}
		if ancestor.next[a.letter] != nil {
			a.suffixLink = ancestor.next[a.letter]
			break
		}
		ancestor = ancestor.suffixLink
	}
}

func (a *Node) Search(s string) (string, bool) {
	curr := a
	match := false
	fmt.Print()
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
