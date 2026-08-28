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

	params := decodo.NewGoogleAdsParams()
	params.Query = decodo.Ptr("laptop")
	params.Headless = decodo.Ptr("html")
	params.Parse = decodo.Ptr(true)

	res, err := client.WebScrapingAPI.Scrape(context.Background(), params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
}
