package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/decodo/sdk-go"
)

func main() {
	client := decodo.NewClient(decodo.Config{
		WebScrapingAPI: &decodo.WebScrapingAPIConfig{APIKey: os.Getenv("DECODO_API_KEY")},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	pages, err := client.WebScrapingAPI.Crawl(ctx, decodo.StartCrawlRequest{
		URL:    "https://decodo.com",
		Limit:  decodo.Ptr(10),
		Output: decodo.Ptr("markdown"),
	}, decodo.WithPollInterval(5*time.Second))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, page := range pages {
		fmt.Println(page.URL, len(page.Content))
	}
}
