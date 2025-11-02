package botmanager

import (
	"testing"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/ahocorasick"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/parser"
)

const (
	exampleShortString = "GPTBot"
	exampleLongString  = "a really long string that happens to have a match that we care about GPTBot before the end"
	exampleSource      = "https://cdn.jsdelivr.net/gh/ai-robots-txt/ai.robots.txt/robots.json"
)

var (
	bM = BotUAManager{botIndex: testGetIndex()}
)

func testGetIndex() parser.RobotsIndex {
	u := parser.Source{URL: exampleSource}
	r, _ := u.GetIndex()
	return r
}

func BenchmarkSimpleSearchShort(b *testing.B) {
	for range b.N {
		_, _ = bM.slowSearch(exampleShortString)
	}
}
func BenchmarkSimpleSearchLong(b *testing.B) {
	for range b.N {
		_, _ = bM.slowSearch(exampleLongString)
	}
}

func BenchmarkAhoCorsasickSearchShort(b *testing.B) {
	bM.ahoCorasick = ahocorasick.NewFromIndex(bM.botIndex)
	for range b.N {
		_, _ = bM.fastSearch(exampleShortString)
	}
}

func BenchmarkAhoCorsasickSearchLong(b *testing.B) {
	bM.ahoCorasick = ahocorasick.NewFromIndex(bM.botIndex)
	for range b.N {
		_, _ = bM.fastSearch(exampleLongString)
	}
}
