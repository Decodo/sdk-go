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

	res, err := client.WebScrapingAPI.Search(context.Background(), decodo.RunSearchRequest{
		Query: "shoes",
		Geo:   decodo.Ptr("United States"),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
}
