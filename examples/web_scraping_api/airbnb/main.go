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

	params := decodo.NewAirbnbParams()
	params.URL = decodo.Ptr("https://www.airbnb.com/s/New-York-City--New-York--United-States/homes?refinement_paths%5B%5D=%2Fhomes&place_id=ChIJOwg_06VPwokRYv534QaPC8g&location_bb=QiOru8KTZn1CIegEwpSEhw%3D%3D&acp_id=6333bccb-ed88-460f-950a-ed9788913f4d&date_picker_type=calendar&search_type=autocomplete_click")

	res, err := client.WebScrapingAPI.Scrape(context.Background(), params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
}
