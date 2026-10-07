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
		WebScrapingAPI: &decodo.WebScrapingAPIConfig{APIKey: os.Getenv("DECODO_API_KEY")},
	})

	params := decodo.NewMobileParams()
	params.URL = decodo.Ptr("https://suchen.mobile.de/fahrzeuge/search.html?dam=false&isSearchRequest=true&ref=quickSearch&s=Car&vc=Car")

	res, err := client.WebScrapingAPI.Scrape(context.Background(), params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
}
