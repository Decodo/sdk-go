# Decodo Go SDK

![Go](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8)
![License](https://img.shields.io/badge/license-MIT-blue)

[![](https://dcbadge.limes.pink/api/server/https://discord.gg/Ja8dqKgvbZ)](https://discord.gg/Ja8dqKgvbZ)

<p align="center">
<a href="https://dashboard.decodo.com/integrations?utm_source=github&utm_medium=social&utm_campaign=go_sdk"> <img src="https://github.com/user-attachments/assets/a1e52a9e-3da1-4081-b3c6-053aafb8f196"/></a>

The official Go SDK for the Decodo [Web Scraping API](http://decodo.com/scraping/web).

Build scraping workflows for search engines, eCommerce platforms, social media, AI tools, and more using the Decodo Web Scraping API.

- Typed, target-specific request parameters with editor autocomplete
- Sync, async, and batch scraping methods
- Runtime validation against the current Decodo target schema
- Typed error set for safer integrations
- One dependency beyond the standard library, Go 1.21+

# What is Decodo Go SDK?

Decodo Go SDK is the official Go SDK for the Decodo Web Scraping API. It provides a typed interface for interacting with Decodo targets like Google, Amazon, TikTok, Reddit, YouTube, ChatGPT, Perplexity, and more.

Instead of assembling HTTP requests and validating payloads by hand, you work with typed constructors and target-specific parameter structs directly in your editor.

# Why use the SDK?

- **Typing and autocomplete**. Each target has its own parameter struct, so your editor knows which fields it accepts.
- **Unified scraping interface**. Work with search engines, eCommerce platforms, social media, and AI tools through one SDK.
- **Async and batch workflows**. Create scraping tasks, poll statuses, and process batches at scale.
- **Typed errors**. Handle authentication, validation, timeout, and rate-limit failures explicitly.
- **Generated from the schema**. Target constants and parameter structs come from the Decodo schema, which is also used to validate requests at runtime.
- **Minimal setup**. One dependency for schema validation, and the standard library for everything else.

## Requirements

- Go 1.21 or later. Run `go version` to check your current version, or [install Go](https://go.dev/doc/install).

## Installation

```bash
go get github.com/decodo/sdk-go
```

## Quick start

Create a new project:

```sh
mkdir scraper
cd scraper

go mod init scraper
go get github.com/decodo/sdk-go

touch main.go
```

Get your Web Scraping API token from the [Decodo dashboard](https://dashboard.decodo.com/welcome). The token is the base64-encoded `user:password` value from the Basic Auth credentials shown in the dashboard.

Keep the token out of your source by exporting it:

```sh
export DECODO_TOKEN=your_token
```

Every target has a constructor that returns its parameter struct. Create the one you need, set its fields, and pass it to `Scrape`:

```go
// main.go
package main

import (
	"context"
	"fmt"
	"log"
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
	params.Query = decodo.Ptr("coffee shops")
	params.Geo = decodo.Ptr("United States")
	params.Parse = decodo.Ptr(true)

	result, err := client.WebScrapingAPI.Scrape(context.Background(), params)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Results[0].Content)
}
```

Run the program:

```sh
go run main.go
```

Each constructor pins its own target and returns a struct that carries only the fields that target supports, so you never pass a target name or assemble a generic parameter map. Your editor flags misspelled or unsupported fields as you type, and the SDK validates parameter values against the target schema before sending the request.

`NewGoogleSearchParams()` is the constructor for the `google_search` target. The naming follows a simple pattern covered in [Targets and parameter constructors](#targets-and-parameter-constructors).

<details>
<summary>Example response</summary>

`result.Results[0].Content` maps to `results[0].content` in the raw API response below.

```json
{
  "results": [
    {
      "content": {
        "results": {
          "last_visible_page": 10,
          "page": 1,
          "parse_status_code": 12000,
          "results": {
            "local_pack": [
              {
                "items": [
                  {
                    "address": "Rochester, NY",
                    "paid": false,
                    "pos": 1,
                    "rating": 4.9,
                    "rating_count": 1700,
                    "subtitle": "Coffee shop",
                    "title": "Albunn Coffee House"
                  },
                  {
                    "address": "Rochester, NY",
                    "paid": false,
                    "pos": 2,
                    "rating": 4.8,
                    "rating_count": 937,
                    "subtitle": "Coffee shop",
                    "title": "Layali Coffee House"
                  },
                  {
                    "address": "Ocean Township, NJ",
                    "paid": false,
                    "pos": 3,
                    "rating": 4.9,
                    "rating_count": 124,
                    "subtitle": "Coffee shop",
                    "title": "Ocean Brew Co."
                  }
                ],
                "pos_overall": 1
              }
            ],
            "organic": [
              {
                "desc": "For an alternate, local view, Eater has an interesting list of what it considers Philadelphia's 21 Essential Coffee Shops.",
                "pos": 1,
                "pos_overall": 2,
                "title": "Philadelphia",
                "url": "https://www.brian-coffee-spot.com/the-coffee-spot-guide-to/usa-canada/philadelphia/"
              }
            ],
            "search_information": {
              "query": "coffee shops",
              "total_results_count": 403000000
            }
          },
          "url": "https://www.google.com/search?q=coffee+shops&hl=en&gl=us"
        },
        "errors": [],
        "status_code": 12000,
        "task_id": "7463508496927950850"
      },
      "status_code": 200,
      "url": "https://www.google.com/search?q=coffee+shops&hl=en&gl=us",
      "task_id": "7463508496927950850",
      "created_at": "2026-05-22 08:38:10",
      "updated_at": "2026-05-22 08:38:13"
    }
  ]
}
```

</details>

## Setting optional parameters

Optional parameters are pointer fields, so an unset field is distinguishable from a zero value. Use `decodo.Ptr` to set them inline instead of declaring a temporary variable for each value:

```go
params.Query = decodo.Ptr("shoes")
params.Parse = decodo.Ptr(true)
params.PageFrom = decodo.Ptr(1)
```

## Configuration

```go
client := decodo.NewClient(decodo.Config{
	WebScrapingAPI: &decodo.WebScrapingAPIConfig{
		Token:             "<basic_auth_token>",
		IntegrationHeader: "my-app",
	},
	TimeoutMs: 120_000, // optional, request timeout in ms (default: 180000)
})
```

| Parameter | Description |
| --- | --- |
| `WebScrapingAPI.Token` | Web Scraping API basic auth token, the base64-encoded `user:password` string from the Decodo dashboard. Required |
| `WebScrapingAPI.IntegrationHeader` | Identifies the calling integration in requests (default: `sdk-go`) |
| `TimeoutMs` | Request timeout in milliseconds (default: 180000) |
| `Schema` | Target schema used for request validation (default: loaded automatically). See [Schema validation](#schema-validation) |

## Web Scraping API

Access the API via `client.WebScrapingAPI`.

The snippets below assume you have already constructed a client. See [Configuration](#configuration) for how to build one.

| Method | Returns |
| --- | --- |
| `Scrape(ctx, params)` | The scraping result, after waiting for it |
| `ScrapeAsync(ctx, params)` | A scraping task, immediately |
| `GetStatus(ctx, taskID)` | The current status of a task, as a `TaskStatus` constant |
| `GetResults(ctx, taskID)` | The results of a task, or `nil` if they are not ready |
| `ScrapeBatch(ctx, params)` | A batch task containing one query per input |

### Sync scrape

Waits for the scraping result before returning:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/decodo/sdk-go"
)

func main() {
	client := decodo.NewClient(decodo.Config{
		WebScrapingAPI: &decodo.WebScrapingAPIConfig{
			Token: os.Getenv("DECODO_TOKEN"),
		},
	})

	params := decodo.NewAmazonProductParams()
	params.Query = decodo.Ptr("B09H74FXNW")
	params.Parse = decodo.Ptr(true)

	// Run the scrape and wait for the result.
	result, err := client.WebScrapingAPI.Scrape(context.Background(), params)
	if err != nil {
		log.Fatal(err)
	}

	// Print the completed response.
	fmt.Println(result.Results[0].Content)
}
```

### Async scrape

Creates a scraping task and returns immediately, then you poll for status and results:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/decodo/sdk-go"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client := decodo.NewClient(decodo.Config{
		WebScrapingAPI: &decodo.WebScrapingAPIConfig{
			Token: os.Getenv("DECODO_TOKEN"),
		},
	})

	params := decodo.NewGoogleSearchParams()
	params.Query = decodo.Ptr("laptop reviews")
	params.Parse = decodo.Ptr(true)

	// Submit the scraping task and return immediately.
	task, err := client.WebScrapingAPI.ScrapeAsync(ctx, params)
	if err != nil {
		log.Fatal(err)
	}

	// Poll until the task finishes.
	for {
		status, err := client.WebScrapingAPI.GetStatus(ctx, task.ID)
		if err != nil {
			log.Fatal(err)
		}

		if status.Status == decodo.TaskStatusDone {
			break
		}

		if status.Status == decodo.TaskStatusFaulted {
			log.Fatalf("Scraping task %s failed.", task.ID)
		}

		time.Sleep(2 * time.Second)
	}

	// Retrieve and print the completed result.
	results, err := client.WebScrapingAPI.GetResults(ctx, task.ID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(results.Results[0].Content)
}
```

`GetStatus` returns one of `decodo.TaskStatusPending`, `decodo.TaskStatusDone`, or `decodo.TaskStatusFaulted`. `GetResults` returns `nil` until the results are available.

### Batch scrape

Every target also has a batch constructor, `New<Target>BatchParams()`, where `URL` and `Query` take a slice. Each value becomes its own task:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/decodo/sdk-go"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client := decodo.NewClient(decodo.Config{
		WebScrapingAPI: &decodo.WebScrapingAPIConfig{
			Token: os.Getenv("DECODO_TOKEN"),
		},
	})

	params := decodo.NewGoogleSearchBatchParams()
	params.Query = []string{"coffee", "tea", "juice"}
	params.Parse = decodo.Ptr(true)

	batch, err := client.WebScrapingAPI.ScrapeBatch(ctx, params)
	if err != nil {
		log.Fatal(err)
	}

	// Each input became its own task. Wait for each one and print its result.
	for _, query := range batch.Queries {
		for {
			status, err := client.WebScrapingAPI.GetStatus(ctx, query.ID)
			if err != nil {
				log.Fatal(err)
			}

			if status.Status == decodo.TaskStatusDone {
				break
			}

			if status.Status == decodo.TaskStatusFaulted {
				log.Fatalf("Scraping task %s failed.", query.ID)
			}

			time.Sleep(2 * time.Second)
		}

		result, err := client.WebScrapingAPI.GetResults(ctx, query.ID)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(result.Results[0].Content)
	}
}
```

`batch.ID` identifies the batch itself, and each entry in `batch.Queries` carries the task ID for one input. Poll those exactly as you would an async task. A batch runs against a single target, which the constructor pins for you.

Two details worth knowing:

- **`URL` and `Query` are the fields that take multiple values**. Every other field keeps the type it has on the regular params struct.
- **Some targets are async-only for now**. `walmart_product`, `target_product`, `tiktok_shop_product`, `chatgpt`, `gemini`, and `perplexity` aren't available through `ScrapeBatch` yet. Use `ScrapeAsync` for those.
- **Batch requests are limited to one per second**. How many inputs a single batch can carry depends on your plan's rate limit.
- **Batch requests skip schema validation**. The target schema describes single values, so arrays wouldn't pass it. Parameters are still typed at compile time, but a bad value only surfaces as an API error.

## Targets and parameter constructors

Every target has a matching constructor. The constructor name is the target name in Pascal case, prefixed with `New` and suffixed with `Params`:

```
google_search → NewGoogleSearchParams()
chatgpt       → NewChatgptParams()
airbnb        → NewAirbnbParams()
```

Two things follow from this:

- **You don't need to pass a target**. Each constructor pins its own target, so `NewGoogleSearchParams()` is complete on its own.
- **The target stays readable**. Each struct carries a `Target` field, and `GetTarget()` returns it as a string, so you can tell which target a set of parameters belongs to.
- **Batch has its own constructors**. `NewGoogleSearchBatchParams()` mirrors `NewGoogleSearchParams()` for use with `ScrapeBatch`. See [Batch scrape](#batch-scrape).

Each target takes one primary input field, `URL`, `Query`, `ProductID`, or `Prompt`, together with optional configuration such as `Parse`, `Markdown`, `Geo`, or `CallbackURL`. Which optional fields exist varies by target, so lean on autocomplete or the [parameters documentation](https://help.decodo.com/docs/web-scraping-api-parameters). The tables below show each target's primary input.

### Search engines

| Target | Description | Primary input | Constructor |
| --- | --- | --- | --- |
| `google_search` | Google Search results for a query | `Query` | `NewGoogleSearchParams()` |
| `google_maps` | Google Maps search results | `Query` | `NewGoogleMapsParams()` |
| `google_shopping_search` | Google Shopping search results | `Query` | `NewGoogleShoppingSearchParams()` |
| `google_shopping_product` | Google Shopping product page | `Query` | `NewGoogleShoppingProductParams()` |
| `google_suggest` | Google Autocomplete suggestions | `Query` | `NewGoogleSuggestParams()` |
| `google_lens` | Google Lens reverse image search | `Query` | `NewGoogleLensParams()` |
| `google_travel_hotels` | Google Travel hotel listings | `Query` | `NewGoogleTravelHotelsParams()` |
| `google_trends_explore` | Google Trends explore data | `Query` | `NewGoogleTrendsExploreParams()` |
| `google_ads` | Google Ads results for a query | `Query` | `NewGoogleAdsParams()` |
| `bing_search` | Bing Search results | `Query` | `NewBingSearchParams()` |
| `bing` | Raw Bing URL scraping | `URL` | `NewBingParams()` |

### eCommerce

| Target | Description | Primary input | Constructor |
| --- | --- | --- | --- |
| `amazon_product` | Amazon product detail page by ASIN | `Query` | `NewAmazonProductParams()` |
| `amazon_search` | Amazon search results | `Query` | `NewAmazonSearchParams()` |
| `amazon_pricing` | Amazon pricing and offers | `Query` | `NewAmazonPricingParams()` |
| `amazon_sellers` | Amazon seller listings | `Query` | `NewAmazonSellersParams()` |
| `amazon_bestsellers` | Amazon bestsellers by category | `Query` | `NewAmazonBestsellersParams()` |
| `walmart_product` | Walmart product page by product ID | `ProductID` | `NewWalmartProductParams()` |
| `walmart_search` | Walmart search results | `Query` | `NewWalmartSearchParams()` |
| `walmart` | Raw Walmart URL scraping | `URL` | `NewWalmartParams()` |
| `target_product` | Target.com product page by product ID | `ProductID` | `NewTargetProductParams()` |
| `target_search` | Target.com search results | `Query` | `NewTargetSearchParams()` |
| `target` | Raw Target.com URL scraping | `URL` | `NewTargetParams()` |
| `lowes_search` | Lowe's search results | `Query` | `NewLowesSearchParams()` |
| `ecommerce` | Generic eCommerce page with parser | `URL` | `NewEcommerceParams()` |

### Social media

| Target | Description | Primary input | Constructor |
| --- | --- | --- | --- |
| `reddit_post` | Reddit post by URL | `URL` | `NewRedditPostParams()` |
| `reddit_subreddit` | Reddit subreddit by URL | `URL` | `NewRedditSubredditParams()` |
| `reddit_user` | Reddit user profile by URL | `URL` | `NewRedditUserParams()` |
| `youtube_video` | YouTube video by ID | `Query` | `NewYoutubeVideoParams()` |
| `youtube_search` | YouTube search results | `Query` | `NewYoutubeSearchParams()` |
| `youtube_search_max` | YouTube search results (extended) | `Query` | `NewYoutubeSearchMaxParams()` |
| `youtube_metadata` | YouTube video metadata by ID | `Query` | `NewYoutubeMetadataParams()` |
| `youtube_transcript` | YouTube video transcript by ID | `Query` | `NewYoutubeTranscriptParams()` |
| `youtube_subtitles` | YouTube video subtitles by ID | `Query` | `NewYoutubeSubtitlesParams()` |
| `youtube_channel` | YouTube channel by handle or ID | `Query` | `NewYoutubeChannelParams()` |
| `tiktok_post` | TikTok post by URL | `URL` | `NewTiktokPostParams()` |
| `tiktok_shop_search` | TikTok Shop search results | `Query` | `NewTiktokShopSearchParams()` |
| `tiktok_shop_product` | TikTok Shop product page by product ID | `ProductID` | `NewTiktokShopProductParams()` |
| `tiktok` | Raw TikTok URL scraping | `URL` | `NewTiktokParams()` |
| `instagram_graphql_profile` | Instagram profile via GraphQL | `Query` | `NewInstagramGraphqlProfileParams()` |

### AI tools

| Target | Description | Primary input | Constructor |
| --- | --- | --- | --- |
| `chatgpt` | ChatGPT response for a prompt | `Prompt` | `NewChatgptParams()` |
| `perplexity` | Perplexity response for a prompt | `Prompt` | `NewPerplexityParams()` |
| `gemini` | Gemini response for a prompt | `Prompt` | `NewGeminiParams()` |
| `google_ai_mode` | Google AI Mode response | `Query` | `NewGoogleAiModeParams()` |

### Other

| Target | Description | Primary input | Constructor |
| --- | --- | --- | --- |
| `bbb` | Better Business Bureau listing by URL | `URL` | `NewBbbParams()` |
| `autotrader` | Autotrader listing by URL | `URL` | `NewAutotraderParams()` |
| `mobile` | Mobile.de listing by URL | `URL` | `NewMobileParams()` |
| `airbnb` | Airbnb listing by URL | `URL` | `NewAirbnbParams()` |
| `apple_app_store` | Apple App Store app by URL | `URL` | `NewAppleAppStoreParams()` |

### Universal scraping

| Target | Description | Primary input | Constructor |
| --- | --- | --- | --- |
| `universal` | Any URL via the universal scraper | `URL` | `NewUniversalParams()` |
| `google` | Raw Google URL scraping | `URL` | `NewGoogleParams()` |
| `amazon` | Raw Amazon URL scraping | `URL` | `NewAmazonParams()` |

> `universal_ecommerce` isn't listed above. Its parameter struct currently exposes only `Target` and `CallbackURL`, so reach for `ecommerce` for generic product pages, or `universal` for any URL.

For the full target list and parameter details, see the API documentation:

- [Target list](https://help.decodo.com/docs/web-scraping-api-targets)
- [Parameters](https://help.decodo.com/docs/web-scraping-api-parameters)

## Schema validation

Go checks that a field exists and holds the right type. Values are checked separately, at runtime, against the Decodo target schema.

When no `Schema` is configured, the client uses `decodo.SharedDefaultSchema`. It loads the latest schema on first use from the public `decodo-sdk-config` bucket on Google Cloud Storage, the same source `cmd/codegen` reads, and caches it locally at `~/.decodo/decodo.ir.json` for 24 hours. If the fetch fails, the SDK writes a warning to stderr and carries on with validation disabled. The load runs once per process, so it won't retry afterwards.

`LoadRemoteSchema` gives you control over that. `TTLMs` sets how long the cached copy stays valid, in milliseconds, where `0` means it never expires. `CachePath` overrides the cache location.

```go
schema, err := decodo.LoadRemoteSchema(decodo.RemoteSchemaOptions{
	TTLMs: 3_600_000, // re-fetch after an hour instead of the default 24
})
if err != nil {
	log.Fatal(err)
}

client := decodo.NewClient(decodo.Config{
	WebScrapingAPI: &decodo.WebScrapingAPIConfig{Token: token},
	Schema:         schema,
})
```

## Error handling

The SDK returns typed errors that map to API error codes:

```go
result, err := client.WebScrapingAPI.Scrape(ctx, params)
if err != nil {
	switch e := err.(type) {
	case *decodo.AuthenticationError:
		// Handle authentication failures.
		log.Fatalf("Invalid token (HTTP %d). Check the Basic authentication token in your dashboard.", e.StatusCode)
	case *decodo.RateLimitError:
		// Retry once after a short delay.
		time.Sleep(5 * time.Second)
		result, err = client.WebScrapingAPI.Scrape(ctx, params)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(result.Results[0].Content)
	case *decodo.ValidationError:
		// Handle invalid request parameters.
		fmt.Println("Payload rejected:", e.Errors)
	case *decodo.TimeoutError:
		// Handle request timeouts.
		fmt.Println("Timed out. Increase TimeoutMs or switch to ScrapeAsync for slow targets:", e.Msg)
	case *decodo.CancellationError:
		// The caller's context was cancelled.
		fmt.Println("Cancelled:", e.Msg)
	default:
		// Catch any other API errors.
		fmt.Printf("Request failed: %v\n", err)
	}
}
```

`AuthenticationError`, `RateLimitError`, and `ValidationError` embed `DecodoError` and implement `Unwrap`, so `errors.Is` and `errors.As` also work against `*decodo.DecodoError`.

| Error | Returned when | Useful fields |
| --- | --- | --- |
| `AuthenticationError` | The API returns `401` or `403` | `StatusCode`, `APIStatus`, `Message` |
| `RateLimitError` | The API returns `429` | `StatusCode`, `APIStatus`, `Message` |
| `ValidationError` | The API returns `422`, or `400` with validation errors | `Errors`, plus the `DecodoError` fields |
| `TimeoutError` | The request exceeds `TimeoutMs` | `Msg` |
| `CancellationError` | The caller's context is cancelled | `Msg` |
| `DecodoError` | Any other unsuccessful response. Embedded by the three HTTP errors above | `StatusCode`, `APIStatus`, `Message` |

Three things to keep in mind:

- **`TimeoutError` and `CancellationError` sit outside the `DecodoError` hierarchy**. They carry only `Msg`, so a check against `*decodo.DecodoError` won't match them. Handle them separately, as in the example above.
- **Typed parameters fail earlier than this**. An unknown field or a wrong type is a compile error, before any request is made. `ValidationError` covers requests that compile but are rejected by the schema or the API.
- **Schema validation can be skipped**. If the target schema can't be fetched, validation is skipped for that session and invalid values reach the API instead. See [Schema validation](#schema-validation).

## Examples

[`examples/web_scraping_api/`](./examples/web_scraping_api/) holds a complete, runnable program for most targets, one directory per target:

- [`google_search/`](./examples/web_scraping_api/google_search/), [`amazon_product/`](./examples/web_scraping_api/amazon_product/), [`youtube_search/`](./examples/web_scraping_api/youtube_search/), [`chatgpt/`](./examples/web_scraping_api/chatgpt/), and so on
- [`universal/`](./examples/web_scraping_api/universal/) for the universal scraper
- [`async/google_search_async/`](./examples/web_scraping_api/async/google_search_async/) for a task with polling
- [`batch/google_search_batch/`](./examples/web_scraping_api/batch/google_search_batch/) for a batch task with polling

Raw URL targets use a `_url` suffix, so `google` is [`google_url/`](./examples/web_scraping_api/google_url/) and `amazon` is [`amazon_url/`](./examples/web_scraping_api/amazon_url/). A few directories are named after the site rather than the target, such as [`mobile_de/`](./examples/web_scraping_api/mobile_de/) for `mobile` and [`instagram_profile/`](./examples/web_scraping_api/instagram_profile/) for `instagram_graphql_profile`.

Run any of them with:

```bash
DECODO_TOKEN=your_token go run ./examples/web_scraping_api/google_search
```

## Code generation

Target constants and parameter structs are generated from the Decodo IR (intermediate representation) schema. The generated files are committed, so installing the SDK gives you the typed constructors with no extra step. This section is for contributors updating the SDK to a newer schema.

Regenerate with:

```bash
go run ./cmd/codegen
```

This writes:

- `targets.go` – the `Target` enum and all target constants
- `generated_params.go` – typed parameter structs for each target
- `generated_parameters.go` – parameter metadata

The generator always looks for the latest `decodo-ir-v*.json` in the public `decodo-sdk-config` bucket on Google Cloud Storage, then caches it to `inputs/decodo.ir.json`, which is gitignored. If the fetch fails it falls back to that cache, or to `/tmp/decodo.ir.json`. Re-run it to pick up new targets or changed parameters.

## Related repositories

- [Decodo Python SDK](https://github.com/Decodo/sdk-python)
- [Decodo TypeScript SDK](https://github.com/Decodo/sdk-ts)
- [Web Scraping API](https://github.com/Decodo/Web-Scraping-API)
- [Decodo MCP Server](https://github.com/Decodo/mcp-server)
- [Decodo OpenClaw Skill](https://github.com/Decodo/decodo-openclaw-skill)

## Get started

Build scraping workflows with the Decodo Web Scraping API:

- [Start free plan](https://dashboard.decodo.com/)
- [Documentation](https://help.decodo.com/docs/introduction)
- [Discord](https://discord.gg/Ja8dqKgvbZ)

## License

Released under the [MIT License](https://github.com/Decodo/Decodo/blob/master/LICENSE).
