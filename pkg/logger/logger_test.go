package logger

import (
	"bytes"
	"testing"
	"regexp"
)

// init sets up the testing environment and helpers
var testStdOut bytes.Buffer //nolint:gochecknoglobals
var testStdErr bytes.Buffer //nolint:gochecknoglobals
//nolint:gochecknoinits
func init() {
	stdOut = &testStdOut
	stdErr = &testStdErr
}


// TestNewLogDebug calls Log.New() with a DEBUG log level and validates its output
func TestNewLogDebug(t *testing.T) {
	testStdOut.Reset()
	testStdErr.Reset()
	log := New("DEBUG")
	msg := "Test debug!"
	want := regexp.MustCompile("DEBUG - .+" + msg + "\n")
	log.Debug(msg)
	got := testStdOut.String()
	if !want.MatchString(got) {
		t.Errorf("Log.Debug() did not write the expected string to StdOut! Wanted '%s', Got '%s'", want, got)
	}
	got = testStdErr.String()
	if got != "" {
		t.Errorf("Log.Debug() unexpectedly wrote to StdErr! Got '%s'", got)
	}
}

// TestNewLogInfo calls Log.New() with an INFO log level and validates its output
func TestNewLogInfo(t *testing.T) {
	testStdOut.Reset()
	testStdErr.Reset()
	log := New("INFO")
	msg := "Test info!"
	want := regexp.MustCompile("INFO - .+" + msg + "\n")
	log.Info(msg)
	got := testStdOut.String()
	if !want.MatchString(got) {
		t.Errorf("Log.Info() did not write the expected string to StdOut! Wanted '%s', Got '%s'", want, got)
	}
	got = testStdErr.String()
	if got != "" {
		t.Errorf("Log.Info() unexpectedly wrote to StdErr! Got '%s'", got)
	}
}

// TestNewLogWarn calls Log.New() with a WARN log level and validates its output
func TestNewLogWarn(t *testing.T) {
	testStdOut.Reset()
	testStdErr.Reset()
	log := New("WARN")
	msg := "Test warn!"
	want := regexp.MustCompile("WARN - .+" + msg + "\n")
	log.Warn(msg)
	got := testStdOut.String()
	if !want.MatchString(got) {
		t.Errorf("Log.Warn() did not write the expected string to StdOut! Wanted '%s', Got '%s'", want, got)
	}
	got = testStdErr.String()
	if got != "" {
		t.Errorf("Log.Warn() unexpectedly wrote to StdErr! Got '%s'", got)
	}
}


// TestNewLogError calls Log.New() with an ERROR log level and validates its output
func TestNewLogError(t *testing.T) {
	testStdOut.Reset()
	testStdErr.Reset()
	log := New("ERROR")
	msg := "Test error!"
	want := regexp.MustCompile("ERROR - .+" + msg + "\n")
	log.Error(msg)
	got := testStdErr.String()
	if !want.MatchString(got) {
		t.Errorf("Log.Error() did not write the expected string to StdErr! Wanted '%s', Got '%s'", want, got)
	}
	got = testStdOut.String()
	if got != "" {
		t.Errorf("Log.Error() unexpectedly wrote to StdOut! Got '%s'", got)
	}
}
