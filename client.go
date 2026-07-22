package decodo

const (
	webAPIBaseURL    = "https://scraper-api.decodo.com"
	defaultTimeoutMs = 180_000
)

// WebScrapingAPIConfig contains configuration for the web scraping API.
type WebScrapingAPIConfig struct {
	// Token is the authentication token (required).
	Token string
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
		httpCfg := httpClientConfig{
			baseURL:           webAPIBaseURL,
			authType:          authTypeBasic,
			authToken:         config.WebScrapingAPI.Token,
			timeoutMs:         timeoutMs,
			integrationHeader: integrationHeader,
		}
		c.WebScrapingAPI = newWebScrapingAPI(newHTTPClient(httpCfg), schema)
	}

	return c
}
