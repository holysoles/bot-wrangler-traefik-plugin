// Package proxy provides a reverse proxy to send bot requests through
package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// BotProxy is a wrapper around httputil.ReverseProxy() to proxy bot requests to a backend server.
type BotProxy struct {
	Proxy    *httputil.ReverseProxy
}

// New returns a new BotProxy instance that acts as a reverse proxy to the provided url.
func New(u string) (*BotProxy) {
	// we don't error check since it was already done in ValidateConfig()
	dURL, _ := url.Parse(u)
	// we could check connectivity to the URL before setting up here, but in case the destination wants real requests
	// or is just temporarily unavailable, we won't fail the initialization
	rP := httputil.NewSingleHostReverseProxy(dURL)
	// since we're likely sending this request to a "tarpit" style application, we shouldn't buffer the response for performance
	rP.BufferPool = nil
	bP := BotProxy{rP}

	return &bP
}

// ServeHTTP Handles forwarding the request to the designated destination.
func (bP *BotProxy) ServeHTTP (w http.ResponseWriter, r *http.Request) {
	// we assume NewSingleHostReverseProxy gives us a ReverseProxy that automatically handles forwarded headers (e.g. X-Forwarded-For)
	bP.Proxy.ServeHTTP(w, r)
}