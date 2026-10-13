package decodo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const defaultCrawlPollInterval = 5 * time.Second

// Crawl states reported in GetCrawlStatusResponse.Status.
const (
	CrawlStatusPending   = "pending"
	CrawlStatusDone      = "done"
	CrawlStatusFaulted   = "faulted"
	CrawlStatusCancelled = "cancelled"
	CrawlStatusTruncated = "truncated"
)

// CrawlPage is a single crawled page.
type CrawlPage = ListCrawlPagesResponseURLItem

type crawlOptions struct {
	pollInterval time.Duration
	metadataOnly bool
}

// CrawlOption configures Crawl and GetCrawlResults. Crawl applies WithPollInterval
// to its polling and passes the rest on to GetCrawlResults.
type CrawlOption func(*crawlOptions)

// WithPollInterval sets how often Crawl checks the crawl status (default: 5s).
func WithPollInterval(d time.Duration) CrawlOption {
	return func(o *crawlOptions) { o.pollInterval = d }
}

// WithMetadataOnly omits page content from the results. The API serves content in
// smaller pages, so leaving it on makes more requests.
func WithMetadataOnly() CrawlOption {
	return func(o *crawlOptions) { o.metadataOnly = true }
}

func newCrawlOptions(opts []CrawlOption) crawlOptions {
	o := crawlOptions{pollInterval: defaultCrawlPollInterval}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// requireDataAPI guards the Data API endpoints, which only accept an API key.
func (api *WebScrapingAPI) requireDataAPI(caller string) error {
	if api.configErr != nil {
		return api.configErr
	}
	if !api.routes.dataAPI {
		return &ConfigurationError{Msg: fmt.Sprintf(
			"%s requires an API key. Configure WebScrapingAPIConfig.APIKey instead of Token.", caller)}
	}
	return nil
}

func (api *WebScrapingAPI) dataAPIPath(path, caller string) (string, error) {
	if err := api.requireDataAPI(caller); err != nil {
		return "", err
	}
	return path, nil
}

func (api *WebScrapingAPI) crawlPath(path, crawlID, caller string) (string, error) {
	path, err := api.dataAPIPath(path, caller)
	if err != nil {
		return "", err
	}
	if crawlID == "" {
		return "", &ConfigurationError{Msg: fmt.Sprintf("%s requires a crawl ID.", caller)}
	}
	return strings.Replace(path, "{id}", url.PathEscape(crawlID), 1), nil
}

func (api *WebScrapingAPI) call(ctx context.Context, method, path string, params, out interface{}) error {
	var body []byte
	var err error
	if method == "GET" {
		body, err = api.http.get(ctx, path, nil)
	} else {
		body, err = api.http.post(ctx, path, params)
	}
	if err != nil {
		return asValidationError(err)
	}
	if body == nil {
		return fmt.Errorf("unexpected empty response from server")
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

// asValidationError reports the Data API's plain 400s, which carry no errors array,
// as *ValidationError so callers can handle rejected input the same way as on Scrape.
func asValidationError(err error) error {
	if base, ok := err.(*DecodoError); ok && base.StatusCode == 400 {
		return &ValidationError{DecodoError: *base}
	}
	return err
}

// Search runs a web search. Requires an API key.
func (api *WebScrapingAPI) Search(ctx context.Context, params RunSearchRequest) (*RunSearchResponse, error) {
	path, err := api.dataAPIPath(dataAPIPathRunSearch, "Search")
	if err != nil {
		return nil, err
	}
	var result RunSearchResponse
	if err := api.call(ctx, "POST", path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Map discovers the links of a site. Requires an API key.
func (api *WebScrapingAPI) Map(ctx context.Context, params MapSiteRequest) (*MapSiteResponse, error) {
	path, err := api.dataAPIPath(dataAPIPathMapSite, "Map")
	if err != nil {
		return nil, err
	}
	var result MapSiteResponse
	if err := api.call(ctx, "POST", path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// StartCrawl starts a crawl and returns immediately. Requires an API key.
func (api *WebScrapingAPI) StartCrawl(ctx context.Context, params StartCrawlRequest) (*StartCrawlResponse, error) {
	path, err := api.dataAPIPath(dataAPIPathStartCrawl, "StartCrawl")
	if err != nil {
		return nil, err
	}
	var result StartCrawlResponse
	if err := api.call(ctx, "POST", path, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCrawlStatus retrieves the progress of a crawl. Requires an API key.
func (api *WebScrapingAPI) GetCrawlStatus(ctx context.Context, crawlID string) (*GetCrawlStatusResponse, error) {
	path, err := api.crawlPath(dataAPIPathGetCrawlStatus, crawlID, "GetCrawlStatus")
	if err != nil {
		return nil, err
	}
	var result GetCrawlStatusResponse
	if err := api.call(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCrawlResults returns every page of a crawl, following the next link until the
// last page. Pages carry their content unless WithMetadataOnly is passed.
// Requires an API key.
func (api *WebScrapingAPI) GetCrawlResults(ctx context.Context, crawlID string, opts ...CrawlOption) ([]CrawlPage, error) {
	path, err := api.crawlPath(dataAPIPathListCrawlPages, crawlID, "GetCrawlResults")
	if err != nil {
		return nil, err
	}
	if !newCrawlOptions(opts).metadataOnly {
		path += "?type=content"
	}

	var pages []CrawlPage
	for path != "" {
		var page ListCrawlPagesResponse
		if err := api.call(ctx, "GET", path, nil, &page); err != nil {
			return nil, err
		}
		pages = append(pages, page.URL...)

		path = ""
		if page.Next != "" {
			// Only the path and query are kept so the request goes to the configured host.
			next, err := url.Parse(page.Next)
			if err != nil {
				return nil, fmt.Errorf("parsing next page link: %w", err)
			}
			path = next.RequestURI()
		}
	}
	return pages, nil
}

// CancelCrawl stops a running crawl. Requires an API key.
func (api *WebScrapingAPI) CancelCrawl(ctx context.Context, crawlID string) (*CancelCrawlResponse, error) {
	path, err := api.crawlPath(dataAPIPathCancelCrawl, crawlID, "CancelCrawl")
	if err != nil {
		return nil, err
	}
	var result CancelCrawlResponse
	if err := api.call(ctx, "POST", path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Crawl starts a crawl, polls until the server settles it and returns every page
// with its content. Requires an API key.
//
// The wait is bounded by ctx. When ctx expires, Crawl returns a *CrawlTimeoutError
// (or *CancellationError if ctx was cancelled) and the crawl keeps running on the
// server; stop it with CancelCrawl. A crawl that ends faulted or cancelled returns
// a *CrawlError. A truncated crawl returns the pages it collected. Any other error
// after the crawl started is wrapped in a *CrawlRequestError so the crawl ID is not lost.
func (api *WebScrapingAPI) Crawl(ctx context.Context, params StartCrawlRequest, opts ...CrawlOption) ([]CrawlPage, error) {
	o := newCrawlOptions(opts)
	if o.pollInterval <= 0 {
		return nil, &ConfigurationError{Msg: "poll interval must be positive."}
	}
	if err := api.requireDataAPI("Crawl"); err != nil {
		return nil, err
	}

	job, err := api.StartCrawl(ctx, params)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, &TimeoutError{Msg: "context deadline exceeded while starting the crawl; it may have been created, check your crawl list"}
		}
		return nil, err
	}

	for {
		status, err := api.GetCrawlStatus(ctx, job.CrawlID)
		if err != nil {
			return nil, crawlContextError(ctx, job.CrawlID, err)
		}

		switch status.Status {
		case CrawlStatusFaulted, CrawlStatusCancelled:
			return nil, &CrawlError{CrawlID: job.CrawlID, Status: status.Status, Reason: status.Reason}
		case CrawlStatusDone, CrawlStatusTruncated:
			pages, err := api.GetCrawlResults(ctx, job.CrawlID, opts...)
			if err != nil {
				return nil, crawlContextError(ctx, job.CrawlID, err)
			}
			return pages, nil
		}

		select {
		case <-ctx.Done():
			return nil, crawlContextError(ctx, job.CrawlID, ctx.Err())
		case <-time.After(o.pollInterval):
		}
	}
}

// crawlContextError attaches the crawl ID to every error that occurs once the crawl
// exists, so the caller can still cancel or collect a crawl that outlived the wait.
func crawlContextError(ctx context.Context, crawlID string, err error) error {
	switch ctx.Err() {
	case context.DeadlineExceeded:
		return &CrawlTimeoutError{CrawlID: crawlID}
	case context.Canceled:
		return &CancellationError{Msg: fmt.Sprintf("crawl %s was cancelled while waiting; it is still running, call CancelCrawl to stop it", crawlID)}
	}
	return &CrawlRequestError{CrawlID: crawlID, Err: err}
}
