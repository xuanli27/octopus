package helper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xuanli27/octopus/internal/model"
	"github.com/xuanli27/octopus/internal/transformer/outbound"
)

func TestOpenAIModelListURLs(t *testing.T) {
	// Site root: probe multiple common paths (issue #91).
	got := openAIModelListURLs("https://example.com")
	want := []string{
		"https://example.com/models",
		"https://example.com/v1/models",
		"https://example.com/api/v1/models",
		"https://example.com/v1beta/models",
	}
	if len(got) != len(want) {
		t.Fatalf("root candidates: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("root candidates[%d]: got %q want %q", i, got[i], want[i])
		}
	}

	// Already versioned: only /models.
	got = openAIModelListURLs("https://example.com/v1/")
	if len(got) != 1 || got[0] != "https://example.com/v1/models" {
		t.Fatalf("versioned base: got %v", got)
	}
	got = openAIModelListURLs("https://example.com/api/v1")
	if len(got) != 1 || got[0] != "https://example.com/api/v1/models" {
		t.Fatalf("api/v1 base: got %v", got)
	}
}

func TestFetchModelsProbesV1WhenRootReturns404(t *testing.T) {
	hits := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.Path)
		switch r.URL.Path {
		case "/models":
			http.NotFound(w, r)
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"gpt-4o","object":"model"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	models, err := FetchModels(context.Background(), model.Channel{
		Type:     outbound.OutboundTypeOpenAIChat,
		BaseUrls: []model.BaseUrl{{URL: server.URL}},
		Keys:     []model.ChannelKey{{Enabled: true, ChannelKey: "sk-test"}},
	})
	if err != nil {
		t.Fatalf("FetchModels: %v", err)
	}
	if len(models) != 1 || models[0] != "gpt-4o" {
		t.Fatalf("models: got %v", models)
	}
	// Must have tried root /models before /v1/models.
	if len(hits) < 2 || hits[0] != "/models" || hits[1] != "/v1/models" {
		t.Fatalf("probe order: got %v", hits)
	}
}

func TestFetchModelsUsesBrowserHeadersAndSummarizesHTMLError(t *testing.T) {
	observedUserAgent := ""
	observedAccept := ""
	observedAcceptLanguage := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedUserAgent = r.Header.Get("User-Agent")
		observedAccept = r.Header.Get("Accept")
		observedAcceptLanguage = r.Header.Get("Accept-Language")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="en-US"><head><title>Just a moment...</title></head><body>blocked</body></html>`))
	}))
	defer server.Close()

	_, err := FetchModels(context.Background(), model.Channel{
		Type:     outbound.OutboundTypeOpenAIChat,
		BaseUrls: []model.BaseUrl{{URL: server.URL, Delay: 0}},
		Keys:     []model.ChannelKey{{Enabled: true, ChannelKey: "managed-key"}},
	})
	if err == nil {
		t.Fatalf("expected FetchModels to fail")
	}
	if !strings.Contains(err.Error(), "http 403: Just a moment...") {
		t.Fatalf("expected summarized HTML error, got %v", err)
	}
	if !strings.Contains(observedUserAgent, "Mozilla/5.0") {
		t.Fatalf("expected browser user-agent, got %q", observedUserAgent)
	}
	if observedAccept == "" {
		t.Fatalf("expected Accept header to be set")
	}
	if observedAcceptLanguage == "" {
		t.Fatalf("expected Accept-Language header to be set")
	}
}
