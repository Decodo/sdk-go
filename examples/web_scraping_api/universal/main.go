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
		WebScrapingAPI: &decodo.WebScrapingAPIConfig{Token: os.Getenv("DECODO_TOKEN")},
	})

	params := decodo.NewUniversalParams()
	params.URL = decodo.Ptr("https://www.example.com")
	params.Geo = decodo.Ptr("United States")
	params.Markdown = decodo.Ptr(true)

	res, err := client.WebScrapingAPI.Scrape(context.Background(), params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
}
