package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/decodo/sdk-go"
)

func main() {
	client := decodo.NewClient(decodo.Config{
		WebScrapingAPI: &decodo.WebScrapingAPIConfig{Token: "<web_auth_token>"},
	})
	params := decodo.NewGoogleSearchParams()
	params.Query = decodo.Ptr("shoes")
	params.Parse = decodo.Ptr(true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	batch, err := client.WebScrapingAPI.ScrapeBatch(ctx, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if len(batch.Queries) == 0 {
		fmt.Fprintln(os.Stderr, "No queries in batch response")
		os.Exit(1)
	}
	firstID := batch.Queries[0].ID
	fmt.Printf("Batch created, first task: %s\n", firstID)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "Timed out waiting for results")
			os.Exit(1)
		case <-ticker.C:
			fmt.Println("Polling for results...")
			status, err := client.WebScrapingAPI.GetStatus(ctx, firstID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting status: %v\n", err)
				os.Exit(1)
			}
			if status.Status == decodo.TaskStatusFaulted {
				fmt.Fprintln(os.Stderr, "Task faulted")
				os.Exit(1)
			}
			if status.Status != decodo.TaskStatusDone {
				continue
			}
			results, err := client.WebScrapingAPI.GetResults(ctx, firstID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting results: %v\n", err)
				os.Exit(1)
			}
			if results == nil || len(results.Results) == 0 {
				fmt.Fprintln(os.Stderr, "No results")
				os.Exit(1)
			}
			out, err := json.MarshalIndent(results.Results[0].Content, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling results: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(out))
			return
		}
	}
}
