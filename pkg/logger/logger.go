package logger

import (
	"io"
	"log"
	"os"
)

type Log struct {
	logDebug *log.Logger
	logInfo  *log.Logger
	logWarn  *log.Logger
	logError *log.Logger
}

// New Initializes the logger for the plugin. Output configured by logLevel parameter
func New(logLevel string) *Log {
	sourceName := "BotWranglerTraefikPlugin"
	logDebug := log.New(io.Discard, "DEBUG - "+sourceName+": ", log.Ldate|log.Ltime)
	logInfo := log.New(io.Discard, "INFO - "+sourceName+": ", log.Ldate|log.Ltime)
	logWarn := log.New(io.Discard, "WARN - "+sourceName+": ", log.Ldate|log.Ltime)
	logError := log.New(io.Discard, "ERROR - "+sourceName+": ", log.Ldate|log.Ltime)

	logError.SetOutput(os.Stderr)
	switch logLevel {
	case "DEBUG":
		logDebug.SetOutput(os.Stdout)
		fallthrough
	case "INFO":
		logInfo.SetOutput(os.Stdout)
		fallthrough
	case "WARN":
		logWarn.SetOutput(os.Stdout)
	}

	return &Log{
		logDebug: logDebug,
		logInfo:  logInfo,
		logWarn:  logWarn,
		logError: logError,
	}
}

func (l *Log) Debug(str string) {
	l.logDebug.Printf("%s", str)
}

func (l *Log) Info(str string) {
	l.logInfo.Printf("%s", str)
}

func (l *Log) Warn(str string) {
	l.logWarn.Printf("%s", str)
}

func (l *Log) Error(str string) {
	l.logError.Printf("%s", str)
}
