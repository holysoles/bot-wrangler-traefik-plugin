// Package logger is a simple wrapper around the builtin "log" package that provides log levels.
// It'd be nice to use zerolog to match Traefik's log output, but Yaegi does not allow use of the "unsafe" module, which is a transient dependency.
package logger

import (
	"io"
	"log/slog"
	"os"
)

// Log struct for the logger.
type Log struct {
	*slog.Logger
}

// New initializes the logger for the plugin. Output configured by lvl parameter.
func New(lvl string) *Log {
	return NewFromWriter(lvl, os.Stdout)
}

// NewFromWriter initializes the logger to write to the provided io.Writer. Output configured by lvl parameter.
func NewFromWriter(lvl string, w io.Writer) *Log {
	var sLvl slog.Level
	// Level.UnmarshalText handles string comp. we already handle string validation in config.ValidateConfig()
	_ = sLvl.UnmarshalText([]byte(lvl))
	defaultAttrs := []slog.Attr{
		slog.String("pluginName", "bot-wrangler-traefik-plugin"),
	}
	// we can't set just HandlerOptions.AddSource=true, it'll just showup as reflect src
	log := slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: sLvl}).WithAttrs(defaultAttrs))
	slog.SetDefault(log)
	return &Log{log}
}
