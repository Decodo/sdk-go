package decodo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// newTestWebScrapingAPI creates a WebScrapingAPI pointed at srv with no schema validation.
func newTestWebScrapingAPI(t *testing.T, srv *httptest.Server) *WebScrapingAPI {
	t.Helper()
	cfg := httpClientConfig{
		baseURL:           srv.URL,
		authType:          authTypeBasic,
		authToken:         "test-token",
		timeoutMs:         5_000,
		integrationHeader: "sdk-go-test",
	}
	return newWebScrapingAPI(newHTTPClient(cfg), nil)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func TestScrapeBatch_SendsQueryAsArray(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		writeJSON(w, map[string]any{
			"id":      12345,
			"queries": []any{},
		})
	}))
	defer srv.Close()

	api := newTestWebScrapingAPI(t, srv)
	params := NewGoogleSearchBatchParams()
	params.Query = []string{"coffee", "tea"}

	_, err := api.ScrapeBatch(context.Background(), params)
	if err != nil {
		t.Fatalf("ScrapeBatch error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}

	raw, ok := payload["query"]
	if !ok {
		t.Fatal("request body missing 'query' field")
	}
	queries, ok := raw.([]any)
	if !ok {
		t.Fatalf("'query' is %T, want []any (JSON array)", raw)
	}
	if len(queries) != 2 {
		t.Fatalf("query array len = %d, want 2", len(queries))
	}
	if queries[0] != "coffee" || queries[1] != "tea" {
		t.Fatalf("query = %v, want [coffee tea]", queries)
	}
}

func TestScrapeBatch_ParsesNumericBatchID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API returns a numeric id — must not panic or error.
		writeJSON(w, map[string]any{
			"id": 7495756599328274433,
			"queries": []any{
				map[string]any{"id": "abc123", "status": "pending"},
			},
		})
	}))
	defer srv.Close()

	api := newTestWebScrapingAPI(t, srv)
	params := NewGoogleSearchBatchParams()
	params.Query = []string{"coffee"}

	resp, err := api.ScrapeBatch(context.Background(), params)
	if err != nil {
		t.Fatalf("ScrapeBatch error: %v", err)
	}
	if resp.ID == 0 {
		t.Fatal("BatchResponse.ID not parsed (got 0)")
	}
	if len(resp.Queries) != 1 {
		t.Fatalf("Queries len = %d, want 1", len(resp.Queries))
	}
	if resp.Queries[0].ID != "abc123" {
		t.Fatalf("Queries[0].ID = %q, want abc123", resp.Queries[0].ID)
	}
}

func TestScrapeBatch_SkipsSchemaValidation(t *testing.T) {
	// Schema that records if GetRequestSchema is called — ScrapeBatch must not call it.
	rejectAll := &rejectAllSchema{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"id": 1, "queries": []any{}})
	}))
	defer srv.Close()

	cfg := httpClientConfig{
		baseURL:           srv.URL,
		authType:          authTypeBasic,
		authToken:         "test-token",
		timeoutMs:         5_000,
		integrationHeader: "sdk-go-test",
	}
	api := newWebScrapingAPI(newHTTPClient(cfg), rejectAll)

	params := NewGoogleSearchBatchParams()
	params.Query = []string{"coffee", "tea"}

	_, err := api.ScrapeBatch(context.Background(), params)
	if err != nil {
		t.Fatalf("ScrapeBatch should skip schema validation, got error: %v", err)
	}
	if rejectAll.called {
		t.Fatal("ScrapeBatch must not call schema validation")
	}
}

func TestScrapeBatch_ReturnsBothQueryTasks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"id": 99,
			"queries": []any{
				map[string]any{"id": "task-1", "query": "coffee", "status": "pending"},
				map[string]any{"id": "task-2", "query": "tea", "status": "pending"},
			},
		})
	}))
	defer srv.Close()

	api := newTestWebScrapingAPI(t, srv)
	params := NewGoogleSearchBatchParams()
	params.Query = []string{"coffee", "tea"}

	resp, err := api.ScrapeBatch(context.Background(), params)
	if err != nil {
		t.Fatalf("ScrapeBatch error: %v", err)
	}
	if len(resp.Queries) != 2 {
		t.Fatalf("Queries len = %d, want 2", len(resp.Queries))
	}
	if resp.Queries[0].ID != "task-1" || resp.Queries[1].ID != "task-2" {
		t.Fatalf("unexpected query IDs: %v", resp.Queries)
	}
}

// rejectAllSchema is a Schema that records if GetRequestSchema is ever called.
type rejectAllSchema struct {
	called bool
}

func (s *rejectAllSchema) GetRequestSchema(target string) *jsonschema.Schema {
	s.called = true
	return nil
}

func (s *rejectAllSchema) ListTargets() []string                          { return nil }
func (s *rejectAllSchema) GetTargetMeta(target string) *TargetInfo        { return nil }
func (s *rejectAllSchema) GetTargetParameterSchema(string) map[string]any { return nil }
func (s *rejectAllSchema) GetSharedParameters() map[string]any            { return nil }
func (s *rejectAllSchema) Version() string                                { return "" }
