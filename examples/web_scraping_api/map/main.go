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

	res, err := client.WebScrapingAPI.Map(context.Background(), decodo.MapSiteRequest{
		URL:   "https://decodo.com",
		Limit: decodo.Ptr(100),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(res.Results[0].Content.Links, "", "  ")
	fmt.Println(string(out))
}
