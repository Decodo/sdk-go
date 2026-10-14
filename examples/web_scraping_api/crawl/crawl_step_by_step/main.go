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

	job, err := client.WebScrapingAPI.StartCrawl(ctx, decodo.StartCrawlRequest{
		URL:   "https://decodo.com",
		Limit: decodo.Ptr(10),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Crawl started: %s\n", job.CrawlID)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "Timed out waiting for the crawl")
			os.Exit(1)
		case <-ticker.C:
			progress, err := client.WebScrapingAPI.GetCrawlStatus(ctx, job.CrawlID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%s: %d/%d pages\n", progress.Status, progress.Completed, progress.Discovered)
			if progress.Status == decodo.CrawlStatusPending {
				continue
			}

			pages, err := client.WebScrapingAPI.GetCrawlResults(ctx, job.CrawlID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			for _, page := range pages {
				fmt.Println(page.URL, len(page.Content))
			}
			return
		}
	}
}
