// Package logger is a simple wrapper around the builtin "log" package that provides log levels.
// It'd be nice to use zerolog to match Traefik's log output, but Yaegi does not allow use of the "unsafe" module, which is a transient dependency.
package logger

import (
	"io"
	"log"
	"os"
)

// Log struct for the logger.
type Log struct {
	logDebug *log.Logger
	logInfo  *log.Logger
	logWarn  *log.Logger
	logError *log.Logger
}

// New initializes the logger for the plugin. Output configured by logLevel parameter.
func New(logLevel string) *Log {
	return NewFromWriters(logLevel, os.Stdout, os.Stderr)
}

// NewFromWriters initializes the logger with the provided io.Writers for the standard log stream, and error log stream.
func NewFromWriters(logLevel string, o io.Writer, e io.Writer) *Log {
	sourceName := "BotWranglerTraefikPlugin"
	logDebug := log.New(io.Discard, "DEBUG - "+sourceName+": ", log.Ldate|log.Ltime)
	logInfo := log.New(io.Discard, "INFO - "+sourceName+": ", log.Ldate|log.Ltime)
	logWarn := log.New(io.Discard, "WARN - "+sourceName+": ", log.Ldate|log.Ltime)
	logError := log.New(io.Discard, "ERROR - "+sourceName+": ", log.Ldate|log.Ltime)

	logError.SetOutput(e)
	switch logLevel {
	case "DEBUG":
		logDebug.SetOutput(o)
		fallthrough
	case "INFO":
		logInfo.SetOutput(o)
		fallthrough
	case "WARN":
		logWarn.SetOutput(o)
	}

	return &Log{
		logDebug: logDebug,
		logInfo:  logInfo,
		logWarn:  logWarn,
		logError: logError,
	}
}

// Debug writes a Debug level message to the log.
func (l *Log) Debug(str string) {
	l.logDebug.Printf("%s", str)
}

// Info writes a Info level message to the log.
func (l *Log) Info(str string) {
	l.logInfo.Printf("%s", str)
}

// Warn writes a Warn (Warning) level message to the log.
func (l *Log) Warn(str string) {
	l.logWarn.Printf("%s", str)
}

// Error writes a Error level message to the log.
func (l *Log) Error(str string) {
	l.logError.Printf("%s", str)
}
