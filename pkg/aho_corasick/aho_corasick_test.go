package aho_corasick

import (
	"fmt"
	"testing"
	"time"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/parser"
)

const (
	testIterations = 10
)

func timer(name string) func() time.Duration {
	start := time.Now()
	return func() time.Duration {
		fmt.Printf("%s took %v\n", name, time.Since(start))
		return time.Since(start)
	}
}

var (
	r = parser.RobotsIndex{
		"Go":       parser.BotUserAgent{},
		"GPTBot":   parser.BotUserAgent{},
		"GPTOther": parser.BotUserAgent{},
	}
)

// TODO localized tests. need to make mock data
// test match of exact string
// test match of starting string
// test match of ending string
// test no match

// benchmarks? node sizes?
