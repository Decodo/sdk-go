package decodo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// WebScrapingAPI provides methods for the Decodo web scraping API.
type WebScrapingAPI struct {
	http   *httpClient
	schema Schema
}

func newWebScrapingAPI(http *httpClient, schema Schema) *WebScrapingAPI {
	if schema == nil {
		schema = SharedDefaultSchema
	}
	return &WebScrapingAPI{http: http, schema: schema}
}

func (api *WebScrapingAPI) validate(params ScrapeRequest) error {
	compiledSchema := api.schema.GetRequestSchema(params.GetTarget())
	if compiledSchema == nil {
		return nil
	}

	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshaling params for validation: %w", err)
	}

	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("unmarshaling params for validation: %w", err)
	}

	if err := compiledSchema.Validate(v); err != nil {
		return &ValidationError{
			DecodoError: DecodoError{
				StatusCode: 0,
				Message:    err.Error(),
			},
		}
	}
	return nil
}

// Scrape performs a synchronous scrape request.
func (api *WebScrapingAPI) Scrape(ctx context.Context, params ScrapeRequest) (*SyncResponse, error) {
	if err := api.validate(params); err != nil {
		return nil, err
	}
	body, err := api.http.post(ctx, "/v2/scrape", params)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf("unexpected empty response from server")
	}
	var result SyncResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// ScrapeAsync creates an async scrape task.
func (api *WebScrapingAPI) ScrapeAsync(ctx context.Context, params ScrapeRequest) (*AsyncTaskResponse, error) {
	if err := api.validate(params); err != nil {
		return nil, err
	}
	body, err := api.http.post(ctx, "/v3/task", params)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf("unexpected empty response from server")
	}
	var result AsyncTaskResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// ScrapeBatch creates a batch scrape task.
func (api *WebScrapingAPI) ScrapeBatch(ctx context.Context, params ScrapeRequest) (*BatchResponse, error) {
	if err := api.validate(params); err != nil {
		return nil, err
	}
	body, err := api.http.post(ctx, "/v3/task/batch", params)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf("unexpected empty response from server")
	}
	var result BatchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// GetStatus retrieves the status of an async task.
func (api *WebScrapingAPI) GetStatus(ctx context.Context, taskID string) (*TaskMetadata, error) {
	body, err := api.http.get(ctx, "/v3/task/"+url.PathEscape(taskID), nil)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, fmt.Errorf("unexpected empty response from server")
	}
	var result TaskMetadata
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// GetResults retrieves the results of a completed async task. Returns nil if not yet available.
func (api *WebScrapingAPI) GetResults(ctx context.Context, taskID string) (*TaskResultsResponse, error) {
	body, err := api.http.get(ctx, "/v3/task/"+url.PathEscape(taskID)+"/results", nil)
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}
	var result TaskResultsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	if len(result.Results) == 0 {
		return nil, nil
	}
	return &result, nil
}
