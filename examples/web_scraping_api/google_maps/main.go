package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/decodo/sdk-golang"
)

func main() {
	client := decodo.NewClient(decodo.Config{
		WebScrapingAPI: &decodo.WebScrapingAPIConfig{Token: "<web_auth_token>"},
	})

	params := decodo.NewGoogleMapsParams()
	params.Query = decodo.Ptr("coffee shops brooklyn")
	params.Geo = decodo.Ptr("United States")
	params.Locale = decodo.Ptr("en")

	res, err := client.WebScrapingAPI.Scrape(context.Background(), params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
}
