package logger

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var testLogOut bytes.Buffer //nolint:gochecknoglobals

// TestNewLog tests that a logger can be initialized by the simpler New() function
func TestNewLog(t *testing.T) {
	log := New("DEBUG")
	got := reflect.TypeOf(log).String()
	if got != "*logger.Log" {
		t.Error("Unexpected type returned from logger.New() constructor. Got: " + got)
	}
}

// TestLogLevel calls Log.NewFromWriters() with each log level and validates its output
func TestLogLevel(t *testing.T) {
	testLogOut.Reset()

	methods := []string{"Debug", "Info", "Warn", "Error"}
	for _, m := range methods {
		lvl := strings.ToUpper(m)
		log := NewFromWriter(lvl, &testLogOut)
		msg := fmt.Sprintf("Test %s!", lvl)
		want := regexp.MustCompile(fmt.Sprintf(".* level=%s.* msg=\"%s\" pluginName=bot-wrangler-traefik-plugin.*", lvl, msg))
		reflect.ValueOf(log).MethodByName(m).Call([]reflect.Value{reflect.ValueOf(msg)})
		log.Debug(msg)
		got := testLogOut.String()
		if !want.MatchString(got) {
			t.Errorf("Log.%s() did not write the expected string as output! Wanted '%s', Got '%s'", m, want, got)
		}
	}
}
