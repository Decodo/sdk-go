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
	params.Geo = decodo.Ptr("United States")
	params.Parse = decodo.Ptr(true)

	task, err := client.WebScrapingAPI.ScrapeAsync(context.Background(), params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Task created: %s\n", task.ID)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		fmt.Println("Polling for results...")
		results, err := client.WebScrapingAPI.GetResults(context.Background(), task.ID)
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
