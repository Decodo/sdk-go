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

	batch, err := client.WebScrapingAPI.ScrapeBatch(context.Background(), params)
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
	for range ticker.C {
		fmt.Println("Polling for results...")
		results, err := client.WebScrapingAPI.GetResults(context.Background(), firstID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if results != nil {
			out, _ := json.MarshalIndent(results.Results[0].Content, "", "  ")
			fmt.Println(string(out))
			return
		}
	}
}
