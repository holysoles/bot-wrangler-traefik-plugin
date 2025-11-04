package logger

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"
)

var testLogOut bytes.Buffer //nolint:gochecknoglobals

// TestNewLog tests that a logger can be initialized by the simpler New() function
func TestNewLog(t *testing.T) {
	log := New("DEBUG")
	got := fmt.Sprintf("%T", log)
	// yaegi docs indicate that type names may differ between compiled and interpreted modes
	compiledType := "*logger.Log"
	yaegiType := "*struct { Logger *slog.Logger }" //
	if got != compiledType && got != yaegiType {
		t.Error("Unexpected type returned from logger.New() constructor. Got: " + got)
	}
}

// TestLogLevel calls Log.NewFromWriters() with each log level and validates its output
func TestLogLevel(t *testing.T) {
	testLogOut.Reset()

	lvl := "DEBUG"
	t.Run(lvl, func(t *testing.T) {
		log := NewFromWriter(lvl, &testLogOut)
		msg := fmt.Sprintf("Test %s!", lvl)
		want := regexp.MustCompile(fmt.Sprintf(".* level=%s.* msg=\"%s\" pluginName=bot-wrangler-traefik-plugin.*", lvl, msg))
		log.Debug(msg)
		got := testLogOut.String()
		if !want.MatchString(got) {
			t.Errorf("Log.%s() did not write the expected string as output! Wanted '%s', Got '%s'", lvl, want, got)
		}
	})
	lvl = "INFO"
	t.Run(lvl, func(t *testing.T) {
		log := NewFromWriter(lvl, &testLogOut)
		msg := fmt.Sprintf("Test %s!", lvl)
		want := regexp.MustCompile(fmt.Sprintf(".* level=%s.* msg=\"%s\" pluginName=bot-wrangler-traefik-plugin.*", lvl, msg))
		log.Info(msg)
		got := testLogOut.String()
		if !want.MatchString(got) {
			t.Errorf("Log.%s() did not write the expected string as output! Wanted '%s', Got '%s'", lvl, want, got)
		}
	})
	lvl = "WARN"
	t.Run(lvl, func(t *testing.T) {
		log := NewFromWriter(lvl, &testLogOut)
		msg := fmt.Sprintf("Test %s!", lvl)
		want := regexp.MustCompile(fmt.Sprintf(".* level=%s.* msg=\"%s\" pluginName=bot-wrangler-traefik-plugin.*", lvl, msg))
		log.Warn(msg)
		got := testLogOut.String()
		if !want.MatchString(got) {
			t.Errorf("Log.%s() did not write the expected string as output! Wanted '%s', Got '%s'", lvl, want, got)
		}
	})
	lvl = "ERROR"
	t.Run(lvl, func(t *testing.T) {
		log := NewFromWriter(lvl, &testLogOut)
		msg := fmt.Sprintf("Test %s!", lvl)
		want := regexp.MustCompile(fmt.Sprintf(".* level=%s.* msg=\"%s\" pluginName=bot-wrangler-traefik-plugin.*", lvl, msg))
		log.Error(msg)
		got := testLogOut.String()
		if !want.MatchString(got) {
			t.Errorf("Log.%s() did not write the expected string as output! Wanted '%s', Got '%s'", lvl, want, got)
		}
	})
}
