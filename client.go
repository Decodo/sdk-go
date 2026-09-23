package decodo

import "strings"

const (
	webAPIBaseURL    = "https://scraper-api.decodo.com"
	dataAPIBaseURL   = "https://data.decodo.com"
	defaultTimeoutMs = 180_000
)

// WebScrapingAPIConfig contains configuration for the web scraping API.
type WebScrapingAPIConfig struct {
	Token  string
	APIKey string
	// IntegrationHeader overrides the x-integration header (default: "sdk-go").
	IntegrationHeader string
}

// Config contains configuration for the Decodo client.
type Config struct {
	// WebScrapingAPI configures the web scraping API.
	WebScrapingAPI *WebScrapingAPIConfig
	// TimeoutMs is the request timeout in milliseconds (default: 180000).
	TimeoutMs int
	// Schema overrides the default schema for validation. If nil, a lazy-loaded remote schema is used.
	Schema Schema
}

// Client is the main Decodo SDK client.
type Client struct {
	// WebScrapingAPI provides web scraping functionality.
	WebScrapingAPI *WebScrapingAPI
}

type transport struct {
	baseURL    string
	authType   authType
	credential string
	routes     webScrapingAPIRoutes
}

func resolveTransport(cfg *WebScrapingAPIConfig) (transport, error) {
	token := strings.TrimSpace(cfg.Token)
	apiKey := strings.TrimSpace(cfg.APIKey)

	switch {
	case token != "" && apiKey != "":
		return transport{}, &ConfigurationError{Msg: "WebScrapingAPI accepts either Token or APIKey, not both."}
	case apiKey != "":
		return transport{
			baseURL:    dataAPIBaseURL,
			authType:   authTypeBearer,
			credential: cfg.APIKey,
			routes:     dataAPIRoutes,
		}, nil
	case token != "":
		return transport{
			baseURL:    webAPIBaseURL,
			authType:   authTypeBasic,
			credential: cfg.Token,
			routes:     scraperAPIRoutes,
		}, nil
	default:
		return transport{}, &ConfigurationError{Msg: "WebScrapingAPI requires Token or APIKey."}
	}
}

// NewClient creates a new Decodo client with the given configuration.
func NewClient(config Config) *Client {
	timeoutMs := config.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = defaultTimeoutMs
	}

	schema := config.Schema
	if schema == nil {
		schema = SharedDefaultSchema
	}

	c := &Client{}

	if config.WebScrapingAPI != nil {
		integrationHeader := config.WebScrapingAPI.IntegrationHeader
		if integrationHeader == "" {
			integrationHeader = "sdk-go"
		}

		tr, err := resolveTransport(config.WebScrapingAPI)
		if err != nil {
			c.WebScrapingAPI = newMisconfiguredWebScrapingAPI(err)
			return c
		}

		httpCfg := httpClientConfig{
			baseURL:           tr.baseURL,
			authType:          tr.authType,
			credential:        tr.credential,
			timeoutMs:         timeoutMs,
			integrationHeader: integrationHeader,
		}
		c.WebScrapingAPI = newWebScrapingAPI(newHTTPClient(httpCfg), schema, tr.routes)
	}

	return c
}
