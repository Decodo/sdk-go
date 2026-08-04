package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/decodo/sdk-golang"
)

func main() {
	client := decodo.NewClient(decodo.Config{
		WebScrapingAPI: &decodo.WebScrapingAPIConfig{Token: "<web_auth_token>"},
	})
	params := decodo.NewGoogleSearchParams()
	params.Query = decodo.Ptr("shoes")
	params.Geo = decodo.Ptr("United States")
	params.Parse = decodo.Ptr(true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	task, err := client.WebScrapingAPI.ScrapeAsync(ctx, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Task created: %s\n", task.ID)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "Timed out waiting for results")
			os.Exit(1)
		case <-ticker.C:
			fmt.Println("Polling for results...")
			status, err := client.WebScrapingAPI.GetStatus(ctx, task.ID)
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
			results, err := client.WebScrapingAPI.GetResults(ctx, task.ID)
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
