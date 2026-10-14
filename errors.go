package decodo

import "fmt"

// DecodoError is the base error type for all Decodo API errors.
type DecodoError struct {
	StatusCode int
	APIStatus  string
	Message    string
}

func (e *DecodoError) Error() string {
	return e.Message
}

// AuthenticationError is returned on 401/403 responses.
type AuthenticationError struct {
	DecodoError
}

func (e *AuthenticationError) Unwrap() error { return &e.DecodoError }

// RateLimitError is returned on 429 responses.
type RateLimitError struct {
	DecodoError
}

func (e *RateLimitError) Unwrap() error { return &e.DecodoError }

// ValidationError is returned on 422/400 responses with validation errors.
type ValidationError struct {
	DecodoError
	Errors []interface{}
}

func (e *ValidationError) Unwrap() error { return &e.DecodoError }

type ConfigurationError struct {
	Msg string
}

func (e *ConfigurationError) Error() string { return e.Msg }

// TimeoutError is returned when a request times out.
type TimeoutError struct {
	Msg string
}

func (e *TimeoutError) Error() string {
	return e.Msg
}

// CrawlError is returned by Crawl when the crawl ends faulted or cancelled.
type CrawlError struct {
	CrawlID string
	Status  string
	// Reason is set when the API says why the crawl stopped, for example no_pages_found.
	Reason string
}

func (e *CrawlError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("crawl %s ended with status %s (%s)", e.CrawlID, e.Status, e.Reason)
	}
	return fmt.Sprintf("crawl %s ended with status %s", e.CrawlID, e.Status)
}

// CrawlRequestError is returned by Crawl when a request fails after the crawl was
// started, for example a 5xx while polling. Unless Finished is set, the crawl keeps
// running on the server. It unwraps to the underlying error, so errors.As still
// matches the typed API errors.
type CrawlRequestError struct {
	CrawlID string
	// Finished is set when the crawl had already finished and reading its results failed.
	Finished bool
	Err      error
}

func (e *CrawlRequestError) Error() string {
	if e.Finished {
		return fmt.Sprintf("crawl %s finished but reading its results failed, call GetCrawlResults to collect it: %v", e.CrawlID, e.Err)
	}
	return fmt.Sprintf("crawl %s is still running, call CancelCrawl to stop it: %v", e.CrawlID, e.Err)
}

func (e *CrawlRequestError) Unwrap() error { return e.Err }

// CrawlTimeoutError is returned by Crawl when its context deadline passes. Unless
// Finished is set, the crawl keeps running on the server.
type CrawlTimeoutError struct {
	CrawlID string
	// Finished is set when the deadline passed while reading a finished crawl's results.
	Finished bool
}

func (e *CrawlTimeoutError) Error() string {
	if e.Finished {
		return fmt.Sprintf("crawl %s finished but the context deadline passed while reading its results, call GetCrawlResults to collect it", e.CrawlID)
	}
	return fmt.Sprintf("crawl %s did not finish before the context deadline; it is still running, call CancelCrawl to stop it or GetCrawlResults to collect it later", e.CrawlID)
}

// CrawlCancellationError is returned by Crawl when its context is cancelled. Unless
// Finished is set, the crawl keeps running on the server. It unwraps to
// *CancellationError, so errors.As still matches that type.
type CrawlCancellationError struct {
	CrawlID string
	// Finished is set when the context was cancelled while reading a finished crawl's results.
	Finished bool
}

func (e *CrawlCancellationError) Error() string {
	if e.Finished {
		return fmt.Sprintf("crawl %s finished but the context was cancelled while reading its results, call GetCrawlResults to collect it", e.CrawlID)
	}
	return fmt.Sprintf("crawl %s was cancelled while waiting; it is still running, call CancelCrawl to stop it", e.CrawlID)
}

func (e *CrawlCancellationError) Unwrap() error { return &CancellationError{Msg: e.Error()} }

// CancellationError is returned when the caller's context is cancelled.
type CancellationError struct {
	Msg string
}

func (e *CancellationError) Error() string { return e.Msg }

func mapHTTPError(statusCode int, message, apiStatus string, errors []interface{}) error {
	base := DecodoError{StatusCode: statusCode, APIStatus: apiStatus, Message: message}
	switch {
	case statusCode == 401 || statusCode == 403:
		return &AuthenticationError{base}
	case statusCode == 429:
		return &RateLimitError{base}
	case statusCode == 422 || (statusCode == 400 && len(errors) > 0):
		return &ValidationError{DecodoError: base, Errors: errors}
	default:
		return &base
	}
}
