# Decodo Go SDK

Official Go SDK for the [Decodo](https://decodo.com) Web Scraping API. Scrape Google, Amazon, Walmart, Reddit, YouTube, TikTok, and many more — with built-in proxy rotation, CAPTCHA handling, and JavaScript rendering.

## Installation

```bash
go get github.com/decodo/sdk-go
```

**Requires Go 1.21+**

## Generate types

After cloning the repository, run the type generator to create typed target parameters:

```bash
go run ./cmd/codegen
```

This fetches the latest API schema from the Decodo registry and writes:
- `targets.go` — committed file with the `Target` enum and all target constants
- `generated_params.go` — gitignored file with typed parameter structs for each target
- `generated_parameters.go` — gitignored file with parameter metadata

Re-run this command whenever Decodo publishes an updated schema to pick up new targets or changed parameters.

## Quick Start

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/decodo/sdk-go"
)

func main() {
	client := decodo.NewClient(decodo.Config{
		WebScrapingAPI: &decodo.WebScrapingAPIConfig{
			Token: os.Getenv("DECODO_TOKEN"),
		},
	})

	params := decodo.NewGoogleSearchParams()
	params.Query = decodo.Ptr("web scraping")
	params.Geo = decodo.Ptr("United States")
	params.Parse = decodo.Ptr(true)

	result, err := client.WebScrapingAPI.Scrape(context.Background(), params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}
```

## Configuration

```go
client := decodo.NewClient(decodo.Config{
	WebScrapingAPI: &decodo.WebScrapingAPIConfig{
		Token:             "YOUR_TOKEN", // required
		IntegrationHeader: "my-app",     // optional; default "sdk-go"
	},
	TimeoutMs: 60_000, // optional; default 180000ms (3 minutes)
})
```

Your API token can be found in the [Decodo dashboard](https://app.decodo.com). Set it via the `DECODO_TOKEN` environment variable or pass it directly.

## API Methods

### Scrape (synchronous)

Performs a synchronous scrape and waits for the result.

```go
result, err := client.WebScrapingAPI.Scrape(ctx, params)
// result.Results[0].Content contains the scraped data
```

### ScrapeAsync (asynchronous)

Creates an async scrape task and returns immediately.

```go
task, err := client.WebScrapingAPI.ScrapeAsync(ctx, params)
fmt.Println("Task ID:", task.ID)
```

### GetStatus

Retrieves the current status of an async task.

```go
status, err := client.WebScrapingAPI.GetStatus(ctx, taskID)
// status.Status: "pending", "done", or "faulted"
```

### GetResults

Retrieves the results of a completed async task. Returns `nil` if results are not yet available.

```go
results, err := client.WebScrapingAPI.GetResults(ctx, taskID)
if results != nil {
	// results.Results[0].Content
}
```

### ScrapeBatch

Submits a batch scrape task.

```go
batch, err := client.WebScrapingAPI.ScrapeBatch(ctx, params)
firstID := batch.Queries[0].ID
```

## Async Polling Example

```go
task, err := client.WebScrapingAPI.ScrapeAsync(ctx, params)
if err != nil {
	log.Fatal(err)
}

ticker := time.NewTicker(3 * time.Second)
defer ticker.Stop()
for range ticker.C {
	results, err := client.WebScrapingAPI.GetResults(ctx, task.ID)
	if err != nil {
		log.Fatal(err)
	}
	if results != nil {
		fmt.Println(results.Results[0].Content)
		break
	}
}
```

## Error Handling

The SDK returns typed errors you can inspect:

```go
result, err := client.WebScrapingAPI.Scrape(ctx, params)
if err != nil {
	switch e := err.(type) {
	case *decodo.AuthenticationError:
		fmt.Printf("Auth failed (HTTP %d): %s\n", e.StatusCode, e.Message)
	case *decodo.RateLimitError:
		fmt.Printf("Rate limited: %s\n", e.Message)
	case *decodo.ValidationError:
		fmt.Printf("Invalid params: %s\n", e.Message)
	case *decodo.TimeoutError:
		fmt.Printf("Timeout: %s\n", e.Msg)
	default:
		fmt.Printf("Error: %v\n", err)
	}
}
```

| Error Type | HTTP Status | Description |
|---|---|---|
| `AuthenticationError` | 401, 403 | Invalid or missing token |
| `RateLimitError` | 429 | Too many requests |
| `ValidationError` | 422, 400 | Invalid request parameters |
| `TimeoutError` | — | Request exceeded timeout |
| `DecodoError` | Other | Generic API error |

## Helper: Ptr

Use `decodo.Ptr[T]` to set optional pointer fields:

```go
params.Query = decodo.Ptr("shoes")
params.Parse = decodo.Ptr(true)
params.PageFrom = decodo.Ptr(1)
```

## Schema Validation

By default, the SDK auto-loads the latest target schema from the Decodo CDN on first use and caches it locally for 24 hours (`~/.decodo/decodo.ir.json`). If the fetch fails, validation is silently skipped for that session.

To customise caching behaviour or pin a specific schema URL:

```go
schema, err := decodo.LoadRemoteSchema(decodo.RemoteSchemaOptions{
	TTLMs: 3_600_000, // cache for 1 hour instead of 24
})
if err != nil {
	log.Fatal(err)
}

client := decodo.NewClient(decodo.Config{
	WebScrapingAPI: &decodo.WebScrapingAPIConfig{Token: token},
	Schema:         schema,
})
```

## Supported Targets

### Google
| Target | Constructor |
|---|---|
| `google_search` | `NewGoogleSearchParams()` |
| `google` | `NewGoogleParams()` |
| `google_ads` | `NewGoogleAdsParams()` |
| `google_ai_mode` | `NewGoogleAiModeParams()` |
| `google_lens` | `NewGoogleLensParams()` |
| `google_maps` | `NewGoogleMapsParams()` |
| `google_shopping_product` | `NewGoogleShoppingProductParams()` |
| `google_shopping_search` | `NewGoogleShoppingSearchParams()` |
| `google_suggest` | `NewGoogleSuggestParams()` |
| `google_travel_hotels` | `NewGoogleTravelHotelsParams()` |
| `google_trends_explore` | `NewGoogleTrendsExploreParams()` |

### Amazon
| Target | Constructor |
|---|---|
| `amazon` | `NewAmazonParams()` |
| `amazon_bestsellers` | `NewAmazonBestsellersParams()` |
| `amazon_pricing` | `NewAmazonPricingParams()` |
| `amazon_product` | `NewAmazonProductParams()` |
| `amazon_search` | `NewAmazonSearchParams()` |
| `amazon_sellers` | `NewAmazonSellersParams()` |

### Walmart
| Target | Constructor |
|---|---|
| `walmart` | `NewWalmartParams()` |
| `walmart_product` | `NewWalmartProductParams()` |
| `walmart_search` | `NewWalmartSearchParams()` |

### Target
| Target | Constructor |
|---|---|
| `target` | `NewTargetParams()` |
| `target_product` | `NewTargetProductParams()` |
| `target_search` | `NewTargetSearchParams()` |

### Bing
| Target | Constructor |
|---|---|
| `bing` | `NewBingParams()` |
| `bing_search` | `NewBingSearchParams()` |

### YouTube
| Target | Constructor |
|---|---|
| `youtube_channel` | `NewYoutubeChannelParams()` |
| `youtube_metadata` | `NewYoutubeMetadataParams()` |
| `youtube_search` | `NewYoutubeSearchParams()` |
| `youtube_search_max` | `NewYoutubeSearchMaxParams()` |
| `youtube_subtitles` | `NewYoutubeSubtitlesParams()` |
| `youtube_transcript` | `NewYoutubeTranscriptParams()` |
| `youtube_video` | `NewYoutubeVideoParams()` |

### Reddit
| Target | Constructor |
|---|---|
| `reddit_post` | `NewRedditPostParams()` |
| `reddit_subreddit` | `NewRedditSubredditParams()` |
| `reddit_user` | `NewRedditUserParams()` |

### TikTok
| Target | Constructor |
|---|---|
| `tiktok` | `NewTiktokParams()` |
| `tiktok_post` | `NewTiktokPostParams()` |
| `tiktok_shop_product` | `NewTiktokShopProductParams()` |
| `tiktok_shop_search` | `NewTiktokShopSearchParams()` |

### AI Tools
| Target | Constructor |
|---|---|
| `chatgpt` | `NewChatgptParams()` |
| `gemini` | `NewGeminiParams()` |
| `google_ai_mode` | `NewGoogleAiModeParams()` |
| `perplexity` | `NewPerplexityParams()` |

### Other
| Target | Constructor |
|---|---|
| `airbnb` | `NewAirbnbParams()` |
| `apple_app_store` | `NewAppleAppStoreParams()` |
| `autotrader` | `NewAutotraderParams()` |
| `bbb` | `NewBbbParams()` |
| `ecommerce` | `NewEcommerceParams()` |
| `instagram_graphql_profile` | `NewInstagramGraphqlProfileParams()` |
| `lowes_search` | `NewLowesSearchParams()` |
| `mobile` | `NewMobileParams()` |
| `universal` | `NewUniversalParams()` |
| `universal_ecommerce` | `NewUniversalEcommerceParams()` |

## Code Generation

The `targets.go`, `parameters.go`, and `request_schemas.go` files are generated from the Decodo IR (intermediate representation). To regenerate:

```bash
go run ./cmd/codegen
```

The codegen reads from `/tmp/decodo.ir.json` (or `inputs/decodo.ir.json`) or fetches from GCS automatically.

## Examples

See the [`examples/`](./examples/) directory for complete working examples:

- [`examples/web_scraping_api/google_search/`](./examples/web_scraping_api/google_search/) — synchronous Google Search scrape
- [`examples/web_scraping_api/amazon_product/`](./examples/web_scraping_api/amazon_product/) — Amazon product page scrape
- [`examples/web_scraping_api/universal/`](./examples/web_scraping_api/universal/) — universal URL scrape
- [`examples/web_scraping_api/async/google_search_async/`](./examples/web_scraping_api/async/google_search_async/) — async task with polling
- [`examples/web_scraping_api/batch/google_search_batch/`](./examples/web_scraping_api/batch/google_search_batch/) — batch task with polling

Run an example:

```bash
DECODO_TOKEN=your_token go run ./examples/web_scraping_api/google_search
```

## License

MIT
