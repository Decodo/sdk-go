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

	params := decodo.NewBbbParams()
	params.URL = decodo.Ptr("https://www.bbb.org/search?find_text=Tree+Service&find_entity=&find_type=&find_loc=New+York%2C+NY&find_country=USA")

	res, err := client.WebScrapingAPI.Scrape(context.Background(), params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
}
