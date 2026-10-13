package decodo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newDataAPITestAPI(srv *httptest.Server) *WebScrapingAPI {
	cfg := httpClientConfig{
		baseURL:           srv.URL,
		authType:          authTypeBearer,
		credential:        "test-key",
		timeoutMs:         5_000,
		integrationHeader: "sdk-go-test",
	}
	routes := webScrapingAPIRoutes{scrape: dataAPIScrapePath, task: dataAPITaskPath, dataAPI: true}
	return newWebScrapingAPI(newHTTPClient(cfg), nil, routes)
}

func crawlStatusBody(status string) map[string]any {
	return map[string]any{"crawl_id": "c1", "status": status, "target": "crawl"}
}

func TestSearch_PostsToSearchRoute(t *testing.T) {
	var gotPath, gotMethod, gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod, gotAuth = r.URL.Path, r.Method, r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		writeJSON(w, map[string]any{
			"status":  "done",
			"organic": []any{map[string]any{"title": "Shoes", "link": "https://a.test", "rank": 1}},
		})
	}))
	defer srv.Close()

	res, err := newDataAPITestAPI(srv).Search(context.Background(), RunSearchRequest{Query: "shoes", Geo: Ptr("United States")})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if gotMethod != "POST" || gotPath != "/v1/search" || gotAuth != "Bearer test-key" {
		t.Fatalf("got %s %s auth=%q", gotMethod, gotPath, gotAuth)
	}
	if gotBody["query"] != "shoes" || gotBody["geo"] != "United States" {
		t.Fatalf("body = %v", gotBody)
	}
	if _, ok := gotBody["gl"]; ok {
		t.Fatalf("unset optional field was sent: %v", gotBody)
	}
	if len(res.Organic) != 1 || res.Organic[0].Title != "Shoes" {
		t.Fatalf("organic = %+v", res.Organic)
	}
}

func TestMap_SendsPathListsAsArrays(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		writeJSON(w, map[string]any{"results": []any{map[string]any{
			"content": map[string]any{"links": []any{map[string]any{"url": "https://a.test/x"}}},
			"map_id":  "m1",
		}}})
	}))
	defer srv.Close()

	res, err := newDataAPITestAPI(srv).Map(context.Background(), MapSiteRequest{
		URL:         "https://a.test",
		SelectPaths: []string{"^/blog"},
		Limit:       Ptr(100),
	})
	if err != nil {
		t.Fatalf("Map error: %v", err)
	}
	if gotPath != "/v1/map" {
		t.Fatalf("path = %q", gotPath)
	}
	if paths, ok := gotBody["select_paths"].([]any); !ok || len(paths) != 1 {
		t.Fatalf("select_paths = %v", gotBody["select_paths"])
	}
	if got := res.Results[0].Content.Links[0].URL; got != "https://a.test/x" {
		t.Fatalf("link = %q", got)
	}
}

func TestDataAPI_RejectsTokenWithoutCallingHTTP(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	defer srv.Close()

	api := newTestWebScrapingAPI(t, srv)
	ctx := context.Background()

	calls_ := map[string]func() error{
		"Search":          func() error { _, err := api.Search(ctx, RunSearchRequest{Query: "q"}); return err },
		"Map":             func() error { _, err := api.Map(ctx, MapSiteRequest{URL: "https://a.test"}); return err },
		"StartCrawl":      func() error { _, err := api.StartCrawl(ctx, StartCrawlRequest{URL: "https://a.test"}); return err },
		"GetCrawlStatus":  func() error { _, err := api.GetCrawlStatus(ctx, "c1"); return err },
		"GetCrawlResults": func() error { _, err := api.GetCrawlResults(ctx, "c1"); return err },
		"CancelCrawl":     func() error { _, err := api.CancelCrawl(ctx, "c1"); return err },
		"Crawl":           func() error { _, err := api.Crawl(ctx, StartCrawlRequest{URL: "https://a.test"}); return err },
	}
	for name, call := range calls_ {
		var cfgErr *ConfigurationError
		err := call()
		if !errors.As(err, &cfgErr) || !strings.Contains(cfgErr.Msg, "requires an API key") || !strings.Contains(cfgErr.Msg, name) {
			t.Errorf("%s error = %v, want ConfigurationError naming the method and API key", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("server was called %d times", calls.Load())
	}
}

func TestCrawlLowLevel_Routes(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.EscapedPath())
		writeJSON(w, crawlStatusBody("pending"))
	}))
	defer srv.Close()

	api := newDataAPITestAPI(srv)
	ctx := context.Background()
	job, err := api.StartCrawl(ctx, StartCrawlRequest{URL: "https://a.test"})
	if err != nil || job.CrawlID != "c1" {
		t.Fatalf("StartCrawl = %+v, %v", job, err)
	}
	if _, err := api.GetCrawlStatus(ctx, "c/1"); err != nil {
		t.Fatal(err)
	}
	if _, err := api.CancelCrawl(ctx, "c1"); err != nil {
		t.Fatal(err)
	}

	want := []string{"POST /v1/crawl", "GET /v1/crawl/c%2F1/status", "POST /v1/crawl/c1/cancel"}
	assertEqualSlices(t, "requests", seen, want)
}

func TestGetCrawlResults_FollowsNextAgainstConfiguredHost(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.RequestURI())
		if r.URL.Query().Get("cursor") == "" {
			writeJSON(w, map[string]any{
				"crawl_id": "c1",
				"url":      []any{map[string]any{"url": "https://a.test/1", "content": "one"}},
				"next":     "https://other.example/v1/crawl/c1?type=content&cursor=abc",
			})
			return
		}
		writeJSON(w, map[string]any{
			"crawl_id": "c1",
			"url":      []any{map[string]any{"url": "https://a.test/2", "content": "two"}},
		})
	}))
	defer srv.Close()

	pages, err := newDataAPITestAPI(srv).GetCrawlResults(context.Background(), "c1")
	if err != nil {
		t.Fatalf("GetCrawlResults error: %v", err)
	}
	if len(pages) != 2 || pages[0].Content != "one" || pages[1].Content != "two" {
		t.Fatalf("pages = %+v", pages)
	}
	assertEqualSlices(t, "requests", seen, []string{
		"/v1/crawl/c1?type=content",
		"/v1/crawl/c1?type=content&cursor=abc",
	})
}

func TestGetCrawlResults_MetadataOnlyOmitsContentType(t *testing.T) {
	var gotURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURI = r.URL.RequestURI()
		writeJSON(w, map[string]any{"crawl_id": "c1", "url": []any{}})
	}))
	defer srv.Close()

	pages, err := newDataAPITestAPI(srv).GetCrawlResults(context.Background(), "c1", WithMetadataOnly())
	if err != nil {
		t.Fatal(err)
	}
	if gotURI != "/v1/crawl/c1" || len(pages) != 0 {
		t.Fatalf("uri = %q, pages = %v", gotURI, pages)
	}
}

func TestCrawl_PollsUntilDoneThenReturnsPages(t *testing.T) {
	var statusCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST":
			writeJSON(w, crawlStatusBody("pending"))
		case strings.HasSuffix(r.URL.Path, "/status"):
			status := "pending"
			if statusCalls.Add(1) >= 3 {
				status = "truncated"
			}
			writeJSON(w, crawlStatusBody(status))
		default:
			writeJSON(w, map[string]any{"crawl_id": "c1", "url": []any{map[string]any{"url": "https://a.test"}}})
		}
	}))
	defer srv.Close()

	pages, err := newDataAPITestAPI(srv).Crawl(context.Background(), StartCrawlRequest{URL: "https://a.test"}, WithPollInterval(time.Millisecond))
	if err != nil {
		t.Fatalf("Crawl error: %v", err)
	}
	if len(pages) != 1 || statusCalls.Load() != 3 {
		t.Fatalf("pages = %d, status calls = %d", len(pages), statusCalls.Load())
	}
}

func TestCrawl_FailedStatusReturnsCrawlError(t *testing.T) {
	for _, status := range []string{CrawlStatusFaulted, CrawlStatusCancelled} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body := crawlStatusBody(status)
			if strings.HasSuffix(r.URL.Path, "/status") {
				body["reason"] = "no_pages_found"
			}
			writeJSON(w, body)
		}))

		_, err := newDataAPITestAPI(srv).Crawl(context.Background(), StartCrawlRequest{URL: "https://a.test"}, WithPollInterval(time.Millisecond))
		srv.Close()

		var crawlErr *CrawlError
		if !errors.As(err, &crawlErr) || crawlErr.CrawlID != "c1" || crawlErr.Status != status || crawlErr.Reason != "no_pages_found" {
			t.Fatalf("%s: error = %v", status, err)
		}
	}
}

func TestCrawl_DeadlineReturnsTimeoutErrorWithCrawlID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, crawlStatusBody("pending"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := newDataAPITestAPI(srv).Crawl(ctx, StartCrawlRequest{URL: "https://a.test"}, WithPollInterval(10*time.Millisecond))

	var timeoutErr *CrawlTimeoutError
	if !errors.As(err, &timeoutErr) || timeoutErr.CrawlID != "c1" {
		t.Fatalf("error = %v, want CrawlTimeoutError for c1", err)
	}
}

func TestCrawl_ContextCancelReturnsCancellationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		writeJSON(w, crawlStatusBody("pending"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	_, err := newDataAPITestAPI(srv).Crawl(ctx, StartCrawlRequest{URL: "https://a.test"}, WithPollInterval(10*time.Millisecond))

	var cancelErr *CancellationError
	if !errors.As(err, &cancelErr) || !strings.Contains(cancelErr.Msg, "c1") {
		t.Fatalf("error = %v, want CancellationError naming c1", err)
	}
}

func TestCrawl_RejectsNonPositivePollInterval(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server must not be called")
	}))
	defer srv.Close()

	_, err := newDataAPITestAPI(srv).Crawl(context.Background(), StartCrawlRequest{URL: "https://a.test"}, WithPollInterval(0))
	var cfgErr *ConfigurationError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("error = %v, want ConfigurationError", err)
	}
}

func TestCrawl_FailedPollKeepsCrawlID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			writeJSON(w, crawlStatusBody("pending"))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]any{"message": "nope"})
	}))
	defer srv.Close()

	_, err := newDataAPITestAPI(srv).Crawl(context.Background(), StartCrawlRequest{URL: "https://a.test"}, WithPollInterval(time.Millisecond))

	var reqErr *CrawlRequestError
	var authErr *AuthenticationError
	if !errors.As(err, &reqErr) || reqErr.CrawlID != "c1" || !errors.As(err, &authErr) {
		t.Fatalf("error = %v, want CrawlRequestError for c1 wrapping AuthenticationError", err)
	}
}

func TestCrawl_ForwardsMetadataOnlyToResults(t *testing.T) {
	var resultsURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST":
			writeJSON(w, crawlStatusBody("pending"))
		case strings.HasSuffix(r.URL.Path, "/status"):
			writeJSON(w, crawlStatusBody("done"))
		default:
			resultsURI = r.URL.RequestURI()
			writeJSON(w, map[string]any{"crawl_id": "c1", "url": []any{}})
		}
	}))
	defer srv.Close()

	_, err := newDataAPITestAPI(srv).Crawl(context.Background(), StartCrawlRequest{URL: "https://a.test"}, WithMetadataOnly())
	if err != nil || resultsURI != "/v1/crawl/c1" {
		t.Fatalf("uri = %q, err = %v", resultsURI, err)
	}
}

func TestCrawlMethods_RejectEmptyCrawlID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server must not be called")
	}))
	defer srv.Close()

	api := newDataAPITestAPI(srv)
	ctx := context.Background()
	var cfgErr *ConfigurationError
	if _, err := api.GetCrawlStatus(ctx, ""); !errors.As(err, &cfgErr) {
		t.Fatalf("GetCrawlStatus error = %v", err)
	}
	if _, err := api.GetCrawlResults(ctx, ""); !errors.As(err, &cfgErr) {
		t.Fatalf("GetCrawlResults error = %v", err)
	}
	if _, err := api.CancelCrawl(ctx, ""); !errors.As(err, &cfgErr) {
		t.Fatalf("CancelCrawl error = %v", err)
	}
}

func TestDataAPIErrors_ReadErrorFieldAndMapBadRequestToValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/status") {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]any{"error": "crawl c9 not found"})
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]any{"error": "limit must be between 1 and 10000"})
	}))
	defer srv.Close()
	api := newDataAPITestAPI(srv)

	_, err := api.StartCrawl(context.Background(), StartCrawlRequest{URL: "https://a.test", Limit: Ptr(0)})
	var val *ValidationError
	if !errors.As(err, &val) || val.StatusCode != 400 || val.Message != "limit must be between 1 and 10000" {
		t.Fatalf("StartCrawl error = %T %v", err, err)
	}

	_, err = api.GetCrawlStatus(context.Background(), "c9")
	var base *DecodoError
	if !errors.As(err, &base) || base.StatusCode != 404 || base.Message != "crawl c9 not found" {
		t.Fatalf("GetCrawlStatus error = %T %v", err, err)
	}
	if errors.As(err, &val) {
		t.Fatalf("404 must not be a ValidationError")
	}
}

func TestScrape_BadRequestWithoutErrorsStaysDecodoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]any{"message": "bad"})
	}))
	defer srv.Close()

	_, err := newTestWebScrapingAPI(t, srv).ScrapeAsync(context.Background(), NewGoogleSearchParams())
	var val *ValidationError
	if errors.As(err, &val) {
		t.Fatalf("existing Scrape behaviour changed: %T %v", err, err)
	}
}
