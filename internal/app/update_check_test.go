package toudocu

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseUpdateCheckFlag(t *testing.T) {
	options, _, _, err := ParseArguments([]string{"serve", "./docs", "--no-update-check"})
	if err != nil || !options.NoUpdateCheck {
		t.Fatalf("options=%#v err=%v", options, err)
	}
	if _, _, _, err := ParseArguments([]string{"build", "./docs", "--no-update-check"}); err == nil || !strings.Contains(err.Error(), "only for serve") {
		t.Fatalf("build accepted --no-update-check: %v", err)
	}
}

func TestUpdateCheckerDoesNotFollowRedirects(t *testing.T) {
	checker := newUpdateChecker()
	client, ok := checker.client.(*http.Client)
	if !ok || client.CheckRedirect == nil {
		t.Fatal("update checker must configure an explicit redirect policy")
	}
	if err := client.CheckRedirect(&http.Request{}, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("redirect policy error = %v, want http.ErrUseLastResponse", err)
	}
}

func TestUpdateChecker(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantStatus string
		wantLatest string
		wantURL    string
	}{
		{name: "newer", statusCode: 200, body: `{"tag_name":"0.0.2","draft":false,"prerelease":false}`, wantStatus: "update-available", wantLatest: "0.0.2", wantURL: releasePageBase + "0.0.2"},
		{name: "same", statusCode: 200, body: `{"tag_name":"0.0.1"}`, wantStatus: "up-to-date", wantLatest: "0.0.1"},
		{name: "older", statusCode: 200, body: `{"tag_name":"0.0.0"}`, wantStatus: "up-to-date", wantLatest: "0.0.0"},
		{name: "prerelease", statusCode: 200, body: `{"tag_name":"0.0.2","prerelease":true}`, wantStatus: "unavailable"},
		{name: "draft", statusCode: 200, body: `{"tag_name":"0.0.2","draft":true}`, wantStatus: "unavailable"},
		{name: "invalid version", statusCode: 200, body: `{"tag_name":"v0.0.2"}`, wantStatus: "unavailable"},
		{name: "malformed", statusCode: 200, body: `{`, wantStatus: "unavailable"},
		{name: "upstream error", statusCode: 503, body: `{}`, wantStatus: "unavailable"},
		{name: "oversized", statusCode: 200, body: strings.Repeat("x", maxReleaseResponseSize+1), wantStatus: "unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Accept") != "application/vnd.github+json" || r.Header.Get("User-Agent") != "toudocu/0.0.1" {
					t.Fatalf("headers=%v", r.Header)
				}
				w.WriteHeader(test.statusCode)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			got := checkLatestVersion(server.Client(), server.URL, "0.0.1")
			if got.SchemaVersion != 1 || got.CurrentVersion != "0.0.1" || got.Status != test.wantStatus || got.LatestVersion != test.wantLatest || got.ReleaseURL != test.wantURL {
				t.Fatalf("response=%#v", got)
			}
		})
	}

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(50 * time.Millisecond)
			_, _ = w.Write([]byte(`{"tag_name":"0.0.2"}`))
		}))
		defer server.Close()
		client := server.Client()
		client.Timeout = time.Millisecond
		if got := checkLatestVersion(client, server.URL, "0.0.1"); got.Status != "unavailable" {
			t.Fatalf("response=%#v", got)
		}
	})

	t.Run("memoized", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			_, _ = w.Write([]byte(`{"tag_name":"0.0.2"}`))
		}))
		defer server.Close()
		checker := &updateChecker{client: server.Client(), endpoint: server.URL, current: "0.0.1"}
		results := make(chan versionCheckResponse, 8)
		for range 8 {
			go func() { results <- checker.check() }()
		}
		for range 8 {
			if result := <-results; result.Status != "update-available" {
				t.Fatalf("response=%#v", result)
			}
		}
		if calls.Load() != 1 {
			t.Fatalf("calls=%d", calls.Load())
		}
	})
}

func TestVersionEndpoint(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"tag_name":"0.0.2"}`))
	}))
	defer upstream.Close()
	server := &documentationServer{updateChecker: &updateChecker{client: upstream.Client(), endpoint: upstream.URL, current: "0.0.1"}}

	get := httptest.NewRecorder()
	server.ServeHTTP(get, httptest.NewRequest(http.MethodGet, versionEndpoint, nil))
	if get.Code != http.StatusOK || get.Header().Get("Cache-Control") != "no-store" || get.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("GET status=%d headers=%v", get.Code, get.Header())
	}
	var payload versionCheckResponse
	if err := json.Unmarshal(get.Body.Bytes(), &payload); err != nil || payload.Status != "update-available" || payload.LatestVersion != "0.0.2" {
		t.Fatalf("payload=%#v err=%v", payload, err)
	}

	head := httptest.NewRecorder()
	server.ServeHTTP(head, httptest.NewRequest(http.MethodHead, versionEndpoint, nil))
	if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("HEAD status=%d headers=%v body=%q", head.Code, head.Header(), head.Body.String())
	}

	post := httptest.NewRecorder()
	server.ServeHTTP(post, httptest.NewRequest(http.MethodPost, versionEndpoint, nil))
	if post.Code != http.StatusMethodNotAllowed || post.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("POST status=%d headers=%v", post.Code, post.Header())
	}
	if calls.Load() != 1 {
		t.Fatalf("upstream calls=%d", calls.Load())
	}
}

func TestVersionEndpointDisabled(t *testing.T) {
	var calls atomic.Int32
	client := roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, nil
	})
	for name, server := range map[string]*documentationServer{
		"flag": {options: Options{NoUpdateCheck: true}, updateChecker: &updateChecker{client: client, endpoint: latestReleaseAPI, current: Version}},
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, versionEndpoint, nil))
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d", response.Code)
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("outbound calls=%d", calls.Load())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) Do(request *http.Request) (*http.Response, error) {
	return function(request)
}
