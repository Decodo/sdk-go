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

	params := decodo.NewGoogleLensParams()
	params.Query = decodo.Ptr("https://www.humanesociety.org/sites/default/files/2021-06/hamster-540188.jpg")
	params.Headless = decodo.Ptr("html")

	res, err := client.WebScrapingAPI.Scrape(context.Background(), params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
}
