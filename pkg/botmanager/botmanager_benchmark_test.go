package botmanager

import (
	"bytes"
	"testing"

	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/ahocorasick"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/config"
	"github.com/holysoles/bot-wrangler-traefik-plugin/pkg/logger"
)

const (
	exampleShortString = "GPTBot"
	exampleLongString  = "a really long string that happens to have a match that we care about GPTBot before the end"
	exampleSource      = "https://cdn.jsdelivr.net/gh/ai-robots-txt/ai.robots.txt/robots.json"
)

var (
	log   = logger.NewFromWriter("ERROR", &testLogOut)
	c     = config.New()
	bM, _ = New(exampleSource, c.CacheUpdateInterval, log, c.CacheSize, c.UseFastMatch, c.RobotsTXTDisallowAll, c.RobotsTXTFilePath, c.RobotsSourceRetryInterval)
)

func BenchmarkSimpleSearchShort(b *testing.B) {
	// yaegi doesn't like a range over int loop
	// https://github.com/traefik/yaegi/issues/1701
	for i := 0; i < b.N; i++ { //nolint:intrange,modernize
		_ = bM.slowSearch(exampleShortString)
	}
}
func BenchmarkSimpleSearchLong(b *testing.B) {
	for i := 0; i < b.N; i++ { //nolint:intrange,modernize
		_ = bM.slowSearch(exampleLongString)
	}
}

func BenchmarkAhoCorsasickSearchShort(b *testing.B) {
	bM.ahoCorasick = ahocorasick.NewFromIndex(bM.botIndex)
	for i := 0; i < b.N; i++ { //nolint:intrange,modernize
		_ = bM.fastSearch(exampleShortString)
	}
}

func BenchmarkAhoCorsasickSearchLong(b *testing.B) {
	bM.ahoCorasick = ahocorasick.NewFromIndex(bM.botIndex)
	for i := 0; i < b.N; i++ { //nolint:intrange,modernize
		_ = bM.fastSearch(exampleLongString)
	}
}

func BenchmarkRobotsTxtRenderCache(b *testing.B) {
	for i := 0; i < b.N; i++ { //nolint:intrange,modernize
		w := &bytes.Buffer{}
		_ = bM.RenderRobotsTxt(w, true)
	}
}
func BenchmarkRobotsTxtRenderNoCache(b *testing.B) {
	for i := 0; i < b.N; i++ { //nolint:intrange,modernize
		w := &bytes.Buffer{}
		_ = bM.RenderRobotsTxt(w, false)
	}
}
