package proxy

import (
	"fmt"
	"context"
	"testing"
	"net/http"
	"net/http/httptest"
)

// TestBotProxyNew tests the default initialization behavior of proxy.New()
func TestBotProxyNew(_ *testing.T) {
	_ = New("http://localhost")
}

// TestBotProxyServe tests that the BotProxy actually forwards a request to the backend server
func TestBotProxyServe(t *testing.T) {
	want := "the backend server has been reached by the reverse proxy"
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Helper()
		_, err := fmt.Fprint(w, want)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer backendServer.Close()

	ctx := context.Background()

	p := New(backendServer.URL)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}
	p.ServeHTTP(recorder, req)
	got := recorder.Body.String()
	if got != want {
		t.Error("the BotProxy did not forward the response to the backend server")
	}
}

// TestBotProxyNoBuffering tests that the ReverseProxy is not buffering the backend server's response
func TestReverseProxyNoBuffering(t *testing.T) {
    backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        // Send a large response body w appropriate headers
		w.Header().Set("Content-Length", "1024")
		_, err := w.Write(make([]byte, 1024))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
			return
		}
    }))
    defer backendServer.Close()

	p := New(backendServer.URL)
	ctx := context.Background()
	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}
	p.ServeHTTP(recorder, req)
	res := recorder.Result()

	// Check if the response is chunked
	rTE := res.TransferEncoding
	if len(rTE) > 0 {
		if rTE[0] == "chunked" {
			t.Errorf("Reverse proxy is buffering the response")
		}
	}

	// Check if the Content-Length header matches the expected size
	resCL := res.Header.Get("Content-Length")
	if resCL != "1024" {
		t.Errorf("Unexpected Content-Length: %s", resCL)
	}
}
