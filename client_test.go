package decodo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newRecordingServer(t *testing.T) (*httptest.Server, *[]string, *[]string) {
	t.Helper()
	var paths, auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		auths = append(auths, r.Header.Get("Authorization"))
		writeJSON(w, map[string]any{"results": []any{}})
	}))
	t.Cleanup(srv.Close)
	return srv, &paths, &auths
}

func newTransportAPI(srv *httptest.Server, tr transport) *WebScrapingAPI {
	cfg := httpClientConfig{
		baseURL:           srv.URL,
		authType:          tr.authType,
		credential:        tr.credential,
		timeoutMs:         5_000,
		integrationHeader: "sdk-go-test",
	}
	return newWebScrapingAPI(newHTTPClient(cfg), nil, tr.routes)
}

func exerciseAllRoutes(t *testing.T, api *WebScrapingAPI) {
	t.Helper()
	ctx := context.Background()
	params := NewUniversalParams()
	params.URL = Ptr("https://example.com")

	if _, err := api.Scrape(ctx, params); err != nil {
		t.Fatalf("Scrape: %v", err)
	}
	if _, err := api.ScrapeAsync(ctx, params); err != nil {
		t.Fatalf("ScrapeAsync: %v", err)
	}
	if _, err := api.ScrapeBatch(ctx, params); err != nil {
		t.Fatalf("ScrapeBatch: %v", err)
	}
	if _, err := api.GetStatus(ctx, "task-1"); err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if _, err := api.GetResults(ctx, "task-1"); err != nil {
		t.Fatalf("GetResults: %v", err)
	}
}

func assertEqualSlices(t *testing.T, name string, got, want []string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("%s:\n got: %q\nwant: %q", name, got, want)
	}
}

func TestResolveTransport_Token(t *testing.T) {
	tr, err := resolveTransport(&WebScrapingAPIConfig{Token: "test-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.baseURL != "https://scraper-api.decodo.com" {
		t.Fatalf("baseURL = %q", tr.baseURL)
	}
	if tr.authType != authTypeBasic || tr.credential != "test-token" {
		t.Fatalf("auth = (%v, %q)", tr.authType, tr.credential)
	}
	if tr.routes != (webScrapingAPIRoutes{scrape: scraperAPIScrapePath, task: scraperAPITaskPath}) {
		t.Fatalf("routes = %+v", tr.routes)
	}
}

func TestResolveTransport_APIKey(t *testing.T) {
	tr, err := resolveTransport(&WebScrapingAPIConfig{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.baseURL != "https://data.decodo.com" {
		t.Fatalf("baseURL = %q", tr.baseURL)
	}
	if tr.authType != authTypeBearer || tr.credential != "test-key" {
		t.Fatalf("auth = (%v, %q)", tr.authType, tr.credential)
	}
	if tr.routes != (webScrapingAPIRoutes{scrape: dataAPIScrapePath, task: dataAPITaskPath}) {
		t.Fatalf("routes = %+v", tr.routes)
	}
}

func TestResolveTransport_Invalid(t *testing.T) {
	cases := []struct {
		name string
		cfg  WebScrapingAPIConfig
		want string
	}{
		{"both", WebScrapingAPIConfig{Token: "t", APIKey: "k"}, "either Token or APIKey"},
		{"neither", WebScrapingAPIConfig{}, "requires Token or APIKey"},
		{"blank token", WebScrapingAPIConfig{Token: "   "}, "requires Token or APIKey"},
		{"blank api key", WebScrapingAPIConfig{APIKey: "   "}, "requires Token or APIKey"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveTransport(&tc.cfg)
			var cfgErr *ConfigurationError
			if !errors.As(err, &cfgErr) {
				t.Fatalf("error = %v, want *ConfigurationError", err)
			}
			if !strings.Contains(cfgErr.Msg, tc.want) {
				t.Fatalf("message %q does not contain %q", cfgErr.Msg, tc.want)
			}
		})
	}
}

func TestToken_UsesScraperAPIRoutesAndBasicAuth(t *testing.T) {
	srv, paths, auths := newRecordingServer(t)
	tr, _ := resolveTransport(&WebScrapingAPIConfig{Token: "test-token"})

	exerciseAllRoutes(t, newTransportAPI(srv, tr))

	assertEqualSlices(t, "paths", *paths, []string{
		"/v2/scrape",
		"/v3/task",
		"/v3/task/batch",
		"/v3/task/task-1",
		"/v3/task/task-1/results",
	})
	for _, a := range *auths {
		if a != "Basic test-token" {
			t.Fatalf("Authorization = %q, want Basic test-token", a)
		}
	}
}

func TestAPIKey_UsesDataAPIRoutesAndBearerAuth(t *testing.T) {
	srv, paths, auths := newRecordingServer(t)
	tr, _ := resolveTransport(&WebScrapingAPIConfig{APIKey: "test-key"})

	exerciseAllRoutes(t, newTransportAPI(srv, tr))

	assertEqualSlices(t, "paths", *paths, []string{
		"/v1/scrape",
		"/v1/task",
		"/v1/task/batch",
		"/v1/task/task-1",
		"/v1/task/task-1/results",
	})
	for _, a := range *auths {
		if a != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", a)
		}
	}
}

func TestNewClient_MisconfiguredCredentialsReturnConfigurationError(t *testing.T) {
	client := NewClient(Config{
		WebScrapingAPI: &WebScrapingAPIConfig{Token: "t", APIKey: "k"},
	})
	if client.WebScrapingAPI == nil {
		t.Fatal("WebScrapingAPI is nil")
	}

	ctx := context.Background()
	params := NewUniversalParams()
	params.URL = Ptr("https://example.com")

	_, err := client.WebScrapingAPI.Scrape(ctx, params)
	var cfgErr *ConfigurationError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("Scrape error = %v, want *ConfigurationError", err)
	}
	if _, err := client.WebScrapingAPI.GetStatus(ctx, "task-1"); !errors.As(err, &cfgErr) {
		t.Fatalf("GetStatus error = %v, want *ConfigurationError", err)
	}
}
